package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeff/sched-cli/internal/paths"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("DirectoryStyle", func(t *testing.T) {
		if cfg.DirectoryStyle != paths.StylePlatform {
			t.Errorf("DirectoryStyle = %q, want %q", cfg.DirectoryStyle, paths.StylePlatform)
		}
	})

	t.Run("LogRetentionDays", func(t *testing.T) {
		if cfg.LogRetentionDays != 90 {
			t.Errorf("LogRetentionDays = %d, want 90", cfg.LogRetentionDays)
		}
	})

	t.Run("Syslog", func(t *testing.T) {
		if cfg.Syslog != false {
			t.Error("Syslog = true, want false")
		}
	})

	t.Run("CacheTTLHours", func(t *testing.T) {
		if cfg.CacheTTLHours != 48 {
			t.Errorf("CacheTTLHours = %d, want 48", cfg.CacheTTLHours)
		}
	})

	t.Run("EventURL_empty", func(t *testing.T) {
		if cfg.EventURL != "" {
			t.Errorf("EventURL = %q, want empty", cfg.EventURL)
		}
	})

	t.Run("Username_empty", func(t *testing.T) {
		if cfg.Username != "" {
			t.Errorf("Username = %q, want empty", cfg.Username)
		}
	})

	t.Run("Token_empty", func(t *testing.T) {
		if cfg.Token != "" {
			t.Errorf("Token = %q, want empty", cfg.Token)
		}
	})

	t.Run("HasAuth_false", func(t *testing.T) {
		if cfg.HasAuth() {
			t.Error("HasAuth() = true, want false for default config")
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("returns_DefaultConfig_when_file_missing", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		want := DefaultConfig()
		if cfg.DirectoryStyle != want.DirectoryStyle {
			t.Errorf("DirectoryStyle = %q, want %q", cfg.DirectoryStyle, want.DirectoryStyle)
		}
		if cfg.LogRetentionDays != want.LogRetentionDays {
			t.Errorf("LogRetentionDays = %d, want %d", cfg.LogRetentionDays, want.LogRetentionDays)
		}
		if cfg.Syslog != want.Syslog {
			t.Errorf("Syslog = %v, want %v", cfg.Syslog, want.Syslog)
		}
		if cfg.CacheTTLHours != want.CacheTTLHours {
			t.Errorf("CacheTTLHours = %d, want %d", cfg.CacheTTLHours, want.CacheTTLHours)
		}
	})

	t.Run("loads_valid_config", func(t *testing.T) {
		dir := t.TempDir()
		data := []byte(`{
			"event_url": "https://example.sched.com",
			"username": "testuser",
			"auth_method": "token",
			"token": "abc123",
			"ucontext": "ctx-val",
			"directory_style": "xdg",
			"log_retention_days": 30,
			"syslog": true,
			"cache_ttl_hours": 24
		}`)
		if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0600); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}

		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.EventURL != "https://example.sched.com" {
			t.Errorf("EventURL = %q, want %q", cfg.EventURL, "https://example.sched.com")
		}
		if cfg.Username != "testuser" {
			t.Errorf("Username = %q, want %q", cfg.Username, "testuser")
		}
		if cfg.AuthMethod != AuthToken {
			t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, AuthToken)
		}
		if cfg.Token != "abc123" {
			t.Errorf("Token = %q, want %q", cfg.Token, "abc123")
		}
		if cfg.UContext != "ctx-val" {
			t.Errorf("UContext = %q, want %q", cfg.UContext, "ctx-val")
		}
		if cfg.DirectoryStyle != paths.StyleXDG {
			t.Errorf("DirectoryStyle = %q, want %q", cfg.DirectoryStyle, paths.StyleXDG)
		}
		if cfg.LogRetentionDays != 30 {
			t.Errorf("LogRetentionDays = %d, want 30", cfg.LogRetentionDays)
		}
		if cfg.Syslog != true {
			t.Error("Syslog = false, want true")
		}
		if cfg.CacheTTLHours != 24 {
			t.Errorf("CacheTTLHours = %d, want 24", cfg.CacheTTLHours)
		}
	})

	t.Run("round_trip_preserves_all_fields", func(t *testing.T) {
		dir := t.TempDir()
		original := &Config{
			EventURL:         "https://myevent.sched.com",
			Username:         "roundtripuser",
			AuthMethod:       AuthBrowser,
			Token:            "tok-roundtrip",
			UContext:         "uctx-roundtrip",
			DirectoryStyle:   paths.StyleXDG,
			LogRetentionDays: 7,
			Syslog:           true,
			CacheTTLHours:    12,
		}
		if err := Save(dir, original); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		loaded, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.EventURL != original.EventURL {
			t.Errorf("EventURL = %q, want %q", loaded.EventURL, original.EventURL)
		}
		if loaded.Username != original.Username {
			t.Errorf("Username = %q, want %q", loaded.Username, original.Username)
		}
		if loaded.AuthMethod != original.AuthMethod {
			t.Errorf("AuthMethod = %q, want %q", loaded.AuthMethod, original.AuthMethod)
		}
		if loaded.Token != original.Token {
			t.Errorf("Token = %q, want %q", loaded.Token, original.Token)
		}
		if loaded.UContext != original.UContext {
			t.Errorf("UContext = %q, want %q", loaded.UContext, original.UContext)
		}
		if loaded.DirectoryStyle != original.DirectoryStyle {
			t.Errorf("DirectoryStyle = %q, want %q", loaded.DirectoryStyle, original.DirectoryStyle)
		}
		if loaded.LogRetentionDays != original.LogRetentionDays {
			t.Errorf("LogRetentionDays = %d, want %d", loaded.LogRetentionDays, original.LogRetentionDays)
		}
		if loaded.Syslog != original.Syslog {
			t.Errorf("Syslog = %v, want %v", loaded.Syslog, original.Syslog)
		}
		if loaded.CacheTTLHours != original.CacheTTLHours {
			t.Errorf("CacheTTLHours = %d, want %d", loaded.CacheTTLHours, original.CacheTTLHours)
		}
	})

	t.Run("returns_error_for_invalid_JSON", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{not json`), 0600); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		_, err := Load(dir)
		if err == nil {
			t.Fatal("Load() expected error for invalid JSON, got nil")
		}
	})

	t.Run("partial_config_gets_defaults_for_missing_fields", func(t *testing.T) {
		dir := t.TempDir()
		data := []byte(`{"event_url": "https://partial.sched.com", "token": "partial-tok"}`)
		if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0600); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
		cfg, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		// Explicitly set fields
		if cfg.EventURL != "https://partial.sched.com" {
			t.Errorf("EventURL = %q, want %q", cfg.EventURL, "https://partial.sched.com")
		}
		if cfg.Token != "partial-tok" {
			t.Errorf("Token = %q, want %q", cfg.Token, "partial-tok")
		}
		// Fields missing from JSON should retain defaults
		if cfg.DirectoryStyle != paths.StylePlatform {
			t.Errorf("DirectoryStyle = %q, want default %q", cfg.DirectoryStyle, paths.StylePlatform)
		}
		if cfg.LogRetentionDays != 90 {
			t.Errorf("LogRetentionDays = %d, want default 90", cfg.LogRetentionDays)
		}
		if cfg.Syslog != false {
			t.Error("Syslog = true, want default false")
		}
		if cfg.CacheTTLHours != 48 {
			t.Errorf("CacheTTLHours = %d, want default 48", cfg.CacheTTLHours)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("creates_directory_if_not_exists", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "config", "dir")
		cfg := DefaultConfig()
		if err := Save(dir, cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		info, err := os.Stat(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatalf("config.json not created: %v", err)
		}
		if info.IsDir() {
			t.Error("config.json is a directory, want file")
		}
	})

	t.Run("writes_valid_JSON", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{
			EventURL: "https://test.sched.com",
			Token:    "test-token",
		}
		if err := Save(dir, cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatalf("ReadFile error = %v", err)
		}
		var parsed Config
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("config.json is not valid JSON: %v", err)
		}
		if parsed.EventURL != "https://test.sched.com" {
			t.Errorf("parsed EventURL = %q, want %q", parsed.EventURL, "https://test.sched.com")
		}
		if parsed.Token != "test-token" {
			t.Errorf("parsed Token = %q, want %q", parsed.Token, "test-token")
		}
	})

	t.Run("sets_file_permissions_0600", func(t *testing.T) {
		dir := t.TempDir()
		cfg := DefaultConfig()
		if err := Save(dir, cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		info, err := os.Stat(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatalf("Stat error = %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	})

	t.Run("overwrites_existing_config", func(t *testing.T) {
		dir := t.TempDir()
		first := &Config{EventURL: "https://first.sched.com", Token: "first-token"}
		if err := Save(dir, first); err != nil {
			t.Fatalf("Save(first) error = %v", err)
		}
		second := &Config{EventURL: "https://second.sched.com", Token: "second-token"}
		if err := Save(dir, second); err != nil {
			t.Fatalf("Save(second) error = %v", err)
		}
		loaded, err := Load(dir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded.EventURL != "https://second.sched.com" {
			t.Errorf("EventURL = %q, want %q", loaded.EventURL, "https://second.sched.com")
		}
		if loaded.Token != "second-token" {
			t.Errorf("Token = %q, want %q", loaded.Token, "second-token")
		}
	})

	t.Run("writes_human_readable_indented_JSON", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{EventURL: "https://indent.sched.com"}
		if err := Save(dir, cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatalf("ReadFile error = %v", err)
		}
		content := string(data)
		// MarshalIndent with "  " prefix produces lines starting with two spaces
		if len(content) == 0 {
			t.Fatal("config.json is empty")
		}
		// Verify indentation: re-marshal with same settings and compare
		var parsed Config
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}
		expected, err := json.MarshalIndent(&parsed, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent error = %v", err)
		}
		if content != string(expected) {
			t.Errorf("config.json not indented as expected.\ngot:\n%s\nwant:\n%s", content, string(expected))
		}
	})
}

func TestConfigPath(t *testing.T) {
	t.Run("returns_correct_full_path", func(t *testing.T) {
		dir := "/some/config/dir"
		got := ConfigPath(dir)
		want := filepath.Join(dir, "config.json")
		if got != want {
			t.Errorf("ConfigPath(%q) = %q, want %q", dir, got, want)
		}
	})
}

func TestHasAuth(t *testing.T) {
	t.Run("true_when_token_set", func(t *testing.T) {
		cfg := &Config{Token: "my-secret-token"}
		if !cfg.HasAuth() {
			t.Error("HasAuth() = false, want true when token is set")
		}
	})

	t.Run("false_when_token_empty", func(t *testing.T) {
		cfg := &Config{Token: ""}
		if cfg.HasAuth() {
			t.Error("HasAuth() = true, want false when token is empty")
		}
	})
}

func TestEnvToken(t *testing.T) {
	t.Run("returns_value_when_set", func(t *testing.T) {
		t.Setenv("SCHED_TOKEN", "env-token-value")
		got := EnvToken()
		if got != "env-token-value" {
			t.Errorf("EnvToken() = %q, want %q", got, "env-token-value")
		}
	})

	t.Run("returns_empty_when_not_set", func(t *testing.T) {
		t.Setenv("SCHED_TOKEN", "")
		got := EnvToken()
		if got != "" {
			t.Errorf("EnvToken() = %q, want empty", got)
		}
	})
}

func TestEnvCredentials(t *testing.T) {
	t.Run("returns_both_and_true_when_both_set", func(t *testing.T) {
		t.Setenv("SCHED_EMAIL", "user@example.com")
		t.Setenv("SCHED_PASSWORD", "s3cret")
		email, password, ok := EnvCredentials()
		if !ok {
			t.Error("ok = false, want true when both env vars set")
		}
		if email != "user@example.com" {
			t.Errorf("email = %q, want %q", email, "user@example.com")
		}
		if password != "s3cret" {
			t.Errorf("password = %q, want %q", password, "s3cret")
		}
	})

	t.Run("returns_false_when_only_email_set", func(t *testing.T) {
		t.Setenv("SCHED_EMAIL", "user@example.com")
		t.Setenv("SCHED_PASSWORD", "")
		_, _, ok := EnvCredentials()
		if ok {
			t.Error("ok = true, want false when only email is set")
		}
	})

	t.Run("returns_false_when_only_password_set", func(t *testing.T) {
		t.Setenv("SCHED_EMAIL", "")
		t.Setenv("SCHED_PASSWORD", "s3cret")
		_, _, ok := EnvCredentials()
		if ok {
			t.Error("ok = true, want false when only password is set")
		}
	})

	t.Run("returns_false_when_neither_set", func(t *testing.T) {
		t.Setenv("SCHED_EMAIL", "")
		t.Setenv("SCHED_PASSWORD", "")
		_, _, ok := EnvCredentials()
		if ok {
			t.Error("ok = true, want false when neither env var is set")
		}
	})
}

func TestAuthMethodConstants(t *testing.T) {
	t.Run("AuthCredentials", func(t *testing.T) {
		if AuthCredentials != "credentials" {
			t.Errorf("AuthCredentials = %q, want %q", AuthCredentials, "credentials")
		}
	})

	t.Run("AuthToken", func(t *testing.T) {
		if AuthToken != "token" {
			t.Errorf("AuthToken = %q, want %q", AuthToken, "token")
		}
	})

	t.Run("AuthBrowser", func(t *testing.T) {
		if AuthBrowser != "browser" {
			t.Errorf("AuthBrowser = %q, want %q", AuthBrowser, "browser")
		}
	})

	t.Run("AuthFromBrowser", func(t *testing.T) {
		if AuthFromBrowser != "from-browser" {
			t.Errorf("AuthFromBrowser = %q, want %q", AuthFromBrowser, "from-browser")
		}
	})
}
