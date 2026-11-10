-- name: count-campaign-audience
-- How many subscribers a campaign of type $2 on lists $1 goes to: each
-- person once however many of the lists they're on, with the same rules
-- next-campaigns uses for to_send (no blocklisted subscribers, no
-- unsubscriptions, double opt-in lists only to confirmed subscribers, and
-- an opt-in campaign only to unconfirmed ones). Live, unlike the lists'
-- subscriber counts, which come from a periodically refreshed view and
-- count every subscription status. Runs in the workspace's session, so RLS
-- scopes it.
SELECT COUNT(DISTINCT sl.subscriber_id)
FROM subscriber_lists sl
JOIN lists l ON (l.id = sl.list_id)
JOIN subscribers s ON (s.id = sl.subscriber_id AND s.status != 'blocklisted')
WHERE sl.list_id = ANY($1::INT[])
    AND (
        CASE
            WHEN $2 = 'optin' THEN sl.status = 'unconfirmed' AND l.optin = 'double'
            WHEN l.optin = 'double' THEN sl.status = 'confirmed'
            ELSE sl.status != 'unsubscribed'
        END
    );

-- name: get-inbox-check-stats
-- What the inbox check's audience checks read (internal/inboxcheck) besides
-- the audience (count-campaign-audience): the workspace's campaign sends,
-- hard bounces and spam complaints over the last $1 days. Runs in the
-- workspace's session, so RLS scopes it.
SELECT
    (SELECT COALESCE(SUM(sent), 0) FROM campaigns
        WHERE started_at >= NOW() - MAKE_INTERVAL(days => $1)) AS sent,
    (SELECT COUNT(*) FROM bounces WHERE type = 'hard' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $1)) AS hard_bounces,
    (SELECT COUNT(*) FROM bounces WHERE type = 'complaint' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $1)) AS complaints;

-- name: get-sending-health
-- The workspace's campaign sends, hard bounces and spam complaints over the
-- last $1 days, and its campaign templates (to find the ones without an
-- unsubscribe link). For the operator API's sending health; runs in the
-- workspace's session, so RLS scopes it.
SELECT
    (SELECT COALESCE(SUM(sent), 0) FROM campaigns
        WHERE started_at >= NOW() - MAKE_INTERVAL(days => $1)) AS sent,
    (SELECT COUNT(*) FROM bounces WHERE type = 'hard' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $1)) AS hard_bounces,
    (SELECT COUNT(*) FROM bounces WHERE type = 'complaint' AND campaign_id IS NOT NULL
        AND created_at >= NOW() - MAKE_INTERVAL(days => $1)) AS complaints;

-- name: get-unsent-campaign-bodies
-- The workspace's latest unsent campaigns with their template bodies, to
-- find the ones without an unsubscribe link (likely to land in spam).
-- $1 = how many.
SELECT campaigns.id, campaigns.name, campaigns.status, campaigns.content_type, campaigns.body,
    COALESCE(templates.body, '') AS template_body
FROM campaigns
LEFT JOIN templates ON (templates.id = campaigns.template_id)
WHERE campaigns.status IN ('draft', 'scheduled', 'paused')
ORDER BY campaigns.updated_at DESC
LIMIT $1;
