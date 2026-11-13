-- name: create-user
INSERT INTO users (username, password_login, password, email, name, type, user_role_id, list_role_id, status, tenant_id)
    VALUES($1, $2, (
        CASE
            -- For user types with password_login enabled, bcrypt and store the hash of the password.
            WHEN $6::user_type != 'api' AND $2 AND $3 != ''
                THEN CRYPT($3, GEN_SALT('bf'))
            WHEN $6 = 'api'
            -- For APIs, store the password (token) as-is.
                THEN $3
            ELSE NULL
        END
    ), $4, $5, $6, (SELECT id FROM roles WHERE id = $7 AND type = 'user'), (SELECT id FROM roles WHERE id = $8 AND type = 'list'), $9, $10) RETURNING id;

-- name: update-user
WITH sa AS (
    -- The workspace's own Super Admin role (roles.super), not a hardcoded
    -- id: role ids come from one sequence across tenants, so only tenant
    -- 1's has id 1, and checking for 1 rejected every other workspace's
    -- role and status edits.
    SELECT id FROM roles WHERE super AND tenant_id = (SELECT tenant_id FROM users WHERE id = $1)
),
u AS (
    -- Edit is only allowed if there are more than 1 active super users or
    -- if the only superadmin user's status/role isn't being changed.
    SELECT
        CASE
            WHEN (SELECT COUNT(*) FROM users WHERE id != $1 AND status = 'enabled' AND type = 'user' AND user_role_id = (SELECT id FROM sa)) = 0  AND ($8 != (SELECT id FROM sa) OR $10 != 'enabled')
            THEN FALSE
            ELSE TRUE
        END AS canEdit
)
UPDATE users SET
    username=(CASE WHEN $2 != '' THEN $2 ELSE username END),
    password_login=$3,
    password=(CASE WHEN $3 = TRUE THEN (CASE WHEN $4 != '' THEN CRYPT($4, GEN_SALT('bf')) ELSE password END) ELSE NULL END),
    email=(CASE WHEN $5 != '' THEN $5 ELSE email END),
    name=(CASE WHEN $6 != '' THEN $6 ELSE name END),
    type=(CASE WHEN $7 != '' THEN $7::user_type ELSE type END),
    user_role_id=(CASE WHEN $8 != 0 THEN (SELECT id FROM roles WHERE id = $8 AND type = 'user') ELSE user_role_id END),
    list_role_id=(
        CASE
            WHEN $9 < 0 THEN NULL
            WHEN $9 > 0 THEN (SELECT id FROM roles WHERE id = $9 AND type = 'list')
            ELSE list_role_id END
    ),
    status=(CASE WHEN $10 != '' THEN $10::user_status ELSE status END),
    updated_at=NOW()
    WHERE id=$1 AND (SELECT canEdit FROM u) = TRUE;

-- name: delete-users
WITH sa AS (
    -- The workspace's own Super Admin role; see update-user.
    SELECT id FROM roles WHERE super AND tenant_id = (SELECT tenant_id FROM users WHERE id = ANY($1) LIMIT 1)
),
u AS (
    SELECT COUNT(*) AS num FROM users WHERE NOT(id = ANY($1)) AND user_role_id=(SELECT id FROM sa) AND type='user' AND status='enabled'
)
DELETE FROM users WHERE id = ALL($1) AND (SELECT num FROM u) > 0;

