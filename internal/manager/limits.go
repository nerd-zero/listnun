package manager

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SendLimits is one tenant's own sending limits (its Settings ›
// Performance), applied to its campaigns only.
type SendLimits struct {
	// PerSecond caps the tenant's messages per second across all workers.
	// 0 is no cap of its own (the process-wide per-worker rate still
	// applies).
	PerSecond int

	// Window and WindowMax cap the tenant's messages per sliding window,
	// e.g. 10,000 per hour for its provider's quota. 0 for either is no
	// window.
	Window    time.Duration
	WindowMax int
}

// windowOn says whether l has a sliding window.
func (l SendLimits) windowOn() bool {
	return l.WindowMax > 0 && l.Window >= time.Second
}

// tenantLimiter is one tenant's sending state against its SendLimits.
type tenantLimiter struct {
	limits SendLimits
	rate   *rate.Limiter

	windowStart time.Time
	windowCount int
}

// limiters holds each tenant's limiter. The limits used to be one global
// set, tenant 1's: its sliding window slept the single loop that feeds
// every campaign, so one tenant reaching its hourly quota paused sending
// for all of them.
type limiters struct {
	get func(tenantID int) SendLimits
	now func() time.Time

	mu sync.Mutex
	by map[int]*tenantLimiter
}

func newLimiters(get func(tenantID int) SendLimits) *limiters {
	if get == nil {
		get = func(int) SendLimits { return SendLimits{} }
	}
	return &limiters{get: get, now: time.Now, by: map[int]*tenantLimiter{}}
}

// limiter returns tenantID's limiter, updated to its current limits lim
// (read before taking the lock, since reading them can hit the database).
// Must be called with l.mu held.
func (l *limiters) limiter(tenantID int, lim SendLimits) *tenantLimiter {
	t, ok := l.by[tenantID]
	if !ok {
		t = &tenantLimiter{windowStart: l.now()}
		l.by[tenantID] = t
	}
	if !ok || t.limits != lim {
		t.limits = lim
		t.rate = nil
		if lim.PerSecond > 0 {
			t.rate = rate.NewLimiter(rate.Limit(lim.PerSecond), lim.PerSecond)
		}
	}
	return t
}

// wait blocks until tenantID may send another message under its own
// per-second cap.
func (l *limiters) wait(tenantID int) {
	lim := l.get(tenantID)
	l.mu.Lock()
	r := l.limiter(tenantID, lim).rate
	l.mu.Unlock()
	if r != nil {
		_ = r.Wait(context.Background())
	}
}

// allowance returns how many more messages tenantID may send in its
// current window (max when it has no window), or, when none, how long
// until the window resets.
func (l *limiters) allowance(tenantID, max int) (int, time.Duration) {
	lim := l.get(tenantID)
	l.mu.Lock()
	defer l.mu.Unlock()

	t := l.limiter(tenantID, lim)
	if !t.limits.windowOn() {
		return max, 0
	}

	now := l.now()
	if now.Sub(t.windowStart) >= t.limits.Window {
		t.windowStart, t.windowCount = now, 0
	}
	left := t.limits.WindowMax - t.windowCount
	if left <= 0 {
		return 0, t.limits.Window - now.Sub(t.windowStart)
	}
	return min(left, max), 0
}

// sent counts n messages against tenantID's window.
func (l *limiters) sent(tenantID, n int) {
	lim := l.get(tenantID)
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.limiter(tenantID, lim)
	if t.limits.windowOn() {
		t.windowCount += n
	}
}
