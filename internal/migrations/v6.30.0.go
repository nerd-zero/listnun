package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_30_0 adds two settings from the handover: security.require_2fa (off),
// and upload.max_file_size (5000 KB) for workspaces created before it was
// seeded. Existing values are kept, so running this twice is harmless.
func V6_30_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (tenant_id, key, value)
		SELECT id, 'security.require_2fa', 'false'::JSONB FROM tenants
		ON CONFLICT (tenant_id, key) DO NOTHING;

		INSERT INTO settings (tenant_id, key, value)
		SELECT id, 'upload.max_file_size', '5000'::JSONB FROM tenants
		ON CONFLICT (tenant_id, key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
