-- name: get-segments
SELECT * FROM segments ORDER BY LOWER(name);

-- name: get-segment
SELECT * FROM segments WHERE id = $1;

-- name: create-segment
INSERT INTO segments (tenant_id, uuid, name, match, conditions, created_by_name)
    VALUES ($1, gen_random_uuid(), $2, $3, $4, $5)
    RETURNING *;

-- name: update-segment
UPDATE segments SET name = $2, match = $3, conditions = $4, updated_at = NOW()
    WHERE id = $1
    RETURNING *;

-- name: delete-segment
DELETE FROM segments WHERE id = $1;
