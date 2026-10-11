package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_15_0 adds tenants.custom_domain -- lets a tenant be reached at a
// domain the org owns (mail.acme.com) instead of only <slug>.root_domain.
// Nullable and unset by default; the listnun-side operator client sets it
// only once the customer's domain ownership has been verified externally
// (e.g. a Cloudflare Custom Hostname's DCV) - listnun itself does no
// verification of its own, it just resolves whatever's stored here.
// internal/tenant/resolve.go's Middleware checks it before falling back to
// subdomain stripping. See listnun's docs/custom-domains.md.
//
// v6.14.0 is already claimed by a different, unmerged migration (operator
// role grants, fix/operator-grants-full-dml) - this is v6.15.0 to avoid
// colliding with it regardless of which merges first.
func V6_15_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		ALTER TABLE tenants ADD COLUMN IF NOT EXISTS custom_domain TEXT UNIQUE;
	`)
	return err
}
