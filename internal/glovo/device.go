package glovo

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Glovo's gateway routes API calls on a set of glovo-* identity headers, and a
// call missing them is answered with 503 "no available server" rather than
// anything that reads like a client error. The device record holds the two
// pieces of that set worth keeping between runs: the perseus client id every
// call carries, and the device URN /oauth/token insists on (it rejects a login
// from an unregistered device with "device_urn_not_found").
type device struct {
	URN            string `json:"urn"`
	Fingerprint    string `json:"fingerprint"`
	ExpirationTime int64  `json:"fingerprints_expiration_time"`
	PerseusClient  string `json:"perseus_client_id"`
}

var deviceMu sync.Mutex

func (c *Client) devicePath() string {
	return filepath.Join(configDir(), "device.json")
}

func (c *Client) loadDevice() *device {
	if c.device != nil {
		return c.device
	}
	b, err := os.ReadFile(c.devicePath())
	if err != nil {
		return nil
	}
	var d device
	if json.Unmarshal(b, &d) != nil {
		return nil
	}
	c.device = &d
	return c.device
}

func (c *Client) saveDevice(d *device) {
	c.device = d
	_ = os.MkdirAll(filepath.Dir(c.devicePath()), 0o700)
	if b, err := json.MarshalIndent(d, "", "  "); err == nil {
		_ = os.WriteFile(c.devicePath(), b, 0o600)
	}
}

// clientID returns this install's stable perseus client id, minting one on
// first use. It never touches the network.
func (c *Client) clientID() string {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	d := c.loadDevice()
	if d == nil {
		d = &device{}
	}
	if d.PerseusClient == "" {
		d.PerseusClient = newUUID()
		c.saveDevice(d)
	}
	return d.PerseusClient
}

// ensureDevice returns a device URN registered with Glovo, registering one on
// first use and renewing it when its fingerprints expire. Only the login
// endpoints need it, so the registration call happens there rather than on
// every request. The fingerprint stands in for the browser fingerprint the web
// client computes with ThumbmarkJS; Glovo only stores it, so a random but
// stable value keeps one identity across runs the way a browser would.
func (c *Client) ensureDevice() (*device, error) {
	deviceMu.Lock()
	defer deviceMu.Unlock()
	d := c.loadDevice()
	switch {
	case d == nil || d.URN == "" || d.Fingerprint == "":
		fingerprint := newFingerprint()
		if d != nil && d.Fingerprint != "" {
			fingerprint = d.Fingerprint
		}
		return c.registerDevice(fingerprint, d)
	case d.ExpirationTime > 0 && time.Now().UnixMilli() > d.ExpirationTime:
		renewed, err := c.renewDevice(d)
		if err == nil {
			return renewed, nil
		}
		c.log("device renewal failed (%v), registering a new one", err)
		return c.registerDevice(d.Fingerprint, d)
	default:
		return d, nil
	}
}

func (c *Client) registerDevice(fingerprint string, prev *device) (*device, error) {
	var resp struct {
		URN            string `json:"urn"`
		ExpirationTime int64  `json:"fingerprintsExpirationTime"`
	}
	status, err := c.doJSON("POST", c.apiBase+"/identity/v4/devices", fingerprintBody(fingerprint), &resp)
	if err != nil {
		return nil, err
	}
	if (status != 200 && status != 201) || resp.URN == "" {
		return nil, fmt.Errorf("couldn't register a device with Glovo (http %d)", status)
	}
	d := &device{URN: resp.URN, Fingerprint: fingerprint, ExpirationTime: resp.ExpirationTime}
	if prev != nil {
		d.PerseusClient = prev.PerseusClient
	}
	if d.PerseusClient == "" {
		d.PerseusClient = newUUID()
	}
	c.saveDevice(d)
	return d, nil
}

func (c *Client) renewDevice(d *device) (*device, error) {
	var resp struct {
		URN            string `json:"urn"`
		ExpirationTime int64  `json:"fingerprintsExpirationTime"`
	}
	status, err := c.doJSON("PUT", c.apiBase+"/identity/v4/devices/"+d.URN, fingerprintBody(d.Fingerprint), &resp)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 || resp.URN == "" {
		return nil, fmt.Errorf("http %d", status)
	}
	renewed := &device{URN: resp.URN, Fingerprint: d.Fingerprint, ExpirationTime: resp.ExpirationTime, PerseusClient: d.PerseusClient}
	c.saveDevice(renewed)
	return renewed, nil
}

func fingerprintBody(fingerprint string) map[string]any {
	return map[string]any{"fingerprints": []map[string]string{{"provider": "ThumbmarkJS", "fingerprint": fingerprint}}}
}

// apiHeaders builds the identity, session and routing headers every
// api.glovoapp.com call carries. Callers merge their own headers on top.
func (c *Client) apiHeaders() map[string]string {
	h := baseHeaders()
	h["accept-language"] = "en"
	h["glovo-client-info"] = "web-customer-web-react/" + appVersion + " project:customer-web"
	h["glovo-request-id"] = newUUID()
	h["glovo-request-ttl"] = "7500"
	h["glovo-dynamic-session-id"] = c.sessionID
	h["glovo-perseus-session-id"] = c.sessionID
	h["glovo-perseus-session-timestamp"] = fmt.Sprint(c.sessionStart.UnixMilli())
	h["glovo-perseus-consent"] = "essential_functional"
	h["glovo-perseus-client-id"] = c.clientID()
	if d := c.loadDevice(); d != nil && d.URN != "" {
		h["glovo-device-urn"] = d.URN
	}
	return h
}

// authHeaders are the apiHeaders plus a registered device URN, which the login
// endpoints require.
func (c *Client) authHeaders() (map[string]string, error) {
	d, err := c.ensureDevice()
	if err != nil {
		return nil, err
	}
	h := c.apiHeaders()
	h["glovo-device-urn"] = d.URN
	return h, nil
}

func newFingerprint() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
