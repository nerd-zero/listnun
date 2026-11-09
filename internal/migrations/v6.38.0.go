package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_38_0 adds imports: one row per finished subscriber import, so the
// Import page can list recent ones after the importer's in-memory status
// is gone.
func V6_38_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS imports (
			id          SERIAL PRIMARY KEY,
			tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			name        TEXT NOT NULL,
			mode        TEXT NOT NULL,
			status      TEXT NOT NULL,
			total       INTEGER NOT NULL DEFAULT 0,
			imported    INTEGER NOT NULL DEFAULT 0,
			added       INTEGER NOT NULL DEFAULT 0,
			skipped     INTEGER NOT NULL DEFAULT 0,
			failed      INTEGER NOT NULL DEFAULT 0,
			created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_imports_tenant ON imports(tenant_id, created_at DESC);

		ALTER TABLE imports ENABLE ROW LEVEL SECURITY;
		ALTER TABLE imports FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON imports;
		CREATE POLICY tenant_isolation ON imports
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`)
	return err
}
