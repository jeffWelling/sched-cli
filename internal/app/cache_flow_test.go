package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jeff/sched-cli/internal/rate"
	"github.com/jeff/sched-cli/internal/store"
	_ "modernc.org/sqlite"
)

// newTestApp creates a minimal App with a real store and limiter for cache flow tests.
func newTestApp(t *testing.T) *App {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	a := &App{
		store:   s,
		limiter: rate.New(s, rate.DefaultLimit, rate.DefaultWindow),
	}
	return a
}

// insertSession inserts a session into the test store.
func insertSession(t *testing.T, s *store.Store, hexID, title string) {
	t.Helper()
	sess := store.Session{
		HexID:     hexID,
		ShortID:   hexID,
		Title:     title,
		StartTime: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC),
		FetchedAt: time.Now().UTC(),
	}
	if err := s.UpsertSession(sess); err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
}

func TestCacheFlow_FreshCache_ReturnsData(t *testing.T) {
	a := newTestApp(t)

	// Insert cached sessions.
	insertSession(t, a.store, "s1", "Talk 1")
	insertSession(t, a.store, "s2", "Talk 2")

	// Mark cache as fresh (TTL of 1 hour, just fetched).
	if err := a.store.SetCacheMeta("sessions", "", 3600); err != nil {
		t.Fatal(err)
	}

	fetchCalled := false
	fetchFn := func() ([]store.Session, error) {
		fetchCalled = true
		return nil, nil
	}

	result, err := a.FetchWithCache("sessions", fetchFn)
	if err != nil {
		t.Fatalf("FetchWithCache error: %v", err)
	}
	if fetchCalled {
		t.Error("fetchFn should not be called when cache is fresh")
	}
	if !result.FromCache {
		t.Error("expected FromCache=true")
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result.Data))
	}
	if result.Warning != "" {
		t.Errorf("expected no warning, got %q", result.Warning)
	}
}

func TestCacheFlow_StaleCache_UnderBudget_Fetches(t *testing.T) {
	a := newTestApp(t)

	// Insert a session so the store has data.
	insertSession(t, a.store, "s1", "Talk 1")

	// Mark cache as stale (TTL 1s, fetched 1 hour ago).
	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err := a.store.DB().Exec(
		"INSERT INTO cache_meta (resource, fetched_at, etag, ttl_seconds) VALUES (?, ?, ?, ?)",
		"sessions", past, "", 1,
	)
	if err != nil {
		t.Fatal(err)
	}

	// No API calls logged — budget is at 0%, well under 50%.
	freshSessions := []store.Session{
		{HexID: "fresh1", Title: "Fresh Talk"},
	}
	fetchCalled := false
	fetchFn := func() ([]store.Session, error) {
		fetchCalled = true
		return freshSessions, nil
	}

	result, err := a.FetchWithCache("sessions", fetchFn)
	if err != nil {
		t.Fatalf("FetchWithCache error: %v", err)
	}
	if !fetchCalled {
		t.Error("fetchFn should be called when cache is stale and under budget")
	}
	if result.FromCache {
		t.Error("expected FromCache=false")
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result.Data))
	}
	if result.Data[0].HexID != "fresh1" {
		t.Errorf("expected fresh1, got %q", result.Data[0].HexID)
	}
	if result.Warning != "" {
		t.Errorf("expected no warning, got %q", result.Warning)
	}
}

func TestCacheFlow_StaleCache_OverBudget_ReturnsCachedWithWarning(t *testing.T) {
	a := newTestApp(t)

	// Insert cached sessions.
	insertSession(t, a.store, "s1", "Cached Talk")

	// Mark cache as stale.
	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err := a.store.DB().Exec(
		"INSERT INTO cache_meta (resource, fetched_at, etag, ttl_seconds) VALUES (?, ?, ?, ?)",
		"sessions", past, "", 1,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Log enough API calls to exceed 50% budget (default limit=30, need >=15).
	for i := 0; i < 16; i++ {
		if err := a.store.LogAPICall("/test", "GET"); err != nil {
			t.Fatal(err)
		}
	}

	fetchCalled := false
	fetchFn := func() ([]store.Session, error) {
		fetchCalled = true
		return nil, nil
	}

	result, err := a.FetchWithCache("sessions", fetchFn)
	if err != nil {
		t.Fatalf("FetchWithCache error: %v", err)
	}
	if fetchCalled {
		t.Error("fetchFn should NOT be called when over budget with cached data")
	}
	if !result.FromCache {
		t.Error("expected FromCache=true")
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 session, got %d", len(result.Data))
	}
	if result.Warning == "" {
		t.Error("expected warning about rate budget, got empty")
	}
}

func TestCacheFlow_NoCache_FetchesRegardless(t *testing.T) {
	a := newTestApp(t)

	// No cache meta, no sessions — completely empty store.
	// Log enough API calls to exceed budget.
	for i := 0; i < 20; i++ {
		if err := a.store.LogAPICall("/test", "GET"); err != nil {
			t.Fatal(err)
		}
	}

	freshSessions := []store.Session{
		{HexID: "essential1", Title: "Essential Talk"},
	}
	fetchCalled := false
	fetchFn := func() ([]store.Session, error) {
		fetchCalled = true
		return freshSessions, nil
	}

	result, err := a.FetchWithCache("sessions", fetchFn)
	if err != nil {
		t.Fatalf("FetchWithCache error: %v", err)
	}
	if !fetchCalled {
		t.Error("fetchFn should be called even when over budget if no cache exists")
	}
	if result.FromCache {
		t.Error("expected FromCache=false")
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result.Data))
	}
	if result.Data[0].HexID != "essential1" {
		t.Errorf("expected essential1, got %q", result.Data[0].HexID)
	}
}
