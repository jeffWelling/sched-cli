package rate

import (
	"context"
	"fmt"
	"time"

	"github.com/jeff/sched-cli/internal/store"
)

const (
	DefaultLimit  = 30
	DefaultWindow = time.Minute
)

// Status holds current rate limit information.
type Status struct {
	CallsInWindow int           `json:"calls_in_window"`
	Limit         int           `json:"limit"`
	Window        time.Duration `json:"window"`
	BudgetUsed    float64       `json:"budget_used"`    // 0.0 to 1.0
	Remaining     int           `json:"remaining"`
}

// Limiter tracks API call rates using the store's api_calls table.
type Limiter struct {
	store  *store.Store
	limit  int
	window time.Duration
}

// New creates a Limiter with the given limits.
func New(s *store.Store, limit int, window time.Duration) *Limiter {
	return &Limiter{
		store:  s,
		limit:  limit,
		window: window,
	}
}

// Record logs an API call.
func (l *Limiter) Record(endpoint, method string) error {
	return l.store.LogAPICall(endpoint, method)
}

// Allow checks if another API call is within the rate limit.
func (l *Limiter) Allow() (bool, error) {
	count, err := l.store.GetAPICallCount(l.window)
	if err != nil {
		return false, err
	}
	return count < l.limit, nil
}

// Wait blocks until a call is allowed, or the context is cancelled.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		allowed, err := l.Allow()
		if err != nil {
			return err
		}
		if allowed {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			// poll every second
		}
	}
}

// BudgetUsed returns the fraction of rate budget consumed (0.0 to 1.0).
func (l *Limiter) BudgetUsed() (float64, error) {
	count, err := l.store.GetAPICallCount(l.window)
	if err != nil {
		return 0, err
	}
	return float64(count) / float64(l.limit), nil
}

// SmartRefreshAllowed returns true if under 50% of the rate budget.
func (l *Limiter) SmartRefreshAllowed() (bool, error) {
	used, err := l.BudgetUsed()
	if err != nil {
		return false, err
	}
	return used < 0.5, nil
}

// Status returns the current rate limit status.
func (l *Limiter) Status() (*Status, error) {
	count, err := l.store.GetAPICallCount(l.window)
	if err != nil {
		return nil, fmt.Errorf("getting api call count: %w", err)
	}
	used := float64(count) / float64(l.limit)
	return &Status{
		CallsInWindow: count,
		Limit:         l.limit,
		Window:        l.window,
		BudgetUsed:    used,
		Remaining:     l.limit - count,
	}, nil
}
