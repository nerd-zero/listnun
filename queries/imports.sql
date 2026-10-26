-- name: count-existing-emails
-- How many of the given addresses are already subscribers. Addresses are
-- stored lower case, as the importer writes them.
SELECT COUNT(*) FROM subscribers WHERE email = ANY($1::TEXT[]);

-- name: record-import
INSERT INTO imports (tenant_id, name, mode, status, total, imported, added, skipped, failed)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: get-recent-imports
SELECT * FROM imports ORDER BY created_at DESC LIMIT $1;
