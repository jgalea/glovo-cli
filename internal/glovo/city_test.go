package glovo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Lisboa":                "lisboa",
		"Barcelona":             "barcelona",
		"Comunidad de Madrid":   "comunidad-de-madrid",
		"Águeda":                "agueda",
		"Alcobaça":              "alcobaca",
		"A Coruña":              "a-coruna",
		"Saint-Étienne":         "saint-etienne",
		"  Vila Real de Trás  ": "vila-real-de-tras",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// cityServer stands in for both APIs the slug lookup uses: the geocoder that
// names the places around a point, and the city pages that say which city code
// each slug serves.
func cityServer(t *testing.T, pages map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v3/addresslookup"):
			_, _ = w.Write([]byte(`{"cityCode":"LIS","countryCode":"PT","addressComponents":{"locality":"Cascais","administrative_area_level_2":"Cascais","administrative_area_level_1":"Lisboa"}}`))
		case strings.HasSuffix(r.URL.Path, "/cities"):
			_, _ = w.Write([]byte(`{"cities":[{"translatedName":"Cascais","slug":"cascais"},{"translatedName":"Lisboa","slug":"lisboa"}]}`))
		default:
			slug := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			code, ok := pages[slug]
			if !ok {
				t.Errorf("unexpected city page %s", r.URL.Path)
			}
			if code == "" {
				_, _ = w.Write([]byte(`<html>pick an address</html>`))
				return
			}
			_, _ = w.Write([]byte(`<html>\"cityCode\":\"` + code + `\"</html>`))
		}
	}))
}

func TestCitySlugSkipsTownsThatServeNoCity(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	// Cascais has an SEO page but no city code of its own; Lisboa serves it.
	srv := cityServer(t, map[string]string{"cascais": "", "lisboa": "LIS"})
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.webBase = srv.URL
	slug, err := c.CitySlug(38.7223, -9.1393, "LIS", "PT")
	if err != nil || slug != "lisboa" {
		t.Fatalf("slug = %q err = %v", slug, err)
	}

	// The answer is cached, so a second lookup needs no city pages at all.
	dead := cityServer(t, map[string]string{})
	dead.Close()
	cached := NewClient(nil)
	cached.apiBase = dead.URL
	cached.webBase = dead.URL
	if slug, err := cached.CitySlug(38.7223, -9.1393, "LIS", "PT"); err != nil || slug != "lisboa" {
		t.Fatalf("cached slug = %q err = %v", slug, err)
	}
}

func TestCitySlugErrorsWhenNothingServesTheCode(t *testing.T) {
	t.Setenv("GLOVO_CONFIG_DIR", t.TempDir())
	srv := cityServer(t, map[string]string{"cascais": "", "lisboa": ""})
	defer srv.Close()

	c := NewClient(nil)
	c.apiBase = srv.URL
	c.webBase = srv.URL
	if _, err := c.CitySlug(38.7223, -9.1393, "LIS", "PT"); err == nil {
		t.Fatal("expected an error naming --city-slug")
	} else if !strings.Contains(err.Error(), "--city-slug") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeliveryAddressCookieCarriesThePoint(t *testing.T) {
	cookie := deliveryAddressCookie(Location{Lat: 38.7223, Lng: -9.1393, CityCode: "LIS", CountryCode: "PT"})
	name, value, ok := strings.Cut(cookie, "=")
	if !ok || name != "glovo_delivery_address" {
		t.Fatalf("cookie = %q", cookie)
	}
	raw, err := url.QueryUnescape(value)
	if err != nil {
		t.Fatal(err)
	}
	var addr map[string]any
	if err := json.Unmarshal([]byte(raw), &addr); err != nil {
		t.Fatalf("cookie value is not JSON: %v", err)
	}
	if addr["cityCode"] != "LIS" || addr["countryCode"] != "PT" || addr["latitude"] != 38.7223 {
		t.Fatalf("addr = %v", addr)
	}
}
