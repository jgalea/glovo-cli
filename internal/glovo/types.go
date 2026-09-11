package glovo

// Store is one restaurant/store in a search or feed listing.
type Store struct {
	ID              int64   `json:"id"`
	Slug            string  `json:"slug"`
	Name            string  `json:"name"`
	Rating          int     `json:"rating,omitempty"`      // percent, e.g. 97
	RatingCount     int     `json:"ratingCount,omitempty"` // orders the rating is based on
	ETAMinutesLow   int     `json:"etaMinutesLow,omitempty"`
	ETAMinutesHigh  int     `json:"etaMinutesHigh,omitempty"`
	DeliveryFee     float64 `json:"deliveryFee,omitempty"`
	Currency        string  `json:"currency,omitempty"`
	Open            bool    `json:"open"`
	ScheduledOpenAt string  `json:"scheduledOpenAt,omitempty"` // e.g. "19:15" when closed
}

// MenuItem is one purchasable product on a store's menu.
type MenuItem struct {
	StoreProductID   int64   `json:"storeProductId"`
	StoreProductUUID string  `json:"storeProductUuid,omitempty"` // the UUID the basket POST calls storeProductId
	ExternalID       string  `json:"externalId,omitempty"`       // e.g. "p_1179695"
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	Price            float64 `json:"price"`
	PromoPrice       float64 `json:"promoPrice,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	CategoryID       int64   `json:"categoryId,omitempty"`
}

// Category is a named group of menu items.
type Category struct {
	ID    int64      `json:"id"`
	Name  string     `json:"name"`
	Items []MenuItem `json:"items"`
}

// Menu is a store's full menu plus the store identity needed to build a
// basket POST (storeId, storeAddressId).
type Menu struct {
	StoreID        int64      `json:"storeId"`
	StoreAddressID int64      `json:"storeAddressId"`
	Categories     []Category `json:"categories"`
}

// BasketLine is one line in the authenticated basket.
type BasketLine struct {
	ProductID       int64   `json:"productId"`
	Name            string  `json:"name,omitempty"` // not present on the real basket response; filled in by the caller via a menu join, if at all
	Qty             int     `json:"qty"`
	UnitPrice       float64 `json:"unitPrice"`
	BasketProductID string  `json:"basketProductId,omitempty"` // opaque id the PATCH set/remove call needs, from a prior GET
}

// Basket is a snapshot of the user's authenticated basket for one store.
type Basket struct {
	CustomerID      int64        `json:"customerId"`
	StoreID         int64        `json:"storeId"`
	StoreAddressID  int64        `json:"storeAddressId,omitempty"`
	StoreCategoryID int64        `json:"storeCategoryId,omitempty"`
	BasketID        string       `json:"basketId,omitempty"`      // needed, with BasketVersion, for the PATCH set/remove call
	BasketVersion   string       `json:"basketVersion,omitempty"` // optimistic-concurrency version from the last GET/POST
	Lines           []BasketLine `json:"lines"`
	Subtotal        float64      `json:"subtotal"`
	Fees            float64      `json:"fees"`
	Total           float64      `json:"total"`
	Currency        string       `json:"currency"`
}

// CartProduct is the identity a basket POST needs for one product: the
// numeric SKU plus the externalId and storeProductId UUID the menu carries.
type CartProduct struct {
	ID               int64
	ExternalID       string
	StoreProductUUID string
}

// Address is a resolved delivery address.
type Address struct {
	PlaceID   string  `json:"placeId"`
	Label     string  `json:"label"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
