package main

import (
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/nerd-zero/listnun/internal/manager"
	"github.com/nerd-zero/listnun/models"
)

// tenantTrackingTTL is how long a tenant's tracking settings are cached.
// Every tracked link and view pixel asks, so they can't be read fresh each
// time; a minute keeps other replicas close behind a settings save.
const tenantTrackingTTL = time.Minute

// tenantTracking resolves each tenant's own link/view tracking settings
// (privacy.disable_tracking, privacy.individual_tracking). These used to
// come from the boot-time config, which is tenant 1's settings, so every
// other tenant's toggles were ignored.
type tenantTracking struct {
	q   *models.Queries
	log *log.Logger
	now func() time.Time

	mu    sync.Mutex
	cache map[int]tenantTrackingEntry
}

type tenantTrackingEntry struct {
	tr      manager.Tracking
	fetched time.Time
}

func newTenantTracking(q *models.Queries, lo *log.Logger) *tenantTracking {
	return &tenantTracking{q: q, log: lo, now: time.Now, cache: map[int]tenantTrackingEntry{}}
}

// For returns tenantID's tracking settings. If they can't be read, the last
// cached value is kept; with none, tracking stays on without per-subscriber
// IDs (the schema default), so a database hiccup never records people
// against a tenant's wishes.
func (t *tenantTracking) For(tenantID int) manager.Tracking {
	t.mu.Lock()
	e, ok := t.cache[tenantID]
	t.mu.Unlock()
	if ok && t.now().Sub(e.fetched) < tenantTrackingTTL {
		return e.tr
	}

	var tr manager.Tracking
	if err := t.q.GetTenantTracking.Get(&tr, tenantID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.log.Printf("error reading tenant %d's tracking settings: %v", tenantID, err)
		tr = e.tr
	}

	t.mu.Lock()
	t.cache[tenantID] = tenantTrackingEntry{tr: tr, fetched: t.now()}
	t.mu.Unlock()
	return tr
}

// invalidate drops tenantID's cached settings after they're saved.
func (t *tenantTracking) invalidate(tenantID int) {
	t.mu.Lock()
	delete(t.cache, tenantID)
	t.mu.Unlock()
}
