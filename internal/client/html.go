package client

import (
	"bytes"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ScheduleEntry represents a session extracted from schedule HTML.
type ScheduleEntry struct {
	HexID      string
	ShortID    string
	Title      string
	OnSchedule bool // true if the user has this session on their schedule
}

// ParseScheduleHTML parses the HTML from /list/simple or /list/descriptions
// and extracts session entries.
func ParseScheduleHTML(data []byte) ([]ScheduleEntry, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var entries []ScheduleEntry

	doc.Find("span.event").Each(func(_ int, s *goquery.Selection) {
		entry := ScheduleEntry{
			OnSchedule: s.HasClass("sub"),
		}

		link := s.Find("a.name")
		if link.Length() == 0 {
			return
		}

		entry.HexID, _ = link.Attr("id")
		// Remove venue span (.vs) before extracting title text
		// Real Sched HTML: <a class="name">Title <span class="vs">Venue</span></a>
		linkClone := link.Clone()
		linkClone.Find("span.vs").Remove()
		entry.Title = strings.TrimSpace(linkClone.Text())

		if href, exists := link.Attr("href"); exists {
			entry.ShortID = extractShortID(href)
		}

		entries = append(entries, entry)
	})

	return entries, nil
}

// extractShortID parses a short ID from an href like "event/2J09B/taming-the-unpredictable".
func extractShortID(href string) string {
	// Trim leading slash if present
	href = strings.TrimPrefix(href, "/")

	parts := strings.SplitN(href, "/", 3)
	if len(parts) >= 2 && parts[0] == "event" {
		return parts[1]
	}
	return ""
}
