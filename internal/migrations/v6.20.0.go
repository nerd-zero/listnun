package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_20_0 adds usage_counters, the billing ledger listnun reads through
// the Operator API: emails each tenant sent and Scrub validations it
// submitted, counted at the moment they happen (internal/usage) rather
// than read back later from Postmark's or Scrub's own stats, which
// listnun doesn't control the retention or granularity of. No FK to
// tenants, so a deleted tenant's already-incurred usage can still be
// invoiced.
func V6_20_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS usage_counters (
			tenant_id   INTEGER NOT NULL,
			kind        TEXT NOT NULL,
			bucket      TIMESTAMP WITH TIME ZONE NOT NULL,
			count       BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (tenant_id, kind, bucket)
		);
		-- Cross-tenant window scans (the Operator API's platform usage).
		CREATE INDEX IF NOT EXISTS idx_usage_counters_bucket ON usage_counters(bucket);

		ALTER TABLE usage_counters ENABLE ROW LEVEL SECURITY;
		ALTER TABLE usage_counters FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON usage_counters;
		CREATE POLICY tenant_isolation ON usage_counters
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
