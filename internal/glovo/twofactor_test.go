package glovo

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// loginServer answers the device registration plus whatever the test wants for
// the token endpoints.
func loginServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/identity/v4/devices" {
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"urn":"glv:device:t","fingerprintsExpirationTime":` + futureMillis() + `}`))
			return
		}
		body, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected call to %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if r.Header.Get("glovo-device-urn") != "glv:device:t" {
			t.Errorf("%s sent without a device urn", r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
}

func TestLoginPasswordReturnsTwoFactorChallenge(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	srv := loginServer(t, map[string]string{
		"/oauth/token": `{"twoFactor":{"twoFactorToken":"2FT","expiresIn":600}}`,
	})
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	err := c.LoginPassword("someone@example.com", "pw")

	var challenge *TwoFactorRequired
	if !errors.As(err, &challenge) {
		t.Fatalf("err = %v, want a TwoFactorRequired", err)
	}
	if challenge.Token != "2FT" || challenge.ExpiresIn != 600 {
		t.Fatalf("challenge = %+v", challenge)
	}
	if c.LoggedIn() {
		t.Error("a pending challenge should not count as a session")
	}
}

func TestLoginPasswordStoresNestedAccessSession(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	token := fakeJWT(`"{\"userId\":4242,\"deviceId\":\"d\",\"grantType\":\"g\"}"`)
	srv := loginServer(t, map[string]string{
		"/oauth/token": `{"access":{"accessToken":"` + token + `","refreshToken":"RT"}}`,
	})
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	if err := c.LoginPassword("someone@example.com", "pw"); err != nil {
		t.Fatal(err)
	}
	s := c.loadAuth()
	if s == nil || s.AccessToken != token || s.RefreshToken != "RT" || s.CustomerID != 4242 {
		t.Fatalf("session = %+v", s)
	}
}

func TestValidateTwoFactorStoresSession(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	token := fakeJWT(`"{\"userId\":7,\"deviceId\":\"d\",\"grantType\":\"g\"}"`)
	var sent map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/identity/v4/devices" {
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"urn":"glv:device:t","fingerprintsExpirationTime":` + futureMillis() + `}`))
			return
		}
		if r.URL.Path != "/oauth/2fa/phone_verification/validate" {
			t.Errorf("unexpected call to %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&sent)
		_, _ = w.Write([]byte(`{"accessToken":"` + token + `","refreshToken":"RT"}`))
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	if err := c.ValidateTwoFactor("2FT", "123456"); err != nil {
		t.Fatal(err)
	}
	if sent["twoFactorToken"] != "2FT" || sent["code"] != "123456" {
		t.Fatalf("sent = %v", sent)
	}
	s := c.loadAuth()
	if s == nil || s.CustomerID != 7 || s.RefreshToken != "RT" {
		t.Fatalf("session = %+v", s)
	}
}
