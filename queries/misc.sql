-- name: get-dashboard-charts
SELECT data FROM mat_dashboard_charts WHERE tenant_id = $1;

-- name: get-dashboard-counts
SELECT data FROM mat_dashboard_counts WHERE tenant_id = $1;

-- name: get-settings
SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings WHERE tenant_id = $1 ORDER BY key) t;

-- name: get-tenant-root-url
-- Used by internal/manager to resolve each tenant's own app.root_url for
-- the URLs it generates (tracking pixel, link clicks, unsubscribe, optin,
-- archive) instead of the boot-time global config, which is always pinned
-- to tenant 1 (see cmd/init.go's initSettings). Returns no rows if the
-- tenant has no app.root_url set, which the caller falls back on.
SELECT value #>> '{}' FROM settings WHERE tenant_id = $1 AND key = 'app.root_url';

-- name: update-settings
UPDATE settings AS s SET value = c.value
    -- For each key in the incoming JSON map, update the row with the key and its value.
    -- The tenant_id filter is essential here, not just defense-in-depth: key names repeat
    -- across tenants by design, so without it this would update every tenant's row that
    -- happens to share a key name in the incoming map.
    FROM(SELECT * FROM JSONB_EACH($1)) AS c(key, value) WHERE s.key = c.key AND s.tenant_id = $2;

-- name: update-settings-by-key
UPDATE settings SET value = $2, updated_at = NOW() WHERE key = $1 AND tenant_id = $3;

-- name: get-db-info
SELECT JSON_BUILD_OBJECT('version', (SELECT VERSION()),
                        'size_mb', (SELECT ROUND(pg_database_size((SELECT CURRENT_DATABASE()))/(1024^2)))) AS info;

-- name: get-dashboard-overview
-- Live dashboard numbers for the last $1 days (today included) for tenant
-- $2: per-day sends (campaign sends counted on the day the campaign
-- started, plus transactional deliveries), bounces, transactional
-- failures and new subscribers, and the counts behind "Needs attention".
WITH since AS (SELECT (CURRENT_DATE - ($1::INT - 1)) AS d),
days AS (SELECT GENERATE_SERIES((SELECT d FROM since), CURRENT_DATE, '1 day')::DATE AS day),
camp AS (
    SELECT started_at::DATE AS day, SUM(sent) AS n FROM campaigns
    WHERE tenant_id = $2 AND started_at >= (SELECT d FROM since) GROUP BY 1
),
tx AS (
    SELECT created_at::DATE AS day,
        COUNT(*) FILTER (WHERE status IN ('delivered', 'sent')) AS delivered,
        COUNT(*) FILTER (WHERE status = 'failed') AS failed
    FROM tx_messages WHERE tenant_id = $2 AND created_at >= (SELECT d FROM since) GROUP BY 1
),
bnc AS (
    SELECT created_at::DATE AS day, COUNT(*) AS n FROM bounces
    WHERE tenant_id = $2 AND created_at >= (SELECT d FROM since) GROUP BY 1
),
subs AS (
    SELECT created_at::DATE AS day, COUNT(*) AS n FROM subscribers
    WHERE tenant_id = $2 AND created_at >= (SELECT d FROM since) GROUP BY 1
)
SELECT JSON_BUILD_OBJECT(
    'days', (
        SELECT JSON_AGG(JSON_BUILD_OBJECT(
            'date', days.day,
            'delivered', COALESCE(camp.n, 0) + COALESCE(tx.delivered, 0),
            'bounced', COALESCE(bnc.n, 0),
            'failed', COALESCE(tx.failed, 0),
            'new_subscribers', COALESCE(subs.n, 0)
        ) ORDER BY days.day)
        FROM days
        LEFT JOIN camp ON camp.day = days.day
        LEFT JOIN tx ON tx.day = days.day
        LEFT JOIN bnc ON bnc.day = days.day
        LEFT JOIN subs ON subs.day = days.day
    ),
    'subscribers', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = $2),
    'unsubscribed', (
        SELECT COUNT(DISTINCT subscriber_id) FROM subscriber_lists
        WHERE tenant_id = $2 AND status = 'unsubscribed' AND updated_at >= (SELECT d FROM since)
    ),
    'campaigns_started', (SELECT COUNT(*) FROM campaigns WHERE tenant_id = $2 AND started_at >= (SELECT d FROM since)),
    'running', (SELECT COUNT(*) FROM campaigns WHERE tenant_id = $2 AND status = 'running'),
    'queued', (
        (SELECT COALESCE(SUM(GREATEST(to_send - sent, 0)), 0) FROM campaigns WHERE tenant_id = $2 AND status = 'running')
        + (SELECT COUNT(*) FROM tx_messages WHERE tenant_id = $2 AND status = 'queued' AND created_at > NOW() - INTERVAL '1 day')
    ),
    'stale_drafts', (
        SELECT COUNT(*) FROM campaigns WHERE tenant_id = $2 AND status = 'draft' AND updated_at < NOW() - INTERVAL '30 days'
    ),
    'bounce_hotspot', (
        SELECT JSON_BUILD_OBJECT('list_id', COALESCE(cl.list_id, 0), 'list_name', cl.list_name, 'hard', COUNT(*))
        FROM bounces b JOIN campaign_lists cl ON cl.campaign_id = b.campaign_id
        WHERE b.tenant_id = $2 AND b.type = 'hard' AND b.created_at > NOW() - INTERVAL '7 days'
        GROUP BY cl.list_id, cl.list_name ORDER BY COUNT(*) DESC LIMIT 1
    )
) AS data;
