package main

import (
	"fmt"
	"os"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull fresh data from Sched",
	Long: `Fetch all sessions and your personal schedule from the Sched API and store
them in the local cache. This downloads the full iCal feed for the event
and your personal schedule.

Run this after "sched-cli config init" and periodically to pick up new
sessions or schedule changes made on the Sched.com website. Requires
authentication.`,
	Example: `  # Pull fresh data from Sched
  sched-cli sync

  # Sync with debug output to see API calls
  sched-cli sync --debug`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "sync")
		if err != nil {
			return err
		}
		defer a.Close()

		if err := a.RequireAuth(); err != nil {
			return err
		}

		// Fetch all sessions from /all.ics.
		fmt.Fprintln(os.Stderr, "Fetching all sessions...")
		sessions, err := a.Client().FetchAllSessions()
		if err != nil {
			return fmt.Errorf("fetching sessions: %w", err)
		}

		for _, sess := range sessions {
			if err := a.Store().UpsertSession(sess); err != nil {
				return fmt.Errorf("storing session '%s': %w", sess.Title, err)
			}
		}

		// Fetch user's schedule from /{username}.ics.
		username := a.Config().Username
		var scheduleCount int
		if username != "" {
			fmt.Fprintln(os.Stderr, "Fetching your schedule...")
			scheduled, err := a.Client().FetchUserSchedule(username)
			if err != nil {
				return fmt.Errorf("fetching your schedule: %w", err)
			}

			for _, sess := range scheduled {
				if err := a.Store().AddToSchedule(sess.HexID, "sync"); err != nil {
					return fmt.Errorf("updating schedule: %w", err)
				}
			}
			scheduleCount = len(scheduled)
		}

		fmt.Fprintf(os.Stderr, "Synced %d sessions", len(sessions))
		if username != "" {
			fmt.Fprintf(os.Stderr, ", %d on your schedule", scheduleCount)
		}
		fmt.Fprintln(os.Stderr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
