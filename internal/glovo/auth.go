package glovo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type authSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	CustomerID   int64  `json:"customer_id"`
}

func (c *Client) authPath() string {
	dir := os.Getenv("GLOVO_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".glovo")
	}
	return filepath.Join(dir, "auth.json")
}

func (c *Client) loadAuth() *authSession {
	if c.auth != nil {
		return c.auth
	}
	b, err := os.ReadFile(c.authPath())
	if err != nil {
		return nil
	}
	var s authSession
	if json.Unmarshal(b, &s) != nil || s.AccessToken == "" {
		return nil
	}
	c.auth = &s
	return c.auth
}

func (c *Client) saveAuth(s *authSession) {
	c.auth = s
	_ = os.MkdirAll(filepath.Dir(c.authPath()), 0o700)
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		_ = os.WriteFile(c.authPath(), b, 0o600)
	}
}

func (c *Client) LoggedIn() bool { return c.loadAuth() != nil }

func (c *Client) doAuthedJSON(method, url string, body any, out any) (int, error) {
	return c.doAuthedJSONH(method, url, nil, body, out)
}

func (c *Client) doAuthedJSONH(method, url string, extra map[string]string, body, out any) (int, error) {
	s := c.loadAuth()
	if s == nil {
		return 0, fmt.Errorf("not logged in — run: glovo login")
	}
	status, err := c.authedOnce(method, url, extra, body, out, s.AccessToken)
	if err != nil {
		return status, err
	}
	if status == 401 {
		if err := c.refresh(); err != nil {
			return status, fmt.Errorf("session expired and refresh failed — run: glovo login")
		}
		status, err = c.authedOnce(method, url, extra, body, out, c.loadAuth().AccessToken)
		if err != nil {
			return status, err
		}
		if status == 401 {
			return status, fmt.Errorf("session expired — run: glovo login")
		}
		return status, nil
	}
	return status, nil
}

func (c *Client) authedOnce(method, url string, extra map[string]string, body, out any, token string) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rdr)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if os.Getenv("GLOVO_DEBUG") != "" && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		fmt.Fprintf(os.Stderr, "glovo: %s %s -> %d\n%s\n", method, url, resp.StatusCode, string(raw))
	}
	if resp.StatusCode == 401 {
		return 401, nil
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode %s: %w", url, err)
		}
	}
	return resp.StatusCode, nil
}

// LoginToken stores a refresh token pasted from a logged-in browser, then
// exchanges it for an access token and the customer id. Endpoint shape is a
// documented-shape placeholder pending reconciliation against a live capture.
func (c *Client) LoginToken(refreshToken string) error {
	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		CustomerID   int64  `json:"customerId"`
	}
	status, err := c.doJSON("POST", c.apiBase+"/oauth/refresh",
		map[string]string{"refreshToken": refreshToken, "grantType": "refresh_token"}, &resp)
	if err != nil {
		return err
	}
	if status != 200 || resp.AccessToken == "" {
		return fmt.Errorf("could not validate token (http %d) — copy a fresh glovo_refresh_token from your browser", status)
	}
	s := &authSession{AccessToken: resp.AccessToken, RefreshToken: refreshToken, CustomerID: resp.CustomerID}
	if resp.RefreshToken != "" {
		s.RefreshToken = resp.RefreshToken
	}
	c.saveAuth(s)
	return nil
}

// LoginPassword exchanges the user's own credentials for tokens. Endpoint shape
// is a documented-shape placeholder pending reconciliation against a live
// capture; the password is never stored.
func (c *Client) LoginPassword(email, password string) error {
	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		CustomerID   int64  `json:"customerId"`
	}
	status, err := c.doJSON("POST", c.apiBase+"/oauth/login",
		map[string]string{"email": email, "password": password, "grantType": "password"}, &resp)
	if err != nil {
		return err
	}
	if status == 403 || status == 401 {
		return fmt.Errorf("login rejected (http %d) — Glovo may require a browser step; use: glovo login --token", status)
	}
	if status != 200 || resp.AccessToken == "" {
		return fmt.Errorf("login failed: http %d", status)
	}
	c.saveAuth(&authSession{AccessToken: resp.AccessToken, RefreshToken: resp.RefreshToken, CustomerID: resp.CustomerID})
	return nil
}

// LoginAccessToken stores a pasted access token directly, deriving the
// customer id from the token's payload claim unless one is passed explicitly.
func (c *Client) LoginAccessToken(accessToken string, customerID int64) error {
	if customerID == 0 {
		id, err := decodeCustomerID(accessToken)
		if err != nil || id == 0 {
			return fmt.Errorf("couldn't read the customer id from the access token — pass it explicitly with --customer-id")
		}
		customerID = id
	}
	c.saveAuth(&authSession{AccessToken: accessToken, CustomerID: customerID})
	return nil
}

func decodeCustomerID(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("not a JWT")
	}
	seg := parts[1]
	seg = strings.ReplaceAll(seg, "-", "+")
	seg = strings.ReplaceAll(seg, "_", "/")
	if pad := len(seg) % 4; pad != 0 {
		seg += strings.Repeat("=", 4-pad)
	}
	raw, err := base64.StdEncoding.DecodeString(seg)
	if err != nil {
		return 0, err
	}
	var claims struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return 0, err
	}
	var payload struct {
		UserID json.RawMessage `json:"userId"`
	}
	if err := json.Unmarshal([]byte(claims.Payload), &payload); err != nil {
		return 0, err
	}
	uid, err := strconv.ParseInt(strings.Trim(string(payload.UserID), `"`), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("userId not an integer: %w", err)
	}
	return uid, nil
}

// refresh hits the token-refresh endpoint. The path and field names are a
// documented-shape placeholder pending reconciliation against a live capture.
func (c *Client) refresh() error {
	s := c.loadAuth()
	if s == nil {
		return fmt.Errorf("no session")
	}
	var resp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	status, err := c.doJSON("POST", c.apiBase+"/oauth/refresh",
		map[string]string{"refreshToken": s.RefreshToken, "grantType": "refresh_token"}, &resp)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("refresh: http %d", status)
	}
	if resp.AccessToken == "" {
		return fmt.Errorf("refresh: empty access token in response")
	}
	s.AccessToken = resp.AccessToken
	if resp.RefreshToken != "" {
		s.RefreshToken = resp.RefreshToken
	}
	c.saveAuth(s)
	return nil
}
