-- media
-- name: insert-media
INSERT INTO media (uuid, filename, thumb, content_type, provider, meta, created_at, tenant_id, size) VALUES($1, $2, $3, $4, $5, $6, NOW(), $7, $8) RETURNING id;

-- name: get-media-usage
-- Storage used by the tenant's media. Files never measured (NULL) or that
-- couldn't be measured (negative) are counted as unknown, not as zero.
SELECT COALESCE(SUM(size) FILTER (WHERE size >= 0), 0) AS bytes,
    COUNT(*) AS files,
    COUNT(*) FILTER (WHERE size IS NULL OR size < 0) AS unknown
FROM media;

-- name: get-unsized-media
-- Files uploaded before sizes were recorded, across tenants, for the
-- backfill.
SELECT id, tenant_id, filename, provider FROM media WHERE size IS NULL ORDER BY id LIMIT $1;

-- name: set-media-size
UPDATE media SET size = $2 WHERE id = $1;

-- name: query-media
SELECT COUNT(*) OVER () AS total, * FROM media
    WHERE ($1 = '' OR filename ILIKE $1) AND provider=$2 ORDER BY created_at DESC OFFSET $3 LIMIT $4;

-- name: get-media
SELECT * FROM media WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN filename = $3    
        ELSE false
    END;

-- name: delete-media
DELETE FROM media WHERE id=$1 RETURNING filename;

