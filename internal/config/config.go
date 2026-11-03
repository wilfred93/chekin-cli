// Package config stores the CLI's saved login.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Credentials is what `chekin login` saves. The password is never stored.
type Credentials struct {
	Token   string `json:"token"`
	Email   string `json:"email,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

// Dir returns the config directory: $CHEKIN_CONFIG_DIR, else
// $XDG_CONFIG_HOME/chekin, else ~/.config/chekin.
func Dir() (string, error) {
	if d := os.Getenv("CHEKIN_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "chekin"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "chekin"), nil
}

func path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "credentials.json"), nil
}

// Load returns the saved credentials, or nil if there are none.
func Load() (*Credentials, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s is corrupt (run `chekin logout`): %w", p, err)
	}
	return &c, nil
}

// Save writes credentials readable only by the current user.
func Save(c *Credentials) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Remove deletes the saved credentials. It is not an error if there are none.
func Remove() error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
