package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	Use:   "sched-cli",
	Short: "CLI tool for Sched.com conference schedules",
	Long:  "Browse sessions, manage your schedule, compare with friends, and plan conference attendance from the terminal.",
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
