package main

import (
	"context"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
)

// tenantImporters lazily builds and caches a *subimporter.Importer per
// tenant, the same pattern as tenantMedia/tenantMessengers. The old
// boot-time-global importer had two bugs under multi-tenancy: its
// Status/lock is a single field on the struct, so only one tenant could
// ever import at a time and every tenant's Import page showed whichever
// tenant's run happened to be the shared instance's current/last one
// (found live via a real cross-tenant leak - one tenant's admin session
// showed a completely different tenant's import filename/status); and
// its domain blocklist/allowlist were read once at boot from tenant 1's
// settings via the global koanf instance, even though those are
// per-tenant settings (phase 5) - so SanitizeEmail/ValidateFields
// (used well beyond bulk CSV import: the public subscription form,
// bounce processing, campaign from-email checks, tx messages, direct
// subscriber creation) validated every tenant's e-mails against tenant
// 1's rules.
//
// The cache is never invalidated within a process's lifetime - matches
// every other per-tenant resolver added this session (settings updates
// already require a full process restart to take effect).
type tenantImporters struct {
	q       *models.Queries
	db      *sqlx.DB
	core    *core.Core
	manager *manager.Manager
	i18n    *i18n.I18n

	mu    sync.Mutex
	cache map[int]*subimporter.Importer
}

func newTenantImporters(q *models.Queries, db *sqlx.DB, co *core.Core, mgr *manager.Manager, i *i18n.I18n) *tenantImporters {
	return &tenantImporters{
		q:       q,
		db:      db,
		core:    co,
		manager: mgr,
		i18n:    i,
		cache:   make(map[int]*subimporter.Importer),
	}
}

// Get returns the given tenant's importer, building and caching it from
// that tenant's own settings on first use.
func (t *tenantImporters) Get(ctx context.Context, tenantID int) (*subimporter.Importer, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if imp, ok := t.cache[tenantID]; ok {
		return imp, nil
	}

	settings, err := t.core.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	imp := subimporter.New(subimporter.Options{
		DomainBlocklist:    settings.DomainBlocklist,
		DomainAllowlist:    settings.DomainAllowlist,
		UpsertStmt:         t.q.UpsertSubscriber.Stmt,
		BlocklistStmt:      t.q.UpsertBlocklistSubscriber.Stmt,
		UpdateListDateStmt: t.q.UpdateListsDate.Stmt,

		// Scrub validate-on-import -- baked in from this tenant's settings
		// at first use, same "requires a process restart to pick up a
		// settings change" precedent as DomainBlocklist/DomainAllowlist
		// above (this whole importer is cached per-tenant for the
		// process's lifetime).
		ScrubEnabled:       settings.Scrub.Enabled && settings.Scrub.URL != "" && settings.Scrub.APIKey != "",
		ScrubBaseURL:       settings.Scrub.URL,
		ScrubAPIKey:        settings.Scrub.APIKey,
		TenantID:           tenantID,
		SetScrubStatusStmt: t.q.SetSubscribersScrubStatusByEmail.Stmt,
		OnRiskySubscriber: func(listIDs []int) error {
			pauseCampaignsForRiskySubscriberWith(context.Background(), t.core, t.manager, tenantID, listIDs)
			return nil
		},

		// Hook for triggering admin notifications and refreshing stats
		// materialized views after a successful import. RefreshMatViews
		// itself isn't tenant-scoped - one global REFRESH still updates
		// every tenant's row in the shared matview, matching the
		// existing documented default (see v6.7.0's migration).
		PostCB: func(subject string, data any) error {
			t.core.RefreshMatViews(true)
			notifs.NotifySystem(subject, notifs.TplImport, data, nil)
			return nil
		},
	}, t.db.DB, t.i18n)

	t.cache[tenantID] = imp

	return imp, nil
}
