-- api keys (Developers area). Secrets are stored as SHA-256 hashes.

-- name: get-api-keys
SELECT id, name, secret_hint, scopes, status, created_by, created_by_name,
    prev_expires_at, last_used_at, revoked_at, created_at, updated_at,
    COUNT(*) OVER () AS total
    FROM api_keys
    WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
    ORDER BY (status = 'revoked'), created_at DESC;

-- name: get-api-key
SELECT id, name, secret_hint, scopes, status, created_by, created_by_name,
    prev_expires_at, last_used_at, revoked_at, created_at, updated_at
    FROM api_keys WHERE id = $1;

-- name: get-api-key-auth
-- Fetches what's needed to authenticate a request. Revoked keys are
-- excluded, and an expired previous secret is returned as NULL.
SELECT id, tenant_id, name, scopes, secret_hash,
    (CASE WHEN prev_expires_at > NOW() THEN prev_secret_hash ELSE NULL END) AS prev_secret_hash
    FROM api_keys WHERE tenant_id = $1 AND name = $2 AND status = 'active';

-- name: create-api-key
INSERT INTO api_keys (name, secret_hash, secret_hint, scopes, created_by, created_by_name, tenant_id)
    VALUES ($1, $2, $3, $4, NULLIF($5, 0), $6, $7) RETURNING id;

-- name: update-api-key-scopes
UPDATE api_keys SET scopes = $2, updated_at = NOW() WHERE id = $1 AND status = 'active';

-- name: rotate-api-key
-- The current secret becomes the previous one, valid until $4.
UPDATE api_keys SET
    prev_secret_hash = secret_hash,
    prev_expires_at = $4,
    secret_hash = $2,
    secret_hint = $3,
    updated_at = NOW()
    WHERE id = $1 AND status = 'active';

-- name: revoke-api-key
UPDATE api_keys SET status = 'revoked', revoked_at = NOW(),
    prev_secret_hash = NULL, prev_expires_at = NULL, updated_at = NOW()
    WHERE id = $1 AND status = 'active';

-- name: delete-api-key
DELETE FROM api_keys WHERE id = $1;

-- name: touch-api-key
-- Records use at most once a minute per key to keep writes off the hot path.
UPDATE api_keys SET last_used_at = NOW()
    WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '1 minute');
