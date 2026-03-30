package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jeff/sched-cli/internal/paths"
)

// AuthMethod identifies how the user authenticated.
type AuthMethod string

const (
	AuthCredentials AuthMethod = "credentials"
	AuthToken       AuthMethod = "token"
	AuthBrowser     AuthMethod = "browser"
	AuthFromBrowser AuthMethod = "from-browser"
)

// Config holds all persistent configuration.
type Config struct {
	EventURL         string              `json:"event_url"`
	Username         string              `json:"username"`
	AuthMethod       AuthMethod          `json:"auth_method"`
	Token            string              `json:"token"`
	UContext         string              `json:"ucontext"`
	DirectoryStyle   paths.DirectoryStyle `json:"directory_style"`
	LogRetentionDays int                 `json:"log_retention_days"`
	Syslog           bool                `json:"syslog"`
	CacheTTLHours    int                 `json:"cache_ttl_hours"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DirectoryStyle:   paths.StylePlatform,
		LogRetentionDays: 90,
		Syslog:           false,
		CacheTTLHours:    48,
	}
}

// Load reads the config file from the platform-appropriate directory.
// Returns DefaultConfig if the file does not exist.
func Load(configDir string) (*Config, error) {
	p := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to the platform-appropriate directory.
// Creates the directory if it does not exist. Sets file permissions to 0600.
func Save(configDir string, cfg *Config) error {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "config.json"), data, 0600)
}

// ConfigPath returns the full path to the config file.
func ConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.json")
}

// HasAuth returns true if the config has valid auth credentials.
func (c *Config) HasAuth() bool {
	return c.Token != ""
}

// EnvToken returns the SCHED_TOKEN environment variable if set.
func EnvToken() string {
	return os.Getenv("SCHED_TOKEN")
}

// EnvCredentials returns SCHED_EMAIL and SCHED_PASSWORD if both are set.
func EnvCredentials() (email, password string, ok bool) {
	email = os.Getenv("SCHED_EMAIL")
	password = os.Getenv("SCHED_PASSWORD")
	ok = email != "" && password != ""
	return
}
