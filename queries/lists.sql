-- lists
-- name: get-lists
SELECT * FROM lists WHERE (CASE WHEN $1 = '' THEN 1=1 ELSE type=$1::list_type END)
    AND (CASE WHEN $2 = '' THEN 1=1 ELSE status=$2::list_status END)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $4 = TRUE THEN TRUE ELSE id = ANY($5::INT[])
    END
    ORDER BY CASE WHEN $3 = 'id' THEN id END, CASE WHEN $3 = 'name' THEN name END;

-- name: query-lists
-- Sorted before the page is cut, so every page follows the same order
-- (including by subscriber_count, which needs the stats joined first).
WITH statuses AS (
    SELECT
        list_id,
        COALESCE(JSONB_OBJECT_AGG(status, subscriber_count) FILTER (WHERE status IS NOT NULL), '{}') AS subscriber_statuses,
        SUM(subscriber_count) AS subscriber_count
    FROM mat_list_subscriber_stats
    GROUP BY list_id
),
ls AS (
    SELECT COUNT(*) OVER () AS total, lists.*,
        COALESCE(ss.subscriber_statuses, '{}') AS subscriber_statuses,
        COALESCE(ss.subscriber_count, 0) AS subscriber_count
    FROM lists LEFT JOIN statuses ss ON (lists.id = ss.list_id)
    WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN (TO_TSVECTOR(name) @@ TO_TSQUERY ($3) OR name ILIKE $3)
        ELSE TRUE
    END
    AND ($4 = '' OR type = $4::list_type)
    AND ($5 = '' OR optin = $5::list_optin)
    AND ($6 = '' OR status = $6::list_status)
    AND (CARDINALITY($7::VARCHAR(100)[]) = 0 OR $7 <@ tags)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $8 = TRUE THEN TRUE ELSE id = ANY($9::INT[])
    END
    ORDER BY %order%, id
    OFFSET $10 LIMIT (CASE WHEN $11 < 1 THEN NULL ELSE $11 END)
)
SELECT * FROM ls ORDER BY %order%, id;

-- name: get-lists-by-optin
-- Can have a list of IDs or a list of UUIDs.
SELECT * FROM lists WHERE (CASE WHEN $1 != '' THEN optin=$1::list_optin ELSE TRUE END) AND
    (CASE WHEN $2::INT[] IS NOT NULL THEN id = ANY($2::INT[])
          WHEN $3::UUID[] IS NOT NULL THEN uuid = ANY($3::UUID[])
    END) ORDER BY name;

-- name: get-list-types
-- Retrieves the private|public type of lists by ID or uuid. Used for filtering.
SELECT id, uuid, type FROM lists WHERE
    (CASE WHEN $1::INT[] IS NOT NULL THEN id = ANY($1::INT[])
          WHEN $2::UUID[] IS NOT NULL THEN uuid = ANY($2::UUID[])
    END);

-- name: get-list-ids-by-uuids
-- Resolves list UUIDs to IDs, tenant-scoped -- used by processSubForm
-- (public.go), which works in UUIDs throughout, to get the int list IDs
-- the Scrub risky-subscriber auto-pause path needs.
SELECT id FROM lists WHERE tenant_id = $1 AND uuid = ANY($2::UUID[]);

-- name: create-list
INSERT INTO lists (uuid, name, type, optin, status, tags, description, tenant_id) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;

-- name: update-list
WITH l AS (
    UPDATE lists SET
        name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
        type=(CASE WHEN $3 != '' THEN $3::list_type ELSE type END),
        optin=(CASE WHEN $4 != '' THEN $4::list_optin ELSE optin END),
        status=(CASE WHEN $5 != '' THEN $5::list_status ELSE status END),
        tags=$6::VARCHAR(100)[],
        description=(CASE WHEN $7 != '' THEN $7 ELSE description END),
        updated_at=NOW()
    WHERE id = $1
    RETURNING id, name
),
c AS (
    UPDATE campaign_lists SET list_name = l.name FROM l WHERE campaign_lists.list_id = l.id RETURNING 1
)
SELECT COUNT(*) FROM l, c;

-- name: update-lists-date
UPDATE lists SET updated_at=NOW() WHERE id = ANY($1);

-- name: get-list-scrub-status
-- Lightweight per-list Scrub summary for GetScrubListStatus -- deliberately
-- not the full get-lists/query-lists shape (subscriber counts etc.),
-- which that handler doesn't need.
SELECT id, scrub_last_validated_at, scrub_last_valid_count, scrub_last_invalid_count
FROM lists WHERE tenant_id = $1;

