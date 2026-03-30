package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jeff/sched-cli/internal/store"
	"golang.org/x/term"
)

// Formatter handles output rendering in table or JSON format.
type Formatter struct {
	w          io.Writer
	jsonMode   bool
	prettyJSON bool
}

// CompareResult holds the result of a schedule comparison.
type CompareResult struct {
	Overlaps []OverlapEntry `json:"overlaps,omitempty"`
	Gaps     []GapEntry     `json:"gaps,omitempty"`
	Matrix   []MatrixRow    `json:"matrix,omitempty"`
}

// OverlapEntry is a session where 2+ people are attending.
type OverlapEntry struct {
	Session   store.Session `json:"session"`
	Attendees []string      `json:"attendees"`
}

// GapEntry is a session flagged as interesting but nobody committed to.
type GapEntry struct {
	Session      store.Session `json:"session"`
	InterestedBy []string      `json:"interested_by"`
}

// MatrixRow shows a session with attendance flags per person.
type MatrixRow struct {
	Session    store.Session      `json:"session"`
	Attendance map[string]string  `json:"attendance"` // person -> "going"/"interested"/""
}

// RateStatus mirrors rate.Status for output purposes.
type RateStatus struct {
	CallsInWindow int     `json:"calls_in_window"`
	Limit         int     `json:"limit"`
	BudgetUsed    float64 `json:"budget_used"`
	Remaining     int     `json:"remaining"`
}

// New creates a Formatter that writes to w. If jsonMode is true, all output is JSON.
// If prettyJSON is true, JSON output is indented; otherwise it is compact NDJSON.
func New(w io.Writer, jsonMode bool, prettyJSON bool) *Formatter {
	return &Formatter{w: w, jsonMode: jsonMode, prettyJSON: prettyJSON}
}

// AutoDetect creates a Formatter, choosing JSON if stdout is not a terminal.
// prettyJSON controls whether JSON output is indented (--json-pretty) or compact NDJSON.
func AutoDetect(jsonMode bool, prettyJSON bool) *Formatter {
	isTerminal := IsTerminal(os.Stdout.Fd())
	return &Formatter{
		w:          os.Stdout,
		jsonMode:   jsonMode || !isTerminal,
		prettyJSON: prettyJSON,
	}
}

// IsTerminal returns true if the file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// FormatSessions renders a list of sessions.
func (f *Formatter) FormatSessions(sessions []store.Session) error {
	if f.jsonMode {
		return f.writeJSON(sessions)
	}
	return f.writeSessionTable(sessions)
}

// FormatSchedule renders the user's schedule.
func (f *Formatter) FormatSchedule(sessions []store.Session) error {
	if f.jsonMode {
		return f.writeJSON(sessions)
	}
	return f.writeSessionTable(sessions)
}

// FormatFriends renders the friends list.
func (f *Formatter) FormatFriends(friends []store.Friend) error {
	if f.jsonMode {
		return f.writeJSON(friends)
	}

	tw := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NICKNAME\tUSERNAME")
	for _, friend := range friends {
		fmt.Fprintf(tw, "%s\t%s\n", friend.Nickname, friend.Username)
	}
	return tw.Flush()
}

// FormatComparison renders a schedule comparison.
func (f *Formatter) FormatComparison(result CompareResult) error {
	if f.jsonMode {
		return f.writeJSON(result)
	}

	// Overlaps
	if len(result.Overlaps) > 0 {
		fmt.Fprintln(f.w, "OVERLAPS (2+ attending):")
		tw := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ID\tTITLE\tATTENDEES")
		for _, o := range result.Overlaps {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", o.Session.ShortID, o.Session.Title, strings.Join(o.Attendees, ", "))
		}
		tw.Flush()
		fmt.Fprintln(f.w)
	}

	// Gaps
	if len(result.Gaps) > 0 {
		fmt.Fprintln(f.w, "GAPS (interested but not committed):")
		tw := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ID\tTITLE\tINTERESTED")
		for _, g := range result.Gaps {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", g.Session.ShortID, g.Session.Title, strings.Join(g.InterestedBy, ", "))
		}
		tw.Flush()
	}

	return nil
}

// FormatRateStatus renders rate limit information.
func (f *Formatter) FormatRateStatus(status RateStatus) error {
	if f.jsonMode {
		return f.writeJSON(status)
	}

	fmt.Fprintf(f.w, "API Rate Limit Status\n")
	fmt.Fprintf(f.w, "  Calls in window: %d/%d\n", status.CallsInWindow, status.Limit)
	fmt.Fprintf(f.w, "  Budget used:     %.0f%%\n", status.BudgetUsed*100)
	fmt.Fprintf(f.w, "  Remaining:       %d\n", status.Remaining)
	return nil
}

// FormatSessionDetail renders detailed info for a single session.
func (f *Formatter) FormatSessionDetail(session store.Session) error {
	if f.jsonMode {
		return f.writeJSON(session)
	}

	fmt.Fprintf(f.w, "%s  %s\n", session.ShortID, session.Title)
	fmt.Fprintf(f.w, "  Time:     %s - %s\n", session.StartTime.Format("Mon Jan 2 15:04"), session.EndTime.Format("15:04"))
	fmt.Fprintf(f.w, "  Location: %s\n", session.Location)
	fmt.Fprintf(f.w, "  Track:    %s\n", session.Category)
	if session.Description != "" {
		fmt.Fprintf(f.w, "\n%s\n", session.Description)
	}
	return nil
}

func (f *Formatter) writeSessionTable(sessions []store.Session) error {
	tw := tabwriter.NewWriter(f.w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTIME\tTITLE\tLOCATION\tTRACK")
	for _, s := range sessions {
		timeStr := s.StartTime.Format("Mon 15:04")
		title := truncate(s.Title, 50)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", s.ShortID, timeStr, title, s.Location, s.Category)
	}
	return tw.Flush()
}

func (f *Formatter) writeJSON(v interface{}) error {
	enc := json.NewEncoder(f.w)
	if f.prettyJSON {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
