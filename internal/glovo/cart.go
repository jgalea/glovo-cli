package glovo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// apiBasket matches the real basket response captured in
// CAPTURE-basket.md: qty lives in quantity.items (the resulting count, not
// the increments delta) and unit price in price.unitaryTotalPrice.major. The
// basket-level total/fees/currency fields were truncated in the capture, so
// they aren't modeled here — toBasket falls back to a line-sum total.
type apiBasket struct {
	BasketID         string             `json:"basketId"`
	BasketVersion    string             `json:"basketVersion"`
	HandlingStrategy string             `json:"handlingStrategy"`
	CustomerID       int64              `json:"customerId"`
	StoreID          int64              `json:"storeId"`
	StoreAddressID   int64              `json:"storeAddressId"`
	StoreCategoryID  int64              `json:"storeCategoryId"`
	Products         []apiBasketProduct `json:"products"`
}

type apiBasketProduct struct {
	IDs      apiBasketIDs      `json:"ids"`
	Quantity apiBasketQuantity `json:"quantity"`
	Price    apiBasketPrice    `json:"price"`
}

type apiBasketIDs struct {
	BasketProductID string          `json:"basketProductId"`
	LegacyID        json.RawMessage `json:"legacyId"` // number or string across responses
	ID              json.RawMessage `json:"id"`       // number or string across responses
	ExternalID      string          `json:"externalId"`
	StoreProductID  string          `json:"storeProductId"`
}

// asID parses an id Glovo returns as either a JSON number or a quoted string.
func asID(raw json.RawMessage) int64 {
	n, _ := strconv.ParseInt(strings.Trim(string(raw), `"`), 10, 64)
	return n
}

type apiBasketQuantity struct {
	Increments      int    `json:"increments"`
	IncrementsLimit int    `json:"incrementsLimit"`
	Items           int    `json:"items"`
	ItemsLimit      int    `json:"itemsLimit"`
	LimitType       string `json:"limitType"`
}

type apiBasketPrice struct {
	TotalFormatted       string               `json:"totalFormatted"`
	UnitaryBasePrice     apiBasketPriceAmount `json:"unitaryBasePrice"`
	UnitaryTotalPrice    apiBasketPriceAmount `json:"unitaryTotalPrice"`
	ProductTotalDiscount float64              `json:"productTotalDiscount"`
}

type apiBasketPriceAmount struct {
	Minor     int64   `json:"minor"`
	Major     float64 `json:"major"`
	Formatted string  `json:"formatted"`
}

func (b apiBasket) toBasket(customerID int64) *Basket {
	out := &Basket{
		CustomerID:      customerID,
		StoreID:         b.StoreID,
		StoreAddressID:  b.StoreAddressID,
		StoreCategoryID: b.StoreCategoryID,
		BasketID:        b.BasketID,
		BasketVersion:   b.BasketVersion,
		Currency:        "EUR", // not present on the product price object; the real currency field lives in the uncaptured basket-level tail
	}
	for _, p := range b.Products {
		pid := asID(p.IDs.ID)
		if pid == 0 {
			pid = asID(p.IDs.LegacyID)
		}
		line := BasketLine{
			ProductID:       pid,
			Qty:             p.Quantity.Items,
			UnitPrice:       p.Price.UnitaryTotalPrice.Major,
			BasketProductID: p.IDs.BasketProductID,
		}
		out.Lines = append(out.Lines, line)
		out.Subtotal += float64(line.Qty) * line.UnitPrice
	}
	// The basket-level pricing object (fees, discounts, grand total) was
	// truncated in the capture below "products", so Total/Subtotal here are
	// a qty*unitPrice line-sum fallback pending a fuller capture.
	out.Total = out.Subtotal
	return out
}

func (c *Client) basketURL(storeID int64) string {
	cid := c.loadAuth().CustomerID
	return fmt.Sprintf("%s/v1/authenticated/customers/%d/baskets/stores/%d", c.apiBase, cid, storeID)
}

