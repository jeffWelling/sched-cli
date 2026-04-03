package main

import (
	"fmt"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/jeff/sched-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	compareWith     []string
	compareWithUser []string
	compareOverlap  bool
	compareGaps     bool
	compareAll      bool
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Cross-schedule analysis",
	Long: `Compare your schedule with one or more friends to find overlapping sessions
and gaps. Overlaps are sessions where two or more people are attending. Gaps
are sessions you flagged as interesting but have not scheduled yet.

Friends are identified by nickname (see "sched-cli friends") or by raw
Sched username with --with-user. Friend schedules are fetched from Sched
and cached locally. By default both overlaps and gaps are shown; use
--overlap or --gaps to filter.`,
	Example: `  # Compare with a friend by nickname
  sched-cli compare --with alice

  # Compare with multiple friends
  sched-cli compare --with alice --with bob

  # Compare by raw Sched username
  sched-cli compare --with-user alice.smith42

  # Show only overlapping sessions
  sched-cli compare --with alice --overlap

  # Show only gaps (interested but not scheduled)
  sched-cli compare --with alice --gaps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "compare")
		if err != nil {
			return err
		}
		defer a.Close()

		if err := a.RequireAuth(); err != nil {
			return err
		}

		// 1. Resolve --with nicknames to usernames.
		var usernames []string
		for _, nick := range compareWith {
			friend, err := a.Store().GetFriendByNickname(nick)
			if err != nil {
				return fmt.Errorf("looking up friend '%s': %w", nick, err)
			}
			if friend == nil {
				return fmt.Errorf("friend '%s' not found. Use 'sched-cli friends add' first", nick)
			}
			usernames = append(usernames, friend.Username)
		}

		// 2. Add --with-user direct usernames.
		usernames = append(usernames, compareWithUser...)

		if len(usernames) == 0 {
			return fmt.Errorf("specify at least one friend with --with or --with-user")
		}

		// 3. Fetch friend schedules.
		friendSchedules := make(map[string]map[string]bool) // username -> set of hex IDs
		for _, username := range usernames {
			sessions, err := a.Client().FetchUserSchedule(username)
			if err != nil {
				return fmt.Errorf("fetching schedule for %s: %w", username, err)
			}

			hexIDs := make([]string, 0, len(sessions))
			idSet := make(map[string]bool)
			for _, s := range sessions {
				hexIDs = append(hexIDs, s.HexID)
				idSet[s.HexID] = true
			}

			// 4. Store in friend_schedules table.
			if err := a.Store().UpsertFriendSchedule(username, hexIDs); err != nil {
				return fmt.Errorf("storing schedule for %s: %w", username, err)
			}
			friendSchedules[username] = idSet
		}

		// 5. Get user's own schedule.
		mySchedule, err := a.Store().GetSchedule()
		if err != nil {
			return fmt.Errorf("getting your schedule: %w", err)
		}
		myScheduleMap := make(map[string]bool)
		for _, s := range mySchedule {
			myScheduleMap[s.HexID] = true
		}

		// 6. Get user's interests.
		myInterests, err := a.Store().ListInterests()
		if err != nil {
			return fmt.Errorf("getting your interests: %w", err)
		}
		myInterestMap := make(map[string]bool)
		for _, s := range myInterests {
			myInterestMap[s.HexID] = true
		}

		// 7. Build CompareResult.
		result := buildCompareResult(a, myScheduleMap, myInterestMap, friendSchedules, usernames)

		// 8. Filter based on display flags. Default = --all.
		showAll := compareAll || (!compareOverlap && !compareGaps)

		filtered := output.CompareResult{}
		if showAll || compareOverlap {
			filtered.Overlaps = result.Overlaps
		}
		if showAll || compareGaps {
			filtered.Gaps = result.Gaps
		}

		return a.Output().FormatComparison(filtered)
	},
}

func buildCompareResult(a *app.App, mySchedule, myInterests map[string]bool, friendSchedules map[string]map[string]bool, usernames []string) output.CompareResult {
	var result output.CompareResult

	// Collect all hex IDs across everyone.
	allIDs := make(map[string]bool)
	for id := range mySchedule {
		allIDs[id] = true
	}
	for _, sched := range friendSchedules {
		for id := range sched {
			allIDs[id] = true
		}
	}

	for hexID := range allIDs {
		// Who is attending?
		var attendees []string
		if mySchedule[hexID] {
			attendees = append(attendees, "you")
		}
		for _, username := range usernames {
			if friendSchedules[username][hexID] {
				attendees = append(attendees, username)
			}
		}

		session, err := a.Store().GetSession(hexID)
		if err != nil || session == nil {
			// Session not in local cache; skip.
			continue
		}

		// Overlap: 2+ people attending.
		if len(attendees) >= 2 {
			result.Overlaps = append(result.Overlaps, output.OverlapEntry{
				Session:   *session,
				Attendees: attendees,
			})
		}

		// Gap: interested but not on schedule.
		if myInterests[hexID] && !mySchedule[hexID] {
			var interestedBy []string
			interestedBy = append(interestedBy, "you")
			for _, username := range usernames {
				if friendSchedules[username][hexID] {
					interestedBy = append(interestedBy, username)
				}
			}
			result.Gaps = append(result.Gaps, output.GapEntry{
				Session:      *session,
				InterestedBy: interestedBy,
			})
		}
	}

	return result
}

func init() {
	compareCmd.Flags().StringSliceVar(&compareWith, "with", nil, "Friend nicknames to compare with")
	compareCmd.Flags().StringSliceVar(&compareWithUser, "with-user", nil, "Raw Sched usernames to compare with")
	compareCmd.Flags().BoolVar(&compareOverlap, "overlap", false, "Show only overlapping sessions")
	compareCmd.Flags().BoolVar(&compareGaps, "gaps", false, "Show only gap sessions")
	compareCmd.Flags().BoolVar(&compareAll, "all", false, "Show all comparisons (default)")

	rootCmd.AddCommand(compareCmd)
}
