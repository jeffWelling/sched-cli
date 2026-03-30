package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// DirectoryStyle controls how directories are resolved.
type DirectoryStyle string

const (
	StylePlatform DirectoryStyle = "platform"
	StyleXDG      DirectoryStyle = "xdg"
)

// Resolver determines platform-appropriate directories for config, cache, and logs.
type Resolver struct {
	style    DirectoryStyle
	homeDir  string
	platform string // "darwin", "linux", etc.
}

// New creates a Resolver with the given style and auto-detected platform.
func New(style DirectoryStyle) *Resolver {
	home, _ := os.UserHomeDir()
	return &Resolver{
		style:    style,
		homeDir:  home,
		platform: runtime.GOOS,
	}
}

// NewWithOverrides creates a Resolver with explicit home dir and platform (for testing).
func NewWithOverrides(style DirectoryStyle, homeDir, platform string) *Resolver {
	return &Resolver{
		style:    style,
		homeDir:  homeDir,
		platform: platform,
	}
}

// ConfigDir returns the directory for configuration files.
// Priority: SCHED_CONFIG_DIR env var > platform default.
// Note: directory_style does NOT affect config dir (bootstrap path).
func (r *Resolver) ConfigDir() string {
	if dir := os.Getenv("SCHED_CONFIG_DIR"); dir != "" {
		return dir
	}
	return r.platformConfigDir()
}

// CacheDir returns the directory for cache files (SQLite DB).
// Priority: SCHED_CACHE_DIR env var > directory_style > platform default.
func (r *Resolver) CacheDir() string {
	if dir := os.Getenv("SCHED_CACHE_DIR"); dir != "" {
		return dir
	}
	if r.style == StyleXDG {
		return r.xdgCacheDir()
	}
	return r.platformCacheDir()
}

// LogDir returns the directory for log files.
// Priority: SCHED_LOG_DIR env var > directory_style > platform default.
func (r *Resolver) LogDir() string {
	if dir := os.Getenv("SCHED_LOG_DIR"); dir != "" {
		return dir
	}
	if r.style == StyleXDG {
		return r.xdgLogDir()
	}
	return r.platformLogDir()
}

func (r *Resolver) platformConfigDir() string {
	if r.platform == "darwin" {
		return filepath.Join(r.homeDir, "Library", "Application Support", "sched-cli")
	}
	return r.xdgConfigDir()
}

func (r *Resolver) platformCacheDir() string {
	if r.platform == "darwin" {
		return filepath.Join(r.homeDir, "Library", "Caches", "sched-cli")
	}
	return r.xdgCacheDir()
}

func (r *Resolver) platformLogDir() string {
	if r.platform == "darwin" {
		return filepath.Join(r.homeDir, "Library", "Logs", "sched-cli")
	}
	return r.xdgLogDir()
}

func (r *Resolver) xdgConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "sched-cli")
	}
	return filepath.Join(r.homeDir, ".config", "sched-cli")
}

func (r *Resolver) xdgCacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "sched-cli")
	}
	return filepath.Join(r.homeDir, ".cache", "sched-cli")
}

func (r *Resolver) xdgLogDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "sched-cli", "logs")
	}
	return filepath.Join(r.homeDir, ".local", "state", "sched-cli", "logs")
}
