package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_37_0 adds segments: named groups of subscribers a workspace saves
// from the Subscribers condition builder ("Harare, joined recently").
// conditions holds internal/segments' conditions; they're compiled to SQL
// when the segment is used, never stored as SQL.
func V6_37_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS segments (
			id               SERIAL PRIMARY KEY,
			tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			uuid             uuid NOT NULL UNIQUE,
			name             TEXT NOT NULL,
			match            TEXT NOT NULL DEFAULT 'all',
			conditions       JSONB NOT NULL DEFAULT '[]',
			created_by_name  TEXT NOT NULL DEFAULT '',
			created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_segments_tenant ON segments(tenant_id);

		ALTER TABLE segments ENABLE ROW LEVEL SECURITY;
		ALTER TABLE segments FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON segments;
		CREATE POLICY tenant_isolation ON segments
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`)
	return err
}
