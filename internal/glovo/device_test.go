package glovo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnsureDeviceRegistersOnceAndPersists(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	var posts int
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"urn":"glv:device:abc","fingerprintsExpirationTime":` + futureMillis() + `}`))
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	d, err := c.ensureDevice()
	if err != nil || d.URN != "glv:device:abc" {
		t.Fatalf("device = %+v err = %v", d, err)
	}
	fps, _ := gotBody["fingerprints"].([]any)
	if len(fps) != 1 {
		t.Fatalf("body = %v", gotBody)
	}
	if fp, _ := fps[0].(map[string]any); fp["provider"] != "ThumbmarkJS" || fp["fingerprint"] == "" {
		t.Fatalf("fingerprint entry = %v", fps[0])
	}

	// A second client on the same config dir reuses the stored registration.
	fresh := NewClient(nil)
	fresh.apiBase = srv.URL
	if d, err := fresh.ensureDevice(); err != nil || d.URN != "glv:device:abc" {
		t.Fatalf("reuse: device = %+v err = %v", d, err)
	}
	if posts != 1 {
		t.Fatalf("registered %d times, want 1", posts)
	}
}

func TestEnsureDeviceRenewsExpiredFingerprints(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	var puts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
			if r.URL.Path != "/identity/v4/devices/glv:device:old" {
				t.Errorf("renewal path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"urn":"glv:device:old","fingerprintsExpirationTime":` + futureMillis() + `}`))
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveDevice(&device{URN: "glv:device:old", Fingerprint: "ff", ExpirationTime: 1, PerseusClient: "pc"})
	c.device = nil

	d, err := c.ensureDevice()
	if err != nil || d.URN != "glv:device:old" || puts != 1 {
		t.Fatalf("device = %+v err = %v puts = %d", d, err, puts)
	}
	if d.PerseusClient != "pc" {
		t.Fatalf("renewal dropped the perseus client id: %+v", d)
	}
}

func TestAPIHeadersCarryTheRoutingSet(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	c := NewClient(nil)
	h := c.apiHeaders()
	for _, k := range []string{
		"glovo-api-version", "glovo-app-platform", "glovo-app-type", "glovo-app-version",
		"glovo-app-context", "glovo-app-development-state", "glovo-language-code",
		"glovo-client-info", "glovo-request-id", "glovo-request-ttl",
		"glovo-dynamic-session-id", "glovo-perseus-client-id", "glovo-perseus-consent",
		"glovo-perseus-session-id", "glovo-perseus-session-timestamp",
	} {
		if h[k] == "" {
			t.Errorf("missing header %s", k)
		}
	}
	// No device is registered yet, so no URN goes out; ordinary calls don't need one.
	if _, ok := h["glovo-device-urn"]; ok {
		t.Error("device urn sent before any registration")
	}
	// The client id is stable across clients, the request id is not.
	other := NewClient(nil).apiHeaders()
	if other["glovo-perseus-client-id"] != h["glovo-perseus-client-id"] {
		t.Error("perseus client id should persist across runs")
	}
	if other["glovo-request-id"] == h["glovo-request-id"] {
		t.Error("request id should be per request")
	}
}

func futureMillis() string {
	return jsonNumber(time.Now().Add(24 * time.Hour).UnixMilli())
}

func jsonNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
