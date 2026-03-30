package rate

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeff/sched-cli/internal/store"
	_ "modernc.org/sqlite"
)

// newTestStore creates a fresh Store backed by a temporary SQLite database.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// --- Allow ---

func TestAllow_TrueWhenNoCalls(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	allowed, err := limiter.Allow()
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Error("Allow() = false, want true (no calls logged)")
	}
}

func TestAllow_TrueWhenUnderLimit(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	// Log 15 calls (under the limit of 30)
	for i := 0; i < 15; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	allowed, err := limiter.Allow()
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Error("Allow() = false, want true (15 calls, limit 30)")
	}
}

func TestAllow_FalseWhenAtLimit(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 5, time.Minute)

	// Log exactly 5 calls (at the limit)
	for i := 0; i < 5; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	allowed, err := limiter.Allow()
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if allowed {
		t.Error("Allow() = true, want false (5 calls, limit 5)")
	}
}

// --- Record ---

func TestRecord_LogsCallThatCountsTowardLimit(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 2, time.Minute)

	// Initially allowed
	allowed, err := limiter.Allow()
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Fatal("Allow() = false before any calls, want true")
	}

	// Record two calls to fill the budget
	if err := limiter.Record("/api/sessions", "GET"); err != nil {
		t.Fatalf("Record() error: %v", err)
	}
	if err := limiter.Record("/api/schedule", "POST"); err != nil {
		t.Fatalf("Record() error: %v", err)
	}

	// Now should be at limit
	allowed, err = limiter.Allow()
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if allowed {
		t.Error("Allow() = true after 2 calls with limit 2, want false")
	}
}

// --- BudgetUsed ---

func TestBudgetUsed_ZeroWhenNoCalls(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	used, err := limiter.BudgetUsed()
	if err != nil {
		t.Fatalf("BudgetUsed() error: %v", err)
	}
	if used != 0.0 {
		t.Errorf("BudgetUsed() = %f, want 0.0", used)
	}
}

func TestBudgetUsed_CorrectFraction(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	// Log 15 calls => 15/30 = 0.5
	for i := 0; i < 15; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	used, err := limiter.BudgetUsed()
	if err != nil {
		t.Fatalf("BudgetUsed() error: %v", err)
	}
	if used != 0.5 {
		t.Errorf("BudgetUsed() = %f, want 0.5", used)
	}
}

func TestBudgetUsed_OneWhenAtLimit(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 10, time.Minute)

	for i := 0; i < 10; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	used, err := limiter.BudgetUsed()
	if err != nil {
		t.Fatalf("BudgetUsed() error: %v", err)
	}
	if used != 1.0 {
		t.Errorf("BudgetUsed() = %f, want 1.0", used)
	}
}

// --- SmartRefreshAllowed ---

func TestSmartRefreshAllowed_TrueWhenUnder50Percent(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	// Log 14 calls => 14/30 = 0.467 (under 50%)
	for i := 0; i < 14; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	allowed, err := limiter.SmartRefreshAllowed()
	if err != nil {
		t.Fatalf("SmartRefreshAllowed() error: %v", err)
	}
	if !allowed {
		t.Error("SmartRefreshAllowed() = false, want true (14/30 = 46.7%%)")
	}
}

func TestSmartRefreshAllowed_FalseAtExactly50Percent(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	// Log 15 calls => 15/30 = 0.5 (exactly 50%)
	for i := 0; i < 15; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	allowed, err := limiter.SmartRefreshAllowed()
	if err != nil {
		t.Fatalf("SmartRefreshAllowed() error: %v", err)
	}
	if allowed {
		t.Error("SmartRefreshAllowed() = true, want false (15/30 = exactly 50%%)")
	}
}

func TestSmartRefreshAllowed_FalseWhenAbove50Percent(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	// Log 20 calls => 20/30 = 0.667 (above 50%)
	for i := 0; i < 20; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	allowed, err := limiter.SmartRefreshAllowed()
	if err != nil {
		t.Fatalf("SmartRefreshAllowed() error: %v", err)
	}
	if allowed {
		t.Error("SmartRefreshAllowed() = true, want false (20/30 = 66.7%%)")
	}
}

// --- Status ---

func TestStatus_CorrectFieldsNoCalls(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	status, err := limiter.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.CallsInWindow != 0 {
		t.Errorf("CallsInWindow = %d, want 0", status.CallsInWindow)
	}
	if status.Limit != 30 {
		t.Errorf("Limit = %d, want 30", status.Limit)
	}
	if status.Window != time.Minute {
		t.Errorf("Window = %v, want %v", status.Window, time.Minute)
	}
	if status.BudgetUsed != 0.0 {
		t.Errorf("BudgetUsed = %f, want 0.0", status.BudgetUsed)
	}
	if status.Remaining != 30 {
		t.Errorf("Remaining = %d, want 30", status.Remaining)
	}
}

func TestStatus_CorrectFieldsWithCalls(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 20, 5*time.Minute)

	// Log 8 calls
	for i := 0; i < 8; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error on call %d: %v", i, err)
		}
	}

	status, err := limiter.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.CallsInWindow != 8 {
		t.Errorf("CallsInWindow = %d, want 8", status.CallsInWindow)
	}
	if status.Limit != 20 {
		t.Errorf("Limit = %d, want 20", status.Limit)
	}
	if status.Window != 5*time.Minute {
		t.Errorf("Window = %v, want %v", status.Window, 5*time.Minute)
	}
	wantBudget := 8.0 / 20.0
	if status.BudgetUsed != wantBudget {
		t.Errorf("BudgetUsed = %f, want %f", status.BudgetUsed, wantBudget)
	}
	if status.Remaining != 12 {
		t.Errorf("Remaining = %d, want 12 (20 - 8)", status.Remaining)
	}
}

func TestStatus_RemainingCalculation(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 5, time.Minute)

	// Log 3 calls
	for i := 0; i < 3; i++ {
		if err := limiter.Record("/test", "GET"); err != nil {
			t.Fatalf("Record() error: %v", err)
		}
	}

	status, err := limiter.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.Remaining != 2 {
		t.Errorf("Remaining = %d, want 2 (5 - 3)", status.Remaining)
	}
}

// --- Wait ---

func TestWait_ReturnsImmediatelyWhenAllowed(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 30, time.Minute)

	ctx := context.Background()
	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() error: %v", err)
	}
	// Should return almost immediately (well under 100ms)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Wait() took %v, expected near-instant return", elapsed)
	}
}

func TestWait_ReturnsContextCanceledWhenCancelled(t *testing.T) {
	s := newTestStore(t)
	limiter := New(s, 1, time.Minute)

	// Fill the budget so Wait will block
	if err := limiter.Record("/test", "GET"); err != nil {
		t.Fatalf("Record() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := limiter.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}
