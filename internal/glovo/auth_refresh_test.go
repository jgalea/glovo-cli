package glovo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoAuthedRefreshesOn401(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/refresh":
			_, _ = w.Write([]byte(`{"accessToken":"NEW","refreshToken":"r2"}`))
		default:
			calls++
			if r.Header.Get("Authorization") == "Bearer NEW" {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"ok":true}`))
				return
			}
			w.WriteHeader(401)
		}
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "OLD", RefreshToken: "r", CustomerID: 1})

	var out struct {
		OK bool `json:"ok"`
	}
	status, err := c.doAuthedJSON("GET", srv.URL+"/x", nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 || !out.OK {
		t.Fatalf("status=%d out=%+v", status, out)
	}
	if c.loadAuth().AccessToken != "NEW" {
		t.Fatal("token not refreshed in store")
	}
	if calls != 2 {
		t.Fatalf("expected exactly 2 resource-endpoint calls (initial 401 + retry with new token), got %d", calls)
	}
}

// TestDoAuthedStillUnauthorizedAfterRefresh covers the case where refresh
// succeeds (returns a new token) but the resource endpoint still rejects
// the retried request with 401 — e.g. the session was revoked server-side.
// doAuthedJSON must never surface a raw (401, nil) to the caller; it must
// return a clear "session expired" error instead.
func TestDoAuthedStillUnauthorizedAfterRefresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)

	var resourceCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/refresh":
			_, _ = w.Write([]byte(`{"accessToken":"NEW","refreshToken":"r2"}`))
		default:
			resourceCalls++
			w.WriteHeader(401)
		}
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "OLD", RefreshToken: "r", CustomerID: 1})

	var out struct {
		OK bool `json:"ok"`
	}
	status, err := c.doAuthedJSON("GET", srv.URL+"/x", nil, &out)
	if err == nil {
		t.Fatalf("expected an error, got nil (status=%d)", status)
	}
	if status == 401 && err == nil {
		t.Fatal("must never return a raw 401 with a nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "login") {
		t.Fatalf("expected error to mention logging in, got: %v", err)
	}
	// Exactly one refresh call plus at most two resource-endpoint attempts
	// (initial + single retry). Guards against an accidental retry loop.
	if resourceCalls > 2 {
		t.Fatalf("expected at most 2 resource-endpoint calls (initial + single retry), got %d", resourceCalls)
	}
}
