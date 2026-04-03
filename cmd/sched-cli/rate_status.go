package main

import (
	"fmt"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/jeff/sched-cli/internal/output"
	"github.com/spf13/cobra"
)

var rateStatusCmd = &cobra.Command{
	Use:   "rate-status",
	Short: "Show API rate limit usage",
	Long: `Show how many API calls have been made in the current rate-limit window,
the configured limit, and how much budget remains. sched-cli enforces a
local rate limiter to avoid hammering the Sched.com web interface.

Use this to check whether you are close to the limit before running
a batch of operations.`,
	Example: `  # Check rate limit status
  sched-cli rate-status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "rate-status")
		if err != nil {
			return err
		}
		defer a.Close()

		status, err := a.Limiter().Status()
		if err != nil {
			return fmt.Errorf("getting rate status: %w", err)
		}

		return a.Output().FormatRateStatus(output.RateStatus{
			CallsInWindow: status.CallsInWindow,
			Limit:         status.Limit,
			BudgetUsed:    status.BudgetUsed,
			Remaining:     status.Remaining,
		})
	},
}

func init() {
	rootCmd.AddCommand(rateStatusCmd)
}
