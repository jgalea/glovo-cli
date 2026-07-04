package glovo

import (
	"testing"
)

func TestAuthRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)
	if c.LoggedIn() {
		t.Fatal("should not be logged in on empty dir")
	}
	c.saveAuth(&authSession{AccessToken: "a", RefreshToken: "r", CustomerID: 42})
	c2 := NewClient(nil)
	got := c2.loadAuth()
	if got == nil || got.CustomerID != 42 || got.AccessToken != "a" {
		t.Fatalf("loaded = %+v", got)
	}
	if !c2.LoggedIn() {
		t.Fatal("should be logged in")
	}
}
