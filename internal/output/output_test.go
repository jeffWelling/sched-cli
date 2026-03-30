package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jeff/sched-cli/internal/store"
)

// --- Test data helpers ---

func sampleSession() store.Session {
	return store.Session{
		HexID:       "abc123def456",
		ShortID:     "S001",
		Title:       "Introduction to Kubernetes",
		Description: "Learn the basics of container orchestration.",
		StartTime:   time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
		Location:    "Room 101",
		Category:    "DevOps",
		Speakers:    []string{"Alice", "Bob"},
		EventURL:    "https://sched.co/abc123",
		FetchedAt:   time.Date(2025, 6, 14, 12, 0, 0, 0, time.UTC),
	}
}

func sampleSessions() []store.Session {
	return []store.Session{
		sampleSession(),
		{
			HexID:     "def789abc012",
			ShortID:   "S002",
			Title:     "Advanced Go Patterns",
			StartTime: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC),
			Location:  "Room 202",
			Category:  "Programming",
			FetchedAt: time.Date(2025, 6, 14, 12, 0, 0, 0, time.UTC),
		},
	}
}

func sampleFriends() []store.Friend {
	return []store.Friend{
		{Nickname: "alice", Username: "alice@example.com"},
		{Nickname: "bob", Username: "bob@example.com"},
	}
}

func sampleCompareResult() CompareResult {
	s := sampleSession()
	return CompareResult{
		Overlaps: []OverlapEntry{
			{Session: s, Attendees: []string{"alice", "bob"}},
		},
		Gaps: []GapEntry{
			{Session: s, InterestedBy: []string{"charlie"}},
		},
	}
}

func sampleRateStatus() RateStatus {
	return RateStatus{
		CallsInWindow: 15,
		Limit:         100,
		BudgetUsed:    0.15,
		Remaining:     85,
	}
}

// --- JSON Mode ---

func TestFormatSessions_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSessions(sampleSessions())
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatSessions JSON output is not valid JSON")
	}
}

func TestFormatSchedule_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSchedule(sampleSessions())
	if err != nil {
		t.Fatalf("FormatSchedule error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatSchedule JSON output is not valid JSON")
	}
}

func TestFormatFriends_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatFriends(sampleFriends())
	if err != nil {
		t.Fatalf("FormatFriends error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatFriends JSON output is not valid JSON")
	}
}

func TestFormatComparison_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatComparison(sampleCompareResult())
	if err != nil {
		t.Fatalf("FormatComparison error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatComparison JSON output is not valid JSON")
	}
}

func TestFormatRateStatus_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatRateStatus(sampleRateStatus())
	if err != nil {
		t.Fatalf("FormatRateStatus error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatRateStatus JSON output is not valid JSON")
	}
}

func TestFormatSessionDetail_JSON_Valid(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSessionDetail(sampleSession())
	if err != nil {
		t.Fatalf("FormatSessionDetail error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("FormatSessionDetail JSON output is not valid JSON")
	}
}

func TestFormatSessions_JSON_IncludesAllFields(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSessions([]store.Session{sampleSession()})
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	var sessions []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &sessions); err != nil {
		t.Fatalf("unmarshalling JSON: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	requiredFields := []string{"hex_id", "short_id", "title", "start_time", "end_time", "fetched_at"}
	for _, field := range requiredFields {
		if _, ok := s[field]; !ok {
			t.Errorf("missing required field %q in JSON output", field)
		}
	}

	if s["hex_id"] != "abc123def456" {
		t.Errorf("hex_id = %v, want abc123def456", s["hex_id"])
	}
	if s["short_id"] != "S001" {
		t.Errorf("short_id = %v, want S001", s["short_id"])
	}
	if s["title"] != "Introduction to Kubernetes" {
		t.Errorf("title = %v, want Introduction to Kubernetes", s["title"])
	}
}

// --- Table Mode ---

func TestFormatSessions_Table_HasHeaderAndColumns(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatSessions(sampleSessions())
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (header + 2 sessions), got %d", len(lines))
	}

	header := lines[0]
	for _, col := range []string{"ID", "TIME", "TITLE", "LOCATION", "TRACK"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing column %q: %q", col, header)
		}
	}

	// Verify data rows contain expected values
	dataLine := lines[1]
	if !strings.Contains(dataLine, "S001") {
		t.Errorf("first data row missing ShortID 'S001': %q", dataLine)
	}
	if !strings.Contains(dataLine, "Introduction to Kubernetes") {
		t.Errorf("first data row missing title: %q", dataLine)
	}
}

