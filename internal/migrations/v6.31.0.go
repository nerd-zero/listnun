package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_31_0 adds the per-workspace activity log behind Settings › Logs: a
// short, read-only record of what happened in one workspace (settings
// saved, campaigns started and finished, imports, webhook failures). Rows
// are tenant-scoped with row-level security, unlike the process-wide log
// buffer that #98 stopped exposing, and are pruned after 30 days.
func V6_31_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS activity_log (
			id          BIGSERIAL PRIMARY KEY,
			tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			level       TEXT NOT NULL DEFAULT 'info',
			source      TEXT NOT NULL,
			message     TEXT NOT NULL,
			created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_activity_log_tenant ON activity_log(tenant_id, created_at DESC);

		ALTER TABLE activity_log ENABLE ROW LEVEL SECURITY;
		ALTER TABLE activity_log FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON activity_log;
		CREATE POLICY tenant_isolation ON activity_log
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
