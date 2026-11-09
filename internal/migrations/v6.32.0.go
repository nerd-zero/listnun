package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_32_0 adds inactive-subscriber cleanup (the design's Settings › Scrub,
// named apart from the Scrub email-validation service). The setting starts
// disabled in managed mode, and inactive_cleanup_runs records each run so
// the schedule doesn't repeat and the screen can show the last result.
func V6_32_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (tenant_id, key, value)
		SELECT id, 'inactive_cleanup', '{"enabled": false, "mode": "managed", "days": 180, "activity": "either", "action": "unsubscribe", "list_ids": [], "schedule": "0 2 * * 0"}'::JSONB
		FROM tenants
		ON CONFLICT (tenant_id, key) DO NOTHING;

		CREATE TABLE IF NOT EXISTS inactive_cleanup_runs (
			id           SERIAL PRIMARY KEY,
			tenant_id    INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			mode         TEXT NOT NULL,
			action       TEXT NOT NULL,
			matched      INTEGER NOT NULL DEFAULT 0,
			done         INTEGER NOT NULL DEFAULT 0,
			error        TEXT NOT NULL DEFAULT '',
			started_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			finished_at  TIMESTAMP WITH TIME ZONE NULL
		);
		CREATE INDEX IF NOT EXISTS idx_inactive_cleanup_runs ON inactive_cleanup_runs(tenant_id, started_at DESC);

		ALTER TABLE inactive_cleanup_runs ENABLE ROW LEVEL SECURITY;
		ALTER TABLE inactive_cleanup_runs FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON inactive_cleanup_runs;
		CREATE POLICY tenant_isolation ON inactive_cleanup_runs
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
