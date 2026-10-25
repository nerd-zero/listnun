-- name: get-forms
-- Every form, with its signups over the last 30 days.
SELECT f.*, (
    SELECT COUNT(*) FROM form_events e
    WHERE e.form_id = f.id AND e.kind = 'signup' AND e.created_at >= NOW() - INTERVAL '30 days'
) AS signups
FROM forms f ORDER BY f.created_at;

-- name: get-form
SELECT *, 0 AS signups FROM forms WHERE id = $1;

-- name: get-form-by-slug
SELECT *, 0 AS signups FROM forms WHERE slug = $1;

-- name: form-slug-exists
SELECT EXISTS (SELECT 1 FROM forms WHERE slug = $1);

-- name: create-form
INSERT INTO forms (tenant_id, uuid, slug, name, status, heading, intro, list_ids, fields, button_label, success_message, captcha)
    VALUES ($1, gen_random_uuid(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    RETURNING *, 0 AS signups;

-- name: update-form
UPDATE forms SET name = $2, status = $3, heading = $4, intro = $5, list_ids = $6, fields = $7,
    button_label = $8, success_message = $9, captcha = $10, slug = $11, updated_at = NOW()
    WHERE id = $1
    RETURNING *, 0 AS signups;

-- name: delete-form
DELETE FROM forms WHERE id = $1;

-- name: add-form-event
INSERT INTO form_events (tenant_id, form_id, kind, subscriber_id) VALUES ($1, $2, $3, $4);

-- name: get-form-stats
-- A form's views, signups and blocked submissions since $2, and how many
-- of the people who signed up have confirmed a subscription to one of its
-- lists.
WITH f AS (SELECT id, list_ids FROM forms WHERE id = $1),
ev AS (SELECT e.* FROM form_events e, f WHERE e.form_id = f.id AND e.created_at >= $2)
SELECT
    COUNT(*) FILTER (WHERE kind = 'view') AS views,
    COUNT(*) FILTER (WHERE kind = 'signup') AS signups,
    COUNT(*) FILTER (WHERE kind = 'blocked') AS blocked,
    (SELECT COUNT(DISTINCT ev.subscriber_id) FROM ev, f
        WHERE ev.kind = 'signup' AND EXISTS (
            SELECT 1 FROM subscriber_lists sl
            WHERE sl.subscriber_id = ev.subscriber_id AND sl.list_id = ANY(f.list_ids) AND sl.status = 'confirmed'
        )) AS confirmed
FROM ev;
