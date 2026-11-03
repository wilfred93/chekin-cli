package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/wilfred93/chekin-cli/internal/chekin"
	"github.com/wilfred93/chekin-cli/internal/config"
)

// refreshWindow: tokens closer than this to expiring are refreshed first.
const refreshWindow = 5 * time.Minute

var errNotLoggedIn = errors.New("not logged in: run `chekin login`, or set CHEKIN_TOKEN, or CHEKIN_EMAIL and CHEKIN_PASSWORD")

// session finds a usable token and keeps it fresh. Sources, in order:
// CHEKIN_TOKEN, the token saved by `chekin login`, then CHEKIN_EMAIL +
// CHEKIN_PASSWORD (logs in and saves the token).
type session struct {
	app    *App
	client *chekin.Client
	saved  *config.Credentials // nil when the token came from CHEKIN_TOKEN
}

func (a *App) newSession(ctx context.Context) (*session, error) {
	s := &session{app: a, client: a.client("")}

	if t := os.Getenv("CHEKIN_TOKEN"); t != "" {
		s.client.Token = t
		return s, nil
	}

	creds, err := config.Load()
	if err != nil {
		return nil, err
	}
	if creds != nil && creds.Token != "" {
		s.saved = creds
		s.client.Token = creds.Token
		if err := s.refreshIfNeeded(ctx); err != nil {
			return nil, err
		}
		return s, nil
	}

	if err := s.loginFromEnv(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// refreshIfNeeded renews a saved token that is about to expire. An expired
// token can't be refreshed, so it falls back to logging in from env vars.
func (s *session) refreshIfNeeded(ctx context.Context) error {
	claims, err := chekin.ParseClaims(s.client.Token)
	if err != nil || claims.Exp == 0 {
		return nil // unknown format: let the API decide
	}
	left := time.Until(claims.Expiry())
	if left > refreshWindow {
		return nil
	}
	if left > 0 {
		if t, err := s.client.Refresh(ctx, s.client.Token); err == nil {
			return s.store(t)
		}
	}
	if err := s.loginFromEnv(ctx); err != nil {
		if errors.Is(err, errNotLoggedIn) {
			return errors.New("session expired: run `chekin login` again")
		}
		return err
	}
	return nil
}

func (s *session) loginFromEnv(ctx context.Context) error {
	email, password := os.Getenv("CHEKIN_EMAIL"), os.Getenv("CHEKIN_PASSWORD")
	if email == "" || password == "" {
		return errNotLoggedIn
	}
	t, err := s.client.Login(ctx, email, password)
	if err != nil {
		return fmt.Errorf("login with CHEKIN_EMAIL/CHEKIN_PASSWORD failed: %w", err)
	}
	if s.saved == nil {
		s.saved = &config.Credentials{}
	}
	s.saved.Email = email
	return s.store(t)
}

func (s *session) store(token string) error {
	s.client.Token = token
	if s.saved == nil {
		return nil
	}
	s.saved.Token = token
	s.saved.BaseURL = s.app.baseURL()
	return config.Save(s.saved)
}

// call runs fn, and if the API rejects the token, logs in again from env vars
// (when set) and retries once.
func (s *session) call(ctx context.Context, fn func(*chekin.Client) error) error {
	err := fn(s.client)
	if err == nil || !chekin.IsUnauthorized(err) {
		return err
	}
	if loginErr := s.loginFromEnv(ctx); loginErr != nil {
		return fmt.Errorf("%w (session expired or invalid: run `chekin login` again)", err)
	}
	return fn(s.client)
}