func (c *Client) baskets() string {
	cid := c.loadAuth().CustomerID
	return fmt.Sprintf("%s/v1/authenticated/customers/%d/baskets", c.apiBase, cid)
}

func (c *Client) basketQuantityURL(basketID string) string {
	cid := c.loadAuth().CustomerID
	return fmt.Sprintf("%s/v1/authenticated/customers/%d/baskets/%s/products/quantity", c.apiBase, cid, basketID)
}

func (c *Client) BasketGet(storeID int64) (*Basket, error) {
	s := c.loadAuth()
	if s == nil {
		return nil, fmt.Errorf("not logged in — run: glovo login")
	}
	var resp apiBasket
	status, err := c.doAuthedJSONH("GET", c.basketURL(storeID), c.loc, nil, &resp)
	if err != nil {
		return nil, err
	}
	if status == 204 {
		return &Basket{CustomerID: s.CustomerID, StoreID: storeID, Currency: "EUR"}, nil
	}
	if status != 200 {
		return nil, fmt.Errorf("basket: http %d", status)
	}
	return resp.toBasket(s.CustomerID), nil
}

// basketGetRaw fetches the raw basket (before mapping) so callers that must
// re-send the full product list (PUT complete-update) have every id/qty field.
func (c *Client) basketGetRaw(storeID int64) (*apiBasket, int, error) {
	var resp apiBasket
	status, err := c.doAuthedJSONH("GET", c.basketURL(storeID), c.loc, nil, &resp)
	if err != nil {
		return nil, status, err
	}
	return &resp, status, nil
}

func idStr(raw json.RawMessage) string { return strings.Trim(string(raw), `"`) }

// BasketAdd adds delta units of a product. Glovo splits this by basket state:
// an empty basket is created with POST /baskets; a basket that already exists
// must be sent as a complete update via PUT /baskets/{basketId} carrying every
// product (quantity.increments is the resulting per-line total, not a delta).
func (c *Client) BasketAdd(storeID, storeAddressID, storeCategoryID int64, p CartProduct, delta int) (*Basket, error) {
	s := c.loadAuth()
	if s == nil {
		return nil, fmt.Errorf("not logged in — run: glovo login")
	}
	cur, status, err := c.basketGetRaw(storeID)
	if err != nil {
		return nil, err
	}
	newIDs := map[string]any{
		"id":             strconv.FormatInt(p.ID, 10),
		"externalId":     p.ExternalID,
		"legacyId":       strconv.FormatInt(p.ID, 10),
		"storeProductId": p.StoreProductUUID,
	}

	// Empty basket → POST create.
	if status == 204 || cur.BasketID == "" {
		body := map[string]any{
			"products": []map[string]any{{
				"customizations": []any{},
				"ids":            newIDs,
				"quantity":       map[string]any{"increments": delta},
			}},
			"storeId":          storeID,
			"storeAddressId":   storeAddressID,
			"storeCategoryId":  storeCategoryID,
			"handlingStrategy": "DELIVERY",
		}
		var resp apiBasket
		st, err := c.doAuthedJSONH("POST", c.baskets(), c.loc, body, &resp)
		if err != nil {
			return nil, err
		}
		if st < 200 || st >= 300 {
			return nil, fmt.Errorf("basket add: http %d", st)
		}
		return resp.toBasket(s.CustomerID), nil
	}

	// Existing basket → PUT complete-update with every product.
	products := make([]map[string]any, 0, len(cur.Products)+1)
	found := false
	for _, pr := range cur.Products {
		qty := pr.Quantity.Items
		if asID(pr.IDs.ID) == p.ID || asID(pr.IDs.LegacyID) == p.ID {
			qty += delta
			found = true
		}
		products = append(products, map[string]any{
			"ids": map[string]any{
				"id":              idStr(pr.IDs.ID),
				"legacyId":        idStr(pr.IDs.LegacyID),
				"externalId":      pr.IDs.ExternalID,
				"storeProductId":  pr.IDs.StoreProductID,
				"basketProductId": pr.IDs.BasketProductID,
			},
			"customizations": []any{},
			"quantity":       map[string]any{"increments": qty},
		})
	}
	if !found {
		products = append(products, map[string]any{
			"ids":            newIDs,
			"customizations": []any{},
			"quantity":       map[string]any{"increments": delta},
		})
	}
	body := map[string]any{
		"handlingStrategy": "DELIVERY",
		"storeId":          storeID,
		"storeAddressId":   storeAddressID,
		"storeCategoryId":  storeCategoryID,
		"basketId":         cur.BasketID,
		"basketVersion":    cur.BasketVersion,
		"customerId":       strconv.FormatInt(s.CustomerID, 10),
		"products":         products,
	}
	var resp apiBasket
	st, err := c.doAuthedJSONH("PUT", c.basketByIDURL(cur.BasketID), c.loc, body, &resp)
	if err != nil {
		return nil, err
	}
	if st < 200 || st >= 300 {
		return nil, fmt.Errorf("basket add (update): http %d", st)
	}
	return resp.toBasket(s.CustomerID), nil
}

