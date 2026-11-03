// Package chekin is a small client for the API that the Chekin web dashboard
// (dashboard.chekin.com) uses at https://a.chekin.io.
//
// It is not Chekin's official partner API: the endpoints were found by
// inspecting the dashboard, and Chekin may change them without notice.
package chekin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the API host used by the Chekin dashboard.
const DefaultBaseURL = "https://a.chekin.io"

// Client talks to the Chekin API. Token is sent as "Authorization: JWT <token>".
type Client struct {
	BaseURL   string
	Token     string
	UserAgent string
	HTTP      *http.Client
}

// New returns a client for baseURL (DefaultBaseURL when empty).
func New(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Token:     token,
		UserAgent: "chekin-cli",
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a non-2xx response from the API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("chekin API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("chekin API error (HTTP %d)", e.StatusCode)
}

// IsUnauthorized reports whether err means the token is missing, invalid or expired.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden && strings.Contains(strings.ToLower(apiErr.Message), "token"))
}

// do sends a request to path (relative to BaseURL) or to an absolute URL on
// the same host, and decodes a JSON response into out.
func (c *Client) do(ctx context.Context, method, target string, body, out any) error {
	u, err := c.resolve(target)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Source", "DASHBOARD")
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "JWT "+c.Token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", req.URL.Host, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{StatusCode: resp.StatusCode, Message: errorMessage(data), Body: string(data)}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unexpected response from %s: %w", req.URL.Path, err)
	}
	return nil
}

// resolve turns a relative path into a full URL. Absolute URLs (such as
// pagination "next" links) must point at the API host, so the token is never
// sent anywhere else.
func (c *Client) resolve(target string) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", c.BaseURL, err)
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return c.BaseURL + "/" + strings.TrimLeft(target, "/"), nil
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	if u.Host != base.Host {
		return "", fmt.Errorf("refusing to follow link to unexpected host %q", u.Host)
	}
	// The API returns http:// links behind its proxy; keep the base scheme.
	u.Scheme = base.Scheme
	return u.String(), nil
}

// errorMessage pulls a readable message out of the API's error formats:
// {"message": ...}, {"error": ...}, {"detail": ...} and {"non_field_errors": [...]}.
func errorMessage(data []byte) string {
	var body map[string]any
	if json.Unmarshal(data, &body) != nil {
		s := strings.TrimSpace(string(data))
		if len(s) > 200 || strings.HasPrefix(s, "<") {
			return ""
		}
		return s
	}
	for _, key := range []string{"message", "error", "detail"} {
		if s, ok := body[key].(string); ok && s != "" {
			return s
		}
	}
	if list, ok := body["non_field_errors"].([]any); ok && len(list) > 0 {
		if s, ok := list[0].(string); ok {
			return s
		}
	}
	return ""
}
