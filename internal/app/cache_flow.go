package app

import (
	"github.com/jeff/sched-cli/internal/store"
)

// FetchResult holds the outcome of a cache-aware fetch operation.
type FetchResult struct {
	Data      []store.Session
	FromCache bool
	Warning   string // non-empty if stale data returned due to rate budget
}

// FetchWithCache implements rate-budget-aware caching. It checks cache
// freshness and rate budget before deciding whether to fetch fresh data
// or return cached results.
//
// Logic:
//  1. If cache is fresh -> return cached data
//  2. If cache is stale + under budget -> call fetchFn for fresh data
//  3. If cache is stale + over budget + cache exists -> return stale + warning
//  4. If cache is stale + over budget + no cache -> call fetchFn (essential)
func (a *App) FetchWithCache(resource string, fetchFn func() ([]store.Session, error)) (*FetchResult, error) {
	stale, err := a.store.IsCacheStale(resource)
	if err != nil {
		return nil, err
	}

	// Step 1: fresh cache — return what we have.
	if !stale {
		sessions, err := a.store.ListSessions(store.SessionFilters{})
		if err != nil {
			return nil, err
		}
		return &FetchResult{
			Data:      sessions,
			FromCache: true,
		}, nil
	}

	// Cache is stale — check rate budget.
	allowed, err := a.limiter.SmartRefreshAllowed()
	if err != nil {
		return nil, err
	}

	// Step 2: stale + under budget -> fetch fresh data.
	if allowed {
		data, err := fetchFn()
		if err != nil {
			return nil, err
		}
		return &FetchResult{
			Data:      data,
			FromCache: false,
		}, nil
	}

	// Over budget — check if we have any cached data at all.
	count, err := a.store.SessionCount()
	if err != nil {
		return nil, err
	}

	// Step 4: over budget + no cache -> fetch anyway (essential request).
	if count == 0 {
		data, err := fetchFn()
		if err != nil {
			return nil, err
		}
		return &FetchResult{
			Data:      data,
			FromCache: false,
		}, nil
	}

	// Step 3: over budget + cache exists -> return stale data with warning.
	sessions, err := a.store.ListSessions(store.SessionFilters{})
	if err != nil {
		return nil, err
	}
	return &FetchResult{
		Data:      sessions,
		FromCache: true,
		Warning:   "using cached data: rate budget exceeded, refresh skipped",
	}, nil
}
