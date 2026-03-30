package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestStore creates a fresh Store backed by a temporary SQLite database.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// insertTestSession upserts a session with sensible defaults for fields the
// caller doesn't care about.
func insertTestSession(t *testing.T, s *Store, hexID, shortID, title string) {
	t.Helper()
	sess := Session{
		HexID:     hexID,
		ShortID:   shortID,
		Title:     title,
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	}
	if err := s.UpsertSession(sess); err != nil {
		t.Fatalf("UpsertSession(%q): %v", hexID, err)
	}
}

// insertTestSessionFull upserts a fully-populated session.
func insertTestSessionFull(t *testing.T, s *Store, sess Session) {
	t.Helper()
	if err := s.UpsertSession(sess); err != nil {
		t.Fatalf("UpsertSession(%q): %v", sess.HexID, err)
	}
}

// ---------------------------------------------------------------------------
// Schema & Migration
// ---------------------------------------------------------------------------

func TestNew_CreatesTables(t *testing.T) {
	s := newTestStore(t)

	tables := []string{
		"schema_version", "sessions", "schedule", "interests", "friends",
		"friend_schedules", "api_calls", "cache_meta",
	}
	for _, tbl := range tables {
		var name string
		err := s.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist: %v", tbl, err)
		}
	}
}

func TestNew_Idempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "idem.db")
	s1, err := New(dbPath)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	s1.Close()

	s2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	s2.Close()
}

func TestMigrate_FreshDatabaseGetsVersion1(t *testing.T) {
	s := newTestStore(t)

	var version int
	err := s.DB().QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("reading schema_version: %v", err)
	}
	if version != SchemaVersion {
		t.Errorf("schema version = %d; want %d", version, SchemaVersion)
	}
}

func TestMigrate_ExistingV1IsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate_idem.db")

	// First open: creates schema at version 1.
	s1, err := New(dbPath)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	// Insert some data to verify it survives re-open.
	insertTestSession(t, s1, "surv1", "sv1", "Survivor Talk")
	s1.Close()

	// Second open: should detect version 1 and skip migration.
	s2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	defer s2.Close()

	var version int
	err = s2.DB().QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("reading schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("schema version = %d; want 1", version)
	}

	// Verify data survived.
	got, err := s2.GetSession("surv1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got == nil {
		t.Fatal("expected session surv1 to survive re-open")
	}
	if got.Title != "Survivor Talk" {
		t.Errorf("Title = %q; want Survivor Talk", got.Title)
	}
}

func TestMigrate_FutureVersionReturnsError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "future.db")

	// Create a database with schema_version set to a future version.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO schema_version (version) VALUES (999)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Opening with New should fail because 999 > SchemaVersion.
	_, err = New(dbPath)
	if err == nil {
		t.Fatal("expected error for future schema version, got nil")
	}

	expectedSubstring := "newer than supported"
	if !containsString(err.Error(), expectedSubstring) {
		t.Errorf("error = %q; expected it to contain %q", err.Error(), expectedSubstring)
	}
}

// containsString reports whether s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSchema_SessionsColumns(t *testing.T) {
	s := newTestStore(t)

	// Insert a fully-populated row to verify all columns work.
	sess := Session{
		HexID:       "abc123",
		ShortID:     "s1",
		Title:       "Test Talk",
		Description: "A great talk",
		StartTime:   time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		Location:    "Room A",
		Category:    "Go",
		EventURL:    "https://example.com/s1",
		FetchedAt:   time.Now().UTC(),
	}
	if err := s.UpsertSession(sess); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := s.GetSession("abc123")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Location != "Room A" {
		t.Errorf("Location = %q; want %q", got.Location, "Room A")
	}
	if got.Category != "Go" {
		t.Errorf("Category = %q; want %q", got.Category, "Go")
	}
	if got.EventURL != "https://example.com/s1" {
		t.Errorf("EventURL = %q; want %q", got.EventURL, "https://example.com/s1")
	}
}

func TestSchema_SessionsShortIDUnique(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "aaa", "dup", "First")

	// Insert a different hex_id but duplicate short_id — should fail.
	sess := Session{
		HexID:     "bbb",
		ShortID:   "dup",
		Title:     "Second",
		FetchedAt: time.Now().UTC(),
	}
	err := s.UpsertSession(sess)
	if err == nil {
		t.Fatal("expected UNIQUE constraint error on short_id, got nil")
	}
}

