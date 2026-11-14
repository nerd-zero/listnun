package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/internal/manager"
	"github.com/nerd-zero/listnun/internal/notifs"
	"github.com/nerd-zero/listnun/internal/subimporter"
	"github.com/nerd-zero/listnun/models"
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
// A settings save marks the tenant's importer stale (invalidate); it's
// rebuilt from the new settings on next use once it isn't importing, so a
// running import keeps its state.
type tenantImporters struct {
	q *models.Queries
	// sysQ are the system pool's statements, for Scrub batches (cmd/scrub_batch.go).
	sysQ    *models.Queries
	db      *sqlx.DB
	core    *core.Core
	manager *manager.Manager
	sysMail *systemMail
	i18n    *i18n.I18n

	mu    sync.Mutex
	cache map[int]*subimporter.Importer
	stale map[int]bool
}

func newTenantImporters(q, sysQ *models.Queries, db *sqlx.DB, co *core.Core, mgr *manager.Manager, sm *systemMail, i *i18n.I18n) *tenantImporters {
	return &tenantImporters{
		q:       q,
		sysQ:    sysQ,
		db:      db,
		core:    co,
		manager: mgr,
		sysMail: sm,
		i18n:    i,
		cache:   make(map[int]*subimporter.Importer),
		stale:   make(map[int]bool),
	}
}

// invalidate marks tenantID's importer to be rebuilt from its settings
// (domain block/allow lists, Scrub) once it's idle.
func (t *tenantImporters) invalidate(tenantID int) {
	t.mu.Lock()
	t.stale[tenantID] = true
	t.mu.Unlock()
}

// idle says whether imp has no import running.
func idle(imp *subimporter.Importer) bool {
	s := imp.GetStats().Status
	return s != subimporter.StatusImporting && s != subimporter.StatusStopping
}

// Get returns the given tenant's importer, building and caching it from
// that tenant's own settings on first use.
func (t *tenantImporters) Get(ctx context.Context, tenantID int) (*subimporter.Importer, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if imp, ok := t.cache[tenantID]; ok && !(t.stale[tenantID] && idle(imp)) {
		return imp, nil
	}
	delete(t.stale, tenantID)

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
		TenantID:           tenantID,

		// Scrub validate-on-import -- baked in from this tenant's settings
		// at first use, same "requires a process restart to pick up a
		// settings change" precedent as DomainBlocklist/DomainAllowlist
		// above (this whole importer is cached per-tenant for the
		// process's lifetime). Submission is fire-and-forget: results
		// come back later via /webhooks/scrub/batch (cmd/scrub_batch.go),
		// not synchronously here.
		ScrubSubmitFunc: func(listIDs []int, emails []string) error {
			_, err := submitScrubBatch(context.Background(), t.sysQ, tenantID, settings, listIDs, emails, "")
			return err
		},

		// Hook for triggering admin notifications and refreshing stats
		// materialized views after a successful import. RefreshMatViews
		// itself isn't tenant-scoped - one global REFRESH still updates
		// every tenant's row in the shared matview, matching the
		// existing documented default (see v6.7.0's migration).
		PostCB: func(subject string, data any) error {
			if n, ok := data.(subimporter.ImportNotif); ok {
				level := models.ActivityInfo
				if n.Status != subimporter.StatusFinished {
					level = models.ActivityWarn
				}
				t.core.LogActivity(tenantID, level, "import",
					fmt.Sprintf("%s %s, %d of %d records", n.Name, n.Status, n.Imported, n.Total))
				if err := t.core.RecordImport(context.Background(), models.ImportRecord{
					TenantID: tenantID, Name: n.Name, Mode: n.Mode, Status: n.Status, Total: n.Total,
					Imported: n.Imported, Added: n.Added, Skipped: n.Skipped, Failed: n.Failed,
				}); err != nil {
					t.core.LogActivity(tenantID, models.ActivityWarn, "import", fmt.Sprintf("error saving import history: %v", err))
				}
			}
			t.core.RefreshMatViews(true)
			// To this workspace's own notify addresses.
			_ = t.sysMail.notifyAdmins(context.Background(), tenantID, subject, notifs.TplImport, data)
			return nil
		},
	}, t.db.DB, t.i18n)

	t.cache[tenantID] = imp

	return imp, nil
}
