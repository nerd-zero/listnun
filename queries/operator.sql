-- operator
-- These queries run exclusively against the operator DB connection
-- (cmd/operator.go's initOperatorDB), a separate pool using a Postgres
-- role with BYPASSRLS - required because aggregating counts across every
-- tenant at once is impossible under RLS using the normal app role (which
-- must never have BYPASSRLS; see docs/design/multi-tenancy.md's Operator
-- API section). Never load these against the main tenant-app pool.

-- name: operator-create-organization
INSERT INTO organizations (name) VALUES ($1) RETURNING *;

-- name: operator-get-organization
SELECT o.*,
    (SELECT COUNT(*) FROM tenants WHERE organization_id = o.id) AS tenant_count
FROM organizations o WHERE o.id = $1;

-- name: operator-get-organizations
SELECT o.*,
    (SELECT COUNT(*) FROM tenants WHERE organization_id = o.id) AS tenant_count
FROM organizations o ORDER BY o.id;

-- name: operator-get-organization-tenants
SELECT t.*,
    (SELECT COUNT(*) FROM users WHERE tenant_id = t.id) AS user_count,
    (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id) AS subscriber_count
FROM tenants t WHERE t.organization_id = $1 ORDER BY t.id;

-- name: operator-delete-organization
--- organization_id is ON DELETE SET NULL (see docs/design/multi-tenancy.md's
--- Organizations section) - tenants under this organization are kept,
--- just detached from it.
DELETE FROM organizations WHERE id = $1;

-- name: operator-create-tenant
-- $3 (organization_id) is nullable - a tenant doesn't have to belong to
-- an organization.
INSERT INTO tenants (slug, name, status, organization_id) VALUES ($1, $2, 'active', $3) RETURNING *;

-- name: operator-seed-tenant-settings
-- New tenants otherwise get zero settings rows (nothing seeds them -
-- schema.sql's big INSERT INTO settings block relies on tenant_id's
-- DEFAULT 1, so it only ever seeds tenant 1). Every other tenant's
-- Core.GetSettings call - Settings page, per-tenant SMTP/media/OIDC
-- resolution, the bulk importer's domain blocklist - failed with
-- "unexpected end of JSON input" (JSON_OBJECT_AGG over zero rows is
-- SQL NULL) until this ran. Copies tenant 1's current settings as the
-- new tenant's starting defaults - requires the BYPASSRLS operator
-- connection since it reads across tenants.
--
-- app.root_url is deliberately excluded: it's tenant-1-specific, not a
-- sensible default for anyone else, and operator-set-tenant-root-url
-- sets the new tenant's own correct value right after this runs.
INSERT INTO settings (tenant_id, key, value)
SELECT $1, key, value FROM settings WHERE tenant_id = 1 AND key != 'app.root_url'
ON CONFLICT (tenant_id, key) DO NOTHING;

-- name: operator-set-tenant-root-url
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'app.root_url', to_jsonb($2::TEXT))
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: operator-set-tenant-custom-domain
-- $2 is nullable - clears the custom domain (reverting to <slug>.root_domain
-- resolution only) when unset. Paired transactionally with
-- operator-set-tenant-root-url by the Go caller (UpdateOperatorTenantCustomDomain)
-- so tenants.custom_domain (what internal/tenant.Middleware resolves by)
-- and app.root_url (what listmonk generates links with) never drift apart.
UPDATE tenants SET custom_domain = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: operator-set-tenant-smtp
-- Replaces the tenant's smtp setting (copied as tenant 1's placeholder
-- example entries by operator-seed-tenant-settings) with a real entry
-- supplied by the caller - see cmd/operator.go's SetTenantSMTP.
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'smtp', $2::JSONB)
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: operator-set-tenant-scrub
-- Replaces the tenant's scrub setting with a platform-pushed entry - see
-- cmd/operator.go's SetTenantScrub. Always writes managed_by_platform:
-- true (the caller sets it, not the tenant's prior value) so a
-- platform-initiated disable still locks the tenant's own toggle,
-- distinct from a tenant that was never platform-managed at all.
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'scrub', $2::JSONB)
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: operator-get-tenant-scrub-usage
-- Ground truth for the console's per-instance usage display -- Scrub
-- itself has no per-integration usage endpoint (only account-scoped
-- /usage/*, and raw paginated /history with no aggregate), so this
-- reads the same local subscribers.scrub_status data the tenant's own
-- dashboard widget already uses (see internal/migrations/v6.16.0.go).
SELECT
    COUNT(*) FILTER (WHERE scrub_status = 'deliverable') AS deliverable,
    COUNT(*) FILTER (WHERE scrub_status = 'undeliverable') AS undeliverable,
    COUNT(*) FILTER (WHERE scrub_status = 'invalid_syntax') AS invalid_syntax,
    COUNT(*) FILTER (WHERE scrub_status = 'risky') AS risky,
    COUNT(*) FILTER (WHERE scrub_status = 'unchecked_error') AS unchecked_error,
    COUNT(*) FILTER (WHERE scrub_status IS NULL) AS unchecked,
    COUNT(*) AS total_subscribers
FROM subscribers WHERE tenant_id = $1;

-- name: operator-get-tenant
SELECT t.*,
    (SELECT COUNT(*) FROM users WHERE tenant_id = t.id) AS user_count,
    (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id) AS subscriber_count
FROM tenants t WHERE t.id = $1;

-- name: operator-get-tenants
SELECT t.*,
    (SELECT COUNT(*) FROM users WHERE tenant_id = t.id) AS user_count,
    (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id) AS subscriber_count
FROM tenants t ORDER BY t.id;

-- name: operator-update-tenant-status
UPDATE tenants SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: operator-delete-tenant
--- Every tenant-scoped table references tenants(id) ON DELETE CASCADE, so
--- this single DELETE removes all of the tenant's subscribers, campaigns,
--- users, settings, etc. along with it. Irreversible - the caller
--- (cmd/operator.go's DeleteOperatorTenant) refuses to run this against
--- tenant 1, the default tenant every install seeds data against.
DELETE FROM tenants WHERE id = $1;
