package main

import (
	"fmt"
	"os"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/spf13/cobra"
)

var interestCmd = &cobra.Command{
	Use:   "interest",
	Short: "Local what-if planning",
	Long: `Flag sessions as interesting for local what-if planning, without touching
your live Sched.com schedule. Build up a shortlist, compare with friends,
then push your final picks with "sched-cli interest push".

Interests are stored locally until pushed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var interestAddCmd = &cobra.Command{
	Use:   "add SESSION_ID [SESSION_ID...]",
	Short: "Flag session(s) as interested",
	Long: `Flag one or more sessions as interesting by hex ID. This is a local-only
operation — nothing is sent to Sched.com until you run "sched-cli interest push".

Use this to build a shortlist before committing to your schedule.`,
	Example: `  # Flag a single session
  sched-cli interest add e6f499540ac79243410b138edde13b1a

  # Flag multiple sessions
  sched-cli interest add e6f499540ac79243410b138edde13b1a abc123def456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "interest-add")
		if err != nil {
			return err
		}
		defer a.Close()

		for _, id := range args {
			hexID, err := resolveSessionID(a, id)
			if err != nil {
				return err
			}
			if err := a.Store().AddInterest(hexID); err != nil {
				return fmt.Errorf("adding interest for '%s': %w", id, err)
			}
		}

		fmt.Fprintf(os.Stderr, "Flagged %d session(s) as interested\n", len(args))
		return nil
	},
}

var interestRemoveCmd = &cobra.Command{
	Use:   "remove SESSION_ID [SESSION_ID...]",
	Short: "Remove interest flag from session(s)",
	Long: `Remove the interest flag from one or more sessions by hex ID. This only
affects the local interest list; it does not remove the session from your
Sched.com schedule if it was already pushed.`,
	Example: `  # Remove interest from a session
  sched-cli interest remove e6f499540ac79243410b138edde13b1a`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "interest-remove")
		if err != nil {
			return err
		}
		defer a.Close()

		for _, id := range args {
			hexID, err := resolveSessionID(a, id)
			if err != nil {
				return err
			}
			if err := a.Store().RemoveInterest(hexID); err != nil {
				return fmt.Errorf("removing interest for '%s': %w", id, err)
			}
		}

		fmt.Fprintf(os.Stderr, "Removed interest from %d session(s)\n", len(args))
		return nil
	},
}

var interestListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions flagged as interesting",
	Long: `List all sessions currently flagged as interesting. These are sessions you
are considering but have not yet committed to your Sched.com schedule.

Use "sched-cli interest push" to add all of them to your live schedule.`,
	Example: `  # View your interest list
  sched-cli interest list

  # Output as JSON
  sched-cli interest list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "interest-list")
		if err != nil {
			return err
		}
		defer a.Close()

		sessions, err := a.Store().ListInterests()
		if err != nil {
			return fmt.Errorf("listing interests: %w", err)
		}

		return a.Output().FormatSessions(sessions)
	},
}

var interestPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push all interests to your Sched schedule",
	Long: `Add all locally-flagged interesting sessions to your Sched.com schedule in
one batch. Each session is pushed to the Sched API and moved from the
interests table to the schedule table locally.

Requires authentication. Sessions that fail to push are reported but do
not stop the remaining sessions from being processed.`,
	Example: `  # Push all interests to your live schedule
  sched-cli interest push`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "interest-push")
		if err != nil {
			return err
		}
		defer a.Close()

		if err := a.RequireAuth(); err != nil {
			return err
		}

		interests, err := a.Store().ListInterests()
		if err != nil {
			return fmt.Errorf("listing interests: %w", err)
		}

		if len(interests) == 0 {
			fmt.Fprintln(os.Stderr, "No interests to push.")
			return nil
		}

		var succeeded, failed int
		for _, sess := range interests {
			_, err := a.Client().AddToSchedule(sess.HexID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to add '%s': %v\n", sess.Title, err)
				failed++
				continue
			}

			// Move from interests to schedule.
			if err := a.Store().AddToSchedule(sess.HexID, "interest-push"); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update local schedule for '%s': %v\n", sess.Title, err)
				failed++
				continue
			}
			if err := a.Store().RemoveInterest(sess.HexID); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to remove interest flag for '%s': %v\n", sess.Title, err)
			}
			succeeded++
		}

		fmt.Fprintf(os.Stderr, "Pushed %d session(s) to schedule", succeeded)
		if failed > 0 {
			fmt.Fprintf(os.Stderr, " (%d failed)", failed)
		}
		fmt.Fprintln(os.Stderr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(interestCmd)
	interestCmd.AddCommand(interestAddCmd, interestRemoveCmd, interestListCmd, interestPushCmd)
}
