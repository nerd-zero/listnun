package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// platformSMTPHost is the host of the SMTP servers the platform has set up
// through the Operator API so far. Those were stored without a marker, so
// V6_40_0 recognises them by it. From now on the Operator API marks the
// servers it sets up itself.
const platformSMTPHost = "smtp.postmarkapp.com"

// V6_40_0 adds managed_by_platform to every SMTP server: true for those
// the platform set up, which their workspaces then can't see or change
// (models.Settings.SMTPManaged), false for the rest. Servers that already
// have it are left as they are.
func V6_40_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		UPDATE settings SET value = (
		    SELECT COALESCE(JSONB_AGG(
		        CASE WHEN e ? 'managed_by_platform' THEN e
		        ELSE e || JSONB_BUILD_OBJECT('managed_by_platform', LOWER(TRIM(COALESCE(e->>'host', ''))) = $1)
		        END ORDER BY n), '[]'::JSONB)
		    FROM JSONB_ARRAY_ELEMENTS(settings.value) WITH ORDINALITY AS t(e, n)
		)
		WHERE key = 'smtp' AND JSONB_TYPEOF(value) = 'array'
		    AND EXISTS (SELECT 1 FROM JSONB_ARRAY_ELEMENTS(settings.value) e WHERE NOT e ? 'managed_by_platform');
	`, platformSMTPHost)
	return err
}
