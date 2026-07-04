package glovo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasketAddPostsRealShape(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet { // empty basket -> add takes the POST create path
			w.WriteHeader(204)
			return
		}
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Write([]byte(realBasketJSON))
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	p := CartProduct{ID: 42257009854, ExternalID: "p_1179695", StoreProductUUID: "7db2260e-0883-4154-b940-e4ac8188d43c"}
	b, err := c.BasketAdd(27121, 434279, 1, p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if b.Lines[0].Qty != 1 {
		t.Fatalf("qty = %d, want 1", b.Lines[0].Qty)
	}

	if wantSuffix := "/v1/authenticated/customers/42/baskets"; !hasSuffix(gotPath, wantSuffix) {
		t.Fatalf("path = %s, want suffix %s", gotPath, wantSuffix)
	}
	if got := gotBody["storeId"]; got != float64(27121) {
		t.Fatalf("storeId = %v, want 27121", got)
	}
	if got := gotBody["storeAddressId"]; got != float64(434279) {
		t.Fatalf("storeAddressId = %v, want 434279", got)
	}
	if got := gotBody["storeCategoryId"]; got != float64(1) {
		t.Fatalf("storeCategoryId = %v, want 1", got)
	}
	if got := gotBody["handlingStrategy"]; got != "DELIVERY" {
		t.Fatalf("handlingStrategy = %v, want DELIVERY", got)
	}
	products, ok := gotBody["products"].([]any)
	if !ok || len(products) != 1 {
		t.Fatalf("products = %+v", gotBody["products"])
	}
	prod := products[0].(map[string]any)
	qty, ok := prod["quantity"].(map[string]any)
	if !ok {
		t.Fatalf("quantity shape = %+v", prod["quantity"])
	}
	if qty["increments"] != float64(1) {
		t.Fatalf("quantity.increments = %v, want 1 (a delta, not an absolute qty)", qty["increments"])
	}
	ids, ok := prod["ids"].(map[string]any)
	if !ok {
		t.Fatalf("ids shape = %+v", prod["ids"])
	}
	if ids["id"] != "42257009854" || ids["legacyId"] != "42257009854" {
		t.Fatalf("ids.id/legacyId = %+v", ids)
	}
	if ids["externalId"] != "p_1179695" {
		t.Fatalf("ids.externalId = %v", ids["externalId"])
	}
	if ids["storeProductId"] != "7db2260e-0883-4154-b940-e4ac8188d43c" {
		t.Fatalf("ids.storeProductId = %v", ids["storeProductId"])
	}
	if _, ok := prod["customizations"].([]any); !ok {
		t.Fatalf("customizations missing or wrong shape: %+v", prod["customizations"])
	}
}

func TestBasketAddPutsWhenBasketExists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var putPath string
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet: // existing (non-empty) basket -> add takes the PUT complete-update path
			w.Write([]byte(realBasketJSON))
		case http.MethodPut:
			putPath = r.URL.Path
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &putBody)
			w.Write([]byte(realBasketJSON))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	// A product not already in the basket -> it gets appended, so the PUT
	// carries the existing line plus the new one.
	p := CartProduct{ID: 999888777, ExternalID: "p_new", StoreProductUUID: "uuid-new"}
	if _, err := c.BasketAdd(27121, 434279, 1, p, 1); err != nil {
		t.Fatal(err)
	}
	if !hasSuffix(putPath, "/baskets/42_27121_871cb5a8-abcd-4444-9999-000000000000/products") {
		t.Fatalf("PUT path = %s, want the .../baskets/{basketId}/products complete-update endpoint", putPath)
	}
	if putBody["customerId"] != "42" {
		t.Fatalf("customerId = %v, want the string \"42\"", putBody["customerId"])
	}
	if putBody["basketVersion"] != "5f672a97-73c7-4ba0-9c5d-ffbc76599247" {
		t.Fatalf("basketVersion = %v", putBody["basketVersion"])
	}
	products, ok := putBody["products"].([]any)
	if !ok || len(products) != 2 {
		t.Fatalf("PUT products = %+v, want 2 (existing line + appended new product)", putBody["products"])
	}
	newProd := products[1].(map[string]any)
	ids := newProd["ids"].(map[string]any)
	if ids["id"] != "999888777" || ids["externalId"] != "p_new" {
		t.Fatalf("appended product ids = %+v", ids)
	}
}

