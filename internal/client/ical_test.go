package client

import (
	"testing"
	"time"
)

// --- Basic Parsing ---

func TestParseICalFeed_SingleVEVENTAllFields(t *testing.T) {
	ical := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Understanding eBPF
DESCRIPTION:Deep dive into eBPF internals
LOCATION:Room 101
CATEGORIES:Networking
UID:abc123def456
URL:http://srecon26.sched.com/event/2J09B
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}

	s := sessions[0]
	if s.Title != "Understanding eBPF" {
		t.Errorf("Title = %q, want %q", s.Title, "Understanding eBPF")
	}
	if s.Description != "Deep dive into eBPF internals" {
		t.Errorf("Description = %q, want %q", s.Description, "Deep dive into eBPF internals")
	}
	if s.Location != "Room 101" {
		t.Errorf("Location = %q, want %q", s.Location, "Room 101")
	}
	if s.Category != "Networking" {
		t.Errorf("Category = %q, want %q", s.Category, "Networking")
	}
	if s.HexID != "abc123def456" {
		t.Errorf("HexID = %q, want %q", s.HexID, "abc123def456")
	}
	if s.EventURL != "http://srecon26.sched.com/event/2J09B" {
		t.Errorf("EventURL = %q, want %q", s.EventURL, "http://srecon26.sched.com/event/2J09B")
	}
	if s.ShortID != "2J09B" {
		t.Errorf("ShortID = %q, want %q", s.ShortID, "2J09B")
	}

	wantStart := time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC)
	if !s.StartTime.Equal(wantStart) {
		t.Errorf("StartTime = %v, want %v", s.StartTime, wantStart)
	}
	wantEnd := time.Date(2026, 3, 24, 15, 30, 0, 0, time.UTC)
	if !s.EndTime.Equal(wantEnd) {
		t.Errorf("EndTime = %v, want %v", s.EndTime, wantEnd)
	}
}

func TestParseICalFeed_MultipleVEVENTs(t *testing.T) {
	ical := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260324T100000Z
DTEND:20260324T110000Z
SUMMARY:Session One
UID:aaa111
URL:http://example.sched.com/event/S1
END:VEVENT
BEGIN:VEVENT
DTSTART:20260324T110000Z
DTEND:20260324T120000Z
SUMMARY:Session Two
UID:bbb222
URL:http://example.sched.com/event/S2
END:VEVENT
BEGIN:VEVENT
DTSTART:20260324T120000Z
DTEND:20260324T130000Z
SUMMARY:Session Three
UID:ccc333
URL:http://example.sched.com/event/S3
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}
	if sessions[0].Title != "Session One" {
		t.Errorf("sessions[0].Title = %q, want %q", sessions[0].Title, "Session One")
	}
	if sessions[1].Title != "Session Two" {
		t.Errorf("sessions[1].Title = %q, want %q", sessions[1].Title, "Session Two")
	}
	if sessions[2].Title != "Session Three" {
		t.Errorf("sessions[2].Title = %q, want %q", sessions[2].Title, "Session Three")
	}
}

func TestParseICalFeed_EmptyInput(t *testing.T) {
	sessions, err := ParseICalFeed([]byte(""), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

func TestParseICalFeed_NonVEVENTContentIgnored(t *testing.T) {
	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Sched//Test//EN
X-WR-CALNAME:srecon26
BEGIN:VTIMEZONE
TZID:America/Los_Angeles
BEGIN:STANDARD
DTSTART:19701101T020000
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:The One Event
UID:onlyevent
URL:http://example.sched.com/event/ONE
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1 (VTIMEZONE and calendar properties should be ignored)", len(sessions))
	}
	if sessions[0].Title != "The One Event" {
		t.Errorf("Title = %q, want %q", sessions[0].Title, "The One Event")
	}
}

// --- UID Handling ---

func TestParseICalFeed_UID_FullSchedule(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:e6f499540ac79243410b138edde13b1a
URL:http://example.sched.com/event/e6f499540ac79243410b138edde13b1a
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].HexID != "e6f499540ac79243410b138edde13b1a" {
		t.Errorf("HexID = %q, want %q", sessions[0].HexID, "e6f499540ac79243410b138edde13b1a")
	}
}

