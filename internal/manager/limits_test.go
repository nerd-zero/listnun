package manager

import (
	"testing"
	"time"
)

// Sending limits are per tenant. They used to be one set, tenant 1's, whose
// sliding window slept the loop that feeds every tenant's campaigns.
func TestLimiters_WindowPerTenant(t *testing.T) {
	limits := map[int]SendLimits{
		2: {Window: time.Hour, WindowMax: 100},
		3: {},
	}
	l := newLimiters(func(id int) SendLimits { return limits[id] })
	now := time.Now()
	l.now = func() time.Time { return now }

	if n, _ := l.allowance(2, 1000); n != 100 {
		t.Fatalf("tenant 2 allowance = %d, want its window's 100", n)
	}
	l.sent(2, 100)
	n, wait := l.allowance(2, 1000)
	if n != 0 || wait != time.Hour {
		t.Errorf("tenant 2 after 100 sent: %d left, wait %s; want 0 and the rest of the hour", n, wait)
	}
	if n, _ := l.allowance(3, 1000); n != 1000 {
		t.Errorf("tenant 3 allowance = %d, want a full batch: tenant 2's window isn't its", n)
	}

	now = now.Add(time.Hour)
	if n, _ := l.allowance(2, 1000); n != 100 {
		t.Errorf("tenant 2 after its window = %d, want a fresh 100", n)
	}

	// A changed limit is picked up.
	limits[2] = SendLimits{}
	if n, _ := l.allowance(2, 1000); n != 1000 {
		t.Errorf("tenant 2 with no window = %d, want a full batch", n)
	}
}

func TestLimiters_PerSecond(t *testing.T) {
	l := newLimiters(func(id int) SendLimits {
		if id == 2 {
			return SendLimits{PerSecond: 5}
		}
		return SendLimits{}
	})

	// Tenant 3 has no cap of its own: no wait.
	start := time.Now()
	for range 50 {
		l.wait(3)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Errorf("tenant 3 was slowed down by another tenant's cap")
	}

	// Tenant 2 bursts 5, then ~5/s: 8 sends take at least half a second.
	start = time.Now()
	for range 8 {
		l.wait(2)
	}
	if d := time.Since(start); d < 500*time.Millisecond {
		t.Errorf("tenant 2 sent 8 in %s, want its 5/s cap applied", d)
	}
}
