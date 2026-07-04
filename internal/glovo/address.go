package glovo

import "fmt"

func (c *Client) Geocode(lat, lng float64) (*Address, error) {
	url := fmt.Sprintf("%s/v3/addresslookup/pub/coordinates?latitude=%g&longitude=%g&allowFallback=true", c.apiBase, lat, lng)
	var resp struct {
		PlaceID   string  `json:"placeId"`
		Title     string  `json:"title"`
		Subtitle  string  `json:"subtitle"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	status, err := c.doJSON("GET", url, nil, &resp)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("geocode: http %d", status)
	}
	return &Address{PlaceID: resp.PlaceID, Label: resp.Title, Latitude: resp.Latitude, Longitude: resp.Longitude}, nil
}
