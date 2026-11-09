-- outgoing webhooks (Developers area).

-- name: get-webhook-endpoints
-- Endpoints with their last-24h delivery numbers. The secret isn't returned.
SELECT e.id, e.url, e.events, e.status, RIGHT(e.secret, 4) AS secret_hint, e.created_at, e.updated_at,
    COALESCE(s.done, 0) AS done_24h,
    COALESCE(s.ok, 0) AS ok_24h,
    COALESCE(s.median_ms, 0) AS median_ms,
    (SELECT COUNT(*) FROM webhook_deliveries r WHERE r.endpoint_id = e.id AND r.status = 'retrying') AS retrying
    FROM webhook_endpoints e
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) FILTER (WHERE d.status IN ('success', 'failed', 'retrying')) AS done,
            COUNT(*) FILTER (WHERE d.status = 'success') AS ok,
            PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY d.last_duration_ms)
                FILTER (WHERE d.last_duration_ms IS NOT NULL)::INT AS median_ms
        FROM webhook_deliveries d
        WHERE d.endpoint_id = e.id AND d.created_at > NOW() - INTERVAL '24 hours'
    ) s ON TRUE
    WHERE ($1 = 0 OR e.id = $1)
    ORDER BY e.created_at;

-- name: get-webhook-endpoint-secret
SELECT secret FROM webhook_endpoints WHERE id = $1;

-- name: create-webhook-endpoint
INSERT INTO webhook_endpoints (url, events, secret, tenant_id) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: update-webhook-endpoint
UPDATE webhook_endpoints SET url = $2, events = $3, status = $4, updated_at = NOW() WHERE id = $1;

-- name: rotate-webhook-endpoint-secret
UPDATE webhook_endpoints SET secret = $2, updated_at = NOW() WHERE id = $1;

-- name: delete-webhook-endpoint
DELETE FROM webhook_endpoints WHERE id = $1;

-- name: enqueue-webhook-deliveries
-- One delivery per active endpoint of the tenant subscribed to the event.
-- $3 is the payload; the delivery ID is generated per row from $4, a prefix.
INSERT INTO webhook_deliveries (uuid, tenant_id, endpoint_id, event, message_uuid, payload)
    SELECT $4 || e.id::TEXT, e.tenant_id, e.id, $2, $5, $3
    FROM webhook_endpoints e
    WHERE e.tenant_id = $1 AND e.status = 'active' AND $2 = ANY(e.events);

-- name: enqueue-webhook-test
INSERT INTO webhook_deliveries (uuid, tenant_id, endpoint_id, event, payload)
    SELECT $3, tenant_id, id, $2, $4 FROM webhook_endpoints WHERE id = $1
    RETURNING id;

-- name: claim-webhook-deliveries
-- Locks up to $1 due deliveries for this worker for $2 seconds. SKIP LOCKED
-- lets several app instances share the queue.
UPDATE webhook_deliveries d SET locked_until = NOW() + ($2 || ' seconds')::INTERVAL
    FROM webhook_endpoints e
    WHERE d.id IN (
        SELECT id FROM webhook_deliveries
        WHERE status IN ('pending', 'retrying') AND next_attempt_at <= NOW()
            AND (locked_until IS NULL OR locked_until < NOW())
        ORDER BY next_attempt_at
        LIMIT $1
        FOR UPDATE SKIP LOCKED
    ) AND e.id = d.endpoint_id
    RETURNING d.id, d.uuid, d.tenant_id, d.endpoint_id, d.event, d.payload, d.attempts, e.url, e.secret;

-- name: record-webhook-attempt
-- Records one try and moves the delivery on: $7 is the next status and $8
-- when to try again (NULL when done).
WITH a AS (
    INSERT INTO webhook_attempts (delivery_id, tenant_id, attempt, status_code, duration_ms, response, error)
        VALUES ($1, $2, $3, NULLIF($4, 0), $5, $6, $9)
)
UPDATE webhook_deliveries SET
    attempts = $3,
    status = $7,
    next_attempt_at = $8,
    locked_until = NULL,
    last_status_code = NULLIF($4, 0),
    last_duration_ms = $5,
    updated_at = NOW()
    WHERE id = $1;

-- name: query-webhook-deliveries
SELECT d.id, d.uuid, d.endpoint_id, e.url AS endpoint_url, d.event, d.message_uuid, d.status, d.attempts,
    d.next_attempt_at, d.last_status_code, d.last_duration_ms, d.created_at, d.updated_at,
    COUNT(*) OVER () AS total
    FROM webhook_deliveries d
    JOIN webhook_endpoints e ON e.id = d.endpoint_id
    WHERE ($1 = 0 OR d.endpoint_id = $1) AND ($2 = '' OR d.status = $2)
    ORDER BY d.created_at DESC, d.id DESC
    OFFSET $3 LIMIT $4;

-- name: get-webhook-delivery
SELECT d.id, d.uuid, d.endpoint_id, e.url AS endpoint_url, d.event, d.message_uuid, d.payload, d.status, d.attempts,
    d.next_attempt_at, d.last_status_code, d.last_duration_ms, d.created_at, d.updated_at
    FROM webhook_deliveries d
    JOIN webhook_endpoints e ON e.id = d.endpoint_id
    WHERE d.uuid = $1;

-- name: get-webhook-attempts
SELECT attempt, status_code, duration_ms, response, error, created_at
    FROM webhook_attempts WHERE delivery_id = $1 ORDER BY created_at, id;

-- name: retry-webhook-delivery
-- Sends a delivery again now. A finished one gets a fresh set of attempts.
UPDATE webhook_deliveries SET
    status = (CASE WHEN status = 'retrying' THEN 'retrying' ELSE 'pending' END),
    attempts = (CASE WHEN status IN ('success', 'failed') THEN 0 ELSE attempts END),
    next_attempt_at = NOW(), locked_until = NULL, updated_at = NOW()
    WHERE uuid = $1;

-- name: prune-webhook-deliveries
DELETE FROM webhook_deliveries WHERE created_at < NOW() - $1::INTERVAL AND status IN ('success', 'failed');
