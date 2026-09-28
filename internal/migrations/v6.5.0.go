package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_5_0 enables Row Level Security on every tenant-scoped table added in
// v6.4.0, as belt-and-suspenders tenant isolation enforced inside Postgres
// itself rather than solely relying on every hand-written query in
// queries/*.sql remembering a `WHERE tenant_id = ...` clause.
//
// The policy is deliberately permissive while `app.current_tenant` is unset:
// `tenant_id = current_setting('app.current_tenant', true)::INTEGER OR
// current_setting('app.current_tenant', true) IS NULL`. The application does
// not set that session variable yet (that lands in the auth/request-flow
// phase) - a strict "unset context sees nothing" policy would make every
// query return zero rows the moment this migration runs against any
// correctly-permissioned deployment (non-superuser, non-owner app role).
// This fallback keeps the app fully functional in the interim; tighten it
// (drop the `OR ... IS NULL` branch) once the app reliably sets tenant
// context on every request.
//
// FORCE ROW LEVEL SECURITY (in addition to ENABLE) matters because most
// self-hosted listnun installs, including this dev database, use a single
// Postgres role for both schema ownership and the app connection - and
// table owners are exempt from RLS by default, silently making plain
// ENABLE ROW LEVEL SECURITY a no-op for that common setup. Superusers are
// always exempt regardless of FORCE, so this remains inert in the current
// dev sandbox (superuser role) but takes effect for any owner role that
// isn't also a superuser.
func V6_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		ALTER TABLE subscribers      ENABLE ROW LEVEL SECURITY; ALTER TABLE subscribers      FORCE ROW LEVEL SECURITY;
		ALTER TABLE lists            ENABLE ROW LEVEL SECURITY; ALTER TABLE lists            FORCE ROW LEVEL SECURITY;
		ALTER TABLE templates        ENABLE ROW LEVEL SECURITY; ALTER TABLE templates        FORCE ROW LEVEL SECURITY;
		ALTER TABLE campaigns        ENABLE ROW LEVEL SECURITY; ALTER TABLE campaigns        FORCE ROW LEVEL SECURITY;
		ALTER TABLE media            ENABLE ROW LEVEL SECURITY; ALTER TABLE media            FORCE ROW LEVEL SECURITY;
		ALTER TABLE links            ENABLE ROW LEVEL SECURITY; ALTER TABLE links            FORCE ROW LEVEL SECURITY;
		ALTER TABLE bounces          ENABLE ROW LEVEL SECURITY; ALTER TABLE bounces          FORCE ROW LEVEL SECURITY;
		ALTER TABLE roles            ENABLE ROW LEVEL SECURITY; ALTER TABLE roles            FORCE ROW LEVEL SECURITY;
		ALTER TABLE users            ENABLE ROW LEVEL SECURITY; ALTER TABLE users            FORCE ROW LEVEL SECURITY;
		ALTER TABLE subscriber_lists ENABLE ROW LEVEL SECURITY; ALTER TABLE subscriber_lists FORCE ROW LEVEL SECURITY;
		ALTER TABLE campaign_lists   ENABLE ROW LEVEL SECURITY; ALTER TABLE campaign_lists   FORCE ROW LEVEL SECURITY;
		ALTER TABLE campaign_views   ENABLE ROW LEVEL SECURITY; ALTER TABLE campaign_views   FORCE ROW LEVEL SECURITY;
		ALTER TABLE campaign_media   ENABLE ROW LEVEL SECURITY; ALTER TABLE campaign_media   FORCE ROW LEVEL SECURITY;
		ALTER TABLE link_clicks      ENABLE ROW LEVEL SECURITY; ALTER TABLE link_clicks      FORCE ROW LEVEL SECURITY;
		ALTER TABLE settings         ENABLE ROW LEVEL SECURITY; ALTER TABLE settings         FORCE ROW LEVEL SECURITY;

		DROP POLICY IF EXISTS tenant_isolation ON subscribers;
		CREATE POLICY tenant_isolation ON subscribers USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON lists;
		CREATE POLICY tenant_isolation ON lists USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON templates;
		CREATE POLICY tenant_isolation ON templates USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON campaigns;
		CREATE POLICY tenant_isolation ON campaigns USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON media;
		CREATE POLICY tenant_isolation ON media USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON links;
		CREATE POLICY tenant_isolation ON links USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON bounces;
		CREATE POLICY tenant_isolation ON bounces USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON roles;
		CREATE POLICY tenant_isolation ON roles USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON users;
		CREATE POLICY tenant_isolation ON users USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON subscriber_lists;
		CREATE POLICY tenant_isolation ON subscriber_lists USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON campaign_lists;
		CREATE POLICY tenant_isolation ON campaign_lists USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON campaign_views;
		CREATE POLICY tenant_isolation ON campaign_views USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON campaign_media;
		CREATE POLICY tenant_isolation ON campaign_media USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON link_clicks;
		CREATE POLICY tenant_isolation ON link_clicks USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
		DROP POLICY IF EXISTS tenant_isolation ON settings;
		CREATE POLICY tenant_isolation ON settings USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER OR current_setting('app.current_tenant', true) IS NULL);
	`)
	return err
}
