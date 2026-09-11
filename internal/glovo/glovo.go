package glovo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	apiBase = "https://api.glovoapp.com"
	webBase = "https://glovoapp.com"
	// appVersion tracks the web client's build. The gateway routes on it, so a
	// stale value is one of the ways a call ends up with no upstream.
	appVersion = "v1.2674.0"
	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
	maxBody    = 8 << 20
)

type Client struct {
	http         *http.Client
	apiBase      string
	webBase      string
	logf         func(string, ...any)
	auth         *authSession      // populated lazily by auth.go
	device       *device           // populated lazily by device.go
	loc          map[string]string // glovo-location-* / delivery-location headers for authed calls
	sessionID    string
	sessionStart time.Time
}

// SetLocation records the delivery location whose glovo-* headers the
// authenticated basket endpoints require (they reject a POST with "City code
// is required" otherwise).
func (c *Client) SetLocation(cityCode, countryCode string, lat, lng float64) {
	c.loc = locationHeaders(lat, lng, cityCode, countryCode)
}

func NewClient(logf func(string, ...any)) *Client {
	return &Client{
		http:         &http.Client{Timeout: 30 * time.Second},
		apiBase:      apiBase,
		webBase:      webBase,
		logf:         logf,
		sessionID:    newUUID(),
		sessionStart: time.Now(),
	}
}

func configDir() string {
	if dir := os.Getenv("GLOVO_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".glovo")
}

func (c *Client) log(format string, args ...any) {
	if c.logf != nil {
		c.logf(format, args...)
	}
}

// baseHeaders are the app-identity headers every api.glovoapp.com call
// carries, including the device registration that has no device id yet.
func baseHeaders() map[string]string {
	return map[string]string{
		"glovo-api-version":           "14",
		"glovo-app-platform":          "web",
		"glovo-app-type":              "customer",
		"glovo-app-version":           appVersion,
		"glovo-app-context":           "web",
		"glovo-app-development-state": "prod",
		"glovo-language-code":         "en",
	}
}

// getHTML fetches a server-rendered page. Glovo renders store content only for
// a request that carries a delivery address, so cookie carries the
// glovo_delivery_address the page should be rendered against.
func (c *Client) getHTML(url, cookie string) (string, error) {
	body, status, err := c.getPage(url, cookie, false)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("GET %s: http %d", url, status)
	}
	return body, nil
}

// getPage is getHTML without treating a non-2xx as an error, for callers that
// expect misses (a city slug that has no Glovo page 404s). freshConn closes the
// connection afterwards, so a retry is routed again rather than pinned to the
// server that just answered badly.
func (c *Client) getPage(url, cookie string, freshConn bool) (string, int, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Close = freshConn
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(b), resp.StatusCode, nil
}

// doJSON sends an optional JSON body and decodes a JSON response into out (if non-nil).
func (c *Client) doJSON(method, url string, body any, out any) (int, error) {
	return c.doJSONH(method, url, baseHeaders(), body, out)
}

func (c *Client) doJSONH(method, url string, extra map[string]string, body, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rdr)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", webBase)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if os.Getenv("GLOVO_DEBUG") != "" && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		fmt.Fprintf(os.Stderr, "glovo: %s %s -> %d\n%s\n", method, url, resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode %s: %w", url, err)
		}
	}
	return resp.StatusCode, nil
}
