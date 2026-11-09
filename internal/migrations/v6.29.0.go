package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_29_0 turns every existing API user into an API key, as the handover
// asks. The key gets only the scopes the user's role already covers today
// (tx:send, subscribers:read); the newer scopes stay off. Its own secret is
// a value no request can match, so nothing changes for integrations: the
// API user's username:token keeps working, and rotating the key issues its
// first real secret. legacy_user_id links the two so the UI can offer to
// disable the old token once an integration has moved over.
func V6_29_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS legacy_user_id INTEGER NULL REFERENCES users(id) ON DELETE SET NULL;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_legacy_user ON api_keys(legacy_user_id) WHERE legacy_user_id IS NOT NULL;

		INSERT INTO api_keys (tenant_id, name, secret_hash, secret_hint, scopes, status, created_by_name, legacy_user_id)
		SELECT u.tenant_id,
			LEFT(REGEXP_REPLACE(LOWER('api-' || u.username), '[^a-z0-9._-]+', '-', 'g'), 63),
			'unusable:' || MD5(RANDOM()::TEXT || CLOCK_TIMESTAMP()::TEXT || u.id::TEXT),
			'',
			ARRAY_REMOVE(ARRAY[
				CASE WHEN r.id = 1 OR 'tx:send' = ANY(r.permissions) THEN 'tx:send' END,
				CASE WHEN r.id = 1 OR 'subscribers:get_all' = ANY(r.permissions) THEN 'subscribers:read' END
			], NULL),
			'active',
			u.username,
			u.id
		FROM users u
		LEFT JOIN roles r ON r.id = u.user_role_id
		WHERE u.type = 'api'
			AND NOT EXISTS (SELECT 1 FROM api_keys k WHERE k.legacy_user_id = u.id)
		ON CONFLICT (tenant_id, name) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
