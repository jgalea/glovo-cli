package glovo

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlaceOrderNotCaptured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	c := NewClient(nil)
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 1})
	_, err := c.PlaceOrder(27121)
	if !errors.Is(err, ErrCheckoutNotCaptured) {
		t.Fatalf("err = %v, want ErrCheckoutNotCaptured", err)
	}
}

func TestOrderDryRunReturnsBasket(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(realBasketJSON))
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})
	b, err := c.OrderDryRun(27121)
	if err != nil || b.Total != 10.78 {
		t.Fatalf("b=%+v err=%v", b, err)
	}
}
