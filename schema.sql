DROP TYPE IF EXISTS list_type CASCADE; CREATE TYPE list_type AS ENUM ('public', 'private', 'temporary');
DROP TYPE IF EXISTS list_optin CASCADE; CREATE TYPE list_optin AS ENUM ('single', 'double');
DROP TYPE IF EXISTS list_status CASCADE; CREATE TYPE list_status AS ENUM ('active', 'archived');
DROP TYPE IF EXISTS subscriber_status CASCADE; CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blocklisted');
DROP TYPE IF EXISTS subscription_status CASCADE; CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');
DROP TYPE IF EXISTS campaign_status CASCADE; CREATE TYPE campaign_status AS ENUM ('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished');
DROP TYPE IF EXISTS campaign_type CASCADE; CREATE TYPE campaign_type AS ENUM ('regular', 'optin');
DROP TYPE IF EXISTS content_type CASCADE; CREATE TYPE content_type AS ENUM ('richtext', 'html', 'plain', 'markdown', 'visual');
DROP TYPE IF EXISTS bounce_type CASCADE; CREATE TYPE bounce_type AS ENUM ('soft', 'hard', 'complaint');
DROP TYPE IF EXISTS template_type CASCADE; CREATE TYPE template_type AS ENUM ('campaign', 'campaign_visual', 'tx');
DROP TYPE IF EXISTS user_type CASCADE; CREATE TYPE user_type AS ENUM ('user', 'api');
DROP TYPE IF EXISTS user_status CASCADE; CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
DROP TYPE IF EXISTS role_type CASCADE; CREATE TYPE role_type AS ENUM ('user', 'list');
DROP TYPE IF EXISTS twofa_type CASCADE; CREATE TYPE twofa_type AS ENUM ('none', 'totp');
DROP TYPE IF EXISTS tenant_status CASCADE; CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'disabled');

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- organizations
-- A purely cross-tenant grouping construct, managed only via the Operator
-- API (see docs/design/multi-tenancy.md) - never RLS-scoped, never
-- resolved per-request the way tenants are. One organization can own
-- multiple tenants ("listmonks") for different purposes (e.g. separate
-- brands/departments), or none at all - tenants.organization_id is
-- nullable so existing/standalone tenants aren't forced into one.
DROP TABLE IF EXISTS organizations CASCADE;
CREATE TABLE organizations (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- tenants
DROP TABLE IF EXISTS tenants CASCADE;
CREATE TABLE tenants (
    id              SERIAL PRIMARY KEY,
    organization_id INTEGER NULL REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE,
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    status          tenant_status NOT NULL DEFAULT 'active',
    custom_domain   TEXT UNIQUE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_tenants_organization; CREATE INDEX idx_tenants_organization ON tenants(organization_id);
INSERT INTO tenants (id, slug, name, status) VALUES (1, 'default', 'Default', 'active');
SELECT setval('tenants_id_seq', 1);

-- subscribers
DROP TABLE IF EXISTS subscribers CASCADE;
CREATE TABLE subscribers (
    id              SERIAL PRIMARY KEY,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    uuid uuid       NOT NULL UNIQUE,
    email           TEXT NOT NULL,
    name            TEXT NOT NULL,
    attribs         JSONB NOT NULL DEFAULT '{}',
    status          subscriber_status NOT NULL DEFAULT 'enabled',

    -- Set by validate-on-add/import Scrub email validation (see v6.16.0
    -- migration doc comment). NULL = never checked (legacy row or Scrub
    -- not configured for this tenant); unchecked_error is a distinct
    -- sentinel from NULL so a Scrub outage is observable.
    scrub_status      TEXT NULL,
    scrub_checked_at  TIMESTAMP WITH TIME ZONE NULL,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT subscribers_email_key UNIQUE (tenant_id, email)
);
DROP INDEX IF EXISTS idx_subs_email; CREATE UNIQUE INDEX idx_subs_email ON subscribers(tenant_id, LOWER(email));
DROP INDEX IF EXISTS idx_subs_status; CREATE INDEX idx_subs_status ON subscribers(status);
DROP INDEX IF EXISTS idx_subs_id_status; CREATE INDEX idx_subs_id_status ON subscribers(id, status);
DROP INDEX IF EXISTS idx_subs_created_at; CREATE INDEX idx_subs_created_at ON subscribers(created_at);
DROP INDEX IF EXISTS idx_subs_updated_at; CREATE INDEX idx_subs_updated_at ON subscribers(updated_at);
DROP INDEX IF EXISTS idx_subscribers_tenant; CREATE INDEX idx_subscribers_tenant ON subscribers(tenant_id);
DROP INDEX IF EXISTS idx_subscribers_scrub_status; CREATE INDEX idx_subscribers_scrub_status ON subscribers(tenant_id, scrub_status);

-- lists
DROP TABLE IF EXISTS lists CASCADE;
CREATE TABLE lists (
    id              SERIAL PRIMARY KEY,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    uuid            uuid NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    type            list_type NOT NULL,
    optin           list_optin NOT NULL DEFAULT 'single',
    status          list_status NOT NULL DEFAULT 'active',
    tags            VARCHAR(100)[],
    description     TEXT NOT NULL DEFAULT '',

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_lists_type; CREATE INDEX idx_lists_type ON lists(type);
DROP INDEX IF EXISTS idx_lists_optin; CREATE INDEX idx_lists_optin ON lists(optin);
DROP INDEX IF EXISTS idx_lists_status; CREATE INDEX idx_lists_status ON lists(status);
DROP INDEX IF EXISTS idx_lists_name; CREATE INDEX idx_lists_name ON lists(name);
DROP INDEX IF EXISTS idx_lists_created_at; CREATE INDEX idx_lists_created_at ON lists(created_at);
DROP INDEX IF EXISTS idx_lists_updated_at; CREATE INDEX idx_lists_updated_at ON lists(updated_at);
DROP INDEX IF EXISTS idx_lists_tenant; CREATE INDEX idx_lists_tenant ON lists(tenant_id);


DROP TABLE IF EXISTS subscriber_lists CASCADE;
CREATE TABLE subscriber_lists (
    subscriber_id      INTEGER REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
    list_id            INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tenant_id          INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    meta               JSONB NOT NULL DEFAULT '{}',
    status             subscription_status NOT NULL DEFAULT 'unconfirmed',

    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY(subscriber_id, list_id)
);
DROP INDEX IF EXISTS idx_sub_lists_sub_id; CREATE INDEX idx_sub_lists_sub_id ON subscriber_lists(subscriber_id);
DROP INDEX IF EXISTS idx_sub_lists_list_id; CREATE INDEX idx_sub_lists_list_id ON subscriber_lists(list_id);
DROP INDEX IF EXISTS idx_sub_lists_status; CREATE INDEX idx_sub_lists_status ON subscriber_lists(status);
DROP INDEX IF EXISTS idx_subscriber_lists_tenant; CREATE INDEX idx_subscriber_lists_tenant ON subscriber_lists(tenant_id);

-- templates
DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
    id              SERIAL PRIMARY KEY,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    name            TEXT NOT NULL,
    type            template_type NOT NULL DEFAULT 'campaign',
    subject         TEXT NOT NULL,
    body            TEXT NOT NULL,
    body_source     TEXT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT false,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX templates_is_default_idx ON templates (tenant_id, is_default) WHERE is_default = true;
DROP INDEX IF EXISTS idx_templates_tenant; CREATE INDEX idx_templates_tenant ON templates(tenant_id);


-- campaigns
DROP TABLE IF EXISTS campaigns CASCADE;
CREATE TABLE campaigns (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    uuid uuid        NOT NULL UNIQUE,
    name             TEXT NOT NULL,
    subject          TEXT NOT NULL,
    from_email       TEXT NOT NULL,
    body             TEXT NOT NULL,
    body_source      TEXT NULL,
    altbody          TEXT NULL,
    content_type     content_type NOT NULL DEFAULT 'richtext',
    send_at          TIMESTAMP WITH TIME ZONE,
    headers          JSONB NOT NULL DEFAULT '[]',
    attribs          JSONB NOT NULL DEFAULT '{}',
    status           campaign_status NOT NULL DEFAULT 'draft',
    -- Set only when status='paused' was set automatically (not by a user)
    -- -- e.g. a Scrub-flagged risky subscriber landing on a target list,
    -- or internal/manager/pipe.go's pre-existing too-many-errors
    -- auto-pause. Cleared on any manual status change. NULL for a
    -- manually-paused campaign.
    pause_reason     TEXT NULL,
    tags             VARCHAR(100)[],

    -- The subscription statuses of subscribers to which a campaign will be sent.
    -- For opt-in campaigns, this will be 'unsubscribed'.
    type campaign_type DEFAULT 'regular',

    -- The ID of the messenger backend used to send this campaign.
    messenger        TEXT NOT NULL,
    template_id      INTEGER REFERENCES templates(id) ON DELETE SET NULL,

    -- Progress and stats.
    to_send            INT NOT NULL DEFAULT 0,
    sent               INT NOT NULL DEFAULT 0,
    max_subscriber_id  INT NOT NULL DEFAULT 0,
    last_subscriber_id INT NOT NULL DEFAULT 0,

    -- Publishing.
    archive             BOOLEAN NOT NULL DEFAULT false,
    archive_slug        TEXT NULL UNIQUE,
    archive_template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL,
    archive_meta        JSONB NOT NULL DEFAULT '{}',

    started_at       TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_camps_status; CREATE INDEX idx_camps_status ON campaigns(status);
DROP INDEX IF EXISTS idx_camps_name; CREATE INDEX idx_camps_name ON campaigns(name);
DROP INDEX IF EXISTS idx_camps_created_at; CREATE INDEX idx_camps_created_at ON campaigns(created_at);
DROP INDEX IF EXISTS idx_camps_updated_at; CREATE INDEX idx_camps_updated_at ON campaigns(updated_at);
DROP INDEX IF EXISTS idx_campaigns_tenant; CREATE INDEX idx_campaigns_tenant ON campaigns(tenant_id);


DROP TABLE IF EXISTS campaign_lists CASCADE;
CREATE TABLE campaign_lists (
    id           BIGSERIAL PRIMARY KEY,
    campaign_id  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tenant_id    INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Lists may be deleted, so list_id is nullable
    -- and a copy of the original list name is maintained here.
    list_id      INTEGER NULL REFERENCES lists(id) ON DELETE SET NULL ON UPDATE CASCADE,
    list_name    TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ON campaign_lists (campaign_id, list_id);
DROP INDEX IF EXISTS idx_camp_lists_camp_id; CREATE INDEX idx_camp_lists_camp_id ON campaign_lists(campaign_id);
DROP INDEX IF EXISTS idx_camp_lists_list_id; CREATE INDEX idx_camp_lists_list_id ON campaign_lists(list_id);
DROP INDEX IF EXISTS idx_campaign_lists_tenant; CREATE INDEX idx_campaign_lists_tenant ON campaign_lists(tenant_id);

DROP TABLE IF EXISTS campaign_views CASCADE;
CREATE TABLE campaign_views (
    id               BIGSERIAL PRIMARY KEY,
    campaign_id      INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the view counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_views_camp_id; CREATE INDEX idx_views_camp_id ON campaign_views(campaign_id);
DROP INDEX IF EXISTS idx_views_subscriber_id; CREATE INDEX idx_views_subscriber_id ON campaign_views(subscriber_id);
DROP INDEX IF EXISTS idx_views_date; CREATE INDEX idx_views_date ON campaign_views(created_at);
DROP INDEX IF EXISTS idx_campaign_views_tenant; CREATE INDEX idx_campaign_views_tenant ON campaign_views(tenant_id);

-- media
DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    uuid uuid        NOT NULL UNIQUE,
    provider         TEXT NOT NULL DEFAULT '',
    filename         TEXT NOT NULL,
    content_type     TEXT NOT NULL DEFAULT 'application/octet-stream',
    thumb            TEXT NOT NULL,
    meta             JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_media_filename; CREATE INDEX idx_media_filename ON media(provider, filename);
DROP INDEX IF EXISTS idx_media_tenant; CREATE INDEX idx_media_tenant ON media(tenant_id);

-- campaign_media
DROP TABLE IF EXISTS campaign_media CASCADE;
CREATE TABLE campaign_media (
    campaign_id  INTEGER REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tenant_id    INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Media items may be deleted, so media_id is nullable
    -- and a copy of the original name is maintained here.
    media_id     INTEGER NULL REFERENCES media(id) ON DELETE SET NULL ON UPDATE CASCADE,

    filename     TEXT NOT NULL DEFAULT ''
);
DROP INDEX IF EXISTS idx_camp_media_id; CREATE UNIQUE INDEX idx_camp_media_id ON campaign_media (campaign_id, media_id);
DROP INDEX IF EXISTS idx_camp_media_camp_id; CREATE INDEX idx_camp_media_camp_id ON campaign_media(campaign_id);
DROP INDEX IF EXISTS idx_campaign_media_tenant; CREATE INDEX idx_campaign_media_tenant ON campaign_media(tenant_id);


-- links
DROP TABLE IF EXISTS links CASCADE;
CREATE TABLE links (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    uuid uuid        NOT NULL UNIQUE,
    url              TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_links_tenant; CREATE INDEX idx_links_tenant ON links(tenant_id);

DROP TABLE IF EXISTS link_clicks CASCADE;
CREATE TABLE link_clicks (
    id               BIGSERIAL PRIMARY KEY,
    campaign_id      INTEGER NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    link_id          INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the link counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_clicks_camp_id; CREATE INDEX idx_clicks_camp_id ON link_clicks(campaign_id);
DROP INDEX IF EXISTS idx_clicks_link_id; CREATE INDEX idx_clicks_link_id ON link_clicks(link_id);
DROP INDEX IF EXISTS idx_clicks_sub_id; CREATE INDEX idx_clicks_sub_id ON link_clicks(subscriber_id);
DROP INDEX IF EXISTS idx_clicks_date; CREATE INDEX idx_clicks_date ON link_clicks(created_at);
DROP INDEX IF EXISTS idx_link_clicks_tenant; CREATE INDEX idx_link_clicks_tenant ON link_clicks(tenant_id);

-- settings
DROP TABLE IF EXISTS settings CASCADE;
CREATE TABLE settings (
    key             TEXT NOT NULL,
    tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    value           JSONB NOT NULL DEFAULT '{}',
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY (tenant_id, key)
);
DROP INDEX IF EXISTS idx_settings_key; CREATE INDEX idx_settings_key ON settings(key);
DROP INDEX IF EXISTS idx_settings_tenant; CREATE INDEX idx_settings_tenant ON settings(tenant_id);
INSERT INTO settings (key, value) VALUES
    ('app.site_name', '"Mailing list"'),
    ('app.root_url', '"http://localhost:9000"'),
    ('app.favicon_url', '""'),
    ('app.from_email', '"listmonk <noreply@listmonk.yoursite.com>"'),
    ('app.logo_url', '""'),
    ('app.concurrency', '10'),
    ('app.message_rate', '10'),
    ('app.batch_size', '1000'),
    ('app.max_send_errors', '1000'),
    ('app.message_sliding_window', 'false'),
    ('app.message_sliding_window_duration', '"1h"'),
    ('app.message_sliding_window_rate', '10000'),
    ('app.cache_slow_queries', 'false'),
    ('app.cache_slow_queries_interval', '"0 3 * * *"'),
    ('app.enable_public_archive', 'true'),
    ('app.enable_public_subscription_page', 'true'),
    ('app.show_optin_page', 'true'),
    ('app.enable_public_archive_rss_content', 'true'),
    ('app.send_optin_confirmation', 'true'),
    ('app.check_updates', 'true'),
    ('app.notify_emails', '[]'),
    ('app.lang', '"en"'),
    ('privacy.individual_tracking', 'false'),
    ('privacy.disable_tracking', 'false'),
    ('privacy.unsubscribe_header', 'true'),
    ('privacy.allow_blocklist', 'true'),
    ('privacy.allow_export', 'true'),
    ('privacy.allow_wipe', 'true'),
    ('privacy.allow_preferences', 'true'),
    ('privacy.exportable', '["profile", "subscriptions", "campaign_views", "link_clicks"]'),
    ('privacy.domain_blocklist', '[]'),
    ('privacy.domain_allowlist', '[]'),
    ('privacy.record_optin_ip', 'false'),
    ('security.captcha', '{"altcha": {"enabled": false, "complexity": 300000}, "hcaptcha": {"enabled": false, "key": "", "secret": ""}}'),
    ('security.oidc', '{"enabled": false, "provider_url": "", "provider_name": "", "client_id": "", "client_secret": "", "auto_create_users": false, "default_user_role_id": null, "default_list_role_id": null}'),
    ('security.trusted_urls', '[]'),
    ('upload.provider', '"filesystem"'),
    ('upload.max_file_size', '5000'),
    ('upload.extensions', '["jpg","jpeg","png","gif","svg","*"]'),
    ('upload.filesystem.upload_path', '"uploads"'),
    ('upload.filesystem.upload_uri', '"/uploads"'),
    ('upload.s3.url', '"https://ap-south-1.s3.amazonaws.com"'),
    ('upload.s3.public_url', '""'),
    ('upload.s3.aws_access_key_id', '""'),
    ('upload.s3.aws_secret_access_key', '""'),
    ('upload.s3.aws_default_region', '"ap-south-1"'),
    ('upload.s3.bucket', '""'),
    ('upload.s3.bucket_domain', '""'),
    ('upload.s3.bucket_path', '"/"'),
    ('upload.s3.bucket_type', '"public"'),
    ('upload.s3.expiry', '"167h"'),
    ('smtp',
        '[{"enabled":true, "host":"smtp.yoursite.com","port":25,"auth_protocol":"cram","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"msg_retry_delay":"10ms","tls_type":"STARTTLS","tls_skip_verify":false,"email_headers":[], "from_addresses":[]},
          {"enabled":false, "host":"smtp.gmail.com","port":465,"auth_protocol":"login","username":"username@gmail.com","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"msg_retry_delay":"10ms","tls_type":"TLS","tls_skip_verify":false,"email_headers":[], "from_addresses":[]}]'),
    ('messengers', '[]'),
    ('bounce.enabled', 'false'),
    ('bounce.webhooks_enabled', 'false'),
    ('bounce.actions', '{"soft": {"count": 2, "action": "none"}, "hard": {"count": 1, "action": "blocklist"}, "complaint" : {"count": 1, "action": "blocklist"}}'),
    ('bounce.ses_enabled', 'false'),
    ('bounce.azure', '{"enabled": false, "shared_secret": "", "shared_secret_header": ""}'),
    ('bounce.sendgrid_enabled', 'false'),
    ('bounce.sendgrid_key', '""'),
    ('bounce.postmark', '{"enabled": false, "username": "", "password": ""}'),
    ('bounce.forwardemail', '{"enabled": false, "key": ""}'),
    ('bounce.lettermint', '{"enabled": false, "key": ""}'),
    ('bounce.mailboxes',
        '[{"enabled":false, "type": "pop", "host":"pop.yoursite.com","port":995,"auth_protocol":"userpass","username":"username","password":"password","return_path": "bounce@listmonk.yoursite.com","scan_interval":"15m","tls_enabled":true,"tls_skip_verify":false}]'),
    ('appearance.admin.custom_css', '""'),
    ('appearance.admin.custom_js', '""'),
    ('appearance.public.custom_css', '""'),
    ('appearance.public.custom_js', '""'),
    ('maintenance.db', '{"vacuum": false, "vacuum_cron_interval": "0 2 * * *"}'),
    ('scrub', '{"enabled": false, "url": "", "api_key": "", "integration_id": "", "managed_by_platform": false}');

-- bounces
DROP TABLE IF EXISTS bounces CASCADE;
CREATE TABLE bounces (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    subscriber_id    INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
    campaign_id      INTEGER NULL REFERENCES campaigns(id) ON DELETE SET NULL ON UPDATE CASCADE,
    type             bounce_type NOT NULL DEFAULT 'hard',
    source           TEXT NOT NULL DEFAULT '',
    meta             JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_bounces_sub_id; CREATE INDEX idx_bounces_sub_id ON bounces(subscriber_id);
DROP INDEX IF EXISTS idx_bounces_camp_id; CREATE INDEX idx_bounces_camp_id ON bounces(campaign_id);
DROP INDEX IF EXISTS idx_bounces_source; CREATE INDEX idx_bounces_source ON bounces(source);
DROP INDEX IF EXISTS idx_bounces_date; CREATE INDEX idx_bounces_date ON bounces(created_at);
DROP INDEX IF EXISTS idx_bounces_tenant; CREATE INDEX idx_bounces_tenant ON bounces(tenant_id);

-- roles
DROP TABLE IF EXISTS roles CASCADE;
CREATE TABLE roles (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    type             role_type NOT NULL DEFAULT 'user',
    parent_id        INTEGER NULL REFERENCES roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    list_id          INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    permissions      TEXT[] NOT NULL DEFAULT '{}',
    name             TEXT NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_roles ON roles (parent_id, list_id);
CREATE UNIQUE INDEX idx_roles_name ON roles (tenant_id, type, name) WHERE name IS NOT NULL;
DROP INDEX IF EXISTS idx_roles_tenant; CREATE INDEX idx_roles_tenant ON roles(tenant_id);

-- users
DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id               SERIAL PRIMARY KEY,
    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
    username         TEXT NOT NULL,
    password_login   BOOLEAN NOT NULL DEFAULT false,
    password         TEXT NULL,
    email            TEXT NOT NULL,
    name             TEXT NOT NULL,
    avatar           TEXT NULL,
    type             user_type NOT NULL DEFAULT 'user',
    user_role_id     INTEGER NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    list_role_id     INTEGER NULL REFERENCES roles(id) ON DELETE CASCADE,
    status           user_status NOT NULL DEFAULT 'disabled',
    twofa_type       twofa_type NOT NULL DEFAULT 'none',
    twofa_key        TEXT NULL,
    loggedin_at      TIMESTAMP WITH TIME ZONE NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_users_tenant; CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE UNIQUE INDEX users_username_key ON users (tenant_id, username);
CREATE UNIQUE INDEX users_email_key ON users (tenant_id, email);

-- row level security: tenant isolation, enforced inside Postgres itself
-- rather than solely relying on every hand-written query in queries/*.sql
-- remembering a `WHERE tenant_id = ...` clause. FORCE (in addition to
-- ENABLE) matters because most self-hosted listmonk installs use a single
-- Postgres role for both schema ownership and the app connection, and
-- table owners are exempt from RLS by default - plain ENABLE alone would
-- silently be a no-op for that common setup. Superusers are always exempt
-- regardless of FORCE.
--
-- The policy is permissive while `app.current_tenant` is unset (single-
-- tenant/`multi_tenancy_enabled=false` installs never set it) and treats
-- '' the same as unset via NULLIF: Postgres reverts a SET LOCAL custom GUC
-- to '' (not NULL) once any transaction has ever touched it on a given
-- connection, so casting straight to ::INTEGER without NULLIF fails on
-- every later query on that same connection outside internal/core.WithTenant.
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
CREATE POLICY tenant_isolation ON subscribers USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON lists;
CREATE POLICY tenant_isolation ON lists USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON templates;
CREATE POLICY tenant_isolation ON templates USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON campaigns;
CREATE POLICY tenant_isolation ON campaigns USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON media;
CREATE POLICY tenant_isolation ON media USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON links;
CREATE POLICY tenant_isolation ON links USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON bounces;
CREATE POLICY tenant_isolation ON bounces USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON roles;
CREATE POLICY tenant_isolation ON roles USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON users;
CREATE POLICY tenant_isolation ON users USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON subscriber_lists;
CREATE POLICY tenant_isolation ON subscriber_lists USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON campaign_lists;
CREATE POLICY tenant_isolation ON campaign_lists USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON campaign_views;
CREATE POLICY tenant_isolation ON campaign_views USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON campaign_media;
CREATE POLICY tenant_isolation ON campaign_media USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON link_clicks;
CREATE POLICY tenant_isolation ON link_clicks USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
DROP POLICY IF EXISTS tenant_isolation ON settings;
CREATE POLICY tenant_isolation ON settings USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);

-- user sessions
DROP TABLE IF EXISTS sessions CASCADE;
CREATE TABLE sessions (
    id TEXT NOT NULL PRIMARY KEY,
    data JSONB DEFAULT '{}'::jsonb NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now() NOT NULL
);
DROP INDEX IF EXISTS idx_sessions; CREATE INDEX idx_sessions ON sessions (id, created_at);

-- materialized views

-- dashboard stats
DROP MATERIALIZED VIEW IF EXISTS mat_dashboard_counts;
CREATE MATERIALIZED VIEW mat_dashboard_counts AS
    SELECT NOW() AS updated_at, t.id AS tenant_id,
        JSON_BUILD_OBJECT(
            'subscribers', JSON_BUILD_OBJECT(
                'total', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id),
                'blocklisted', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id AND status = 'blocklisted'),
                'orphans', (
                    SELECT COUNT(subscribers.id) FROM subscribers
                    LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
                    WHERE subscribers.tenant_id = t.id AND subscriber_lists.subscriber_id IS NULL
                )
            ),
            'lists', JSON_BUILD_OBJECT(
                'total', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id),
                'private', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND type='private'),
                'public', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND type='public'),
                'optin_single', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND optin='single'),
                'optin_double', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND optin='double')
            ),
            'campaigns', JSON_BUILD_OBJECT(
                'total', (SELECT COUNT(*) FROM campaigns WHERE tenant_id = t.id),
                'by_status', (
                    SELECT COALESCE(JSON_OBJECT_AGG (status, num), '{}'::JSON) FROM
                    (SELECT status, COUNT(*) AS num FROM campaigns WHERE tenant_id = t.id GROUP BY status) r
                )
            ),
            'messages', (SELECT COALESCE(SUM(sent), 0) FROM campaigns WHERE tenant_id = t.id),
            -- Casual/non-urgent count, fine to be eventually-consistent
            -- via this cached matview -- unlike auto-paused campaigns
            -- (get-auto-paused-campaigns), which is a live query since
            -- its whole purpose is prompt visibility right after the
            -- event that causes it.
            'scrub', JSON_BUILD_OBJECT(
                'risky_subscribers', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id AND scrub_status = 'risky')
            )
        ) AS data
    FROM tenants t;
DROP INDEX IF EXISTS mat_dashboard_stats_idx; CREATE UNIQUE INDEX mat_dashboard_stats_idx ON mat_dashboard_counts (tenant_id);


DROP MATERIALIZED VIEW IF EXISTS mat_dashboard_charts;
CREATE MATERIALIZED VIEW mat_dashboard_charts AS
    SELECT NOW() AS updated_at, t.id AS tenant_id,
        JSON_BUILD_OBJECT(
            'link_clicks', COALESCE((
                SELECT JSON_AGG(ROW_TO_JSON(row))
                FROM (
                    WITH viewDates AS (
                      SELECT created_at::DATE AS to_date,
                             created_at::DATE - INTERVAL '30 DAY' AS from_date
                             FROM link_clicks WHERE tenant_id = t.id ORDER BY id DESC LIMIT 1
                    )
                    SELECT COUNT(*) AS count, created_at::DATE as date FROM link_clicks
                      WHERE tenant_id = t.id
                        AND created_at >= (SELECT from_date FROM viewDates)
                        AND created_at < (SELECT to_date FROM viewDates) + INTERVAL '1 day'
                      GROUP by date ORDER BY date
                ) row
            ), '[]'),
            'campaign_views', COALESCE((
                SELECT JSON_AGG(ROW_TO_JSON(row))
                FROM (
                    WITH viewDates AS (
                      SELECT created_at::DATE AS to_date,
                             created_at::DATE - INTERVAL '30 DAY' AS from_date
                             FROM campaign_views WHERE tenant_id = t.id ORDER BY id DESC LIMIT 1
                    )
                    SELECT COUNT(*) AS count, created_at::DATE as date FROM campaign_views
                      WHERE tenant_id = t.id
                        AND created_at >= (SELECT from_date FROM viewDates)
                        AND created_at < (SELECT to_date FROM viewDates) + INTERVAL '1 day'
                      GROUP by date ORDER BY date
                ) row
            ), '[]')
        ) AS data
    FROM tenants t;
DROP INDEX IF EXISTS mat_dashboard_charts_idx; CREATE UNIQUE INDEX mat_dashboard_charts_idx ON mat_dashboard_charts (tenant_id);

-- subscriber counts stats for lists
DROP MATERIALIZED VIEW IF EXISTS mat_list_subscriber_stats;
CREATE MATERIALIZED VIEW mat_list_subscriber_stats AS
    SELECT NOW() AS updated_at, lists.tenant_id AS tenant_id, lists.id AS list_id, subscriber_lists.status, COUNT(subscriber_lists.status) AS subscriber_count FROM lists
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
    GROUP BY lists.tenant_id, lists.id, subscriber_lists.status
    UNION ALL
    SELECT NOW() AS updated_at, subscribers.tenant_id AS tenant_id, 0 AS list_id, NULL AS status, COUNT(id) AS subscriber_count FROM subscribers GROUP BY subscribers.tenant_id;
DROP INDEX IF EXISTS mat_list_subscriber_stats_idx; CREATE UNIQUE INDEX mat_list_subscriber_stats_idx ON mat_list_subscriber_stats (tenant_id, list_id, status);
