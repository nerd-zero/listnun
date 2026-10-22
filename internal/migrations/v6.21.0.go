package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_21_0 adds api_keys for the Developers area: named keys that
// authenticate API calls with HTTP basic auth (key name as the username),
// each with its own scopes. Only a SHA-256 hash of the secret is stored;
// the full secret is shown once, at creation or rotation. Rotating keeps
// the previous hash valid until prev_expires_at so callers can deploy the
// new secret without downtime.
func V6_21_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id                SERIAL PRIMARY KEY,
			tenant_id         INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			name              TEXT NOT NULL,
			secret_hash       TEXT NOT NULL,
			secret_hint       TEXT NOT NULL,
			prev_secret_hash  TEXT NULL,
			prev_expires_at   TIMESTAMP WITH TIME ZONE NULL,
			scopes            TEXT[] NOT NULL DEFAULT '{}',
			status            TEXT NOT NULL DEFAULT 'active',
			created_by        INTEGER NULL REFERENCES users(id) ON DELETE SET NULL,
			created_by_name   TEXT NOT NULL DEFAULT '',
			last_used_at      TIMESTAMP WITH TIME ZONE NULL,
			revoked_at        TIMESTAMP WITH TIME ZONE NULL,
			created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_name ON api_keys(tenant_id, name);
		CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);

		ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
		ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON api_keys;
		CREATE POLICY tenant_isolation ON api_keys
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
