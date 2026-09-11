package glovo

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func b64url(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func fakeJWT(payloadClaims string) string {
	header := b64url(`{"alg":"none"}`)
	claims := b64url(`{"role":"ACCESS","payload":` + payloadClaims + `}`)
	return header + "." + claims + ".x"
}

func TestLoginAccessTokenDecodesCustomerId(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)

	token := fakeJWT(`"{\"userId\":42,\"deviceId\":\"d\",\"grantType\":\"g\"}"`)
	if err := c.LoginAccessToken(token, 0); err != nil {
		t.Fatal(err)
	}

	fresh := NewClient(nil)
	s := fresh.loadAuth()
	if s == nil || s.AccessToken != token || s.CustomerID != 42 {
		t.Fatalf("session = %+v", s)
	}
}

func TestLoginAccessTokenExplicitCustomerId(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)

	if err := c.LoginAccessToken("not-a-jwt", 42); err != nil {
		t.Fatal(err)
	}

	fresh := NewClient(nil)
	s := fresh.loadAuth()
	if s == nil || s.AccessToken != "not-a-jwt" || s.CustomerID != 42 {
		t.Fatalf("session = %+v", s)
	}
}

func TestLoginAccessTokenBadTokenNoIdErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)
	c.saveAuth(&authSession{AccessToken: "GOOD", CustomerID: 7})

	if err := c.LoginAccessToken("not-a-jwt", 0); err == nil {
		t.Fatal("expected error for undecodable token with no explicit customer id")
	}

	fresh := NewClient(nil)
	s := fresh.loadAuth()
	if s == nil || s.AccessToken != "GOOD" || s.CustomerID != 7 {
		t.Fatalf("prior session was clobbered: %+v", s)
	}
}

func TestLoginToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	// The refresh response carries no customer id: it comes from the access
	// token's own payload claim.
	token := fakeJWT(`"{\"userId\":42,\"deviceId\":\"d\",\"grantType\":\"g\"}"`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"` + token + `","refreshToken":"RT2"}`))
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	if err := c.LoginToken("RT"); err != nil {
		t.Fatal(err)
	}
	s := c.loadAuth()
	if s == nil || s.AccessToken != token || s.CustomerID != 42 || s.RefreshToken != "RT2" {
		t.Fatalf("session = %+v", s)
	}
}

func TestLoginTokenFailureKeepsPriorSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)
	c.saveAuth(&authSession{AccessToken: "GOOD", RefreshToken: "GOOD_RT", CustomerID: 42})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c.apiBase = srv.URL

	if err := c.LoginToken("bad-token"); err == nil {
		t.Fatal("expected LoginToken to fail against a 500 exchange response")
	}

	fresh := NewClient(nil)
	s := fresh.loadAuth()
	if s == nil || s.AccessToken != "GOOD" || s.CustomerID != 42 {
		t.Fatalf("prior session was clobbered: %+v", s)
	}
}
