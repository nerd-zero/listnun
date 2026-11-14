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
-- Core.GetSettings call failed with "unexpected end of JSON input" until
-- this ran. Requires the BYPASSRLS operator connection since it reads
-- tenant 1's rows.
--
-- General preferences (sending rate, privacy, opt-in, cleanup, ...) are
-- copied from tenant 1 as the new tenant's starting point. Anything that
-- holds credentials or is tenant 1's own identity or code - SMTP and other
-- messengers, bounce mailboxes and webhook keys, OIDC and captcha secrets,
-- Scrub, custom CSS/JS, From address, notify addresses, trusted URLs -
-- starts from schema.sql's install defaults
-- (keep these in step with it): each tenant sets up its own.
--
-- upload.* is still copied: the Operator API has no way to set a
-- tenant's media storage, so new tenants share tenant 1's (platform) store.
--
-- app.root_url is set by operator-set-tenant-root-url right after this,
-- and replies by the Operator API, so neither is seeded here. branding is
-- tenant 1's own logo, so a new tenant starts without one.
--
-- $1 is cast explicitly: inside a UNION, Postgres resolves an untyped
-- parameter to TEXT before the INSERT's target type is applied, which
-- fails preparation (and so startup) with "tenant_id is of type integer".
WITH defaults(key, value) AS (
    VALUES
        ('app.site_name', '"Mailing list"'::JSONB),
        ('app.favicon_url', '""'::JSONB),
        ('app.from_email', '"listnun <noreply@listnun.yoursite.com>"'::JSONB),
        ('app.logo_url', '""'::JSONB),
        ('app.notify_emails', '[]'::JSONB),
        ('security.captcha', '{"altcha": {"enabled": false, "complexity": 300000}, "hcaptcha": {"enabled": false, "key": "", "secret": ""}, "turnstile": {"enabled": false, "key": "", "secret": ""}, "login": false}'::JSONB),
        ('security.oidc', '{"enabled": false, "provider_url": "", "provider_name": "", "client_id": "", "client_secret": "", "auto_create_users": false, "default_user_role_id": null, "default_list_role_id": null}'::JSONB),
        ('security.trusted_urls', '[]'::JSONB),
        ('smtp', '[{"enabled":true, "host":"smtp.yoursite.com","port":25,"auth_protocol":"cram","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"msg_retry_delay":"10ms","tls_type":"STARTTLS","tls_skip_verify":false,"email_headers":[], "from_addresses":[]},
          {"enabled":false, "host":"smtp.gmail.com","port":465,"auth_protocol":"login","username":"username@gmail.com","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"msg_retry_delay":"10ms","tls_type":"TLS","tls_skip_verify":false,"email_headers":[], "from_addresses":[]}]'::JSONB),
        ('messengers', '[]'::JSONB),
        ('bounce.enabled', 'false'::JSONB),
        ('bounce.webhooks_enabled', 'false'::JSONB),
        ('bounce.actions', '{"soft": {"count": 2, "action": "none"}, "hard": {"count": 1, "action": "blocklist"}, "complaint" : {"count": 1, "action": "blocklist"}}'::JSONB),
        ('bounce.ses_enabled', 'false'::JSONB),
        ('bounce.azure', '{"enabled": false, "shared_secret": "", "shared_secret_header": ""}'::JSONB),
        ('bounce.sendgrid_enabled', 'false'::JSONB),
        ('bounce.sendgrid_key', '""'::JSONB),
        ('bounce.postmark', '{"enabled": false, "username": "", "password": ""}'::JSONB),
        ('bounce.forwardemail', '{"enabled": false, "key": ""}'::JSONB),
        ('bounce.lettermint', '{"enabled": false, "key": ""}'::JSONB),
        ('bounce.mailboxes', '[{"enabled":false, "type": "pop", "host":"pop.yoursite.com","port":995,"auth_protocol":"userpass","username":"username","password":"password","return_path": "bounce@listnun.yoursite.com","scan_interval":"15m","tls_enabled":true,"tls_skip_verify":false}]'::JSONB),
        ('appearance.admin.custom_css', '""'::JSONB),
        ('appearance.admin.custom_js', '""'::JSONB),
        ('appearance.public.custom_css', '""'::JSONB),
        ('appearance.public.custom_js', '""'::JSONB),
        ('scrub', '{"enabled": false, "url": "", "api_key": "", "integration_id": "", "managed_by_platform": false}'::JSONB),
        ('branding', '{"logo": null, "logo_dark": null, "width": 140, "align": "left", "link_url": ""}'::JSONB)
)
INSERT INTO settings (tenant_id, key, value)
SELECT $1::INT, key, value FROM settings
    WHERE tenant_id = 1 AND key NOT IN ('app.root_url', 'replies') AND key NOT IN (SELECT key FROM defaults)
UNION ALL
SELECT $1::INT, key, value FROM defaults
ON CONFLICT (tenant_id, key) DO NOTHING;

-- name: operator-set-tenant-root-url
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'app.root_url', to_jsonb($2::TEXT))
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: operator-set-tenant-from-email
-- Sets the tenant's default From address (app.from_email) - see
-- cmd/operator.go's SetTenantFromEmail.
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'app.from_email', to_jsonb($2::TEXT))
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- name: operator-set-tenant-custom-domain
-- $2 is nullable - clears the custom domain (reverting to <slug>.root_domain
-- resolution only) when unset. Paired transactionally with
-- operator-set-tenant-root-url by the Go caller (UpdateOperatorTenantCustomDomain)
-- so tenants.custom_domain (what internal/tenant.Middleware resolves by)
-- and app.root_url (what listnun generates links with) never drift apart.
UPDATE tenants SET custom_domain = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: operator-set-tenant-smtp
-- Replaces the tenant's smtp setting (copied as tenant 1's placeholder
-- example entries by operator-seed-tenant-settings) with a real entry
-- supplied by the caller - see cmd/operator.go's SetTenantSMTP.
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'smtp', $2::JSONB)
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: operator-set-tenant-settings
-- Sets the given settings keys ($2, a JSON object of key: value) for
-- tenant $1, leaving its other settings as they are.
INSERT INTO settings (tenant_id, key, value)
SELECT $1, key, value FROM JSONB_EACH($2::JSONB)
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- name: operator-set-tenant-replies
-- Replaces the tenant's reply tracking setup (models.TenantReplies) - see
-- cmd/operator.go's SetTenantReplies. Its own "replies" key, which the
-- tenant's Settings UI never reads or writes.
INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'replies', $2::JSONB)
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- name: operator-delete-tenant-replies
DELETE FROM settings WHERE tenant_id = $1 AND key = 'replies';