func TestSchema_FriendsUsernameUnique(t *testing.T) {
	s := newTestStore(t)
	if err := s.AddFriend("alice", "auser"); err != nil {
		t.Fatal(err)
	}
	err := s.AddFriend("bob", "auser")
	if err == nil {
		t.Fatal("expected UNIQUE constraint error on username, got nil")
	}
}

// ---------------------------------------------------------------------------
// Sessions CRUD
// ---------------------------------------------------------------------------

func TestUpsertSession_Insert(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "hex1", "s1", "My Talk")

	got, err := s.GetSession("hex1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.Title != "My Talk" {
		t.Errorf("Title = %q; want %q", got.Title, "My Talk")
	}
}

func TestUpsertSession_Update(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "hex1", "s1", "Original Title")

	// Upsert with same hex_id, new title.
	sess := Session{
		HexID:     "hex1",
		ShortID:   "s1",
		Title:     "Updated Title",
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	}
	if err := s.UpsertSession(sess); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	got, err := s.GetSession("hex1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title = %q; want %q", got.Title, "Updated Title")
	}
}

func TestGetSession_ByHexID(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "hexABC", "short1", "Talk A")

	got, err := s.GetSession("hexABC")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.HexID != "hexABC" {
		t.Errorf("expected session with hex_id hexABC, got %+v", got)
	}
}

func TestGetSession_ByShortID(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "hexDEF", "short2", "Talk B")

	got, err := s.GetSession("short2")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ShortID != "short2" {
		t.Errorf("expected session with short_id short2, got %+v", got)
	}
}

func TestGetSession_Unknown(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetSession("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestListSessions_NoFilters(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "a1", "s1", "Talk 1")
	insertTestSession(t, s, "a2", "s2", "Talk 2")
	insertTestSession(t, s, "a3", "s3", "Talk 3")

	sessions, err := s.ListSessions(SessionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 3 {
		t.Errorf("len = %d; want 3", len(sessions))
	}
}

func TestListSessions_FilterByTrack(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "a1", ShortID: "s1", Title: "Go Talk",
		Category: "Go", FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "a2", ShortID: "s2", Title: "Rust Talk",
		Category: "Rust", FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
	})

	sessions, err := s.ListSessions(SessionFilters{Track: "Go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d; want 1", len(sessions))
	}
	if sessions[0].HexID != "a1" {
		t.Errorf("HexID = %q; want a1", sessions[0].HexID)
	}
}

func TestListSessions_FilterByDay(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "d1", ShortID: "ds1", Title: "Day 1 Talk",
		StartTime: time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "d2", ShortID: "ds2", Title: "Day 2 Talk",
		StartTime: time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})

	sessions, err := s.ListSessions(SessionFilters{Day: "2026-06-15"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d; want 1", len(sessions))
	}
	if sessions[0].HexID != "d1" {
		t.Errorf("HexID = %q; want d1", sessions[0].HexID)
	}
}

func TestListSessions_FilterBySearch_Title(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "x1", ShortID: "xs1", Title: "Advanced Go Patterns",
		FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "x2", ShortID: "xs2", Title: "Intro to Rust",
		FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
	})

	sessions, err := s.ListSessions(SessionFilters{Search: "Go Patterns"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d; want 1", len(sessions))
	}
	if sessions[0].HexID != "x1" {
		t.Errorf("HexID = %q; want x1", sessions[0].HexID)
	}
}

func TestListSessions_FilterBySearch_Description(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "x1", ShortID: "xs1", Title: "Generic Title",
		Description: "This talk covers advanced concurrency",
		FetchedAt:   time.Now().UTC(),
		StartTime:   time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "x2", ShortID: "xs2", Title: "Another Title",
		Description: "Basics of web development",
		FetchedAt:   time.Now().UTC(),
		StartTime:   time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
	})

	sessions, err := s.ListSessions(SessionFilters{Search: "concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d; want 1", len(sessions))
	}
	if sessions[0].HexID != "x1" {
		t.Errorf("HexID = %q; want x1", sessions[0].HexID)
	}
}

