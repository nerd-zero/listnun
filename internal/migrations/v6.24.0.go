package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_24_0 adds the per-tenant `branding` setting: the logo transactional
// templates show through the brand-logo block, its dark-mode variant,
// display width, alignment and link. Every tenant starts without a logo.
func V6_24_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (tenant_id, key, value)
		SELECT id, 'branding', '{"logo": null, "logo_dark": null, "width": 140, "align": "left", "link_url": ""}'::jsonb
		FROM tenants
		WHERE NOT EXISTS (SELECT 1 FROM settings WHERE settings.tenant_id = tenants.id AND settings.key = 'branding')
		ON CONFLICT (tenant_id, key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