-- name: operator-set-tenant-bounce-webhooks
-- Sets the tenant's bounce.webhooks_enabled ($2) and bounce.postmark ($3)
-- settings - see cmd/operator.go's SetTenantBounceWebhooks. Enabling also
-- turns on bounce.enabled, which BounceWebhook requires too; disabling
-- leaves bounce.enabled as the tenant had it.
INSERT INTO settings (tenant_id, key, value)
SELECT $1, k, v FROM (VALUES
    ('bounce.webhooks_enabled', $2::JSONB),
    ('bounce.postmark', $3::JSONB),
    ('bounce.enabled', 'true'::JSONB)
) AS t(k, v)
WHERE k != 'bounce.enabled' OR $2::JSONB = 'true'::JSONB
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
-- root_url is the tenant's own settings.app.root_url -- the fork's real
-- source of truth for where it's reachable, distinct from (and can
-- diverge from) whatever a caller's own root-domain config assumes.
-- Found live: legacy tenants provisioned under an earlier root domain
-- kept settings.app.root_url pointing at that domain even after the
-- provisioner's own config moved on, so reconstructing a tenant's URL
-- from a single global root-domain value silently produces a
-- dead/wrong host for them (see doProvisionScrubIntegration's own doc
-- comment). Exposing this column is the fix -- always ask the fork
-- rather than assume.
SELECT t.*,
    (SELECT COUNT(*) FROM users WHERE tenant_id = t.id) AS user_count,
    (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id) AS subscriber_count,
    (SELECT value #>> '{}' FROM settings WHERE tenant_id = t.id AND key = 'app.root_url') AS root_url
FROM tenants t WHERE t.id = $1;

-- name: operator-get-tenants
SELECT t.*,
    (SELECT COUNT(*) FROM users WHERE tenant_id = t.id) AS user_count,
    (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id) AS subscriber_count,
    (SELECT value #>> '{}' FROM settings WHERE tenant_id = t.id AND key = 'app.root_url') AS root_url
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

-- name: operator-get-tenant-usage
-- A tenant's metered usage over [$2, $3), summed from usage_counters'
-- hourly buckets by bucket start -- back-to-back windows therefore count
-- each hour exactly once. Backs listnun's PAYG billing and console usage.
SELECT
    COALESCE(SUM(count) FILTER (WHERE kind = 'email_sent'), 0) AS emails_sent,
    COALESCE(SUM(count) FILTER (WHERE kind = 'validation'), 0) AS validations
FROM usage_counters
WHERE tenant_id = $1 AND bucket >= $2 AND bucket < $3;

-- name: operator-get-tenant-usage-daily
-- A tenant's usage over [$2, $3) per UTC day, for usage charts. Days with
-- no usage are omitted; the caller fills gaps.
SELECT
    to_char(bucket AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
    COALESCE(SUM(count) FILTER (WHERE kind = 'email_sent'), 0) AS emails_sent,
    COALESCE(SUM(count) FILTER (WHERE kind = 'validation'), 0) AS validations
FROM usage_counters
WHERE tenant_id = $1 AND bucket >= $2 AND bucket < $3
GROUP BY 1 ORDER BY 1;

-- name: operator-get-platform-usage-by-tenant
-- Every tenant's usage over [$1, $2) in one query -- backs listnun's admin
-- overview without a call per tenant. Includes deleted tenants, whose
-- usage_counters rows deliberately outlive them.
SELECT
    tenant_id,
    COALESCE(SUM(count) FILTER (WHERE kind = 'email_sent'), 0) AS emails_sent,
    COALESCE(SUM(count) FILTER (WHERE kind = 'validation'), 0) AS validations
FROM usage_counters
WHERE bucket >= $1 AND bucket < $2
GROUP BY tenant_id ORDER BY tenant_id;

-- name: operator-get-platform-usage-daily
-- All tenants' usage over [$1, $2) per UTC day.
SELECT
    to_char(bucket AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
    COALESCE(SUM(count) FILTER (WHERE kind = 'email_sent'), 0) AS emails_sent,
    COALESCE(SUM(count) FILTER (WHERE kind = 'validation'), 0) AS validations
FROM usage_counters
WHERE bucket >= $1 AND bucket < $2
GROUP BY 1 ORDER BY 1;
