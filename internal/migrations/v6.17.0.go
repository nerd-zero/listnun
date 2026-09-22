package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_17_0 fixes the tenant-level Scrub settings to match Scrub's real
// current API: its integrations table moved from a SERIAL/int primary key
// to a native UUID, so Settings.Scrub.IntegrationID is now a string, not
// an int -- coerce every tenant's already-stored value accordingly (0,
// the unconfigured default, becomes "" so the empty-string "not
// configured" guards at every call site still work, not the misleading
// "0" a naive stringify would produce). Also backfills
// Settings.Scrub.ManagedByPlatform (false unless a listnun-console
// operator push has since set it), the new field gating whether a tenant
// admin's own settings save is allowed to touch this row -- see
// internal/core.UpdateSettings/UpdateSettingsByKey.
//
// Every tenant already has a key='scrub' settings row by the time this
// runs (seeded at tenant-creation via operator-seed-tenant-settings, and
// backfilled for pre-existing tenants by v6.3.0), so this is a plain
// UPDATE -- not v6.3.0's INSERT ... ON CONFLICT (key) shape, which no
// longer works: settings.key's standalone unique constraint was dropped
// in v6.6.0 in favor of PRIMARY KEY (tenant_id, key).
func V6_17_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		UPDATE settings SET value = jsonb_set(
			jsonb_set(value, '{managed_by_platform}', COALESCE(value->'managed_by_platform', 'false'::jsonb)),
			'{integration_id}',
			CASE
				WHEN jsonb_typeof(value->'integration_id') = 'number' THEN
					CASE WHEN (value->>'integration_id')::numeric = 0 THEN '""'::jsonb
						 ELSE to_jsonb(value->>'integration_id') END
				ELSE COALESCE(value->'integration_id', '""'::jsonb)
			END
		)
		WHERE key = 'scrub';
	`); err != nil {
		return err
	}

	// Belt-and-suspenders only -- every tenant should already have this
	// row (see doc comment above). Real constraint is (tenant_id, key),
	// not v6.3.0's now-invalid ON CONFLICT (key).
	if _, err := db.Exec(`
		INSERT INTO settings (tenant_id, key, value)
		SELECT id, 'scrub', '{"enabled":false,"url":"","api_key":"","integration_id":"","managed_by_platform":false}'::jsonb
		FROM tenants
		WHERE NOT EXISTS (SELECT 1 FROM settings WHERE settings.tenant_id = tenants.id AND settings.key = 'scrub')
		ON CONFLICT (tenant_id, key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
