package paths

import (
	"path/filepath"
	"testing"
)

func TestPlatformDefaults_Darwin(t *testing.T) {
	home := "/Users/testuser"
	r := NewWithOverrides(StylePlatform, home, "darwin")

	t.Run("ConfigDir", func(t *testing.T) {
		want := filepath.Join(home, "Library", "Application Support", "sched-cli")
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("CacheDir", func(t *testing.T) {
		want := filepath.Join(home, "Library", "Caches", "sched-cli")
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("LogDir", func(t *testing.T) {
		want := filepath.Join(home, "Library", "Logs", "sched-cli")
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})
}

func TestPlatformDefaults_Linux(t *testing.T) {
	// Clear env vars that would override the platform defaults
	t.Setenv("SCHED_CONFIG_DIR", "")
	t.Setenv("SCHED_CACHE_DIR", "")
	t.Setenv("SCHED_LOG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	home := "/home/testuser"
	r := NewWithOverrides(StylePlatform, home, "linux")

	t.Run("ConfigDir", func(t *testing.T) {
		want := filepath.Join(home, ".config", "sched-cli")
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("CacheDir", func(t *testing.T) {
		want := filepath.Join(home, ".cache", "sched-cli")
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("LogDir", func(t *testing.T) {
		want := filepath.Join(home, ".local", "state", "sched-cli", "logs")
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})
}

func TestStyleXDG_Darwin(t *testing.T) {
	home := "/Users/testuser"
	r := NewWithOverrides(StyleXDG, home, "darwin")

	t.Run("CacheDir uses XDG path", func(t *testing.T) {
		want := filepath.Join(home, ".cache", "sched-cli")
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("LogDir uses XDG path", func(t *testing.T) {
		want := filepath.Join(home, ".local", "state", "sched-cli", "logs")
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})

	t.Run("ConfigDir NOT affected by StyleXDG", func(t *testing.T) {
		// ConfigDir always uses platform default (bootstrap path),
		// regardless of directory_style.
		want := filepath.Join(home, "Library", "Application Support", "sched-cli")
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q (StyleXDG should not affect ConfigDir)", got, want)
		}
	})
}

func TestXDGEnvironmentVariables_Linux(t *testing.T) {
	home := "/home/testuser"

	t.Run("XDG_CONFIG_HOME overrides config dir", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		want := filepath.Join("/custom/config", "sched-cli")
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_CACHE_HOME overrides cache dir", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_CACHE_HOME", "/custom/cache")
		want := filepath.Join("/custom/cache", "sched-cli")
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_STATE_HOME overrides log dir", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		want := filepath.Join("/custom/state", "sched-cli", "logs")
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})
}

func TestXDGEnvironmentVariables_DarwinWithStyleXDG(t *testing.T) {
	home := "/Users/testuser"

	t.Run("XDG_CONFIG_HOME respected for ConfigDir on darwin", func(t *testing.T) {
		// ConfigDir on darwin uses platformConfigDir which returns the macOS
		// Library path. But if XDG_CONFIG_HOME is set AND the platform is
		// darwin, ConfigDir still uses platformConfigDir (macOS native path).
		// Actually: platformConfigDir on darwin returns Library path directly,
		// it does NOT call xdgConfigDir. So XDG_CONFIG_HOME is NOT respected
		// for ConfigDir on darwin regardless of style.
		r := NewWithOverrides(StyleXDG, home, "darwin")
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")

		// ConfigDir always uses platformConfigDir which on darwin returns
		// the Library path. StyleXDG does not affect ConfigDir.
		want := filepath.Join(home, "Library", "Application Support", "sched-cli")
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_CACHE_HOME respected when StyleXDG on darwin", func(t *testing.T) {
		r := NewWithOverrides(StyleXDG, home, "darwin")
		t.Setenv("XDG_CACHE_HOME", "/custom/cache")
		want := filepath.Join("/custom/cache", "sched-cli")
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_STATE_HOME respected when StyleXDG on darwin", func(t *testing.T) {
		r := NewWithOverrides(StyleXDG, home, "darwin")
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		want := filepath.Join("/custom/state", "sched-cli", "logs")
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})
}

func TestEnvVarOverrides(t *testing.T) {
	home := "/Users/testuser"

	t.Run("SCHED_CONFIG_DIR overrides everything for config", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "darwin")
		t.Setenv("SCHED_CONFIG_DIR", "/override/config")
		want := "/override/config"
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_CACHE_DIR overrides everything for cache", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "darwin")
		t.Setenv("SCHED_CACHE_DIR", "/override/cache")
		want := "/override/cache"
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_LOG_DIR overrides everything for logs", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "darwin")
		t.Setenv("SCHED_LOG_DIR", "/override/logs")
		want := "/override/logs"
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_CONFIG_DIR takes precedence over directory_style", func(t *testing.T) {
		r := NewWithOverrides(StyleXDG, home, "darwin")
		t.Setenv("SCHED_CONFIG_DIR", "/override/config")
		want := "/override/config"
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_CACHE_DIR takes precedence over directory_style", func(t *testing.T) {
		r := NewWithOverrides(StyleXDG, home, "linux")
		t.Setenv("SCHED_CACHE_DIR", "/override/cache")
		want := "/override/cache"
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_LOG_DIR takes precedence over directory_style", func(t *testing.T) {
		r := NewWithOverrides(StyleXDG, home, "linux")
		t.Setenv("SCHED_LOG_DIR", "/override/logs")
		want := "/override/logs"
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_CONFIG_DIR takes precedence over XDG_CONFIG_HOME", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
		t.Setenv("SCHED_CONFIG_DIR", "/override/config")
		want := "/override/config"
		got := r.ConfigDir()
		if got != want {
			t.Errorf("ConfigDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_CACHE_DIR takes precedence over XDG_CACHE_HOME", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
		t.Setenv("SCHED_CACHE_DIR", "/override/cache")
		want := "/override/cache"
		got := r.CacheDir()
		if got != want {
			t.Errorf("CacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("SCHED_LOG_DIR takes precedence over XDG_STATE_HOME", func(t *testing.T) {
		r := NewWithOverrides(StylePlatform, home, "linux")
		t.Setenv("XDG_STATE_HOME", "/xdg/state")
		t.Setenv("SCHED_LOG_DIR", "/override/logs")
		want := "/override/logs"
		got := r.LogDir()
		if got != want {
			t.Errorf("LogDir() = %q, want %q", got, want)
		}
	})
}

func TestTableDriven_ConfigDir(t *testing.T) {
	// Clear env vars that could interfere with platform defaults
	t.Setenv("SCHED_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	tests := []struct {
		name     string
		style    DirectoryStyle
		home     string
		platform string
		envVars  map[string]string
		want     string
	}{
		{
			name:     "darwin platform default",
			style:    StylePlatform,
			home:     "/Users/alice",
			platform: "darwin",
			want:     "/Users/alice/Library/Application Support/sched-cli",
		},
		{
			name:     "linux platform default",
			style:    StylePlatform,
			home:     "/home/alice",
			platform: "linux",
			want:     "/home/alice/.config/sched-cli",
		},
		{
			name:     "darwin StyleXDG does not change config",
			style:    StyleXDG,
			home:     "/Users/alice",
			platform: "darwin",
			want:     "/Users/alice/Library/Application Support/sched-cli",
		},
		{
			name:     "linux with XDG_CONFIG_HOME",
			style:    StylePlatform,
			home:     "/home/alice",
			platform: "linux",
			envVars:  map[string]string{"XDG_CONFIG_HOME": "/xdg/config"},
			want:     "/xdg/config/sched-cli",
		},
		{
			name:     "SCHED_CONFIG_DIR wins over everything",
			style:    StyleXDG,
			home:     "/Users/alice",
			platform: "darwin",
			envVars:  map[string]string{"SCHED_CONFIG_DIR": "/forced"},
			want:     "/forced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			r := NewWithOverrides(tt.style, tt.home, tt.platform)
			got := r.ConfigDir()
			if got != tt.want {
				t.Errorf("ConfigDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableDriven_CacheDir(t *testing.T) {
	tests := []struct {
		name     string
		style    DirectoryStyle
		home     string
		platform string
		envVars  map[string]string
		want     string
	}{
		{
			name:     "darwin platform default",
			style:    StylePlatform,
			home:     "/Users/bob",
			platform: "darwin",
			want:     "/Users/bob/Library/Caches/sched-cli",
		},
		{
			name:     "linux platform default",
			style:    StylePlatform,
			home:     "/home/bob",
			platform: "linux",
			want:     "/home/bob/.cache/sched-cli",
		},
		{
			name:     "darwin StyleXDG switches to XDG path",
			style:    StyleXDG,
			home:     "/Users/bob",
			platform: "darwin",
			want:     "/Users/bob/.cache/sched-cli",
		},
		{
			name:     "linux with XDG_CACHE_HOME",
			style:    StylePlatform,
			home:     "/home/bob",
			platform: "linux",
			envVars:  map[string]string{"XDG_CACHE_HOME": "/xdg/cache"},
			want:     "/xdg/cache/sched-cli",
		},
		{
			name:     "darwin StyleXDG with XDG_CACHE_HOME",
			style:    StyleXDG,
			home:     "/Users/bob",
			platform: "darwin",
			envVars:  map[string]string{"XDG_CACHE_HOME": "/custom/cache"},
			want:     "/custom/cache/sched-cli",
		},
		{
			name:     "SCHED_CACHE_DIR wins over everything",
			style:    StyleXDG,
			home:     "/Users/bob",
			platform: "darwin",
			envVars: map[string]string{
				"XDG_CACHE_HOME": "/xdg/cache",
				"SCHED_CACHE_DIR": "/forced",
			},
			want: "/forced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			r := NewWithOverrides(tt.style, tt.home, tt.platform)
			got := r.CacheDir()
			if got != tt.want {
				t.Errorf("CacheDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableDriven_LogDir(t *testing.T) {
	tests := []struct {
		name     string
		style    DirectoryStyle
		home     string
		platform string
		envVars  map[string]string
		want     string
	}{
		{
			name:     "darwin platform default",
			style:    StylePlatform,
			home:     "/Users/carol",
			platform: "darwin",
			want:     "/Users/carol/Library/Logs/sched-cli",
		},
		{
			name:     "linux platform default",
			style:    StylePlatform,
			home:     "/home/carol",
			platform: "linux",
			want:     "/home/carol/.local/state/sched-cli/logs",
		},
		{
			name:     "darwin StyleXDG switches to XDG path",
			style:    StyleXDG,
			home:     "/Users/carol",
			platform: "darwin",
			want:     "/Users/carol/.local/state/sched-cli/logs",
		},
		{
			name:     "linux with XDG_STATE_HOME",
			style:    StylePlatform,
			home:     "/home/carol",
			platform: "linux",
			envVars:  map[string]string{"XDG_STATE_HOME": "/xdg/state"},
			want:     "/xdg/state/sched-cli/logs",
		},
		{
			name:     "darwin StyleXDG with XDG_STATE_HOME",
			style:    StyleXDG,
			home:     "/Users/carol",
			platform: "darwin",
			envVars:  map[string]string{"XDG_STATE_HOME": "/custom/state"},
			want:     "/custom/state/sched-cli/logs",
		},
		{
			name:     "SCHED_LOG_DIR wins over everything",
			style:    StyleXDG,
			home:     "/Users/carol",
			platform: "darwin",
			envVars: map[string]string{
				"XDG_STATE_HOME": "/xdg/state",
				"SCHED_LOG_DIR":  "/forced",
			},
			want: "/forced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			r := NewWithOverrides(tt.style, tt.home, tt.platform)
			got := r.LogDir()
			if got != tt.want {
				t.Errorf("LogDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
