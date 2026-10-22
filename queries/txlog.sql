-- transactional message log (Developers area).

-- name: create-tx-message
-- Inserts a message with its first events in one statement. $13 is a JSON
-- array of {"type", "detail"} events, recorded in order.
WITH m AS (
    INSERT INTO tx_messages (uuid, api_key_id, sender, template_id, template_name, subscriber_id,
        recipient, subject, status, error, messenger, source_ip, payload, body, tenant_id)
    VALUES ($1, NULLIF($2, 0), $3, NULLIF($4, 0), $5, NULLIF($6, 0), $7, $8, $9, $10, $11, $12, $14, $15, $16)
    RETURNING id, tenant_id, created_at
),
ev AS (
    INSERT INTO tx_message_events (message_id, tenant_id, type, detail, created_at)
    SELECT m.id, m.tenant_id, e->>'type', COALESCE(e->>'detail', ''),
        m.created_at + (ord * INTERVAL '1 microsecond')
    FROM m, JSONB_ARRAY_ELEMENTS($13::JSONB) WITH ORDINALITY AS x(e, ord)
)
SELECT id FROM m;

-- name: set-tx-message-status
-- Moves a message to a new status and records the event. A message that
-- already bounced or failed isn't moved back to delivered.
WITH m AS (
    UPDATE tx_messages SET
        status = $2,
        bounce_type = COALESCE(NULLIF($4, ''), bounce_type),
        error = (CASE WHEN $2 = 'failed' THEN $3 ELSE error END),
        updated_at = NOW()
    WHERE id = $1 AND NOT ($2 IN ('sent', 'delivered') AND status IN ('bounced', 'failed'))
    RETURNING id, tenant_id
)
INSERT INTO tx_message_events (message_id, tenant_id, type, detail)
    SELECT id, tenant_id, $5, $3 FROM m
    RETURNING message_id;

-- name: get-tx-message-for-bounce
-- The latest message to an address in the last 72 hours, for attributing a
-- bounce. $1 = 0 matches any tenant (mailbox bounces carry no tenant).
SELECT id FROM tx_messages
    WHERE ($1 = 0 OR tenant_id = $1) AND LOWER(recipient) = LOWER($2)
    AND created_at > NOW() - INTERVAL '72 hours'
    ORDER BY created_at DESC LIMIT 1;

-- name: query-tx-messages
-- $1 search (recipient or message ID), $2 status, $3 template_id,
-- $4 api_key_id, $5 from, $6 to, $7 offset, $8 limit.
SELECT m.id, m.uuid, m.api_key_id, COALESCE(k.name, m.sender) AS sender, m.template_id,
    COALESCE(t.name, m.template_name) AS template_name, m.subscriber_id, m.recipient, m.subject,
    m.status, m.bounce_type, m.error, m.messenger, m.created_at, m.updated_at,
    COUNT(*) OVER () AS total
    FROM tx_messages m
    LEFT JOIN api_keys k ON k.id = m.api_key_id
    LEFT JOIN templates t ON t.id = m.template_id
    WHERE ($1 = '' OR m.uuid = $1 OR m.recipient ILIKE '%' || $1 || '%')
    AND ($2 = '' OR m.status = $2)
    AND ($3 = 0 OR m.template_id = $3)
    AND ($4 = 0 OR m.api_key_id = $4)
    AND ($5::TIMESTAMPTZ IS NULL OR m.created_at >= $5)
    AND ($6::TIMESTAMPTZ IS NULL OR m.created_at < $6)
    ORDER BY m.created_at DESC, m.id DESC
    OFFSET $7 LIMIT $8;

-- name: count-tx-messages-by-status
-- Status counts for the filter pills, with the same filters as
-- query-tx-messages except status.
SELECT status, COUNT(*) AS count FROM tx_messages m
    WHERE ($1 = '' OR m.uuid = $1 OR m.recipient ILIKE '%' || $1 || '%')
    AND ($2 = 0 OR m.template_id = $2)
    AND ($3 = 0 OR m.api_key_id = $3)
    AND ($4::TIMESTAMPTZ IS NULL OR m.created_at >= $4)
    AND ($5::TIMESTAMPTZ IS NULL OR m.created_at < $5)
    GROUP BY status;

-- name: get-tx-message
SELECT m.id, m.uuid, m.api_key_id, COALESCE(k.name, m.sender) AS sender, m.template_id,
    COALESCE(t.name, m.template_name) AS template_name, m.subscriber_id,
    COALESCE(s.status::TEXT, '') AS subscriber_status, m.recipient, m.subject,
    m.status, m.bounce_type, m.error, m.messenger, m.source_ip, m.payload, m.body,
    m.created_at, m.updated_at
    FROM tx_messages m
    LEFT JOIN api_keys k ON k.id = m.api_key_id
    LEFT JOIN templates t ON t.id = m.template_id
    LEFT JOIN subscribers s ON s.id = m.subscriber_id
    WHERE m.uuid = $1;

-- name: get-tx-message-events
SELECT type, detail, created_at FROM tx_message_events
    WHERE message_id = $1 ORDER BY created_at, id;

-- name: get-tx-stats
-- Last-24h numbers for the Developers overview, plus the day before for
-- the trend and the most common failure reason.
SELECT
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') AS sent,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '48 hours' AND created_at <= NOW() - INTERVAL '24 hours') AS sent_prev,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours' AND status = 'delivered') AS delivered,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours' AND status = 'bounced') AS bounced,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours' AND status = 'failed') AS failed,
    COALESCE((SELECT error FROM tx_messages
        WHERE status = 'failed' AND error != '' AND created_at > NOW() - INTERVAL '24 hours'
        GROUP BY error ORDER BY COUNT(*) DESC LIMIT 1), '') AS top_error
    FROM tx_messages WHERE created_at > NOW() - INTERVAL '48 hours';

-- name: prune-tx-messages
DELETE FROM tx_messages WHERE created_at < NOW() - $1::INTERVAL;