func (c *Client) basketByIDURL(basketID string) string {
	cid := c.loadAuth().CustomerID
	return fmt.Sprintf("%s/v1/authenticated/customers/%d/baskets/%s/products", c.apiBase, cid, basketID)
}

// patchBasketProducts sets absolute quantities via the PATCH endpoint
// (quantity:0 removes a line), then re-fetches the basket for a consistent
// return value since the PATCH itself returns 204 with no body.
func (c *Client) patchBasketProducts(storeID int64, basketID, basketVersion string, products []map[string]any) (*Basket, error) {
	if c.loadAuth() == nil {
		return nil, fmt.Errorf("not logged in — run: glovo login")
	}
	body := map[string]any{
		"handlingStrategy": "DELIVERY",
		"basketVersion":    basketVersion,
		"products":         products,
	}
	status, err := c.doAuthedJSONH("PATCH", c.basketQuantityURL(basketID), c.loc, body, nil)
	if err != nil {
		return nil, err
	}
	if status != 204 && (status < 200 || status >= 300) {
		return nil, fmt.Errorf("basket quantity update: http %d", status)
	}
	return c.BasketGet(storeID)
}

// BasketSet sets a line to an absolute quantity via the PATCH set/remove
// endpoint (qty<=0 removes the line). It needs the line's basketProductId
// and the basket's version, both from a prior GET, so it fetches the
// current basket first. If the product isn't in the basket yet and qty>0,
// it falls back to BasketAdd using qty as the delta for a fresh line — that
// fallback needs storeAddressId/storeCategoryId, which the caller passes in
// (the GET's basket-level fields are 0 when the basket was never seeded, so
// they aren't a safe source for these).
func (c *Client) BasketSet(storeID, storeAddressID, storeCategoryID int64, p CartProduct, qty int) (*Basket, error) {
	if qty < 0 {
		qty = 0
	}
	cur, err := c.BasketGet(storeID)
	if err != nil {
		return nil, err
	}
	var basketProductID string
	for _, l := range cur.Lines {
		if l.ProductID == p.ID {
			basketProductID = l.BasketProductID
			break
		}
	}
	if basketProductID == "" {
		if qty <= 0 {
			return cur, nil
		}
		return c.BasketAdd(storeID, storeAddressID, storeCategoryID, p, qty)
	}
	products := []map[string]any{{"basketProductId": basketProductID, "quantity": qty}}
	return c.patchBasketProducts(storeID, cur.BasketID, cur.BasketVersion, products)
}

// BasketClear PATCHes every existing line to quantity 0. No-op if the
// basket is already empty.
func (c *Client) BasketClear(storeID int64) (*Basket, error) {
	cur, err := c.BasketGet(storeID)
	if err != nil {
		return nil, err
	}
	if len(cur.Lines) == 0 {
		return cur, nil
	}
	products := make([]map[string]any, 0, len(cur.Lines))
	for _, l := range cur.Lines {
		products = append(products, map[string]any{"basketProductId": l.BasketProductID, "quantity": 0})
	}
	return c.patchBasketProducts(storeID, cur.BasketID, cur.BasketVersion, products)
}