-- name: has-users
-- Lightweight existence check used to decide whether a tenant needs the
-- first-time-setup flow (see cmd/auth.go's LoginPage) - cheaper than
-- get-users' full role/list-role joins for a plain boolean.
SELECT EXISTS(SELECT 1 FROM users WHERE type = 'user');

-- name: get-users
WITH ur AS (
    SELECT id, name, permissions, super FROM roles WHERE type = 'user' AND parent_id IS NULL
),
lr AS (
    SELECT r.id, r.name, r.permissions, r.list_id, l.name AS list_name
    FROM roles r
    LEFT JOIN lists l ON r.list_id = l.id
    WHERE r.type = 'list' AND r.parent_id IS NULL
),
lp AS (
    SELECT lr.id AS list_role_id,
        JSONB_AGG(
            JSONB_BUILD_OBJECT(
                'id', COALESCE(cr.list_id, lr.list_id),
                'name', COALESCE(cl.name, lr.list_name),
                'permissions', COALESCE(cr.permissions, lr.permissions)
            )
        ) AS list_role_perms
    FROM lr
    LEFT JOIN roles cr ON cr.parent_id = lr.id AND cr.type = 'list'
    LEFT JOIN lists cl ON cr.list_id = cl.id
    GROUP BY lr.id
)
SELECT
    users.*,
    ur.id AS user_role_id,
    ur.name AS user_role_name,
    ur.permissions AS user_role_permissions,
    COALESCE(ur.super, false) AS user_role_super,
    lp.list_role_id,
    lr.name AS list_role_name,
    lp.list_role_perms
FROM users
    LEFT JOIN ur ON users.user_role_id = ur.id
    LEFT JOIN lp ON users.list_role_id = lp.list_role_id
    LEFT JOIN lr ON lp.list_role_id = lr.id
    ORDER BY users.created_at;

-- name: get-user
WITH sel AS (
    SELECT * FROM users
    WHERE
    (
        CASE
            WHEN $1::INT != 0 THEN users.id = $1
            WHEN $2::TEXT != '' THEN username = $2
            WHEN $3::TEXT != '' THEN email = $3
        END
    )
)
SELECT
    sel.*,
    ur.id AS user_role_id,
    ur.name AS user_role_name,
    ur.permissions AS user_role_permissions,
    COALESCE(ur.super, false) AS user_role_super,
    lr.id AS list_role_id,
    lr.name AS list_role_name,
    lp.list_role_perms
FROM sel
    LEFT JOIN roles ur ON sel.user_role_id = ur.id AND ur.type = 'user' AND ur.parent_id IS NULL
    LEFT JOIN (
        SELECT r.id, r.name, r.permissions, r.list_id, l.name AS list_name
        FROM roles r
        LEFT JOIN lists l ON r.list_id = l.id
        WHERE r.type = 'list' AND r.parent_id IS NULL
    ) lr ON sel.list_role_id = lr.id
    LEFT JOIN LATERAL (
        SELECT JSONB_AGG(
                JSONB_BUILD_OBJECT(
                    'id', COALESCE(cr.list_id, lr.list_id),
                    'name', COALESCE(cl.name, lr.list_name),
                    'permissions', COALESCE(cr.permissions, lr.permissions)
                )
            ) AS list_role_perms
        FROM roles cr
        LEFT JOIN lists cl ON cr.list_id = cl.id
        WHERE cr.parent_id = lr.id AND cr.type = 'list'
        GROUP BY lr.id
    ) lp ON TRUE;


-- name: get-api-tokens
SELECT username, password FROM users WHERE status='enabled' AND type='api';

-- name: login-user
WITH u AS (
    SELECT users.*, r.name as role_name, r.permissions FROM users
    LEFT JOIN roles r ON (r.id = users.user_role_id)
    WHERE username = $1 AND status != 'disabled' AND password_login = TRUE
    AND CRYPT($2, password) = password
)
UPDATE users SET loggedin_at = NOW() WHERE id = (SELECT id FROM u) RETURNING *;

-- name: update-user-profile
-- password_login=$4 matters beyond forgot-password's existing use (which
-- always passes back whatever GetUser fetched, so it was a same-value
-- no-op there): cmd/operator.go's doOperatorSetup reuses this query to
-- turn password login on for the first time for a freshly-provisioned,
-- passwordless tenant admin. Without writing this column, the password
-- hash gets set but the account still can't log in with it.
UPDATE users SET name=$2, email=(CASE WHEN password_login THEN $3 ELSE email END),
    password_login=$4,
    password=(CASE WHEN $4 = TRUE THEN (CASE WHEN $5 != '' THEN CRYPT($5, GEN_SALT('bf')) ELSE password END) ELSE NULL END)
    WHERE id=$1;

-- name: update-user-login
UPDATE users SET loggedin_at=NOW(), avatar=(CASE WHEN $2 != '' THEN $2 ELSE avatar END) WHERE id=$1;

-- name: set-user-twofa
UPDATE users SET twofa_type=$2::twofa_type, twofa_key=$3, updated_at=NOW() WHERE id=$1;

-- name: delete-user-sessions
DELETE FROM sessions WHERE data->>'user_id' = $1 AND ($2 = '' OR id != $2);