func TestBasketSetPatchesAbsoluteQuantity(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var patchBody map[string]any
	var patchPath string
	getN := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getN++
			w.Write([]byte(realBasketJSON))
		case http.MethodPatch:
			patchPath = r.URL.Path
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &patchBody)
			w.WriteHeader(204)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	p := CartProduct{ID: 42257009854}
	if _, err := c.BasketSet(27121, 434279, 1, p, 3); err != nil {
		t.Fatal(err)
	}
	if getN != 2 {
		t.Fatalf("GET called %d times, want 2 (one to find basketProductId, one to return the fresh basket)", getN)
	}
	wantPath := "/v1/authenticated/customers/42/baskets/42_27121_871cb5a8-abcd-4444-9999-000000000000/products/quantity"
	if !hasSuffix(patchPath, wantPath) {
		t.Fatalf("PATCH path = %s, want suffix %s", patchPath, wantPath)
	}
	if patchBody["basketVersion"] != "5f672a97-73c7-4ba0-9c5d-ffbc76599247" {
		t.Fatalf("basketVersion = %v", patchBody["basketVersion"])
	}
	if patchBody["handlingStrategy"] != "DELIVERY" {
		t.Fatalf("handlingStrategy = %v", patchBody["handlingStrategy"])
	}
	products, ok := patchBody["products"].([]any)
	if !ok || len(products) != 1 {
		t.Fatalf("products = %+v", patchBody["products"])
	}
	prod := products[0].(map[string]any)
	if prod["basketProductId"] != "SP_7db2260e-0883-4154-b940-e4ac8188d43c_1" {
		t.Fatalf("basketProductId = %v", prod["basketProductId"])
	}
	if prod["quantity"] != float64(3) {
		t.Fatalf("quantity = %v, want an absolute 3", prod["quantity"])
	}
}

func TestBasketSetZeroRemovesLine(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var patchBody map[string]any
	getN := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getN++
			if getN == 1 {
				w.Write([]byte(realBasketJSON))
				return
			}
			w.WriteHeader(204)
		case http.MethodPatch:
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &patchBody)
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	p := CartProduct{ID: 42257009854}
	b, err := c.BasketSet(27121, 434279, 1, p, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Lines) != 0 {
		t.Fatalf("got %d lines, want 0 after remove", len(b.Lines))
	}
	products := patchBody["products"].([]any)
	prod := products[0].(map[string]any)
	if prod["quantity"] != float64(0) {
		t.Fatalf("quantity = %v, want 0 (negative qty clamps to remove)", prod["quantity"])
	}
}

func TestBasketSetFallsBackToAddWhenProductMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var postBody map[string]any
	sawPost := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(204)
		case http.MethodPost:
			sawPost = true
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &postBody)
			w.Write([]byte(realBasketJSON))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	p := CartProduct{ID: 42257009854, ExternalID: "p_1179695", StoreProductUUID: "7db2260e-0883-4154-b940-e4ac8188d43c"}
	if _, err := c.BasketSet(27121, 434279, 1, p, 2); err != nil {
		t.Fatal(err)
	}
	if !sawPost {
		t.Fatal("expected BasketSet to fall back to POST /baskets when the product isn't already in the basket")
	}
	products := postBody["products"].([]any)
	prod := products[0].(map[string]any)
	qty := prod["quantity"].(map[string]any)
	if qty["increments"] != float64(2) {
		t.Fatalf("increments = %v, want 2 (the absolute qty used as the delta for a fresh line)", qty["increments"])
	}
}

func TestBasketSetOnEmptyBasketUsesStoreAddress(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var postBody map[string]any
	sawPost := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(204)
		case http.MethodPost:
			sawPost = true
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &postBody)
			w.Write([]byte(realBasketJSON))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	p := CartProduct{ID: 42257009854, ExternalID: "p_1179695", StoreProductUUID: "7db2260e-0883-4154-b940-e4ac8188d43c"}
	if _, err := c.BasketSet(27121, 434279, 1, p, 2); err != nil {
		t.Fatal(err)
	}
	if !sawPost {
		t.Fatal("expected BasketSet to fall back to POST /baskets when the basket is empty")
	}
	if got := postBody["storeAddressId"]; got != float64(434279) {
		t.Fatalf("storeAddressId = %v, want 434279 (not 0, from the empty GET's zero-value basket)", got)
	}
	if got := postBody["storeCategoryId"]; got != float64(1) {
		t.Fatalf("storeCategoryId = %v, want 1", got)
	}
	products := postBody["products"].([]any)
	prod := products[0].(map[string]any)
	qty := prod["quantity"].(map[string]any)
	if qty["increments"] != float64(2) {
		t.Fatalf("increments = %v, want 2", qty["increments"])
	}
}

func TestBasketClearPatchesAllLinesToZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	var patchBody map[string]any
	getN := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getN++
			if getN == 1 {
				w.Write([]byte(realBasketJSON))
				return
			}
			w.WriteHeader(204)
		case http.MethodPatch:
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &patchBody)
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	b, err := c.BasketClear(27121)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Lines) != 0 {
		t.Fatalf("got %d lines, want 0 after clear", len(b.Lines))
	}
	products := patchBody["products"].([]any)
	if len(products) != 1 {
		t.Fatalf("patched %d lines, want 1 (matching the one line in the fixture)", len(products))
	}
	prod := products[0].(map[string]any)
	if prod["basketProductId"] != "SP_7db2260e-0883-4154-b940-e4ac8188d43c_1" {
		t.Fatalf("basketProductId = %v", prod["basketProductId"])
	}
	if prod["quantity"] != float64(0) {
		t.Fatalf("quantity = %v, want 0", prod["quantity"])
	}
}

func TestBasketClearNoOpWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	patchCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patchCalled = true
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()
	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 42})

	if _, err := c.BasketClear(27121); err != nil {
		t.Fatal(err)
	}
	if patchCalled {
		t.Fatal("BasketClear should be a no-op (no PATCH) when the basket is already empty")
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