func TestListSessions_MultipleFilters(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "m1", ShortID: "ms1", Title: "Go Concurrency",
		Category: "Go", FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "m2", ShortID: "ms2", Title: "Go Testing",
		Category: "Go", FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "m3", ShortID: "ms3", Title: "Rust Concurrency",
		Category: "Rust", FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
	})

	// Track=Go + Day=2026-06-15 should only match m1.
	sessions, err := s.ListSessions(SessionFilters{
		Track: "Go",
		Day:   "2026-06-15",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d; want 1", len(sessions))
	}
	if sessions[0].HexID != "m1" {
		t.Errorf("HexID = %q; want m1", sessions[0].HexID)
	}
}

func TestListSessions_OrderedByStartTime(t *testing.T) {
	s := newTestStore(t)

	// Insert out of chronological order.
	insertTestSessionFull(t, s, Session{
		HexID: "late", ShortID: "sl", Title: "Late Talk",
		FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 16, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "early", ShortID: "se", Title: "Early Talk",
		FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "mid", ShortID: "sm", Title: "Mid Talk",
		FetchedAt: time.Now().UTC(),
		StartTime: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	})

	sessions, err := s.ListSessions(SessionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 3 {
		t.Fatalf("len = %d; want 3", len(sessions))
	}
	if sessions[0].HexID != "early" {
		t.Errorf("first = %q; want early", sessions[0].HexID)
	}
	if sessions[1].HexID != "mid" {
		t.Errorf("second = %q; want mid", sessions[1].HexID)
	}
	if sessions[2].HexID != "late" {
		t.Errorf("third = %q; want late", sessions[2].HexID)
	}
}

func TestSessionCount(t *testing.T) {
	s := newTestStore(t)

	count, err := s.SessionCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("count = %d; want 0", count)
	}

	insertTestSession(t, s, "c1", "cs1", "Talk 1")
	insertTestSession(t, s, "c2", "cs2", "Talk 2")

	count, err = s.SessionCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d; want 2", count)
	}
}

// ---------------------------------------------------------------------------
// Schedule Management
// ---------------------------------------------------------------------------

func TestAddToSchedule(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "sch1", "ss1", "Scheduled Talk")

	if err := s.AddToSchedule("sch1", "manual"); err != nil {
		t.Fatal(err)
	}

	on, err := s.IsOnSchedule("sch1")
	if err != nil {
		t.Fatal(err)
	}
	if !on {
		t.Error("expected session to be on schedule")
	}
}

func TestAddToSchedule_Idempotent(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "sch1", "ss1", "Scheduled Talk")

	if err := s.AddToSchedule("sch1", "manual"); err != nil {
		t.Fatal(err)
	}
	// Second add with different source should just update.
	if err := s.AddToSchedule("sch1", "sync"); err != nil {
		t.Fatal("second AddToSchedule should be idempotent: " + err.Error())
	}

	sched, err := s.GetSchedule()
	if err != nil {
		t.Fatal(err)
	}
	if len(sched) != 1 {
		t.Errorf("len = %d; want 1", len(sched))
	}
}

func TestRemoveFromSchedule(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "sch1", "ss1", "Scheduled Talk")
	if err := s.AddToSchedule("sch1", "manual"); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveFromSchedule("sch1"); err != nil {
		t.Fatal(err)
	}

	on, err := s.IsOnSchedule("sch1")
	if err != nil {
		t.Fatal(err)
	}
	if on {
		t.Error("expected session to NOT be on schedule after removal")
	}
}

func TestRemoveFromSchedule_NonExistent(t *testing.T) {
	s := newTestStore(t)

	// Removing something that was never added should not error.
	if err := s.RemoveFromSchedule("ghost"); err != nil {
		t.Fatalf("RemoveFromSchedule on non-existent should be no-op: %v", err)
	}
}

func TestGetSchedule_Ordered(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "late", ShortID: "sl", Title: "Late",
		StartTime: time.Date(2026, 6, 15, 16, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "early", ShortID: "se", Title: "Early",
		StartTime: time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})

	if err := s.AddToSchedule("late", "manual"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddToSchedule("early", "manual"); err != nil {
		t.Fatal(err)
	}

	sched, err := s.GetSchedule()
	if err != nil {
		t.Fatal(err)
	}
	if len(sched) != 2 {
		t.Fatalf("len = %d; want 2", len(sched))
	}
	if sched[0].HexID != "early" {
		t.Errorf("first = %q; want early", sched[0].HexID)
	}
	if sched[1].HexID != "late" {
		t.Errorf("second = %q; want late", sched[1].HexID)
	}
}

