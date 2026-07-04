package glovo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// realBasketJSON mirrors the shape captured in CAPTURE-basket.md: nested
// ids/quantity/price objects per product, qty in quantity.items, unit price
// in price.unitaryTotalPrice.major. The basket-level pricing tail was not
// captured, so there's no top-level "pricing" object here.
const realBasketJSON = `{
  "basketId": "42_27121_871cb5a8-abcd-4444-9999-000000000000",
  "basketVersion": "5f672a97-73c7-4ba0-9c5d-ffbc76599247",
  "handlingStrategy": "DELIVERY",
  "customerId": 42,
  "storeId": 27121,
  "storeAddressId": 434279,
  "storeCategoryId": 1,
  "products": [
    {
      "ids": {
        "basketProductId": "SP_7db2260e-0883-4154-b940-e4ac8188d43c_1",
        "legacyId": "42257009854",
        "id": "42257009854",
        "externalId": "p_1179695",
        "storeProductId": "7db2260e-0883-4154-b940-e4ac8188d43c"
      },
      "quantity": {
        "increments": 1,
        "incrementsLimit": 10,
        "items": 1,
        "itemsLimit": 10,
        "limitType": "UNIT"
      },
      "price": {
        "totalFormatted": "10.78 €",
        "unitaryBasePrice": {"minor": 1540, "major": 15.40, "formatted": "15.40 €"},
        "unitaryTotalPrice": {"minor": 1078, "major": 10.78, "formatted": "10.78 €"},
        "productTotalDiscount": 4.62
      }
    }
  ]
}`

func TestBasketGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/v1/authenticated/customers/42/baskets/stores/27121"
		if !strings.HasSuffix(r.URL.Path, want) {
			t.Errorf("path = %s", r.URL.Path)
		}
		fmt.Fprint(w, realBasketJSON)
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	b, err := c.BasketGet(27121)
	if err != nil {
		t.Fatal(err)
	}
	if b.StoreID != 27121 {
		t.Fatalf("StoreID = %d, want 27121", b.StoreID)
	}
	if b.StoreAddressID != 434279 {
		t.Fatalf("StoreAddressID = %d, want 434279", b.StoreAddressID)
	}
	if b.BasketID != "42_27121_871cb5a8-abcd-4444-9999-000000000000" {
		t.Fatalf("BasketID = %q", b.BasketID)
	}
	if b.BasketVersion != "5f672a97-73c7-4ba0-9c5d-ffbc76599247" {
		t.Fatalf("BasketVersion = %q", b.BasketVersion)
	}
	if len(b.Lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(b.Lines))
	}
	l := b.Lines[0]
	if l.ProductID != 42257009854 {
		t.Fatalf("ProductID = %d, want 42257009854", l.ProductID)
	}
	if l.Qty != 1 {
		t.Fatalf("Qty = %d, want 1 (from quantity.items)", l.Qty)
	}
	if l.UnitPrice != 10.78 {
		t.Fatalf("UnitPrice = %v, want 10.78 (from price.unitaryTotalPrice.major)", l.UnitPrice)
	}
	if l.BasketProductID != "SP_7db2260e-0883-4154-b940-e4ac8188d43c_1" {
		t.Fatalf("BasketProductID = %q", l.BasketProductID)
	}
	// Basket-level total wasn't captured; Total/Subtotal are a qty*unitPrice
	// line-sum fallback until a fuller capture confirms the real fields.
	if b.Subtotal != 10.78 {
		t.Fatalf("Subtotal = %v, want 10.78 (line-sum fallback)", b.Subtotal)
	}
	if b.Total != 10.78 {
		t.Fatalf("Total = %v, want 10.78 (line-sum fallback)", b.Total)
	}
	if b.Currency != "EUR" {
		t.Fatalf("Currency = %q, want EUR (default; not present on the product price object)", b.Currency)
	}
}

func TestBasketGetEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	b, err := c.BasketGet(27121)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Lines) != 0 {
		t.Fatalf("got %d lines, want 0 for an empty (204) basket", len(b.Lines))
	}
}
