package client

import (
	"os"
	"testing"
)

func loadTestHTML(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/list-descriptions-sample.html")
	if err != nil {
		t.Fatalf("failed to read test HTML: %v", err)
	}
	return data
}

func TestParseScheduleHTML_ExtractsAllEntries(t *testing.T) {
	entries, err := ParseScheduleHTML(loadTestHTML(t))
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
}

func TestParseScheduleHTML_DetectsOnSchedule(t *testing.T) {
	entries, err := ParseScheduleHTML(loadTestHTML(t))
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	// First entry has "sub" class
	if !entries[0].OnSchedule {
		t.Errorf("entries[0].OnSchedule = false, want true")
	}
	// Second entry does NOT have "sub" class
	if entries[1].OnSchedule {
		t.Errorf("entries[1].OnSchedule = true, want false")
	}
	// Third entry has "sub" class
	if !entries[2].OnSchedule {
		t.Errorf("entries[2].OnSchedule = false, want true")
	}
}

func TestParseScheduleHTML_ExtractsHexID(t *testing.T) {
	entries, err := ParseScheduleHTML(loadTestHTML(t))
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	want := []string{
		"e6f499540ac79243410b138edde13b1a",
		"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
		"b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5",
	}
	for i, w := range want {
		if entries[i].HexID != w {
			t.Errorf("entries[%d].HexID = %q, want %q", i, entries[i].HexID, w)
		}
	}
}

func TestParseScheduleHTML_ExtractsShortID(t *testing.T) {
	entries, err := ParseScheduleHTML(loadTestHTML(t))
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	want := []string{"2J09B", "3K10C", "4L11D"}
	for i, w := range want {
		if entries[i].ShortID != w {
			t.Errorf("entries[%d].ShortID = %q, want %q", i, entries[i].ShortID, w)
		}
	}
}

func TestParseScheduleHTML_ExtractsTitle(t *testing.T) {
	entries, err := ParseScheduleHTML(loadTestHTML(t))
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	want := []string{
		"Taming the Unpredictable: Reliability in Chaos",
		"Scaling Beyond Limits",
		"Observability at Scale",
	}
	for i, w := range want {
		if entries[i].Title != w {
			t.Errorf("entries[%d].Title = %q, want %q", i, entries[i].Title, w)
		}
	}
}

func TestParseScheduleHTML_EmptyInput(t *testing.T) {
	entries, err := ParseScheduleHTML([]byte{})
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries from empty input, want 0", len(entries))
	}
}

func TestParseScheduleHTML_NoEvents(t *testing.T) {
	html := []byte(`<html><body><div class="sched-container"><p>No events</p></div></body></html>`)
	entries, err := ParseScheduleHTML(html)
	if err != nil {
		t.Fatalf("ParseScheduleHTML() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries from eventless HTML, want 0", len(entries))
	}
}
