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
	Long:  "Flag sessions as interesting for local planning. Use 'interest push' to commit them to your Sched schedule.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var interestAddCmd = &cobra.Command{
	Use:   "add SESSION_ID [SESSION_ID...]",
	Short: "Flag session(s) as interested",
	Args:  cobra.MinimumNArgs(1),
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
	Args:  cobra.MinimumNArgs(1),
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
	Long:  "Add all locally-flagged interesting sessions to your Sched schedule, then move them from interests to the schedule table.",
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
