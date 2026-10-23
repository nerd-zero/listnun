-- name: get-reply-campaign
-- The campaign an inbound reply answers, for forwarding it. Run under the
-- reply token's tenant ($2); the explicit tenant_id filter backs up RLS on
-- this one cross-tenant entry point (a superuser or BYPASSRLS connection
-- would otherwise see every tenant's campaigns).
SELECT id, tenant_id, name, subject, from_email, headers, messenger FROM campaigns WHERE id = $1 AND tenant_id = $2;

-- name: insert-campaign-reply
-- $3 is the subscriber the reply token names; it's stored NULL if that
-- subscriber no longer exists (or the reply answers a test send, id 0).
-- A redelivered webhook for the same provider message inserts nothing.
INSERT INTO campaign_replies (uuid, tenant_id, campaign_id, subscriber_id, from_email, from_name,
    subject, text_body, html_body, stripped_text, is_auto, provider_message_id)
VALUES ($1, $2, $3, (SELECT id FROM subscribers WHERE id = $4), $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (tenant_id, provider_message_id) DO NOTHING
RETURNING id;

-- name: mark-campaign-reply-forwarded
UPDATE campaign_replies SET forwarded_to = $2, forwarded_at = NOW() WHERE id = $1;

-- name: get-campaign-replies
-- $1 campaign_id, $2 offset, $3 limit. Newest first.
SELECT r.id, r.uuid, r.campaign_id, r.subscriber_id, s.email AS subscriber_email, s.name AS subscriber_name,
    r.from_email, r.from_name, r.subject, r.text_body, r.html_body, r.stripped_text, r.is_auto,
    r.forwarded_to, r.forwarded_at, r.received_at, COUNT(*) OVER () AS total
    FROM campaign_replies r
    LEFT JOIN subscribers s ON s.id = r.subscriber_id
    WHERE r.campaign_id = $1
    ORDER BY r.received_at DESC, r.id DESC
    OFFSET $2 LIMIT $3;

-- name: get-campaign-reply-counts
-- Repliers counts distinct people: the subscriber when known, otherwise
-- the sending address.
SELECT
    COUNT(*) FILTER (WHERE NOT is_auto) AS replies,
    COUNT(DISTINCT COALESCE(subscriber_id::TEXT, LOWER(from_email))) FILTER (WHERE NOT is_auto) AS repliers,
    COUNT(*) FILTER (WHERE is_auto) AS auto
    FROM campaign_replies WHERE campaign_id = $1;

-- name: get-tenant-replies
-- A tenant's reply tracking setup (models.TenantReplies): its own
-- "replies" settings row, kept out of the Settings UI. No rows when reply
-- tracking was never set up for the tenant.
SELECT value FROM settings WHERE tenant_id = $1 AND key = 'replies';
