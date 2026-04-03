package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Set by GoReleaser ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	jsonFlag      bool
	jsonPretty    bool
	noCache       bool
	cacheOnly     bool
	refresh       bool
	cacheTTL      string
	debug         bool
	eventOverride string
)

var rootCmd = &cobra.Command{
	Use:     "sched-cli",
	Short:   "CLI tool for Sched.com conference schedules",
	Version: version,
	Long: `Browse sessions, manage your schedule, compare with friends, and plan
conference attendance from the terminal.

Typical workflow:
  1. sched-cli config init    — authenticate with Sched.com
  2. sched-cli sync           — pull sessions and your schedule
  3. sched-cli sessions list  — browse available sessions
  4. sched-cli interest add   — flag sessions for local what-if planning
  5. sched-cli interest push  — commit your picks to the live schedule

Use --json or --json-pretty on any command to get machine-readable output.
Use --event to temporarily target a different Sched event.`,
	Example: `  # Get started with a new event
  sched-cli config init
  sched-cli sync

  # Browse and plan
  sched-cli sessions list --track "PLENARY"
  sched-cli interest add e6f499540ac79243410b138edde13b1a
  sched-cli compare --with alice`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Validate conflicting flags
		if noCache && cacheOnly {
			return fmt.Errorf("cannot use --no-cache and --cache-only together")
		}
		if refresh && cacheOnly {
			return fmt.Errorf("cannot use --refresh and --cache-only together")
		}
		// --json-pretty implies --json
		if jsonPretty {
			jsonFlag = true
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Force JSON output")
	rootCmd.PersistentFlags().BoolVar(&jsonPretty, "json-pretty", false, "Force indented JSON output")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Don't read or write cache")
	rootCmd.PersistentFlags().BoolVar(&cacheOnly, "cache-only", false, "Never touch network")
	rootCmd.PersistentFlags().BoolVar(&refresh, "refresh", false, "Bypass cache for this request")
	rootCmd.PersistentFlags().StringVar(&cacheTTL, "cache-ttl", "", "Override default cache TTL (e.g., 1h, 30m)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Verbose logging (includes all API I/O)")
	rootCmd.PersistentFlags().StringVar(&eventOverride, "event", "", "Override active event URL")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
