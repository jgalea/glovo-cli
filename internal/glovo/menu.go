package glovo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Location is the delivery point a store is browsed and ordered from.
// Everything but the coordinates is filled in by ResolveLocation.
type Location struct {
	Lat, Lng    float64
	CityCode    string // "LIS"
	CountryCode string // "PT"
	CitySlug    string // "lisboa", the city segment of Glovo's store URLs
	CityName    string
	Label       string // the geocoded street address
	PlaceID     string
}

// StoreMenu fetches and parses the SSR store-detail page for a store,
// returning its menu grouped by category plus the store identity (storeId,
// storeAddressId) the basket POST needs. No account is needed, but the page
// renders a menu only against a delivery address, which the cookie supplies.
func (c *Client) StoreMenu(slug string, loc Location) (*Menu, error) {
	citySlug := loc.CitySlug
	if citySlug == "" {
		resolved, err := c.CitySlug(loc.Lat, loc.Lng, loc.CityCode, loc.CountryCode)
		if err != nil {
			return nil, err
		}
		citySlug = resolved
	}
	u := fmt.Sprintf("%s/en/%s/%s/stores/%s", c.webBase, strings.ToLower(loc.CountryCode), url.PathEscape(citySlug), url.PathEscape(slug))
	cookie := deliveryAddressCookie(loc)
	// Glovo serves the address-picker shell in bursts for pages it renders
	// fine moments later, so an unparseable page is retried with a pause
	// rather than reported as a missing menu.
	backoff := []time.Duration{0, 700 * time.Millisecond, 2 * time.Second}
	for attempt, wait := range backoff {
		if wait > 0 {
			time.Sleep(wait)
		}
		// Retries close the connection so they are not pinned to the server
		// that just answered with the shell.
		html, _, err := c.getPage(u, cookie, attempt > 0)
		if err != nil {
			return nil, err
		}
		if menu := parseMenu(html); menu != nil {
			return menu, nil
		}
		c.log("no menu in %s (attempt %d/%d)", u, attempt+1, len(backoff))
	}
	return nil, fmt.Errorf("no menu on %s — check the store slug, or point --city-slug at the store's own city", u)
}

// deliveryAddressCookie is the cookie the web app sets when you pick an
// address. Without it Glovo serves the address-picker shell in place of the
// store's menu.
func deliveryAddressCookie(loc Location) string {
	addr := map[string]any{
		"latitude":    loc.Lat,
		"longitude":   loc.Lng,
		"cityCode":    loc.CityCode,
		"countryCode": loc.CountryCode,
		"cityName":    loc.CityName,
		"text":        loc.Label,
		"details":     "",
		"placeId":     loc.PlaceID,
		"postalCode":  nil,
		"isVerified":  true,
	}
	b, err := json.Marshal(addr)
	if err != nil {
		return ""
	}
	return "glovo_delivery_address=" + url.QueryEscape(string(b))
}

// parseMenu extracts the store's menu from the store-detail page's SSR
// payload. Categories are the payload's titled sections (each an object with
// "title"/"slug"/"elements"); a section's "elements" holds PRODUCT_ROW
// entries whose "data" is the product tile. Sections may share products
// (e.g. a "Top sellers" section duplicates items from a category section) —
// that mirrors the real page and is preserved here. The same payload also
// carries a "store" object (the only object with an "addressId" key) with
// the numeric storeId ("id") and storeAddressId ("addressId").
func parseMenu(html string) *Menu {
	blob := extractNextChunks(html)
	sections := scanJSONObjects(blob, "elements")

	var cats []Category
	itemCount := 0
	for _, sec := range sections {
		name := asString(sec["title"])
		if name == "" {
			continue
		}
		cat := Category{ID: int64(len(cats) + 1), Name: name}
		for _, raw := range asSlice(sec["elements"]) {
			el := asMap(raw)
			if asString(el["type"]) != "PRODUCT_ROW" {
				continue
			}
			item := parseMenuItem(asMap(el["data"]), cat.ID)
			if item.StoreProductID == 0 {
				continue
			}
			cat.Items = append(cat.Items, item)
			itemCount++
		}
		cats = append(cats, cat)
	}
	if itemCount == 0 {
		return nil
	}
	storeID, storeAddressID := parseStoreIdentity(blob)
	return &Menu{StoreID: storeID, StoreAddressID: storeAddressID, Categories: cats}
}

// parseStoreIdentity finds the payload's store object (identified by its
// unique "addressId" key) and returns its numeric storeId and
// storeAddressId. Both are zero if the object isn't found.
func parseStoreIdentity(blob string) (storeID, storeAddressID int64) {
	stores := scanJSONObjects(blob, "addressId")
	if len(stores) == 0 {
		return 0, 0
	}
	// stores[0] assumes the payload has exactly one object with an "addressId"
	// key; that's empirical from the one page captured so far, not guaranteed.
	store := stores[0]
	return asInt64(store["id"]), asInt64(store["addressId"])
}

func parseMenuItem(tile map[string]any, categoryID int64) MenuItem {
	priceInfo := asMap(tile["priceInfo"])
	item := MenuItem{
		// tile["id"] is the numeric productSKU as a string, not the
		// storeProductId UUID also present on the tile; the cart needs both.
		StoreProductID:   asInt64(tile["id"]),
		StoreProductUUID: asString(tile["storeProductId"]),
		ExternalID:       asString(tile["externalId"]),
		Name:             asString(tile["name"]),
		Description:      asString(tile["description"]),
		Price:            asFloat(priceInfo["amount"]),
		Currency:         asString(priceInfo["currencyCode"]),
		CategoryID:       categoryID,
	}
	if promos := asSlice(tile["promotions"]); len(promos) > 0 {
		promoPriceInfo := asMap(asMap(promos[0])["priceInfo"])
		item.PromoPrice = asFloat(promoPriceInfo["amount"])
	}
	return item
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return 0
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}
