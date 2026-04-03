# sched-cli

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-365%20passing-brightgreen)](#testing)

A CLI tool for [Sched.com](https://sched.com) conference schedules. Browse sessions, manage your schedule, compare with friends, and plan conference attendance from the terminal.

Designed for humans. Optimized for AI agent usage (Claude Code, Gemini CLI).

Single static binary. No runtime dependencies.

## Install

```bash
go install github.com/jeffWelling/sched-cli/cmd/sched-cli@latest
```

Or build from source:

```bash
git clone https://github.com/jeffWelling/sched-cli.git
cd sched-cli
go build -o sched-cli ./cmd/sched-cli/
```

## Quick Start

```bash
# Set up authentication (interactive menu)
sched-cli config init

# Or provide a token directly
sched-cli config init --token <your-sched-token>

# Set your event
sched-cli config set event_url https://srecon26americas.sched.com
sched-cli config set username your.username

# Pull session data
sched-cli sync

# Browse sessions
sched-cli sessions list
sched-cli sessions list --search "kubernetes"
sched-cli sessions list --day 2026-03-24
sched-cli sessions list --time 09:00-12:00

# View your schedule
sched-cli schedule show

# Add a session to your schedule
sched-cli schedule add e6f499540ac79243410b138edde13b1a

# Remove a session
sched-cli schedule remove e6f499540ac79243410b138edde13b1a
```

## Commands

### Sessions

```bash
sched-cli sessions list                          # List all sessions
sched-cli sessions list --search "observability" # Search by keyword
sched-cli sessions list --track "Track 1"        # Filter by track
sched-cli sessions list --day 2026-03-25         # Filter by date
sched-cli sessions list --time 09:00-12:00       # Filter by time range
sched-cli sessions show <session-id>             # Show session details
sched-cli sessions search <query>                # Search sessions
```

### Schedule

```bash
sched-cli schedule show                   # Show your schedule
sched-cli schedule add <id> [<id>...]     # Add session(s) to schedule
sched-cli schedule remove <id> [<id>...]  # Remove session(s) from schedule
```

### Friends

```bash
sched-cli friends add alice alice.username  # Add a friend (nickname -> Sched username)
sched-cli friends remove alice              # Remove a friend
sched-cli friends list                      # List all friends
```

### Compare

```bash
sched-cli compare --with alice,bob           # Compare with friends by nickname
sched-cli compare --with-user some.person    # Compare with any Sched username
sched-cli compare --with alice --overlap     # Show sessions 2+ people attend
sched-cli compare --with alice --gaps        # Show sessions someone flagged but nobody committed to
```

### Interest (Local Planning)

```bash
sched-cli interest add <id>      # Flag a session as "interested" (local only)
sched-cli interest remove <id>   # Remove interest flag
sched-cli interest list          # List interested sessions
sched-cli interest push          # Commit all interests to your Sched schedule
```

### Sync & Cache

```bash
sched-cli sync           # Pull fresh data from Sched
sched-cli cache status   # Show cache health
sched-cli cache clear    # Clear cached data
sched-cli rate-status    # Show API rate limit usage
```

### Config

```bash
sched-cli config init              # Interactive setup
sched-cli config show              # Show current config
sched-cli config set <key> <value> # Set a config value
sched-cli config login             # Re-authenticate
```

## Authentication

Four methods, presented as an interactive menu (or via flags for headless/agent use):

| Method | Flag | Description |
|--------|------|-------------|
| Email + password | `--username <email> --password <pass>` | Direct login to sched.com |
| Token paste | `--token <value>` | Provide session token directly |
| Browser loopback | `--browser` | Log in via browser, cookies captured via localhost |
| Firefox cookies | `--from-browser firefox` | Import cookies from Firefox's cookie database |

**Environment variables** (override config, checked on every command):

- `SCHED_TOKEN` — Session token, skips config init entirely
- `SCHED_EMAIL` + `SCHED_PASSWORD` — Auto-login credentials

**Headless mode:** When stdin is not a terminal, flags are required (no interactive menu). Auth errors are written to stderr with non-zero exit codes.

## Output

- **Terminal (TTY):** Human-readable tables
- **Piped/scripted (non-TTY):** Compact JSON (NDJSON) automatically
- `--json` — Force JSON output
- `--json-pretty` — Force indented JSON output

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Force JSON output |
| `--json-pretty` | Force indented JSON output |
| `--no-cache` | Don't read or write cache |
| `--cache-only` | Never touch network |
| `--refresh` | Bypass cache for this request |
| `--cache-ttl` | Override default 48h cache TTL |
| `--debug` | Verbose logging (all API I/O) |
| `--event` | Override active event URL |

## Directory Layout

Follows platform conventions. Configurable via `directory_style` setting.

| Data | macOS (default) | Linux / XDG |
|------|-----------------|-------------|
| Config | `~/Library/Application Support/sched-cli/` | `~/.config/sched-cli/` |
| Cache | `~/Library/Caches/sched-cli/` | `~/.cache/sched-cli/` |
| Logs | `~/Library/Logs/sched-cli/` | `~/.local/state/sched-cli/logs/` |

Override with env vars: `SCHED_CONFIG_DIR`, `SCHED_CACHE_DIR`, `SCHED_LOG_DIR`

Use XDG layout on macOS: `sched-cli config set directory_style xdg`

## Testing

```bash
go test ./...           # Run all 365+ tests
go test ./... -v        # Verbose output
go test ./... -count=1  # Disable test caching
```

Test coverage spans 10 packages:

| Package | Tests | Coverage |
|---------|-------|----------|
| `cmd/sched-cli` | CLI integration smoke tests | Binary builds, help output, flag validation |
| `internal/app` | App wiring, cache flow, auth errors | Config loading, store access, rate-budget caching |
| `internal/auth` | All 4 auth methods | Credentials, token, browser loopback, Firefox cookies |
| `internal/client` | HTTP client, iCal parser, HTML parser, retry | Real-world Sched data, line folding, escaping |
| `internal/config` | Config load/save, env vars | Round-trip, permissions, defaults |
| `internal/logging` | Structured logging, retention | Date-stamped dirs, cleanup, JSON format |
| `internal/output` | Table/JSON formatting, TTY detection | Compact NDJSON, pretty JSON, truncation |
| `internal/paths` | Platform-native directories | macOS, Linux, XDG override, env vars |
| `internal/rate` | Rate limiting | Rolling window, budget tracking, smart refresh |
| `internal/store` | SQLite CRUD, schema migration | Sessions, schedule, friends, interests, cache meta |

## Shell Completion

Cobra generates completions for bash, zsh, fish, and PowerShell:

```bash
# Bash
sched-cli completion bash > /usr/local/etc/bash_completion.d/sched-cli

# Zsh
sched-cli completion zsh > "${fpath[1]}/_sched-cli"

# Fish
sched-cli completion fish > ~/.config/fish/completions/sched-cli.fish
```

## Architecture

Single static Go binary. No external dependencies at runtime.

- **SQLite** (pure Go via `modernc.org/sqlite`) for caching and local state
- **Cobra** for CLI framework
- **goquery** for HTML parsing
- Hand-rolled **iCal parser** (two-pass: unfold then parse)
- **Rate-budget-aware caching** — smart refresh under 50% of 30 calls/min budget
- **Schema migrations** with version tracking

## Contributing

Issues and pull requests welcome at [github.com/jeffWelling/sched-cli](https://github.com/jeffWelling/sched-cli).

Before submitting:
```bash
go test ./...    # All tests must pass
go build ./...   # Must compile cleanly
```

## License

MIT
