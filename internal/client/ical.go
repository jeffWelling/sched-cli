package client

import (
	"bufio"
	"strings"
	"time"

	"github.com/jeff/sched-cli/internal/store"
)

// ParseICalFeed parses iCal data into sessions.
// If username is non-empty, UIDs are expected in "hexid@username" format (personal schedules).
// If username is empty, UIDs are just hex IDs (full schedule from /all.ics).
func ParseICalFeed(data []byte, username string) ([]store.Session, error) {
	var sessions []store.Session
	var current *store.Session

	// First pass: unfold continuation lines (RFC 5545 line folding).
	// Lines starting with a space or tab are appended to the previous line.
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var lines []string
	for scanner.Scan() {
		text := scanner.Text()
		if len(lines) > 0 && (strings.HasPrefix(text, " ") || strings.HasPrefix(text, "\t")) {
			lines[len(lines)-1] += strings.TrimLeft(text, " \t")
		} else {
			lines = append(lines, text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Second pass: process each unfolded line.
	for _, line := range lines {
		processICalLine(line, &current, &sessions, username)
	}

	return sessions, nil
}

func processICalLine(line string, current **store.Session, sessions *[]store.Session, username string) {
	switch {
	case line == "BEGIN:VEVENT":
		s := &store.Session{FetchedAt: time.Now().UTC()}
		*current = s

	case line == "END:VEVENT":
		if *current != nil {
			*sessions = append(*sessions, **current)
			*current = nil
		}

	case *current == nil:
		return

	case strings.HasPrefix(line, "SUMMARY:"):
		(*current).Title = unescapeICal(strings.TrimPrefix(line, "SUMMARY:"))

	case strings.HasPrefix(line, "DESCRIPTION:"):
		(*current).Description = unescapeICal(strings.TrimPrefix(line, "DESCRIPTION:"))

	case strings.HasPrefix(line, "LOCATION:"):
		(*current).Location = unescapeICal(strings.TrimPrefix(line, "LOCATION:"))

	case strings.HasPrefix(line, "CATEGORIES:"):
		(*current).Category = strings.TrimPrefix(line, "CATEGORIES:")

	case strings.HasPrefix(line, "DTSTART:"):
		if t, err := parseICalTime(strings.TrimPrefix(line, "DTSTART:")); err == nil {
			(*current).StartTime = t
		}

	case strings.HasPrefix(line, "DTEND:"):
		if t, err := parseICalTime(strings.TrimPrefix(line, "DTEND:")); err == nil {
			(*current).EndTime = t
		}

	case strings.HasPrefix(line, "UID:"):
		uid := strings.TrimPrefix(line, "UID:")
		if username != "" && strings.Contains(uid, "@") {
			// Personal schedule: "hexid@username"
			(*current).HexID = strings.Split(uid, "@")[0]
		} else {
			(*current).HexID = uid
		}

	case strings.HasPrefix(line, "URL:"):
		urlStr := strings.TrimPrefix(line, "URL:")
		(*current).EventURL = urlStr
		// Extract short ID from URL: .../event/{shortid} or .../event/{hexid}
		if idx := strings.LastIndex(urlStr, "/event/"); idx >= 0 {
			id := urlStr[idx+7:]
			// Short IDs are typically 5 chars alphanumeric; hex IDs are 32 chars
			if len(id) < 32 {
				(*current).ShortID = id
			}
		}
	}
}

func parseICalTime(s string) (time.Time, error) {
	// Try UTC format first: 20260324T143000Z
	if t, err := time.Parse("20060102T150405Z", s); err == nil {
		return t, nil
	}
	// Try local format: 20260324T143000
	if t, err := time.Parse("20060102T150405", s); err == nil {
		return t, nil
	}
	// Try date-only: 20260324
	return time.Parse("20060102", s)
}

func unescapeICal(s string) string {
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
