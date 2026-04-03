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
	Long: `Browse and search sessions for the active event.

Requires a prior sync. Run "sched-cli sync" to pull session data first.
Subcommands let you list with filters, show full details, or search by keyword.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions with optional filters",
	Long: `List all sessions for the active event, with optional filtering by track,
day, time range, or keyword search. Results are sorted by start time.

Requires a prior sync. Run "sched-cli sync" to pull session data first.
Output is a formatted table in terminal, or JSON when piped or with --json.`,
	Example: `  # List all sessions
  sched-cli sessions list

  # Filter by track
  sched-cli sessions list --track "PLENARY"

  # Sessions on a specific day
  sched-cli sessions list --day 2026-03-25

  # Morning sessions (times are UTC)
  sched-cli sessions list --time 16:00-19:00

  # Search by keyword
  sched-cli sessions list --search "kubernetes"`,
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
	Long: `Show the full details for a single session, including title, description,
speakers, track, location, and time. Accepts a hex session ID.

Use "sched-cli sessions list" or "sched-cli sessions search" to find IDs.`,
	Example: `  # Show session details by hex ID
  sched-cli sessions show e6f499540ac79243410b138edde13b1a

  # Output as JSON for scripting
  sched-cli sessions show e6f499540ac79243410b138edde13b1a --json`,
	Args: cobra.ExactArgs(1),
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
	Long: `Search session titles and descriptions for a keyword or phrase. Multiple
words are joined into a single query. Results are sorted by start time.

This is a convenience shortcut for "sched-cli sessions list --search QUERY".`,
	Example: `  # Search for SRE-related sessions
  sched-cli sessions search SRE

  # Multi-word search
  sched-cli sessions search incident response

  # Combine with JSON output
  sched-cli sessions search observability --json`,
	Args: cobra.MinimumNArgs(1),
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
