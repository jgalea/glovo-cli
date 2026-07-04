package glovo

import "errors"

// ErrCheckoutNotCaptured is returned until the real checkout request has been
// captured from a live order and implemented. Ordering is deliberately inert
// until then rather than sending a guessed request.
var ErrCheckoutNotCaptured = errors.New("checkout not yet implemented — needs a real-order capture")

func (c *Client) OrderDryRun(storeID int64) (*Basket, error) {
	return c.BasketGet(storeID)
}

func (c *Client) PlaceOrder(storeID int64) (string, error) {
	return "", ErrCheckoutNotCaptured
}
