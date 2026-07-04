package glovo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseStoreWall(t *testing.T) {
	body, err := os.ReadFile("testdata/store_wall_min.json")
	if err != nil {
		t.Fatal(err)
	}
	stores := parseStoreWall(body)
	if len(stores) != 2 {
		t.Fatalf("got %d stores, want 2", len(stores))
	}
	s := stores[0]
	if s.ID != 581058 {
		t.Fatalf("ID = %d, want 581058 (numeric shopId)", s.ID)
	}
	if s.Name != "Tacos Galos" || s.Slug != "tacos-galos-barcelona" {
		t.Fatalf("name/slug = %q / %q", s.Name, s.Slug)
	}
	if s.Rating != 95 {
		t.Fatalf("rating = %d, want 95 (from \"95%%\")", s.Rating)
	}
	if s.ETAMinutesLow != 15 || s.ETAMinutesHigh != 25 {
		t.Fatalf("eta = %d-%d", s.ETAMinutesLow, s.ETAMinutesHigh)
	}
	if s.DeliveryFee != 1.49 || s.Currency != "EUR" {
		t.Fatalf("fee = %v %s", s.DeliveryFee, s.Currency)
	}
	if !s.Open {
		t.Fatalf("store0 should be open")
	}
	if stores[1].Open {
		t.Fatalf("store1 should be closed")
	}
}

func TestParseStoreWallGarbageReturnsNil(t *testing.T) {
	got := parseStoreWall([]byte("not json"))
	if got != nil {
		t.Fatalf("got %v, want nil for unparseable envelope", got)
	}
}

func TestParseStoreWallEmptyElementsReturnsNonNilEmpty(t *testing.T) {
	body := []byte(`{"data":{"body":{"data":{"elements":[]}}}}`)
	got := parseStoreWall(body)
	if got == nil {
		t.Fatal("got nil, want non-nil empty slice for a valid envelope with zero elements")
	}
	if len(got) != 0 {
		t.Fatalf("got %d stores, want 0", len(got))
	}
}

func TestSearchSendsAuthAndLocationHeaders(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLOVO_CONFIG_DIR", dir)
	body, _ := os.ReadFile("testdata/store_wall_min.json")

	var gotAuth, gotLat, gotApiVer, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotLat = r.Header.Get("glovo-delivery-location-latitude")
		gotApiVer = r.Header.Get("glovo-api-version")
		gotQuery = r.URL.Query().Get("searchQuery")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.saveAuth(&authSession{AccessToken: "AT", RefreshToken: "RT", CustomerID: 1})

	stores, err := c.Search("pizza", 41.3874, 2.1686, "BCN", "ES")
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 2 {
		t.Fatalf("got %d stores", len(stores))
	}
	if gotAuth != "Bearer AT" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotLat != "41.3874" {
		t.Fatalf("lat header = %q", gotLat)
	}
	if gotApiVer != "14" {
		t.Fatalf("api-version = %q", gotApiVer)
	}
	if gotQuery != "pizza" {
		t.Fatalf("searchQuery = %q", gotQuery)
	}
}
