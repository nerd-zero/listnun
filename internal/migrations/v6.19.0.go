package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_19_0 moves the subscriber-list "Validate" feature (cmd/settings.go's
// ScrubList) off Scrub's list-sync endpoints (POST/GET
// /v1/integrations/{id}/lists/...), which require Scrub to call back into
// listmonk using per-tenant reverse credentials, onto the same async
// batch-submit pipeline v6.18.0 built for CSV import
// (POST /v1/validate/integration + /webhooks/scrub/batch) -- listmonk
// fetches the list's own emails and submits them directly instead.
//
// Adds scrub_validation_jobs: a list-validate action may fan out into
// multiple scrub_validation_batches rows, chunked at Scrub's 30,000-email
// per-request cap, tracked together as one job (see
// cmd/scrub_batch.go's submitScrubListValidation/reconcileScrubJob).
// scrub_validation_batches gains job_id (NULL for a standalone CSV-import
// batch, which never needs more than one chunk) and validated_count (the
// webhook's validated_total_count, previously discarded, needed to answer
// GetScrubListProgress's "validated so far" field).
//
// lists gains scrub_last_validated_at/scrub_last_valid_count/
// scrub_last_invalid_count: GetScrubListStatus's "last_result" now has to
// be persisted locally, since Scrub's own list-tracking is no longer
// queried at all.
func V6_19_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS scrub_last_validated_at TIMESTAMP WITH TIME ZONE NULL;
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS scrub_last_valid_count INTEGER NULL;
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS scrub_last_invalid_count INTEGER NULL;

		CREATE TABLE IF NOT EXISTS scrub_validation_jobs (
			job_id            UUID PRIMARY KEY,
			tenant_id         INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			list_ids          INTEGER[] NOT NULL DEFAULT '{}',
			total_batches     INTEGER NOT NULL,
			completed_batches INTEGER NOT NULL DEFAULT 0,
			submitted_count   INTEGER NOT NULL DEFAULT 0,
			validated_count   INTEGER NOT NULL DEFAULT 0,
			invalid_count     INTEGER NOT NULL DEFAULT 0,
			status            TEXT NOT NULL DEFAULT 'processing',
			created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_scrub_validation_jobs_tenant ON scrub_validation_jobs(tenant_id);

		ALTER TABLE scrub_validation_jobs ENABLE ROW LEVEL SECURITY;
		ALTER TABLE scrub_validation_jobs FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON scrub_validation_jobs;
		CREATE POLICY tenant_isolation ON scrub_validation_jobs
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);

		ALTER TABLE scrub_validation_batches ADD COLUMN IF NOT EXISTS job_id UUID NULL
			REFERENCES scrub_validation_jobs(job_id) ON DELETE CASCADE;
		ALTER TABLE scrub_validation_batches ADD COLUMN IF NOT EXISTS validated_count INTEGER NOT NULL DEFAULT 0;
		CREATE INDEX IF NOT EXISTS idx_scrub_validation_batches_job ON scrub_validation_batches(job_id) WHERE job_id IS NOT NULL;
	`); err != nil {
		return err
	}

	return nil
}