func TestFormatFriends_Table_HasNicknameAndUsername(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatFriends(sampleFriends())
	if err != nil {
		t.Fatalf("FormatFriends error: %v", err)
	}

	output := buf.String()
	header := strings.Split(output, "\n")[0]

	if !strings.Contains(header, "NICKNAME") {
		t.Errorf("header missing NICKNAME column: %q", header)
	}
	if !strings.Contains(header, "USERNAME") {
		t.Errorf("header missing USERNAME column: %q", header)
	}

	if !strings.Contains(output, "alice") {
		t.Error("output missing friend 'alice'")
	}
	if !strings.Contains(output, "bob@example.com") {
		t.Error("output missing username 'bob@example.com'")
	}
}

func TestFormatRateStatus_Table_ShowsAllFields(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatRateStatus(sampleRateStatus())
	if err != nil {
		t.Fatalf("FormatRateStatus error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "15/100") {
		t.Errorf("output missing calls/limit '15/100': %q", output)
	}
	if !strings.Contains(output, "15%") {
		t.Errorf("output missing budget percentage '15%%': %q", output)
	}
	if !strings.Contains(output, "85") {
		t.Errorf("output missing remaining '85': %q", output)
	}
}

func TestFormatSessionDetail_Table_ShowsAllFields(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatSessionDetail(sampleSession())
	if err != nil {
		t.Fatalf("FormatSessionDetail error: %v", err)
	}

	output := buf.String()
	checks := []struct {
		label string
		want  string
	}{
		{"ShortID", "S001"},
		{"Title", "Introduction to Kubernetes"},
		{"Location", "Room 101"},
		{"Track/Category", "DevOps"},
		{"Description", "Learn the basics"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.want) {
			t.Errorf("output missing %s %q: %q", c.label, c.want, output)
		}
	}
}

// --- Truncation ---

func TestTruncate_LongTitleTruncated(t *testing.T) {
	longTitle := strings.Repeat("A", 100)
	result := truncate(longTitle, 50)

	if len(result) != 50 {
		t.Errorf("truncated length = %d, want 50", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("truncated string should end with '...'")
	}
}

func TestTruncate_ShortTitleNotTruncated(t *testing.T) {
	short := "Brief title"
	result := truncate(short, 50)

	if result != short {
		t.Errorf("truncate(%q) = %q, want unchanged", short, result)
	}
}

func TestFormatSessions_Table_TruncatesLongTitles(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	longTitle := strings.Repeat("X", 100)
	sessions := []store.Session{
		{
			ShortID:   "S999",
			Title:     longTitle,
			StartTime: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
			FetchedAt: time.Now(),
		},
	}

	err := f.FormatSessions(sessions)
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	output := buf.String()
	// The full 100-char title should NOT appear
	if strings.Contains(output, longTitle) {
		t.Error("long title should be truncated in table mode")
	}
	// The truncated version with "..." should appear
	if !strings.Contains(output, "...") {
		t.Error("truncated title should contain '...'")
	}
}

// --- Empty Data ---

func TestFormatSessions_EmptySlice_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatSessions([]store.Session{})
	if err != nil {
		t.Fatalf("FormatSessions with empty slice should not error: %v", err)
	}
}

func TestFormatSessions_EmptySlice_JSON_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSessions([]store.Session{})
	if err != nil {
		t.Fatalf("FormatSessions JSON with empty slice should not error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("empty sessions JSON should still be valid")
	}
}

func TestFormatFriends_EmptySlice_OutputsHeader(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatFriends([]store.Friend{})
	if err != nil {
		t.Fatalf("FormatFriends error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NICKNAME") {
		t.Error("empty friends table should still have header")
	}
}

func TestFormatComparison_EmptyResult_RendersCleanly(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	err := f.FormatComparison(CompareResult{})
	if err != nil {
		t.Fatalf("FormatComparison with empty result should not error: %v", err)
	}

	// With no overlaps and no gaps, output should be empty or minimal
	output := buf.String()
	if strings.Contains(output, "OVERLAPS") {
		t.Error("empty comparison should not render OVERLAPS header")
	}
	if strings.Contains(output, "GAPS") {
		t.Error("empty comparison should not render GAPS header")
	}
}

// --- CompareResult formatting ---

func TestFormatComparison_Table_OverlapsSection(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	result := sampleCompareResult()
	err := f.FormatComparison(result)
	if err != nil {
		t.Fatalf("FormatComparison error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "OVERLAPS") {
		t.Error("output missing OVERLAPS header")
	}
	if !strings.Contains(output, "S001") {
		t.Error("overlaps section missing session ShortID")
	}
	if !strings.Contains(output, "alice") {
		t.Error("overlaps section missing attendee 'alice'")
	}
	if !strings.Contains(output, "bob") {
		t.Error("overlaps section missing attendee 'bob'")
	}
}

func TestFormatComparison_Table_GapsSection(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, false, false)

	result := sampleCompareResult()
	err := f.FormatComparison(result)
	if err != nil {
		t.Fatalf("FormatComparison error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "GAPS") {
		t.Error("output missing GAPS header")
	}
	if !strings.Contains(output, "charlie") {
		t.Error("gaps section missing interested party 'charlie'")
	}
}

func TestFormatComparison_Table_OnlyNonEmptySections(t *testing.T) {
	// Only overlaps, no gaps
	var buf bytes.Buffer
	f := New(&buf, false, false)

	result := CompareResult{
		Overlaps: []OverlapEntry{
			{Session: sampleSession(), Attendees: []string{"alice"}},
		},
	}
	err := f.FormatComparison(result)
	if err != nil {
		t.Fatalf("FormatComparison error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "OVERLAPS") {
		t.Error("output should contain OVERLAPS when present")
	}
	if strings.Contains(output, "GAPS") {
		t.Error("output should NOT contain GAPS when empty")
	}

	// Only gaps, no overlaps
	buf.Reset()
	result2 := CompareResult{
		Gaps: []GapEntry{
			{Session: sampleSession(), InterestedBy: []string{"dave"}},
		},
	}
	err = f.FormatComparison(result2)
	if err != nil {
		t.Fatalf("FormatComparison error: %v", err)
	}

	output2 := buf.String()
	if strings.Contains(output2, "OVERLAPS") {
		t.Error("output should NOT contain OVERLAPS when empty")
	}
	if !strings.Contains(output2, "GAPS") {
		t.Error("output should contain GAPS when present")
	}
}

// --- IsTerminal ---

func TestIsTerminal_DoesNotPanic(t *testing.T) {
	_ = IsTerminal(os.Stdout.Fd())
}

// --- JSON output modes ---

func TestWriteJSON_Compact_NoIndentation(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, false)

	err := f.FormatSessions([]store.Session{sampleSession()})
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	output := buf.String()
	if !json.Valid([]byte(output)) {
		t.Fatal("compact JSON output is not valid JSON")
	}

	// Compact JSON should be a single line (no embedded newlines except the trailing one from Encode)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("compact JSON should be a single line, got %d lines", len(lines))
	}
}

func TestWriteJSON_Pretty_HasIndentation(t *testing.T) {
	var buf bytes.Buffer
	f := New(&buf, true, true)

	err := f.FormatSessions([]store.Session{sampleSession()})
	if err != nil {
		t.Fatalf("FormatSessions error: %v", err)
	}

	output := buf.String()
	if !json.Valid([]byte(output)) {
		t.Fatal("pretty JSON output is not valid JSON")
	}

	// Pretty JSON should have multiple lines with indentation
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		t.Error("pretty JSON should span multiple lines")
	}
	if !strings.Contains(output, "  ") {
		t.Error("pretty JSON should contain indentation")
	}
}

func TestAutoDetect_ForcesJSONInNonTTY(t *testing.T) {
	// AutoDetect with jsonMode=false should still enable JSON since tests run in non-TTY
	f := AutoDetect(false, false)
	if f == nil {
		t.Fatal("AutoDetect returned nil")
	}
	// In a test environment (non-TTY), jsonMode should be true
	if !f.jsonMode {
		t.Error("AutoDetect should enable JSON mode when stdout is not a terminal")
	}
}
