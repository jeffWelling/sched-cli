package auth

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jeff/sched-cli/internal/config"
	_ "modernc.org/sqlite"
)

// LoginFromFirefox reads sched.com cookies from the Firefox cookie database.
// It locates the default Firefox profile, copies cookies.sqlite to a temp file
// (Firefox holds a lock while running), and extracts token + ucontext cookies.
func (a *Authenticator) LoginFromFirefox() (*AuthResult, error) {
	profileDir, err := findFirefoxProfile()
	if err != nil {
		return nil, fmt.Errorf("firefox profile: %w", err)
	}

	dbPath := filepath.Join(profileDir, "cookies.sqlite")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("firefox cookies database not found: %w", err)
	}

	// Copy to a temp file — Firefox holds a lock on the original.
	tmpFile, err := os.CreateTemp("", "sched-firefox-cookies-*.sqlite")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	data, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("reading firefox cookies database: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return nil, fmt.Errorf("writing temp cookies database: %w", err)
	}

	return readFirefoxCookies(tmpPath)
}

// readFirefoxCookies opens a SQLite database at dbPath and reads sched.com
// cookies from the moz_cookies table. This is separated from LoginFromFirefox
// to allow direct testing without Firefox profile discovery.
func readFirefoxCookies(dbPath string) (*AuthResult, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening cookies database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT name, value FROM moz_cookies WHERE host = '.sched.com' AND name IN ('token', 'ucontext')`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying cookies: %w", err)
	}
	defer rows.Close()

	cookies := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scanning cookie row: %w", err)
		}
		cookies[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cookie rows: %w", err)
	}

	token := cookies["token"]
	if token == "" {
		return nil, fmt.Errorf("no sched.com token cookie found in Firefox")
	}

	return &AuthResult{
		Token:    token,
		UContext: cookies["ucontext"],
		Method:   config.AuthFromBrowser,
	}, nil
}

// findFirefoxProfile locates the default Firefox profile directory.
func findFirefoxProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	var profilesDir string
	switch runtime.GOOS {
	case "darwin":
		profilesDir = filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	case "linux":
		profilesDir = filepath.Join(home, ".mozilla", "firefox")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return "", fmt.Errorf("reading profiles directory %s: %w", profilesDir, err)
	}

	// Prefer *.default-release, fall back to *.default.
	var defaultProfile string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		matched, _ := filepath.Match("*.default-release", name)
		if matched {
			return filepath.Join(profilesDir, name), nil
		}
		matchedDefault, _ := filepath.Match("*.default", name)
		if matchedDefault && defaultProfile == "" {
			defaultProfile = filepath.Join(profilesDir, name)
		}
	}

	if defaultProfile != "" {
		return defaultProfile, nil
	}

	return "", fmt.Errorf("no default Firefox profile found in %s", profilesDir)
}
