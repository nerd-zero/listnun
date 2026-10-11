package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_12_0 adds organizations - a purely cross-tenant grouping construct,
// managed only via the Operator API. One organization can own multiple
// tenants ("listnuns") for different purposes; tenants.organization_id
// is nullable so existing/standalone tenants aren't forced into one.
// Never RLS-scoped and never resolved per-request the way tenants are -
// this is a management/billing-side concept, not part of the
// tenant-resolution or request-scoping paths.
func V6_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id          SERIAL PRIMARY KEY,
			name        TEXT NOT NULL,
			created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		ALTER TABLE tenants ADD COLUMN IF NOT EXISTS organization_id INTEGER NULL REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE;

		DROP INDEX IF EXISTS idx_tenants_organization;
		CREATE INDEX idx_tenants_organization ON tenants(organization_id);
	`)
	return err
}
