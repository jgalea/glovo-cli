package glovo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Glovo's store pages live under /{lang}/{country}/{city}/stores/{slug}, and
// the city segment has to be the store's own city: any other city renders the
// address-picker shell instead of the menu. The API talks in city codes
// ("LIS") and the web in slugs ("lisboa"), with nothing mapping one to the
// other, so a slug is resolved by geocoding the delivery point, taking the
// place names around it as candidates, and asking each candidate's city page
// which code it serves.

type cityCache struct {
	Slugs map[string]string `json:"slugs"` // "PT/LIS" -> "lisboa"
}

func cityCachePath() string { return filepath.Join(configDir(), "cities.json") }

func loadCityCache() cityCache {
	c := cityCache{Slugs: map[string]string{}}
	b, err := os.ReadFile(cityCachePath())
	if err != nil {
		return c
	}
	if json.Unmarshal(b, &c) != nil || c.Slugs == nil {
		return cityCache{Slugs: map[string]string{}}
	}
	return c
}

func saveCityCache(c cityCache) {
	_ = os.MkdirAll(filepath.Dir(cityCachePath()), 0o700)
	if b, err := json.MarshalIndent(c, "", "  "); err == nil {
		_ = os.WriteFile(cityCachePath(), b, 0o600)
	}
}

// CitySlug returns the web path segment for the city serving a delivery point.
func (c *Client) CitySlug(lat, lng float64, cityCode, countryCode string) (string, error) {
	key := countryCode + "/" + cityCode
	cache := loadCityCache()
	if slug, ok := cache.Slugs[key]; ok && slug != "" {
		return slug, nil
	}
	candidates, err := c.citySlugCandidates(lat, lng, countryCode)
	if err != nil {
		return "", err
	}
	for _, slug := range candidates {
		code, err := c.cityPageCode(strings.ToLower(countryCode), slug)
		if err != nil {
			c.log("city page %s: %v", slug, err)
			continue
		}
		if code == "" {
			continue
		}
		cache.Slugs[countryCode+"/"+code] = slug
		if code == cityCode {
			saveCityCache(cache)
			return slug, nil
		}
	}
	saveCityCache(cache)
	return "", fmt.Errorf("couldn't work out Glovo's city page for %s — pass it with --city-slug", cityCode)
}

// citySlugCandidates lists the plausible city slugs for a delivery point, most
// specific first, filtered against the country's published city list when that
// is available.
func (c *Client) citySlugCandidates(lat, lng float64, countryCode string) ([]string, error) {
	addr, err := c.geocodeComponents(lat, lng)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, k := range []string{"locality", "administrative_area_level_2", "administrative_area_level_1", "postal_town"} {
		if v := addr[k]; v != "" {
			names = append(names, v)
		}
	}
	var candidates []string
	for _, n := range names {
		s := slugify(n)
		if s != "" && !contains(candidates, s) {
			candidates = append(candidates, s)
		}
	}
	known, err := c.countryCitySlugs(countryCode)
	if err != nil || len(known) == 0 {
		return candidates, nil
	}
	var filtered []string
	for _, s := range candidates {
		if contains(known, s) {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return candidates, nil
	}
	return filtered, nil
}

// ResolveCityCodes returns the city and country codes Glovo serves a delivery
// point from, so callers only have to know the coordinates.
func (c *Client) ResolveCityCodes(lat, lng float64) (cityCode, countryCode string, err error) {
	u := fmt.Sprintf("%s/v3/addresslookup/pub/coordinates?latitude=%g&longitude=%g&allowFallback=true", c.apiBase, lat, lng)
	var resp struct {
		CityCode    string `json:"cityCode"`
		CountryCode string `json:"countryCode"`
	}
	status, err := c.doJSON("GET", u, nil, &resp)
	if err != nil {
		return "", "", err
	}
	if status != 200 || resp.CityCode == "" {
		return "", "", fmt.Errorf("couldn't resolve a Glovo city for %g,%g (http %d)", lat, lng, status)
	}
	return resp.CityCode, resp.CountryCode, nil
}

func (c *Client) geocodeComponents(lat, lng float64) (map[string]string, error) {
	u := fmt.Sprintf("%s/v3/addresslookup/pub/coordinates?latitude=%g&longitude=%g&allowFallback=true", c.apiBase, lat, lng)
	var resp struct {
		Components map[string]string `json:"addressComponents"`
	}
	status, err := c.doJSON("GET", u, nil, &resp)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("address lookup: http %d", status)
	}
	return resp.Components, nil
}

func (c *Client) countryCitySlugs(countryCode string) ([]string, error) {
	u := fmt.Sprintf("%s/seo-content/locales/en/countries/%s/cities", c.apiBase, strings.ToLower(countryCode))
	var resp struct {
		Cities []struct {
			Slug string `json:"slug"`
		} `json:"cities"`
	}
	status, err := c.doJSON("GET", u, nil, &resp)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("city list: http %d", status)
	}
	out := make([]string, 0, len(resp.Cities))
	for _, city := range resp.Cities {
		out = append(out, city.Slug)
	}
	return out, nil
}

var cityCodeRe = regexp.MustCompile(`cityCode\\?"\s*:\s*\\?"([A-Z]{2,5})`)

// cityPageCode reads the city code a city landing page serves. A slug with no
// page of its own, or an SEO page for a town Glovo folds into a bigger city,
// carries no code and is not an error.
func (c *Client) cityPageCode(country, slug string) (string, error) {
	html, status, err := c.getPage(fmt.Sprintf("%s/en/%s/%s", c.webBase, country, url.PathEscape(slug)), "")
	if err != nil {
		return "", err
	}
	if status == 404 {
		return "", nil
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("http %d", status)
	}
	m := cityCodeRe.FindStringSubmatch(html)
	if m == nil {
		return "", nil
	}
	return m[1], nil
}

// accents folds the letters that actually turn up in the city names Glovo
// serves; anything outside the set is dropped by slugify.
var accents = map[rune]rune{
	'á': 'a', 'à': 'a', 'ã': 'a', 'â': 'a', 'ä': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'õ': 'o', 'ô': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n', 'ý': 'y', 'ž': 'z', 'š': 's', 'č': 'c', 'ć': 'c', 'đ': 'd',
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if folded, ok := accents[r]; ok {
			r = folded
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
