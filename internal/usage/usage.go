// Package usage meters billable per-tenant activity -- emails sent and
// Scrub validations submitted -- into the usage_counters table, the
// ledger listnun bills PAYG usage from (via the Operator API's
// GET /api/operator/tenants/{id}/usage). Counting happens here, at the
// moment the work is done, rather than being read back later from
// Postmark's or Scrub's own stats, whose retention and day-level
// granularity listnun doesn't control.
//
// Counts are bucketed by UTC hour and are additive: a bucket's count only
// ever grows, so a reader summing buckets in [from, to) counts each hour
// exactly once across back-to-back billing windows.
package usage

import (
	"log"
	"sync"
	"time"
)

// Kinds of usage recorded. Kept in sync with listnun's
// internal/operatorclient, which reads them back by name.
const (
	KindEmailSent  = "email_sent"
	KindValidation = "validation"
)

// BucketSize is the granularity counts are stored at. An hour keeps the
// table small while bounding how far a count can land from its true
// billing-period boundary.
const BucketSize = time.Hour

// Row is one additive increment to a (tenant, kind, bucket) counter.
type Row struct {
	TenantID int
	Kind     string
	Bucket   time.Time
	Count    int64
}

// Store persists increments. AddUsage must add Count to any existing
// counter rather than overwrite it, and must apply all rows or none.
type Store interface {
	AddUsage(rows []Row) error
}

type key struct {
	tenantID int
	kind     string
	bucket   time.Time
}

// Meter buffers increments in memory and writes them to Store in batches,
// so the campaign send loop never waits on a database round trip per
// email. Anything not yet flushed is lost if the process dies without
// Close -- at most one flush interval's worth of counts.
type Meter struct {
	store Store
	log   *log.Logger
	now   func() time.Time

	mu      sync.Mutex
	pending map[key]int64

	stop chan struct{}
	done chan struct{}
}

// New returns a Meter writing to store. Call Run to start periodic
// flushing and Close to stop it.
func New(store Store, l *log.Logger) *Meter {
	return &Meter{
		store:   store,
		log:     l,
		now:     time.Now,
		pending: make(map[key]int64),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Add buffers n units of kind for tenantID in the current hour. Ignores
// non-positive n and tenant IDs (an unscoped, install-level message such
// as a password reset isn't any tenant's usage). Safe on a nil Meter.
func (m *Meter) Add(tenantID int, kind string, n int64) {
	if m == nil || tenantID < 1 || n < 1 {
		return
	}
	k := key{tenantID: tenantID, kind: kind, bucket: m.now().UTC().Truncate(BucketSize)}

	m.mu.Lock()
	m.pending[k] += n
	m.mu.Unlock()
}

// Flush writes everything buffered so far. On failure the increments are
// put back so the next Flush retries them, rather than being dropped.
func (m *Meter) Flush() error {
	m.mu.Lock()
	if len(m.pending) == 0 {
		m.mu.Unlock()
		return nil
	}
	batch := m.pending
	m.pending = make(map[key]int64)
	m.mu.Unlock()

	rows := make([]Row, 0, len(batch))
	for k, n := range batch {
		rows = append(rows, Row{TenantID: k.tenantID, Kind: k.kind, Bucket: k.bucket, Count: n})
	}
	if err := m.store.AddUsage(rows); err != nil {
		m.mu.Lock()
		for k, n := range batch {
			m.pending[k] += n
		}
		m.mu.Unlock()
		return err
	}
	return nil
}

// Run flushes every interval until Close. Blocking; run it in a goroutine.
func (m *Meter) Run(interval time.Duration) {
	defer close(m.done)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := m.Flush(); err != nil {
				m.log.Printf("error flushing usage counters (will retry): %v", err)
			}
		case <-m.stop:
			return
		}
	}
}

// Close stops Run and makes a final flush, so a clean shutdown loses
// nothing. Must be called at most once, after Run was started.
func (m *Meter) Close() error {
	close(m.stop)
	<-m.done
	return m.Flush()
}
