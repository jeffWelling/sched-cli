package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeff/sched-cli/internal/client"
	"github.com/jeff/sched-cli/internal/config"
	"github.com/jeff/sched-cli/internal/logging"
	"github.com/jeff/sched-cli/internal/output"
	"github.com/jeff/sched-cli/internal/paths"
	"github.com/jeff/sched-cli/internal/rate"
	"github.com/jeff/sched-cli/internal/store"
	_ "modernc.org/sqlite"
)

// App wires all internal packages together so CLI commands don't duplicate
// setup logic. Create with New, use accessors, and call Close when done.
type App struct {
	cfg     *config.Config
	paths   *paths.Resolver
	store   *store.Store
	client  *client.Client
	limiter *rate.Limiter
	logger  *logging.Logger
	out     *output.Formatter
	cfgDir  string
}

// New creates and wires all application components.
// debug: enable debug logging
// jsonMode: force JSON output
// prettyJSON: use indented JSON (--json-pretty)
// command: CLI command name (for log file naming)
func New(debug, jsonMode, prettyJSON bool, command string) (*App, error) {
	a := &App{}

	// 1. Bootstrap: create paths resolver with platform default to find config dir.
	bootstrap := paths.New(paths.StylePlatform)

	// 2. Get config dir from resolver.
	a.cfgDir = bootstrap.ConfigDir()

	// 3. Load config (returns defaults if no file exists).
	cfg, err := config.Load(a.cfgDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	a.cfg = cfg

	// 4. Re-create paths resolver with the config's directory_style setting.
	a.paths = paths.New(cfg.DirectoryStyle)

	// 5. Ensure cache dir and log dir exist.
	cacheDir := a.paths.CacheDir()
	logDir := a.paths.LogDir()

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("creating log dir: %w", err)
	}

	// 6. Open store at {cacheDir}/sched.db.
	dbPath := filepath.Join(cacheDir, "sched.db")
	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	a.store = s

	// 7. Create logger.
	retentionDays := cfg.LogRetentionDays
	if retentionDays <= 0 {
		retentionDays = 90
	}
	logger, err := logging.New(logging.LogConfig{
		LogDir:        logDir,
		RetentionDays: retentionDays,
		Debug:         debug,
		SyslogEnabled: cfg.Syslog,
		Command:       command,
	})
	if err != nil {
		a.store.Close()
		return nil, fmt.Errorf("creating logger: %w", err)
	}
	a.logger = logger

	// 8. Create rate limiter from store.
	a.limiter = rate.New(s, rate.DefaultLimit, rate.DefaultWindow)

	// 9. Create output formatter (auto-detect TTY, respect jsonMode/prettyJSON).
	a.out = output.AutoDetect(jsonMode, prettyJSON)

	// 10. If config has auth token, create client with cookies.
	if cfg.HasAuth() {
		a.client = client.New(cfg.EventURL, client.CookieSet{
			Token:    cfg.Token,
			UContext: cfg.UContext,
		})
	}

	return a, nil
}

// Config returns the loaded configuration.
func (a *App) Config() *config.Config {
	return a.cfg
}

// Store returns the SQLite store.
func (a *App) Store() *store.Store {
	return a.store
}

// Client returns the HTTP client for Sched.com. May be nil if not authenticated.
func (a *App) Client() *client.Client {
	return a.client
}

// Limiter returns the API rate limiter.
func (a *App) Limiter() *rate.Limiter {
	return a.limiter
}

// Logger returns the structured logger.
func (a *App) Logger() *logging.Logger {
	return a.logger
}

// Output returns the output formatter.
func (a *App) Output() *output.Formatter {
	return a.out
}

// Paths returns the directory resolver.
func (a *App) Paths() *paths.Resolver {
	return a.paths
}

// RequireAuth checks if config has valid auth. Returns error if not authenticated.
// If authenticated but client hasn't been created yet, creates it.
func (a *App) RequireAuth() error {
	if !a.cfg.HasAuth() {
		return fmt.Errorf("not authenticated. Run 'sched-cli config init' to set up.")
	}
	if a.client == nil {
		a.client = client.New(a.cfg.EventURL, client.CookieSet{
			Token:    a.cfg.Token,
			UContext: a.cfg.UContext,
		})
	}
	return nil
}

// SaveConfig persists the current config to disk.
func (a *App) SaveConfig() error {
	return config.Save(a.cfgDir, a.cfg)
}

// Close cleans up resources (store, logger).
func (a *App) Close() error {
	var firstErr error
	if a.logger != nil {
		if err := a.logger.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if a.store != nil {
		if err := a.store.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
