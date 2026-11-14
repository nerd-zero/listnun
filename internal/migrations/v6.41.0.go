package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_41_0 marks each workspace's Super Admin role. Super Admin used to be
// recognised by role ID 1, which only tenant 1's role has (role IDs are one
// sequence across tenants), so every other workspace's Super Admin was an
// ordinary role: missing permissions added since it was created, and
// editable and deletable. At most one role per workspace is super.
func V6_41_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		ALTER TABLE roles ADD COLUMN IF NOT EXISTS super BOOLEAN NOT NULL DEFAULT false;

		-- Each workspace's Super Admin: role 1 (tenant 1's, whatever it's
		-- called now), else the top-level user role named Super Admin.
		UPDATE roles SET super = true
		    WHERE NOT super AND type = 'user' AND parent_id IS NULL
		    AND (id = 1 OR (name = 'Super Admin' AND NOT EXISTS (
		        SELECT 1 FROM roles r WHERE r.tenant_id = roles.tenant_id AND r.super)));

		DROP INDEX IF EXISTS idx_roles_super;
		CREATE UNIQUE INDEX idx_roles_super ON roles(tenant_id) WHERE super;
	`)
	return err
}
