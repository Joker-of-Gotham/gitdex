package github

import (
	"sync"
	"time"
)

// RateLimitBudget tracks GitHub API rate limit consumption.
type RateLimitBudget struct {
	mu        sync.Mutex
	remaining int
	resetAt   time.Time
	limit     int
}

// NewRateLimitBudget creates a new RateLimitBudget with the given limit.
func NewRateLimitBudget(limit int) *RateLimitBudget {
	return &RateLimitBudget{
		remaining: limit,
		limit:     limit,
	}
}

// Update updates the budget with fresh rate limit info from a response.
func (b *RateLimitBudget) Update(remaining int, resetAt time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.remaining = remaining
	b.resetAt = resetAt
}

// CanProceed returns true if there are remaining requests or the reset time has passed.
func (b *RateLimitBudget) CanProceed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.remaining > 0 {
		return true
	}
	return time.Now().After(b.resetAt)
}

// Remaining returns the number of remaining requests.
func (b *RateLimitBudget) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.remaining
}
