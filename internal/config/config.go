// Package config holds the CLI settings and the account key.
//
// The key lives in the OS keychain (macOS Keychain, libsecret on Linux, Credential Manager on Windows). Without a
// keychain it falls back to ~/.notholy/credentials with mode 600 and says so. NOTHOLY_API_KEY always wins, for CI.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	service     = "notholy"
	account     = "api-key"
	DefaultAPI  = "https://api.notholy.com/v1"
	DefaultApp  = "https://notholy.com"
)

type Config struct {
	APIBase string `json:"api_base,omitempty"`
	AppURL  string `json:"app_url,omitempty"`
	OrgID   string `json:"org_id,omitempty"`
	OrgName string `json:"org_name,omitempty"`
	Model   string `json:"model,omitempty"`
}

// Dir is ~/.notholy, or NOTHOLY_HOME when set (tests).
func Dir() string {
	if d := os.Getenv("NOTHOLY_HOME"); d != "" {
		return d
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ".notholy"
	}
	return filepath.Join(h, ".notholy")
}

func path() string { return filepath.Join(Dir(), "config.json") }

func Load() Config {
	var c Config
	if b, err := os.ReadFile(path()); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if v := os.Getenv("NOTHOLY_API_BASE"); v != "" {
		c.APIBase = v
	}
	if v := os.Getenv("NOTHOLY_APP_URL"); v != "" {
		c.AppURL = v
	}
	if c.APIBase == "" {
		c.APIBase = DefaultAPI
	}
	if c.AppURL == "" {
		c.AppURL = DefaultApp
	}
	// an empty model means the catalog's default, read from the API (api.Client.DefaultModel)
	c.APIBase = strings.TrimRight(c.APIBase, "/")
	c.AppURL = strings.TrimRight(c.AppURL, "/")
	return c
}

func Save(c Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path(), append(b, '\n'), 0o600)
}

// keychain can be turned off (tests, headless boxes) with NOTHOLY_NO_KEYCHAIN=1.
func keychain() bool { return os.Getenv("NOTHOLY_NO_KEYCHAIN") == "" }

func credFile() string { return filepath.Join(Dir(), "credentials") }

// StoreKey saves the key and reports whether it had to use the file fallback.
func StoreKey(key string) (fallback bool, err error) {
	if keychain() {
		if err := keyring.Set(service, account, key); err == nil {
			_ = os.Remove(credFile())
			return false, nil
		}
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return true, err
	}
	return true, os.WriteFile(credFile(), []byte(key+"\n"), 0o600)
}

var ErrNoKey = errors.New("not signed in: run notholy login, or set NOTHOLY_API_KEY")

func Key() (string, error) {
	if v := strings.TrimSpace(os.Getenv("NOTHOLY_API_KEY")); v != "" {
		return v, nil
	}
	if keychain() {
		if v, err := keyring.Get(service, account); err == nil && v != "" {
			return v, nil
		}
	}
	if b, err := os.ReadFile(credFile()); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v, nil
		}
	}
	return "", ErrNoKey
}

func DeleteKey() {
	if keychain() {
		_ = keyring.Delete(service, account)
	}
	_ = os.Remove(credFile())
}

// Mask shows only the prefix of a key.
func Mask(k string) string {
	if len(k) <= 12 {
		return "****"
	}
	return k[:12] + "…"
}