func TestParseICalFeed_UID_PersonalSchedule(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:e6f499540ac79243410b138edde13b1a@jsmith
URL:http://example.sched.com/event/e6f499540ac79243410b138edde13b1a
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "jsmith")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].HexID != "e6f499540ac79243410b138edde13b1a" {
		t.Errorf("HexID = %q, want %q (@ username should be stripped)", sessions[0].HexID, "e6f499540ac79243410b138edde13b1a")
	}
}

func TestParseICalFeed_UID_PersonalScheduleNoUsername(t *testing.T) {
	// When username is empty, even UIDs with @ are treated as-is
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:abc123@someone
URL:http://example.sched.com/event/abc123
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if sessions[0].HexID != "abc123@someone" {
		t.Errorf("HexID = %q, want %q (no username means keep full UID)", sessions[0].HexID, "abc123@someone")
	}
}

// --- Short ID Extraction ---

func TestParseICalFeed_ShortIDFromURL(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:abc123
URL:http://event.sched.com/event/2J09B
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if sessions[0].ShortID != "2J09B" {
		t.Errorf("ShortID = %q, want %q", sessions[0].ShortID, "2J09B")
	}
}

func TestParseICalFeed_HexIDInURLNotShortID(t *testing.T) {
	// When the URL contains a 32-char hex ID, it should NOT be set as ShortID
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:e6f499540ac79243410b138edde13b1a
URL:http://srecon26.sched.com/event/e6f499540ac79243410b138edde13b1a
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if sessions[0].ShortID != "" {
		t.Errorf("ShortID = %q, want empty (hex IDs >= 32 chars should not be treated as short IDs)", sessions[0].ShortID)
	}
}

// --- Time Parsing ---

func TestParseICalFeed_TimeFormats(t *testing.T) {
	tests := []struct {
		name     string
		dtstart  string
		wantTime time.Time
	}{
		{
			name:     "UTC format",
			dtstart:  "20260324T143000Z",
			wantTime: time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC),
		},
		{
			name:     "local format",
			dtstart:  "20260324T143000",
			wantTime: time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC),
		},
		{
			name:     "date-only format",
			dtstart:  "20260324",
			wantTime: time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nDTSTART:" + tt.dtstart + "\nDTEND:" + tt.dtstart + "\nSUMMARY:Test\nUID:timefmt\nURL:http://example.sched.com/event/T1\nEND:VEVENT\nEND:VCALENDAR\n"
			sessions, err := ParseICalFeed([]byte(ical), "")
			if err != nil {
				t.Fatalf("ParseICalFeed() error: %v", err)
			}
			if len(sessions) != 1 {
				t.Fatalf("got %d sessions, want 1", len(sessions))
			}
			if !sessions[0].StartTime.Equal(tt.wantTime) {
				t.Errorf("StartTime = %v, want %v", sessions[0].StartTime, tt.wantTime)
			}
		})
	}
}

// --- iCal Escaping ---

func TestParseICalFeed_EscapedCommas(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
LOCATION:Grand Foyer \, Seattle\, WA
UID:esc1
URL:http://example.sched.com/event/E1
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	want := "Grand Foyer , Seattle, WA"
	if sessions[0].Location != want {
		t.Errorf("Location = %q, want %q", sessions[0].Location, want)
	}
}

func TestParseICalFeed_EscapedSemicolons(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
DESCRIPTION:A\;B\;C
UID:esc2
URL:http://example.sched.com/event/E2
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	want := "A;B;C"
	if sessions[0].Description != want {
		t.Errorf("Description = %q, want %q", sessions[0].Description, want)
	}
}

func TestParseICalFeed_EscapedNewlines(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
DESCRIPTION:Line one\nLine two\nLine three
UID:esc3
URL:http://example.sched.com/event/E3
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	want := "Line one\nLine two\nLine three"
	if sessions[0].Description != want {
		t.Errorf("Description = %q, want %q", sessions[0].Description, want)
	}
}

