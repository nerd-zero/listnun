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
// The cache is never invalidated within a process's lifetime - matches
// every other per-tenant resolver added this session (settings updates
// already require a full process restart to take effect).
type tenantImporters struct {
	q       *models.Queries
	db      *sqlx.DB
	core    *core.Core
	manager *manager.Manager
	sysMail *systemMail
	i18n    *i18n.I18n

	mu    sync.Mutex
	cache map[int]*subimporter.Importer
}

func newTenantImporters(q *models.Queries, db *sqlx.DB, co *core.Core, mgr *manager.Manager, sm *systemMail, i *i18n.I18n) *tenantImporters {
	return &tenantImporters{
		q:       q,
		db:      db,
		core:    co,
		manager: mgr,
		sysMail: sm,
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
		TenantID:           tenantID,

		// Scrub validate-on-import. The tenant's Scrub settings are read
		// at each submission, not baked in like DomainBlocklist above:
		// this importer is cached for the process's lifetime, so a tenant
		// that turned Scrub on after its first import would otherwise
		// never have an import validated until a restart. Submission is
		// fire-and-forget: results come back later via
		// /webhooks/scrub/batch (cmd/scrub_batch.go). A failed submission
		// leaves those subscribers unchecked, so it's recorded in the
		// tenant's activity log, not only the process log -- found live:
		// an import's batches failed with nothing but a log line nobody
		// saw, and its subscribers were never validated.
		ScrubSubmitFunc: func(listIDs []int, emails []string) error {
			ctx := context.Background()
			s, err := t.core.GetSettings(ctx, tenantID)
			if err == nil {
				_, err = submitScrubBatch(ctx, t.q, tenantID, s, listIDs, emails, "")
			}
			if err != nil {
				t.core.LogActivity(tenantID, models.ActivityWarn, "import",
					fmt.Sprintf("%d imported subscribers couldn't be sent to Scrub for validation and are unchecked: %v", len(emails), err))
			}
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
