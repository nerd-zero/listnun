package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_18_0 adds scrub_validation_batches, tracking CSV-import batches
// submitted to Scrub's async POST /v1/validate/integration so the
// /webhooks/scrub/batch callback (which only carries a batch_id, no
// tenant context) can look up which tenant/lists a batch belongs to. That
// callback is authenticated by verifying Scrub's HMAC signature
// (internal/scrub.VerifyWebhook) against the resolved tenant's own
// Settings.Scrub.APIKey -- the same shared platform key every tenant
// already has (pushed down by listnun's operator API) -- not a
// per-batch secret stored on this table. See cmd/scrub_batch.go and
// internal/scrub.Client.SubmitBatch/GetHistory, which replace the old
// synchronous ValidateBulk this version removes.
func V6_18_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS scrub_validation_batches (
			batch_id        UUID PRIMARY KEY,
			tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			list_ids        INTEGER[] NOT NULL DEFAULT '{}',
			status          TEXT NOT NULL DEFAULT 'pending',
			submitted_count INTEGER NOT NULL DEFAULT 0,
			invalid_count   INTEGER NOT NULL DEFAULT 0,
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_scrub_validation_batches_tenant ON scrub_validation_batches(tenant_id);

		ALTER TABLE scrub_validation_batches ENABLE ROW LEVEL SECURITY;
		ALTER TABLE scrub_validation_batches FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON scrub_validation_batches;
		CREATE POLICY tenant_isolation ON scrub_validation_batches
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
