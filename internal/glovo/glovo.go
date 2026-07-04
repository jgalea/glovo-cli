package glovo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiBase   = "https://api.glovoapp.com"
	webBase   = "https://glovoapp.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
	maxBody   = 8 << 20
)

type Client struct {
	http    *http.Client
	apiBase string
	webBase string
	logf    func(string, ...any)
	auth    *authSession      // populated lazily by auth.go
	loc     map[string]string // glovo-location-* / delivery-location headers for authed calls
}

// SetLocation records the delivery location whose glovo-* headers the
// authenticated basket endpoints require (they reject a POST with "City code
// is required" otherwise).
func (c *Client) SetLocation(cityCode, countryCode string, lat, lng float64) {
	c.loc = storeWallHeaders(lat, lng, cityCode, countryCode)
}

func NewClient(logf func(string, ...any)) *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		apiBase: apiBase,
		webBase: webBase,
		logf:    logf,
	}
}

func (c *Client) log(format string, args ...any) {
	if c.logf != nil {
		c.logf(format, args...)
	}
}

func (c *Client) getHTML(url string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GET %s: http %d", url, resp.StatusCode)
	}
	return string(b), nil
}

// doJSON sends an optional JSON body and decodes a JSON response into out (if non-nil).
func (c *Client) doJSON(method, url string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rdr)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode %s: %w", url, err)
		}
	}
	return resp.StatusCode, nil
}
