-- name: record-bounce
-- Insert a bounce and count the bounces for the subscriber and either unsubscribe them,
-- This runs outside any tenant session context (bounce webhooks/POP3 polling
-- have no request-scoped tenant to resolve - see internal/core/bounces.go's
-- RecordBounce), so the inserted bounce's tenant_id is derived from the
-- resolved subscriber's own tenant_id rather than a session variable.
-- $10 is the tenant the bounce came in for (a webhook's request tenant):
-- when non-zero, the subscriber and campaign lookups are limited to it, so
-- a bounce can't match the same email address in another tenant. 0 (POP3
-- mailbox scanning, which has no tenant) keeps the unscoped lookup.
WITH sub AS (
    SELECT id, status, tenant_id FROM subscribers
    WHERE CASE WHEN $1 != '' THEN uuid = $1::UUID ELSE email = $2 END
        AND ($10::INT = 0 OR tenant_id = $10::INT)
),
camp AS (
    SELECT id FROM campaigns WHERE $3 != '' AND uuid = $3::UUID
        AND ($10::INT = 0 OR tenant_id = $10::INT)
),
num AS (
    -- Add a +1 to include the current insertion that is happening.
    SELECT COUNT(*) + 1 AS num FROM bounces WHERE subscriber_id = (SELECT id FROM sub) AND type = $4
),
-- block1 and block2 will run when $8 = 'blocklist' and the number of bounces exceed $8.
block1 AS (
    UPDATE subscribers SET status='blocklisted'
    WHERE $9 = 'blocklist' AND (SELECT num FROM num) >= $8 AND id = (SELECT id FROM sub) AND (SELECT status FROM sub) != 'blocklisted'
),
block2 AS (
    UPDATE subscriber_lists SET status='unsubscribed'
    WHERE $9 = 'unsubscribe' AND (SELECT num FROM num) >= $8 AND subscriber_id = (SELECT id FROM sub) AND (SELECT status FROM sub) != 'blocklisted'
),
bounce AS (
    -- Record the bounce if the subscriber is not already blocklisted;
    INSERT INTO bounces (subscriber_id, campaign_id, type, source, meta, created_at, tenant_id)
    SELECT (SELECT id FROM sub), (SELECT id FROM camp), $4, $5, $6, $7, (SELECT tenant_id FROM sub)
    WHERE NOT EXISTS (SELECT 1 WHERE (SELECT status FROM sub) = 'blocklisted' OR (SELECT num FROM num) > $8)
)
-- This delete  will only run when $9 = 'delete' and the number of bounces exceed $8.
DELETE FROM subscribers
    WHERE $9 = 'delete' AND (SELECT num FROM num) >= $8 AND id = (SELECT id FROM sub);

-- name: query-bounces
SELECT COUNT(*) OVER () AS total,
    bounces.id,
    bounces.type,
    bounces.source,
    bounces.meta,
    bounces.created_at,
    bounces.subscriber_id,
    subscribers.uuid AS subscriber_uuid,
    subscribers.email AS email,
    subscribers.status as subscriber_status,
    (
        CASE WHEN bounces.campaign_id IS NOT NULL
        THEN JSON_BUILD_OBJECT('id', bounces.campaign_id, 'name', campaigns.name)
        ELSE NULL END
    ) AS campaign
FROM bounces
LEFT JOIN subscribers ON (subscribers.id = bounces.subscriber_id)
LEFT JOIN campaigns ON (campaigns.id = bounces.campaign_id)
WHERE ($1 = 0 OR bounces.id = $1)
    AND ($2 = 0 OR bounces.campaign_id = $2)
    AND ($3 = 0 OR bounces.subscriber_id = $3)
    AND ($4 = '' OR bounces.source = $4)
    AND ($5 = '' OR bounces.type::TEXT = $5)
    AND ($6 = '' OR subscribers.email ILIKE '%' || $6 || '%' OR campaigns.name ILIKE '%' || $6 || '%')
ORDER BY %order% OFFSET $7 LIMIT (CASE WHEN $8 < 1 THEN NULL ELSE $8 END);

-- name: get-bounce-type-counts
-- Bounce counts per type for the filters in query-bounces, minus the type
-- filter itself, so the type chips can show what each one would return.
SELECT
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE bounces.type = 'hard') AS hard,
    COUNT(*) FILTER (WHERE bounces.type = 'soft') AS soft,
    COUNT(*) FILTER (WHERE bounces.type = 'complaint') AS complaint
FROM bounces
LEFT JOIN subscribers ON (subscribers.id = bounces.subscriber_id)
LEFT JOIN campaigns ON (campaigns.id = bounces.campaign_id)
WHERE ($1 = 0 OR bounces.campaign_id = $1)
    AND ($2 = '' OR bounces.source = $2)
    AND ($3 = '' OR subscribers.email ILIKE '%' || $3 || '%' OR campaigns.name ILIKE '%' || $3 || '%');

-- name: delete-bounces
-- Deletes the bounces in $1, or with $2, every bounce matching the same
-- filters as query-bounces: $3 campaign, $4 source, $5 type, $6 e-mail or
-- campaign name. Empty filters match everything.
DELETE FROM bounces WHERE id = ANY($1) OR ($2 = TRUE AND id IN (
    SELECT b.id FROM bounces b
    LEFT JOIN subscribers s ON (s.id = b.subscriber_id)
    LEFT JOIN campaigns c ON (c.id = b.campaign_id)
    WHERE ($3 = 0 OR b.campaign_id = $3)
        AND ($4 = '' OR b.source = $4)
        AND ($5 = '' OR b.type::TEXT = $5)
        AND ($6 = '' OR s.email ILIKE '%' || $6 || '%' OR c.name ILIKE '%' || $6 || '%')
));

-- name: delete-bounces-by-subscriber
WITH sub AS (
    SELECT id FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
)
DELETE FROM bounces WHERE subscriber_id = (SELECT id FROM sub);

-- name: blocklist-bounced-subscribers
-- Blocklists everyone with a bounce matching the same filters as
-- query-bounces: $1 campaign, $2 source, $3 type, $4 e-mail or campaign
-- name. Empty filters match everything.
WITH subs AS (
    SELECT DISTINCT b.subscriber_id FROM bounces b
    LEFT JOIN subscribers s ON (s.id = b.subscriber_id)
    LEFT JOIN campaigns c ON (c.id = b.campaign_id)
    WHERE ($1 = 0 OR b.campaign_id = $1)
        AND ($2 = '' OR b.source = $2)
        AND ($3 = '' OR b.type::TEXT = $3)
        AND ($4 = '' OR s.email ILIKE '%' || $4 || '%' OR c.name ILIKE '%' || $4 || '%')
),
b AS (
    UPDATE subscribers SET status='blocklisted', updated_at=NOW()
    WHERE id = ANY(SELECT subscriber_id FROM subs)
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY(SELECT subscriber_id FROM subs);

