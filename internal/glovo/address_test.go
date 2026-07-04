package glovo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeocode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v3/addresslookup/pub/coordinates") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"placeId":"ChIJ","title":"Passeig de Gràcia, 1","subtitle":"Barcelona","latitude":41.388,"longitude":2.169}`))
	}))
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	a, err := c.Geocode(41.388, 2.169)
	if err != nil {
		t.Fatal(err)
	}
	if a.PlaceID != "ChIJ" || a.Label != "Passeig de Gràcia, 1" || a.Latitude != 41.388 {
		t.Fatalf("got %+v", a)
	}
}
