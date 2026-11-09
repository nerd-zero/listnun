-- Inactive-subscriber cleanup (Settings › Inactive cleanup; rules in
-- internal/cleanup). $1 = inactivity window in days, $2 = what counts as
-- activity (opened, clicked, either), $3 = list IDs (empty = all lists).
-- A subscriber is inactive on a list when they joined it before the window,
-- a campaign went to it during the window, and they neither opened nor
-- clicked anything in that time. Blocklisted subscribers are left alone.

-- name: count-inactive-subscribers
WITH target AS (
    SELECT DISTINCT sl.subscriber_id
    FROM subscriber_lists sl
    JOIN subscribers s ON (s.id = sl.subscriber_id)
    WHERE sl.status != 'unsubscribed'
        AND s.status != 'blocklisted'
        AND (CARDINALITY($3::INT[]) = 0 OR sl.list_id = ANY($3::INT[]))
        -- Subscribed before the window: newer subscribers haven't had the time.
        AND sl.created_at < NOW() - MAKE_INTERVAL(days => $1)
        -- Had a chance to engage: a campaign went to this list in the window.
        AND EXISTS (
            SELECT 1 FROM campaign_lists cl JOIN campaigns c ON (c.id = cl.campaign_id)
            WHERE cl.list_id = sl.list_id AND c.status IN ('running', 'finished')
                AND c.started_at >= NOW() - MAKE_INTERVAL(days => $1)
        )
)
SELECT COUNT(*) FROM (SELECT t.subscriber_id FROM target t
WHERE NOT ($2 IN ('opened', 'either') AND EXISTS (
        SELECT 1 FROM campaign_views v WHERE v.subscriber_id = t.subscriber_id
            AND v.created_at >= NOW() - MAKE_INTERVAL(days => $1)))
    AND NOT ($2 IN ('clicked', 'either') AND EXISTS (
        SELECT 1 FROM link_clicks k WHERE k.subscriber_id = t.subscriber_id
            AND k.created_at >= NOW() - MAKE_INTERVAL(days => $1)))) x;

-- name: get-inactive-subscriber-ids
-- One batch of inactive subscriber IDs; $4 = batch size.
WITH target AS (
    SELECT DISTINCT sl.subscriber_id
    FROM subscriber_lists sl
    JOIN subscribers s ON (s.id = sl.subscriber_id)
    WHERE sl.status != 'unsubscribed'
        AND s.status != 'blocklisted'
        AND (CARDINALITY($3::INT[]) = 0 OR sl.list_id = ANY($3::INT[]))
        -- Subscribed before the window: newer subscribers haven't had the time.
        AND sl.created_at < NOW() - MAKE_INTERVAL(days => $1)
        -- Had a chance to engage: a campaign went to this list in the window.
        AND EXISTS (
            SELECT 1 FROM campaign_lists cl JOIN campaigns c ON (c.id = cl.campaign_id)
            WHERE cl.list_id = sl.list_id AND c.status IN ('running', 'finished')
                AND c.started_at >= NOW() - MAKE_INTERVAL(days => $1)
        )
)
SELECT t.subscriber_id FROM target t
WHERE NOT ($2 IN ('opened', 'either') AND EXISTS (
        SELECT 1 FROM campaign_views v WHERE v.subscriber_id = t.subscriber_id
            AND v.created_at >= NOW() - MAKE_INTERVAL(days => $1)))
    AND NOT ($2 IN ('clicked', 'either') AND EXISTS (
        SELECT 1 FROM link_clicks k WHERE k.subscriber_id = t.subscriber_id
            AND k.created_at >= NOW() - MAKE_INTERVAL(days => $1)))
ORDER BY t.subscriber_id LIMIT $4;

-- name: unsubscribe-inactive
-- Unsubscribes the given subscribers from $2's lists, or all lists if empty.
UPDATE subscriber_lists SET status = 'unsubscribed', updated_at = NOW()
    WHERE subscriber_id = ANY($1::INT[]) AND status != 'unsubscribed'
        AND (CARDINALITY($2::INT[]) = 0 OR list_id = ANY($2::INT[]));

-- name: insert-cleanup-run
INSERT INTO inactive_cleanup_runs (tenant_id, mode, action) VALUES ($1, $2, $3) RETURNING id;

-- name: finish-cleanup-run
UPDATE inactive_cleanup_runs SET finished_at = NOW(), matched = $2, done = $3, error = $4 WHERE id = $1;

-- name: get-last-cleanup-run
SELECT id, mode, action, matched, done, error, started_at, finished_at
    FROM inactive_cleanup_runs ORDER BY started_at DESC LIMIT 1;