func TestGetSchedule_Empty(t *testing.T) {
	s := newTestStore(t)

	sched, err := s.GetSchedule()
	if err != nil {
		t.Fatal(err)
	}
	if sched != nil {
		t.Errorf("expected nil slice, got %v", sched)
	}
}

func TestIsOnSchedule(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "on1", "os1", "On Schedule")
	if err := s.AddToSchedule("on1", "manual"); err != nil {
		t.Fatal(err)
	}

	t.Run("true", func(t *testing.T) {
		on, err := s.IsOnSchedule("on1")
		if err != nil {
			t.Fatal(err)
		}
		if !on {
			t.Error("expected true")
		}
	})

	t.Run("false", func(t *testing.T) {
		on, err := s.IsOnSchedule("not_on")
		if err != nil {
			t.Fatal(err)
		}
		if on {
			t.Error("expected false")
		}
	})
}

// ---------------------------------------------------------------------------
// Interest Flags
// ---------------------------------------------------------------------------

func TestAddInterest(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "int1", "is1", "Interesting Talk")

	if err := s.AddInterest("int1"); err != nil {
		t.Fatal(err)
	}

	interested, err := s.IsInterested("int1")
	if err != nil {
		t.Fatal(err)
	}
	if !interested {
		t.Error("expected session to be flagged as interested")
	}
}

func TestAddInterest_Idempotent(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "int1", "is1", "Interesting Talk")

	if err := s.AddInterest("int1"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddInterest("int1"); err != nil {
		t.Fatal("second AddInterest should be no-op: " + err.Error())
	}
}

func TestRemoveInterest(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "int1", "is1", "Interesting Talk")
	if err := s.AddInterest("int1"); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveInterest("int1"); err != nil {
		t.Fatal(err)
	}

	interested, err := s.IsInterested("int1")
	if err != nil {
		t.Fatal(err)
	}
	if interested {
		t.Error("expected session to NOT be interested after removal")
	}
}

func TestListInterests(t *testing.T) {
	s := newTestStore(t)

	insertTestSessionFull(t, s, Session{
		HexID: "i1", ShortID: "is1", Title: "Talk A",
		StartTime: time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})
	insertTestSessionFull(t, s, Session{
		HexID: "i2", ShortID: "is2", Title: "Talk B",
		StartTime: time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	})

	if err := s.AddInterest("i1"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddInterest("i2"); err != nil {
		t.Fatal(err)
	}

	interests, err := s.ListInterests()
	if err != nil {
		t.Fatal(err)
	}
	if len(interests) != 2 {
		t.Fatalf("len = %d; want 2", len(interests))
	}
	// Should be ordered by start_time (i2 at 9:00 before i1 at 14:00).
	if interests[0].HexID != "i2" {
		t.Errorf("first = %q; want i2", interests[0].HexID)
	}
	if interests[1].HexID != "i1" {
		t.Errorf("second = %q; want i1", interests[1].HexID)
	}
}

func TestIsInterested(t *testing.T) {
	s := newTestStore(t)
	insertTestSession(t, s, "int1", "is1", "Interesting Talk")
	if err := s.AddInterest("int1"); err != nil {
		t.Fatal(err)
	}

	t.Run("true", func(t *testing.T) {
		ok, err := s.IsInterested("int1")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected true")
		}
	})

	t.Run("false", func(t *testing.T) {
		ok, err := s.IsInterested("not_interested")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected false")
		}
	})
}

// ---------------------------------------------------------------------------
// Friends
// ---------------------------------------------------------------------------

func TestAddFriend(t *testing.T) {
	s := newTestStore(t)

	if err := s.AddFriend("alice", "alice_sched"); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFriendByNickname("alice")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected friend, got nil")
	}
	if f.Username != "alice_sched" {
		t.Errorf("Username = %q; want alice_sched", f.Username)
	}
}

