package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_27_0 adds Cloudflare Turnstile as a captcha provider and a switch to
// ask for the captcha on the admin sign-in page, both off by default.
// Every tenant already has a security.captcha row; existing values are
// left alone, so running this twice is harmless.
func V6_27_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		UPDATE settings SET value = JSONB_SET(value, '{turnstile}', '{"enabled": false, "key": "", "secret": ""}'::JSONB)
			WHERE key = 'security.captcha' AND NOT (value ? 'turnstile');

		UPDATE settings SET value = JSONB_SET(value, '{login}', 'false'::JSONB)
			WHERE key = 'security.captcha' AND NOT (value ? 'login');
	`); err != nil {
		return err
	}

	return nil
}
