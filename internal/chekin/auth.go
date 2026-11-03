package chekin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type tokenResponse struct {
	Token string `json:"token"`
}

// Login exchanges email and password for a JWT.
//
// It uses the legacy endpoint api/v3/token/. The dashboard's current login
// (api/v5/token/) requires a Cloudflare Turnstile captcha, which a CLI can't
// solve. Accounts that only sign in with Google or GitHub have no password and
// must use a token copied from the browser instead.
func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	var out tokenResponse
	err := c.do(ctx, "POST", "api/v3/token/", map[string]string{"email": email, "password": password}, &out)
	if err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", errors.New("login succeeded but the API returned no token")
	}
	return out.Token, nil
}

// Refresh exchanges a token that is still valid for a new one, the same way
// the dashboard does.
func (c *Client) Refresh(ctx context.Context, token string) (string, error) {
	var out tokenResponse
	if err := c.do(ctx, "POST", "api/v3/api-token-refresh/", map[string]string{"token": token}, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", errors.New("token refresh returned no token")
	}
	return out.Token, nil
}

// Claims are the JWT fields the CLI cares about.
type Claims struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	Exp      int64  `json:"exp"`
}

// Expiry returns when the token expires.
func (c Claims) Expiry() time.Time { return time.Unix(c.Exp, 0) }

// ParseClaims reads a JWT's payload without verifying its signature. It is
// only used to show the account and to know when to refresh.
func ParseClaims(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("not a JWT (expected three dot-separated parts)")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return Claims{}, errors.New("not a JWT (payload is not base64)")
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, errors.New("not a JWT (payload is not JSON)")
	}
	return c, nil
}
