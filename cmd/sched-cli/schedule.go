package main

import (
	"fmt"
	"os"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage personal schedule",
	Long: `View and manage sessions on your personal Sched.com schedule.

Adding or removing sessions requires authentication. Changes are pushed
to the Sched API and also stored locally for offline access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var scheduleShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your scheduled sessions",
	Long: `Display all sessions currently on your personal schedule, sorted by
start time. Data comes from the local cache; run "sched-cli sync" to
refresh from Sched.com.`,
	Example: `  # View your full schedule
  sched-cli schedule show

  # Export schedule as JSON
  sched-cli schedule show --json-pretty`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "schedule-show")
		if err != nil {
			return err
		}
		defer a.Close()

		sessions, err := a.Store().GetSchedule()
		if err != nil {
			return fmt.Errorf("getting schedule: %w", err)
		}

		return a.Output().FormatSchedule(sessions)
	},
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add SESSION_ID [SESSION_ID...]",
	Short: "Add session(s) to your schedule",
	Long: `Add one or more sessions to your Sched.com schedule by hex ID. The change
is pushed to the Sched API immediately and the local cache is updated.

Requires authentication. Run "sched-cli config init" first if needed.
For local-only planning before committing, see "sched-cli interest add".`,
	Example: `  # Add a single session
  sched-cli schedule add e6f499540ac79243410b138edde13b1a

  # Add multiple sessions at once
  sched-cli schedule add e6f499540ac79243410b138edde13b1a abc123def456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "schedule-add")
		if err != nil {
			return err
		}
		defer a.Close()

		if err := a.RequireAuth(); err != nil {
			return err
		}

		// Resolve each ID to a hex ID.
		hexIDs := make([]string, 0, len(args))
		for _, id := range args {
			resolved, err := resolveSessionID(a, id)
			if err != nil {
				return err
			}
			hexIDs = append(hexIDs, resolved)
		}

		// Call the Sched API.
		resp, err := a.Client().AddToSchedule(hexIDs...)
		if err != nil {
			return fmt.Errorf("adding to schedule: %w", err)
		}

		// Update local store.
		for _, hexID := range hexIDs {
			if err := a.Store().AddToSchedule(hexID, "api"); err != nil {
				return fmt.Errorf("updating local schedule: %w", err)
			}
		}

		fmt.Fprintf(os.Stderr, "Added %d session(s) to schedule (status: %s)\n", len(hexIDs), resp.Status)
		return nil
	},
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove SESSION_ID [SESSION_ID...]",
	Short: "Remove session(s) from your schedule",
	Long: `Remove one or more sessions from your Sched.com schedule by hex ID. The
change is pushed to the Sched API immediately and the local cache is updated.

Requires authentication. This cannot be undone through sched-cli; use
"sched-cli schedule add" to re-add a removed session.`,
	Example: `  # Remove a single session
  sched-cli schedule remove e6f499540ac79243410b138edde13b1a

  # Remove multiple sessions at once
  sched-cli schedule remove e6f499540ac79243410b138edde13b1a abc123def456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "schedule-remove")
		if err != nil {
			return err
		}
		defer a.Close()

		if err := a.RequireAuth(); err != nil {
			return err
		}

		// Resolve each ID to a hex ID.
		hexIDs := make([]string, 0, len(args))
		for _, id := range args {
			resolved, err := resolveSessionID(a, id)
			if err != nil {
				return err
			}
			hexIDs = append(hexIDs, resolved)
		}

		// Call the Sched API.
		resp, err := a.Client().RemoveFromSchedule(hexIDs...)
		if err != nil {
			return fmt.Errorf("removing from schedule: %w", err)
		}

		// Update local store.
		for _, hexID := range hexIDs {
			if err := a.Store().RemoveFromSchedule(hexID); err != nil {
				return fmt.Errorf("updating local schedule: %w", err)
			}
		}

		fmt.Fprintf(os.Stderr, "Removed %d session(s) from schedule (status: %s)\n", len(hexIDs), resp.Status)
		return nil
	},
}

// resolveSessionID takes a short ID or hex ID and returns the hex ID.
func resolveSessionID(a *app.App, id string) (string, error) {
	session, err := a.Store().GetSession(id)
	if err != nil {
		return "", fmt.Errorf("looking up session '%s': %w", id, err)
	}
	if session == nil {
		// If it looks like a hex ID, use it directly (may not be cached yet).
		return id, nil
	}
	return session.HexID, nil
}

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleShowCmd, scheduleAddCmd, scheduleRemoveCmd)
}
