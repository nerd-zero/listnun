package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_10_0 fixes a live bug in the RLS policies introduced in v6.5.0: on
// Postgres, once a session's SET LOCAL on a custom GUC ends, the setting
// does not revert to "unset"/NULL - it reverts to that GUC's session-level
// value, and the very first time any backend touches a never-before-set
// custom parameter name (via SET LOCAL, current_setting, anything),
// Postgres materializes it as a placeholder defaulting to ” (empty
// string), not NULL. So after the first internal/core.WithTenant call on
// a given pooled connection, current_setting('app.current_tenant', true)
// returns ” - not NULL - for every later query on that same connection
// that runs outside WithTenant (e.g. queries/campaigns.sql's
// next-campaigns, or Core.GetUserUnscoped used by every session/token
// lookup). The old policy's `::INTEGER` cast then fails on ” with
// "invalid input syntax for type integer", breaking campaign scanning and
// login session validation for the lifetime of that connection - found
// live via a real login failure after switching the dev DB role from a
// superuser (which bypasses RLS entirely, masking this) to a real
// non-superuser role.
//
// Fix: wrap the setting in NULLIF(..., ”) before the cast, so an empty
// string is treated identically to NULL/unset (permissive) and never
// reaches ::INTEGER.
func V6_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		ALTER POLICY tenant_isolation ON subscribers      USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON lists            USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON templates        USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON campaigns        USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON media            USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON links            USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON bounces          USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON roles            USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON users            USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON subscriber_lists USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON campaign_lists   USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON campaign_views   USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON campaign_media   USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON link_clicks      USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER POLICY tenant_isolation ON settings         USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`)
	return err
}
