-- name: insert-activity
INSERT INTO activity_log (tenant_id, level, source, message) VALUES ($1, $2, $3, $4);

-- name: get-activity
-- The workspace's most recent activity, newest first.
SELECT id, level, source, message, created_at FROM activity_log
    ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: prune-activity
DELETE FROM activity_log WHERE created_at < NOW() - $1::INTERVAL;