-- name: update-list-scrub-result
-- Written once a Scrub validation batch/job targeting this list finishes
-- (cmd/scrub_batch.go's reconcileScrubBatch/reconcileScrubJob) -- the
-- persistent home for GetScrubListStatus's "last_result", now that
-- Scrub's own list-tracking is no longer consulted.
UPDATE lists SET scrub_last_validated_at = NOW(), scrub_last_valid_count = $2, scrub_last_invalid_count = $3
WHERE id = $1;

-- name: delete-lists
DELETE FROM lists
WHERE CASE
    WHEN CARDINALITY($1::INT[]) > 0 THEN id = ANY($1)
    ELSE ($2 = '' OR to_tsvector(name) @@ to_tsquery($2))
END
AND CASE
    -- Optional list IDs based on user permission.
    WHEN $3 = TRUE THEN TRUE ELSE id = ANY($4::INT[])
END;


-- name: get-lists-growth
-- Joins and leaves per list since $1. A subscription that's
-- unsubscribed and was last changed since $1 counts as a leave: leaving
-- is the only change that sets that status, and nothing changes the row
-- after it.
SELECT list_id,
    COUNT(*) FILTER (WHERE created_at >= $1) AS joined,
    COUNT(*) FILTER (WHERE status = 'unsubscribed' AND updated_at >= $1) AS left
    FROM subscriber_lists
    WHERE created_at >= $1 OR updated_at >= $1
    GROUP BY list_id;

-- name: get-list-activity
-- Daily joins and leaves on list $1 from day $2 to today, one row per day.
WITH days AS (
    SELECT d::DATE AS day FROM GENERATE_SERIES($2::DATE, NOW()::DATE, '1 day') d
),
joined AS (
    SELECT created_at::DATE AS day, COUNT(*) AS n FROM subscriber_lists
    WHERE list_id = $1 AND created_at >= $2::DATE GROUP BY 1
),
left_ AS (
    SELECT updated_at::DATE AS day, COUNT(*) AS n FROM subscriber_lists
    WHERE list_id = $1 AND status = 'unsubscribed' AND updated_at >= $2::DATE GROUP BY 1
)
SELECT days.day, COALESCE(joined.n, 0) AS joined, COALESCE(left_.n, 0) AS left
    FROM days
    LEFT JOIN joined ON joined.day = days.day
    LEFT JOIN left_ ON left_.day = days.day
    ORDER BY days.day;

-- name: get-list-recent-campaigns
-- The $2 latest campaigns sent, sending or scheduled to list $1, newest
-- (or soonest scheduled) first. Opens and clicks count people, falling
-- back to raw counts when individual tracking is off and no subscriber
-- is recorded.
SELECT c.id, c.uuid, c.name, c.status, c.started_at, c.send_at, c.sent,
    (SELECT CASE WHEN COUNT(subscriber_id) > 0 THEN COUNT(DISTINCT subscriber_id) ELSE COUNT(*) END
        FROM campaign_views WHERE campaign_id = c.id) AS opened,
    (SELECT CASE WHEN COUNT(subscriber_id) > 0 THEN COUNT(DISTINCT subscriber_id) ELSE COUNT(*) END
        FROM link_clicks WHERE campaign_id = c.id) AS clicked
    FROM campaigns c
    JOIN campaign_lists cl ON cl.campaign_id = c.id AND cl.list_id = $1
    WHERE c.status IN ('scheduled', 'running', 'paused', 'finished')
    ORDER BY COALESCE(c.started_at, c.send_at) DESC NULLS LAST, c.id DESC
    LIMIT $2;

-- name: get-optin-pending
-- Up to $3 people waiting to confirm a double opt-in subscription on
-- lists $1 (every list when NULL) who are due a reminder: not sent a
-- confirmation since $2, and no pending subscription added since $2
-- (adding one sends a confirmation itself).
WITH pending AS (
    SELECT sl.subscriber_id AS id, MAX(sl.created_at) AS last_added FROM subscriber_lists sl
    JOIN lists l ON l.id = sl.list_id AND l.optin = 'double'
    WHERE sl.status = 'unconfirmed' AND ($1::INT[] IS NULL OR sl.list_id = ANY($1::INT[]))
    GROUP BY sl.subscriber_id
)
SELECT s.* FROM subscribers s JOIN pending p ON p.id = s.id
    WHERE s.status = 'enabled' AND p.last_added < $2
    AND (s.optin_sent_at IS NULL OR s.optin_sent_at < $2)
    ORDER BY s.id
    LIMIT $3;

-- name: count-optin-pending
-- Everyone get-optin-pending considers, and how many of them are due.
WITH pending AS (
    SELECT sl.subscriber_id AS id, MAX(sl.created_at) AS last_added FROM subscriber_lists sl
    JOIN lists l ON l.id = sl.list_id AND l.optin = 'double'
    WHERE sl.status = 'unconfirmed' AND ($1::INT[] IS NULL OR sl.list_id = ANY($1::INT[]))
    GROUP BY sl.subscriber_id
)
SELECT COUNT(*) AS total,
    COUNT(*) FILTER (WHERE p.last_added < $2 AND (s.optin_sent_at IS NULL OR s.optin_sent_at < $2)) AS due
    FROM subscribers s JOIN pending p ON p.id = s.id
    WHERE s.status = 'enabled';

-- name: mark-optin-sent
UPDATE subscribers SET optin_sent_at = NOW() WHERE id = $1;

-- name: get-lists-summary
-- How many lists there are and the subscriptions on them that aren't
-- unsubscribed, among lists $2 (all when $1), from the cached list stats.
SELECT COUNT(DISTINCT l.id) AS lists, COALESCE(SUM(s.subscriber_count), 0) AS subscriptions
    FROM lists l
    LEFT JOIN mat_list_subscriber_stats s ON s.list_id = l.id AND s.status != 'unsubscribed'
    WHERE l.status = 'active' AND ($1 OR l.id = ANY($2::INT[]));