func TestParseICalFeed_EscapedBackslash(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
DESCRIPTION:path\\to\\file
UID:esc4
URL:http://example.sched.com/event/E4
END:VEVENT
END:VCALENDAR
`
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	want := "path\\to\\file"
	if sessions[0].Description != want {
		t.Errorf("Description = %q, want %q", sessions[0].Description, want)
	}
}

// --- Line Folding ---

func TestParseICalFeed_LineFoldingSpace(t *testing.T) {
	// RFC 5545: long lines may be folded by inserting CRLF + space
	ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nDTSTART:20260324T143000Z\nDTEND:20260324T153000Z\nSUMMARY:Very Long Sess\n ion Title Here\nUID:fold1\nURL:http://example.sched.com/event/F1\nEND:VEVENT\nEND:VCALENDAR\n"
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	want := "Very Long Session Title Here"
	if sessions[0].Title != want {
		t.Errorf("Title = %q, want %q", sessions[0].Title, want)
	}
}

func TestParseICalFeed_LineFoldingTab(t *testing.T) {
	// Continuation lines can also start with tab
	ical := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nDTSTART:20260324T143000Z\nDTEND:20260324T153000Z\nSUMMARY:Folded With\n\tTab Continuation\nUID:fold2\nURL:http://example.sched.com/event/F2\nEND:VEVENT\nEND:VCALENDAR\n"
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	want := "Folded WithTab Continuation"
	if sessions[0].Title != want {
		t.Errorf("Title = %q, want %q", sessions[0].Title, want)
	}
}

func TestParseICalFeed_LineFoldingDoesNotDropSubsequentEvents(t *testing.T) {
	// Regression test: when the first event has a folded line, the second
	// event's fields must not be lost. The original bug had a nested
	// scanner.Scan() loop whose `continue` re-entered the inner loop
	// instead of the outer one, causing non-continuation lines to be skipped.
	ical := "BEGIN:VCALENDAR\n" +
		"VERSION:2.0\n" +
		"BEGIN:VEVENT\n" +
		"DTSTART:20260324T100000Z\n" +
		"DTEND:20260324T110000Z\n" +
		"SUMMARY:First Event With A Ver\n" +
		" y Long Folded Title\n" +
		"DESCRIPTION:Firs\n" +
		" t description als\n" +
		"\to folded with tab\n" +
		"LOCATION:Room A\n" +
		"UID:event1hex\n" +
		"URL:http://example.sched.com/event/E1\n" +
		"END:VEVENT\n" +
		"BEGIN:VEVENT\n" +
		"DTSTART:20260324T120000Z\n" +
		"DTEND:20260324T130000Z\n" +
		"SUMMARY:Second Event\n" +
		"DESCRIPTION:Second description\n" +
		"LOCATION:Room B\n" +
		"CATEGORIES:Networking\n" +
		"UID:event2hex\n" +
		"URL:http://example.sched.com/event/E2\n" +
		"END:VEVENT\n" +
		"END:VCALENDAR\n"

	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	// Verify first event's folded fields were properly unfolded
	if sessions[0].Title != "First Event With A Very Long Folded Title" {
		t.Errorf("sessions[0].Title = %q, want %q", sessions[0].Title, "First Event With A Very Long Folded Title")
	}
	if sessions[0].Description != "First description also folded with tab" {
		t.Errorf("sessions[0].Description = %q, want %q", sessions[0].Description, "First description also folded with tab")
	}
	if sessions[0].Location != "Room A" {
		t.Errorf("sessions[0].Location = %q, want %q", sessions[0].Location, "Room A")
	}

	// Verify second event's fields are NOT dropped (the original bug)
	if sessions[1].Title != "Second Event" {
		t.Errorf("sessions[1].Title = %q, want %q", sessions[1].Title, "Second Event")
	}
	if sessions[1].Description != "Second description" {
		t.Errorf("sessions[1].Description = %q, want %q", sessions[1].Description, "Second description")
	}
	if sessions[1].Location != "Room B" {
		t.Errorf("sessions[1].Location = %q, want %q", sessions[1].Location, "Room B")
	}
	if sessions[1].Category != "Networking" {
		t.Errorf("sessions[1].Category = %q, want %q", sessions[1].Category, "Networking")
	}
	if sessions[1].HexID != "event2hex" {
		t.Errorf("sessions[1].HexID = %q, want %q", sessions[1].HexID, "event2hex")
	}

	wantStart := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	if !sessions[1].StartTime.Equal(wantStart) {
		t.Errorf("sessions[1].StartTime = %v, want %v", sessions[1].StartTime, wantStart)
	}
	wantEnd := time.Date(2026, 3, 24, 13, 0, 0, 0, time.UTC)
	if !sessions[1].EndTime.Equal(wantEnd) {
		t.Errorf("sessions[1].EndTime = %v, want %v", sessions[1].EndTime, wantEnd)
	}
}

// --- Real-world Data ---

const realWorldICalSample = `BEGIN:VCALENDAR
VERSION:2.0
X-WR-CALNAME:srecon26americas
BEGIN:VEVENT
DTSTAMP:20260324T034738Z
DTSTART:20260324T000000Z
DTEND:20260324T020000Z
SUMMARY:Welcome Get-Together
DESCRIPTION:Join us for an evening reception
CATEGORIES:Reception
LOCATION:Grand Foyer \, Seattle\, WA\, USA
SEQUENCE:0
UID:91cce7bfbc34850fb06d3e9f7ebee3c2
URL:http://srecon26americas.sched.com/event/91cce7bfbc34850fb06d3e9f7ebee3c2
END:VEVENT
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T154500Z
SUMMARY:Continental Breakfast
DESCRIPTION:
CATEGORIES:BREAKFAST
LOCATION:Grand Foyer \, Seattle\, WA\, USA
UID:40f2e7387fba2a2df58cd112e6bb28b4@real.jeff.welling
URL:http://srecon26americas.sched.com/event/40f2e7387fba2a2df58cd112e6bb28b4
END:VEVENT
END:VCALENDAR
`

func TestParseICalFeed_RealWorldData_FullSchedule(t *testing.T) {
	sessions, err := ParseICalFeed([]byte(realWorldICalSample), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	// First event
	s0 := sessions[0]
	if s0.Title != "Welcome Get-Together" {
		t.Errorf("sessions[0].Title = %q, want %q", s0.Title, "Welcome Get-Together")
	}
	if s0.Description != "Join us for an evening reception" {
		t.Errorf("sessions[0].Description = %q, want %q", s0.Description, "Join us for an evening reception")
	}
	if s0.Category != "Reception" {
		t.Errorf("sessions[0].Category = %q, want %q", s0.Category, "Reception")
	}
	wantLoc := "Grand Foyer , Seattle, WA, USA"
	if s0.Location != wantLoc {
		t.Errorf("sessions[0].Location = %q, want %q", s0.Location, wantLoc)
	}
	// Full schedule (no username), UID without @ => full UID as HexID
	if s0.HexID != "91cce7bfbc34850fb06d3e9f7ebee3c2" {
		t.Errorf("sessions[0].HexID = %q, want %q", s0.HexID, "91cce7bfbc34850fb06d3e9f7ebee3c2")
	}
	// URL contains 32-char hex, so ShortID should be empty
	if s0.ShortID != "" {
		t.Errorf("sessions[0].ShortID = %q, want empty (hex ID in URL)", s0.ShortID)
	}
	wantStart := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
	if !s0.StartTime.Equal(wantStart) {
		t.Errorf("sessions[0].StartTime = %v, want %v", s0.StartTime, wantStart)
	}
	wantEnd := time.Date(2026, 3, 24, 2, 0, 0, 0, time.UTC)
	if !s0.EndTime.Equal(wantEnd) {
		t.Errorf("sessions[0].EndTime = %v, want %v", s0.EndTime, wantEnd)
	}

	// Second event — no username provided, so UID with @ is kept as-is
	s1 := sessions[1]
	if s1.Title != "Continental Breakfast" {
		t.Errorf("sessions[1].Title = %q, want %q", s1.Title, "Continental Breakfast")
	}
	if s1.Description != "" {
		t.Errorf("sessions[1].Description = %q, want empty", s1.Description)
	}
	if s1.Category != "BREAKFAST" {
		t.Errorf("sessions[1].Category = %q, want %q", s1.Category, "BREAKFAST")
	}
	// No username provided, so full UID preserved (including @)
	if s1.HexID != "40f2e7387fba2a2df58cd112e6bb28b4@real.jeff.welling" {
		t.Errorf("sessions[1].HexID = %q, want %q", s1.HexID, "40f2e7387fba2a2df58cd112e6bb28b4@real.jeff.welling")
	}
}

func TestParseICalFeed_RealWorldData_PersonalSchedule(t *testing.T) {
	sessions, err := ParseICalFeed([]byte(realWorldICalSample), "real.jeff.welling")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	// First event UID has no @, so HexID is full UID even with username
	if sessions[0].HexID != "91cce7bfbc34850fb06d3e9f7ebee3c2" {
		t.Errorf("sessions[0].HexID = %q, want %q", sessions[0].HexID, "91cce7bfbc34850fb06d3e9f7ebee3c2")
	}

	// Second event UID is "hexid@real.jeff.welling" and username is set,
	// so hex ID should be extracted
	if sessions[1].HexID != "40f2e7387fba2a2df58cd112e6bb28b4" {
		t.Errorf("sessions[1].HexID = %q, want %q (@ username should be stripped)", sessions[1].HexID, "40f2e7387fba2a2df58cd112e6bb28b4")
	}
}

func TestParseICalFeed_RealWorldData_EscapedLocation(t *testing.T) {
	sessions, err := ParseICalFeed([]byte(realWorldICalSample), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	want := "Grand Foyer , Seattle, WA, USA"
	if sessions[0].Location != want {
		t.Errorf("Location = %q, want %q", sessions[0].Location, want)
	}
}

func TestParseICalFeed_RealWorldData_TimesParsedCorrectly(t *testing.T) {
	sessions, err := ParseICalFeed([]byte(realWorldICalSample), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}

	// Second event: 14:30 - 15:45
	wantStart := time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 3, 24, 15, 45, 0, 0, time.UTC)
	if !sessions[1].StartTime.Equal(wantStart) {
		t.Errorf("sessions[1].StartTime = %v, want %v", sessions[1].StartTime, wantStart)
	}
	if !sessions[1].EndTime.Equal(wantEnd) {
		t.Errorf("sessions[1].EndTime = %v, want %v", sessions[1].EndTime, wantEnd)
	}
}

func TestParseICalFeed_FetchedAtIsSet(t *testing.T) {
	ical := `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Test
UID:fetched1
URL:http://example.sched.com/event/F1
END:VEVENT
END:VCALENDAR
`
	before := time.Now().UTC().Add(-time.Second)
	sessions, err := ParseICalFeed([]byte(ical), "")
	if err != nil {
		t.Fatalf("ParseICalFeed() error: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)
	if sessions[0].FetchedAt.Before(before) || sessions[0].FetchedAt.After(after) {
		t.Errorf("FetchedAt = %v, want between %v and %v", sessions[0].FetchedAt, before, after)
	}
}

// --- parseICalTime (indirect via full parse) ---

func TestParseICalTime_UTCFormat(t *testing.T) {
	got, err := parseICalTime("20260324T143000Z")
	if err != nil {
		t.Fatalf("parseICalTime() error: %v", err)
	}
	want := time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseICalTime_LocalFormat(t *testing.T) {
	got, err := parseICalTime("20260324T143000")
	if err != nil {
		t.Fatalf("parseICalTime() error: %v", err)
	}
	want := time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseICalTime_DateOnly(t *testing.T) {
	got, err := parseICalTime("20260324")
	if err != nil {
		t.Fatalf("parseICalTime() error: %v", err)
	}
	want := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// --- unescapeICal ---

func TestUnescapeICal_AllEscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"escaped comma", `hello\, world`, "hello, world"},
		{"escaped semicolon", `a\;b`, "a;b"},
		{"escaped newline", `line1\nline2`, "line1\nline2"},
		{"escaped backslash", `path\\file`, `path\file`},
		{"multiple escapes", `a\, b\; c\n d\\e`, "a, b; c\n d\\e"},
		{"no escapes", "plain text", "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeICal(tt.input)
			if got != tt.want {
				t.Errorf("unescapeICal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
