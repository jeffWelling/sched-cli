package app

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDirs(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("SCHED_CONFIG_DIR", filepath.Join(tmpDir, "config"))
	t.Setenv("SCHED_CACHE_DIR", filepath.Join(tmpDir, "cache"))
	t.Setenv("SCHED_LOG_DIR", filepath.Join(tmpDir, "logs"))
	return tmpDir
}

func TestNewApp_CreatesDirectories(t *testing.T) {
	tmpDir := setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	cacheDir := filepath.Join(tmpDir, "cache")
	logDir := filepath.Join(tmpDir, "logs")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("cache dir was not created: %s", cacheDir)
	}
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("log dir was not created: %s", logDir)
	}
}

func TestNewApp_LoadsDefaultConfig(t *testing.T) {
	setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	cfg := a.Config()
	if cfg == nil {
		t.Fatal("Config() returned nil")
	}
	if cfg.LogRetentionDays != 90 {
		t.Errorf("expected default LogRetentionDays=90, got %d", cfg.LogRetentionDays)
	}
	if cfg.CacheTTLHours != 48 {
		t.Errorf("expected default CacheTTLHours=48, got %d", cfg.CacheTTLHours)
	}
	if cfg.Token != "" {
		t.Errorf("expected empty token for default config, got %q", cfg.Token)
	}
}

func TestNewApp_StoreAccessible(t *testing.T) {
	tmpDir := setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	s := a.Store()
	if s == nil {
		t.Fatal("Store() returned nil")
	}

	// Verify store is functional by checking session count.
	count, err := s.SessionCount()
	if err != nil {
		t.Fatalf("SessionCount() error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 sessions in fresh store, got %d", count)
	}

	// Verify the DB file exists.
	dbPath := filepath.Join(tmpDir, "cache", "sched.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file was not created: %s", dbPath)
	}
}

func TestNewApp_Close(t *testing.T) {
	setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := a.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestNewApp_AccessorsNotNil(t *testing.T) {
	setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	if a.Config() == nil {
		t.Error("Config() is nil")
	}
	if a.Store() == nil {
		t.Error("Store() is nil")
	}
	if a.Limiter() == nil {
		t.Error("Limiter() is nil")
	}
	if a.Logger() == nil {
		t.Error("Logger() is nil")
	}
	if a.Output() == nil {
		t.Error("Output() is nil")
	}
	if a.Paths() == nil {
		t.Error("Paths() is nil")
	}
	// Client should be nil when not authenticated.
	if a.Client() != nil {
		t.Error("Client() should be nil without auth")
	}
}

func TestRequireAuth_NoToken(t *testing.T) {
	setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	err = a.RequireAuth()
	if err == nil {
		t.Fatal("RequireAuth() should return error when not authenticated")
	}

	expected := "not authenticated. Run 'sched-cli config init' to set up."
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestRequireAuth_WithToken(t *testing.T) {
	tmpDir := setupTestDirs(t)

	// Write a config file with auth credentials.
	cfgDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	cfgContent := `{
		"event_url": "https://example.sched.com",
		"token": "test-token-abc123",
		"ucontext": "test-ucontext"
	}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	err = a.RequireAuth()
	if err != nil {
		t.Fatalf("RequireAuth() returned error: %v", err)
	}

	if a.Client() == nil {
		t.Error("Client() should not be nil after RequireAuth with valid token")
	}
}

func TestNewApp_WithAuthConfig_ClientCreated(t *testing.T) {
	tmpDir := setupTestDirs(t)

	// Write a config file with auth credentials.
	cfgDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	cfgContent := `{
		"event_url": "https://example.sched.com",
		"token": "test-token",
		"ucontext": "test-ctx"
	}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfgContent), 0600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	// Client should be non-nil when config has a token.
	if a.Client() == nil {
		t.Error("Client() should not be nil when config has auth token")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := setupTestDirs(t)

	a, err := New(false, false, false, "test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer a.Close()

	// Modify config and save.
	a.Config().EventURL = "https://test.sched.com"
	a.Config().Token = "saved-token"

	if err := a.SaveConfig(); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	// Verify the file was written.
	cfgPath := filepath.Join(tmpDir, "config", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading saved config: %v", err)
	}

	content := string(data)
	if !contains(content, "saved-token") {
		t.Errorf("saved config does not contain token: %s", content)
	}
	if !contains(content, "https://test.sched.com") {
		t.Errorf("saved config does not contain event_url: %s", content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
