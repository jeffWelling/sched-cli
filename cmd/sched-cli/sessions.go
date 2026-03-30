package main

import (
	"fmt"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/jeff/sched-cli/internal/store"
	"github.com/spf13/cobra"
)

var (
	sessionsTrack  string
	sessionsDay    string
	sessionsTime   string
	sessionsSearch string
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Browse and search sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions with optional filters",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "sessions-list")
		if err != nil {
			return err
		}
		defer a.Close()

		filters := store.SessionFilters{
			Track:  sessionsTrack,
			Day:    sessionsDay,
			Time:   sessionsTime,
			Search: sessionsSearch,
		}

		sessions, err := a.Store().ListSessions(filters)
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}

		return a.Output().FormatSessions(sessions)
	},
}

var sessionsShowCmd = &cobra.Command{
	Use:   "show SESSION_ID",
	Short: "Show detailed info for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "sessions-show")
		if err != nil {
			return err
		}
		defer a.Close()

		session, err := a.Store().GetSession(args[0])
		if err != nil {
			return fmt.Errorf("getting session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session '%s' not found", args[0])
		}

		return a.Output().FormatSessionDetail(*session)
	},
}

var sessionsSearchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search sessions by keyword",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "sessions-search")
		if err != nil {
			return err
		}
		defer a.Close()

		query := args[0]
		for i := 1; i < len(args); i++ {
			query += " " + args[i]
		}

		filters := store.SessionFilters{
			Search: query,
		}

		sessions, err := a.Store().ListSessions(filters)
		if err != nil {
			return fmt.Errorf("searching sessions: %w", err)
		}

		return a.Output().FormatSessions(sessions)
	},
}

func init() {
	sessionsListCmd.Flags().StringVar(&sessionsTrack, "track", "", "Filter by track/category")
	sessionsListCmd.Flags().StringVar(&sessionsDay, "day", "", "Filter by day (YYYY-MM-DD)")
	sessionsListCmd.Flags().StringVar(&sessionsTime, "time", "", "Filter by time range")
	sessionsListCmd.Flags().StringVar(&sessionsSearch, "search", "", "Search in title and description")

	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.AddCommand(sessionsListCmd, sessionsShowCmd, sessionsSearchCmd)
}
