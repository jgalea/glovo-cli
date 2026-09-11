package glovo

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Client) Search(query string, lat, lng float64, cityCode, countryCode string) ([]Store, error) {
	if c.loadAuth() == nil {
		return nil, fmt.Errorf("search needs your Glovo account — run: glovo login")
	}
	u := fmt.Sprintf("%s/v1/web/store_wall/search?searchQuery=%s", c.apiBase, url.QueryEscape(query))
	extra := c.storeWallHeaders(lat, lng, cityCode, countryCode)
	reqBody := map[string]any{"searchContext": map[string]any{"searchId": newUUID()}}

	var raw json.RawMessage
	status, err := c.doAuthedJSONH("POST", u, extra, reqBody, &raw)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("search: http %d", status)
	}
	stores := parseStoreWall(raw)
	if stores == nil {
		return nil, fmt.Errorf("couldn't parse Glovo's search response, the API shape may have changed")
	}
	return stores, nil
}

// locationHeaders builds the glovo-* routing + delivery-location headers that
// tell the API where the order would be delivered.
func locationHeaders(lat, lng float64, cityCode, countryCode string) map[string]string {
	return map[string]string{
		"glovo-location-city-code":          cityCode,
		"glovo-location-country-code":       countryCode,
		"glovo-delivery-location-latitude":  strconv.FormatFloat(lat, 'f', -1, 64),
		"glovo-delivery-location-longitude": strconv.FormatFloat(lng, 'f', -1, 64),
		"glovo-delivery-location-accuracy":  "0",
		"glovo-delivery-location-timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
}

// storeWallHeaders is the identity + session header set plus the delivery
// location. Dropping any of the identity headers (the perseus ones included)
// makes the gateway answer 503 "no available server" rather than a 4xx.
func (c *Client) storeWallHeaders(lat, lng float64, cityCode, countryCode string) map[string]string {
	h := c.apiHeaders()
	for k, v := range locationHeaders(lat, lng, cityCode, countryCode) {
		h[k] = v
	}
	return h
}

type storeCard struct {
	Type string `json:"type"`
	Data struct {
		Title struct {
			Text struct {
				Text string `json:"text"`
			} `json:"text"`
		} `json:"title"`
		Slug string `json:"slug"`
	} `json:"data"`
	Actions []struct {
		Trigger string `json:"trigger"`
		Data    struct {
			Events []struct {
				Data map[string]string `json:"data"`
			} `json:"events"`
		} `json:"data"`
	} `json:"actions"`
}

func (card storeCard) impression() map[string]string {
	for _, a := range card.Actions {
		if a.Trigger != "onImpression" {
			continue
		}
		for _, e := range a.Data.Events {
			if _, ok := e.Data["shopId"]; ok {
				return e.Data
			}
		}
	}
	return map[string]string{}
}

func parseStoreWall(body []byte) []Store {
	var root struct {
		Data struct {
			Body struct {
				Data struct {
					Elements []json.RawMessage `json:"elements"`
				} `json:"data"`
			} `json:"body"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil
	}
	els := root.Data.Body.Data.Elements
	out := make([]Store, 0, len(els))
	for _, raw := range els {
		var card storeCard
		if json.Unmarshal(raw, &card) != nil || card.Type != "STORE_CARD_V2" {
			continue
		}
		ev := card.impression()
		s := Store{Slug: card.Data.Slug, Name: card.Data.Title.Text.Text}
		s.ID, _ = strconv.ParseInt(ev["shopId"], 10, 64)
		s.Rating = parsePercent(ev["shopRating"])
		s.ETAMinutesLow = atoiSafe(ev["promisedDeliveryTimeRangeLower"])
		s.ETAMinutesHigh = atoiSafe(ev["promisedDeliveryTimeRangeUpper"])
		s.DeliveryFee, _ = strconv.ParseFloat(ev["shopDeliveryFee"], 64)
		s.Currency = ev["shopDeliveryFeeCurrency"]
		s.Open = ev["shopIsOpen"] == "true"
		if s.ID == 0 && s.Slug == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func parsePercent(s string) int { return atoiSafe(strings.TrimSuffix(s, "%")) }

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
