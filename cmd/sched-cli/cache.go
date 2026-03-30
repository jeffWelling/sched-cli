package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Cache management",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "cache-status")
		if err != nil {
			return err
		}
		defer a.Close()

		count, err := a.Store().SessionCount()
		if err != nil {
			return fmt.Errorf("counting sessions: %w", err)
		}

		cacheDir := a.Paths().CacheDir()
		dbPath := filepath.Join(cacheDir, "sched.db")

		var dbSize int64
		if info, err := os.Stat(dbPath); err == nil {
			dbSize = info.Size()
		}

		fmt.Printf("Cache directory: %s\n", cacheDir)
		fmt.Printf("Database:        %s\n", dbPath)
		fmt.Printf("Database size:   %s\n", formatBytes(dbSize))
		fmt.Printf("Sessions cached: %d\n", count)
		return nil
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached data",
	Long:  "Remove the cache database. It will be recreated on next use.",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "cache-clear")
		if err != nil {
			return err
		}

		cacheDir := a.Paths().CacheDir()
		dbPath := filepath.Join(cacheDir, "sched.db")

		// Close the store before removing the file.
		a.Close()

		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing cache database: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Cache cleared.")
		return nil
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatusCmd, cacheClearCmd)
}
