-- name: get-inbox-check-stats
-- What the inbox check's audience checks read (internal/inboxcheck), for
-- the campaign's lists $1: how many subscribers it goes to, and the
-- workspace's campaign sends, hard bounces and spam complaints over the
-- last $2 days. Runs in the workspace's session, so RLS scopes it.
WITH audience AS (
    SELECT DISTINCT sl.subscriber_id
    FROM subscriber_lists sl
    JOIN subscribers s ON (s.id = sl.subscriber_id)
    JOIN lists l ON (l.id = sl.list_id)
    WHERE sl.list_id = ANY($1::INT[])
        AND sl.status != 'unsubscribed'
        AND s.status != 'blocklisted'
        -- Double opt-in lists only send to confirmed subscribers.
        AND NOT (l.optin = 'double' AND sl.status = 'unconfirmed')
)
SELECT
    (SELECT COUNT(*) FROM audience) AS audience,
    (SELECT COALESCE(SUM(sent), 0) FROM campaigns
        WHERE started_at >= NOW() - MAKE_INTERVAL(days => $2)) AS sent,
    (SELECT COUNT(*) FROM bounces WHERE type = 'hard' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $2)) AS hard_bounces,
    (SELECT COUNT(*) FROM bounces WHERE type = 'complaint' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $2)) AS complaints;