func TestAddFriend_UpdateUsername(t *testing.T) {
	s := newTestStore(t)

	if err := s.AddFriend("alice", "old_user"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFriend("alice", "new_user"); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFriendByNickname("alice")
	if err != nil {
		t.Fatal(err)
	}
	if f.Username != "new_user" {
		t.Errorf("Username = %q; want new_user", f.Username)
	}
}

func TestRemoveFriend(t *testing.T) {
	s := newTestStore(t)

	if err := s.AddFriend("bob", "bob_sched"); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveFriend("bob"); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFriendByNickname("bob")
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Errorf("expected nil after removal, got %+v", f)
	}
}

func TestRemoveFriend_Unknown(t *testing.T) {
	s := newTestStore(t)

	err := s.RemoveFriend("nobody")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListFriends(t *testing.T) {
	s := newTestStore(t)

	if err := s.AddFriend("charlie", "c_user"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFriend("alice", "a_user"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFriend("bob", "b_user"); err != nil {
		t.Fatal(err)
	}

	friends, err := s.ListFriends()
	if err != nil {
		t.Fatal(err)
	}
	if len(friends) != 3 {
		t.Fatalf("len = %d; want 3", len(friends))
	}
	// Sorted by nickname.
	if friends[0].Nickname != "alice" {
		t.Errorf("first = %q; want alice", friends[0].Nickname)
	}
	if friends[1].Nickname != "bob" {
		t.Errorf("second = %q; want bob", friends[1].Nickname)
	}
	if friends[2].Nickname != "charlie" {
		t.Errorf("third = %q; want charlie", friends[2].Nickname)
	}
}

func TestListFriends_Empty(t *testing.T) {
	s := newTestStore(t)

	friends, err := s.ListFriends()
	if err != nil {
		t.Fatal(err)
	}
	if friends != nil {
		t.Errorf("expected nil slice, got %v", friends)
	}
}

func TestGetFriendByNickname(t *testing.T) {
	s := newTestStore(t)

	if err := s.AddFriend("dave", "d_user"); err != nil {
		t.Fatal(err)
	}

	f, err := s.GetFriendByNickname("dave")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected friend, got nil")
	}
	if f.Nickname != "dave" || f.Username != "d_user" {
		t.Errorf("got %+v; want {dave d_user}", f)
	}
}

func TestGetFriendByNickname_Unknown(t *testing.T) {
	s := newTestStore(t)

	f, err := s.GetFriendByNickname("ghost")
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Errorf("expected nil, got %+v", f)
	}
}

// ---------------------------------------------------------------------------
// Friend Schedules
// ---------------------------------------------------------------------------

func TestUpsertFriendSchedule(t *testing.T) {
	s := newTestStore(t)

	ids := []string{"hex1", "hex2", "hex3"}
	if err := s.UpsertFriendSchedule("friend_user", ids); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetFriendSchedule("friend_user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d; want 3", len(got))
	}
}

