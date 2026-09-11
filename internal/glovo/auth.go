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
	return filepath.Join(configDir(), "auth.json")
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
	req.Header.Set("Origin", webBase)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.apiHeaders() {
		req.Header.Set(k, v)
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

// LoginToken stores a refresh token pasted from a logged-in browser and
// exchanges it for a session. The refresh response carries no customer id, so
// it comes from the access token's own payload claim.
func (c *Client) LoginToken(refreshToken string) error {
	resp, status, err := c.exchangeRefresh(refreshToken)
	if err != nil {
		return err
	}
	if status != 200 || resp.AccessToken == "" {
		return fmt.Errorf("could not validate token (http %d) — copy a fresh glovo_refresh_token from your browser", status)
	}
	return c.storeSession(resp.AccessToken, firstNonEmpty(resp.RefreshToken, refreshToken))
}

// tokenResponse is the session payload Glovo returns from /oauth/refresh and
// from the 2FA validation, and nests under "access" on /oauth/token.
type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (c *Client) exchangeRefresh(refreshToken string) (tokenResponse, int, error) {
	var resp tokenResponse
	status, err := c.doJSONH("POST", c.apiBase+"/oauth/refresh", c.apiHeaders(),
		map[string]string{"refreshToken": refreshToken, "grantType": "refresh_token"}, &resp)
	return resp, status, err
}

func (c *Client) storeSession(access, refresh string) error {
	cid, err := decodeCustomerID(access)
	if err != nil || cid == 0 {
		return fmt.Errorf("couldn't read the customer id from the access token: %w", err)
	}
	c.saveAuth(&authSession{AccessToken: access, RefreshToken: refresh, CustomerID: cid})
	return nil
}

// TwoFactorRequired reports that Glovo answered the password login with a
// verification challenge instead of a session: it has sent a code to the
// account's phone, which ValidateTwoFactor exchanges for the session.
type TwoFactorRequired struct {
	Token     string
	ExpiresIn int
}

func (e *TwoFactorRequired) Error() string {
	return "Glovo sent a verification code to your phone"
}

// LoginPassword exchanges the user's own email + password for tokens via
// Glovo's OAuth password grant. The password is sent once and never stored —
// only the returned tokens are. A device Glovo hasn't seen before gets a
// TwoFactorRequired back rather than a session.
func (c *Client) LoginPassword(email, password string) error {
	var resp struct {
		Access    tokenResponse `json:"access"`
		TwoFactor *struct {
			Token     string `json:"twoFactorToken"`
			ExpiresIn int    `json:"expiresIn"`
		} `json:"twoFactor"`
	}
	headers, err := c.authHeaders()
	if err != nil {
		return err
	}
	body := map[string]string{"grantType": "password", "username": email, "password": password}
	status, err := c.doJSONH("POST", c.apiBase+"/oauth/token", headers, body, &resp)
	if err != nil {
		return err
	}
	if status == 401 || status == 403 {
		return fmt.Errorf("login rejected (http %d) — check your email/password, or use: glovo login --access-token", status)
	}
	if status != 200 {
		return fmt.Errorf("login failed: http %d", status)
	}
	if resp.TwoFactor != nil && resp.TwoFactor.Token != "" {
		return &TwoFactorRequired{Token: resp.TwoFactor.Token, ExpiresIn: resp.TwoFactor.ExpiresIn}
	}
	if resp.Access.AccessToken == "" {
		return fmt.Errorf("login succeeded but no access token in the response")
	}
	return c.storeSession(resp.Access.AccessToken, resp.Access.RefreshToken)
}

// ValidateTwoFactor completes a login that came back with a TwoFactorRequired,
// exchanging the challenge token and the code Glovo sent for a session.
func (c *Client) ValidateTwoFactor(twoFactorToken, code string) error {
	var resp tokenResponse
	headers, err := c.authHeaders()
	if err != nil {
		return err
	}
	body := map[string]string{"twoFactorToken": twoFactorToken, "code": code}
	status, err := c.doJSONH("POST", c.apiBase+"/oauth/2fa/phone_verification/validate", headers, body, &resp)
	if err != nil {
		return err
	}
	if status == 400 || status == 401 {
		return fmt.Errorf("that code wasn't accepted (http %d) — it may have expired", status)
	}
	if status != 200 || resp.AccessToken == "" {
		return fmt.Errorf("verification failed: http %d", status)
	}
	return c.storeSession(resp.AccessToken, resp.RefreshToken)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstStringField returns the first present key's value as a string (quotes
// trimmed), tolerating either JSON string or number.
func firstStringField(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.Trim(string(v), `"`)
			if s != "" && s != "null" {
				return s
			}
		}
	}
	return ""
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

// refresh swaps the stored refresh token for a new session. Glovo rotates the
// refresh token on every call, so the new one has to replace the old.
func (c *Client) refresh() error {
	s := c.loadAuth()
	if s == nil {
		return fmt.Errorf("no session")
	}
	if s.RefreshToken == "" {
		return fmt.Errorf("no refresh token stored — log in with: glovo login --email you@example.com")
	}
	resp, status, err := c.exchangeRefresh(s.RefreshToken)
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
