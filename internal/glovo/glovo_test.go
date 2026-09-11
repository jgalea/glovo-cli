package glovo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoJSONDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"placeId":"X","title":"T"}`))
	}))
	defer srv.Close()

	c := NewClient(nil)
	var out struct {
		PlaceID string `json:"placeId"`
		Title   string `json:"title"`
	}
	status, err := c.doJSON(http.MethodGet, srv.URL, nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 || out.PlaceID != "X" || out.Title != "T" {
		t.Fatalf("got status=%d out=%+v", status, out)
	}
}

func TestGetHTMLReturnsBody(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		_, _ = w.Write([]byte("<html>hi</html>"))
	}))
	defer srv.Close()
	c := NewClient(nil)
	body, err := c.getHTML(srv.URL, "glovo_delivery_address=x")
	if err != nil || body != "<html>hi</html>" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	if gotCookie != "glovo_delivery_address=x" {
		t.Fatalf("cookie = %q", gotCookie)
	}
}