func TestUpsertFriendSchedule_Replaces(t *testing.T) {
	s := newTestStore(t)

	if err := s.UpsertFriendSchedule("friend_user", []string{"old1", "old2"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFriendSchedule("friend_user", []string{"new1"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetFriendSchedule("friend_user")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d; want 1 after replace", len(got))
	}
	if got[0] != "new1" {
		t.Errorf("got %q; want new1", got[0])
	}
}

func TestGetFriendSchedule_Unknown(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetFriendSchedule("nobody")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Rate Limiting
// ---------------------------------------------------------------------------

func TestLogAPICall(t *testing.T) {
	s := newTestStore(t)

	if err := s.LogAPICall("/sessions", "GET"); err != nil {
		t.Fatal(err)
	}

	count, err := s.GetAPICallCount(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d; want 1", count)
	}
}

func TestGetAPICallCount_WithinWindow(t *testing.T) {
	s := newTestStore(t)

	if err := s.LogAPICall("/a", "GET"); err != nil {
		t.Fatal(err)
	}
	if err := s.LogAPICall("/b", "POST"); err != nil {
		t.Fatal(err)
	}

	count, err := s.GetAPICallCount(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d; want 2", count)
	}
}

func TestGetAPICallCount_OutsideWindow(t *testing.T) {
	s := newTestStore(t)

	// Insert a call manually with a timestamp in the past.
	past := time.Now().UTC().Add(-2 * time.Hour)
	_, err := s.DB().Exec(
		"INSERT INTO api_calls (called_at, endpoint, method) VALUES (?, ?, ?)",
		past, "/old", "GET",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Use a 1-hour window — the old call should be excluded.
	count, err := s.GetAPICallCount(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("count = %d; want 0 (call is outside window)", count)
	}
}

func TestCleanupAPICalls(t *testing.T) {
	s := newTestStore(t)

	// Insert old call.
	past := time.Now().UTC().Add(-2 * time.Hour)
	_, err := s.DB().Exec(
		"INSERT INTO api_calls (called_at, endpoint, method) VALUES (?, ?, ?)",
		past, "/old", "GET",
	)
	if err != nil {
		t.Fatal(err)
	}
	// Insert recent call.
	if err := s.LogAPICall("/new", "GET"); err != nil {
		t.Fatal(err)
	}

	if err := s.CleanupAPICalls(time.Hour); err != nil {
		t.Fatal(err)
	}

	// Only the recent call should remain.
	var total int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM api_calls").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total = %d; want 1 after cleanup", total)
	}
}

// ---------------------------------------------------------------------------
// Cache Metadata
// ---------------------------------------------------------------------------

func TestSetCacheMeta_Create(t *testing.T) {
	s := newTestStore(t)

	if err := s.SetCacheMeta("sessions", `"etag123"`, 3600); err != nil {
		t.Fatal(err)
	}

	cm, err := s.GetCacheMeta("sessions")
	if err != nil {
		t.Fatal(err)
	}
	if cm == nil {
		t.Fatal("expected cache meta, got nil")
	}
	if cm.Resource != "sessions" {
		t.Errorf("Resource = %q; want sessions", cm.Resource)
	}
	if cm.ETag != `"etag123"` {
		t.Errorf("ETag = %q; want \"etag123\"", cm.ETag)
	}
	if cm.TTLSeconds != 3600 {
		t.Errorf("TTLSeconds = %d; want 3600", cm.TTLSeconds)
	}
}

func TestSetCacheMeta_Update(t *testing.T) {
	s := newTestStore(t)

	if err := s.SetCacheMeta("sessions", `"old"`, 100); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCacheMeta("sessions", `"new"`, 200); err != nil {
		t.Fatal(err)
	}

	cm, err := s.GetCacheMeta("sessions")
	if err != nil {
		t.Fatal(err)
	}
	if cm.ETag != `"new"` {
		t.Errorf("ETag = %q; want \"new\"", cm.ETag)
	}
	if cm.TTLSeconds != 200 {
		t.Errorf("TTLSeconds = %d; want 200", cm.TTLSeconds)
	}
}

func TestGetCacheMeta_Unknown(t *testing.T) {
	s := newTestStore(t)

	cm, err := s.GetCacheMeta("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if cm != nil {
		t.Errorf("expected nil, got %+v", cm)
	}
}

func TestIsCacheStale_NoCacheEntry(t *testing.T) {
	s := newTestStore(t)

	stale, err := s.IsCacheStale("missing_resource")
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("expected stale=true when no cache entry exists")
	}
}

func TestIsCacheStale_TTLExceeded(t *testing.T) {
	s := newTestStore(t)

	// Insert cache entry with fetched_at in the past and a short TTL.
	past := time.Now().UTC().Add(-2 * time.Hour)
	_, err := s.DB().Exec(
		"INSERT INTO cache_meta (resource, fetched_at, etag, ttl_seconds) VALUES (?, ?, ?, ?)",
		"stale_resource", past, "", 60, // 60s TTL, fetched 2h ago
	)
	if err != nil {
		t.Fatal(err)
	}

	stale, err := s.IsCacheStale("stale_resource")
	if err != nil {
		t.Fatal(err)
	}
	if !stale {
		t.Error("expected stale=true when TTL exceeded")
	}
}

func TestIsCacheStale_TTLNotExceeded(t *testing.T) {
	s := newTestStore(t)

	// Use SetCacheMeta so fetched_at is ~now, with a long TTL.
	if err := s.SetCacheMeta("fresh_resource", "", 86400); err != nil {
		t.Fatal(err)
	}

	stale, err := s.IsCacheStale("fresh_resource")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("expected stale=false when TTL not exceeded")
	}
}
