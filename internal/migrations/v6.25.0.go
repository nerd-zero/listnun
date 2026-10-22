package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_25_0 adds outgoing webhooks for transactional email events
// (message.sent, .delivered, .bounced, .failed). webhook_endpoints are the
// workspace's HTTPS URLs and their signing secrets; webhook_deliveries is
// one event to one endpoint, retried with backoff for up to 24 hours;
// webhook_attempts records each try's response.
func V6_25_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_endpoints (
			id          SERIAL PRIMARY KEY,
			tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			url         TEXT NOT NULL,
			events      TEXT[] NOT NULL DEFAULT '{}',
			secret      TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'active',
			created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_tenant ON webhook_endpoints(tenant_id);

		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id                BIGSERIAL PRIMARY KEY,
			uuid              TEXT NOT NULL UNIQUE,
			tenant_id         INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			endpoint_id       INTEGER NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
			event             TEXT NOT NULL,
			message_uuid      TEXT NOT NULL DEFAULT '',
			payload           JSONB NOT NULL,
			status            TEXT NOT NULL DEFAULT 'pending',
			attempts          INTEGER NOT NULL DEFAULT 0,
			next_attempt_at   TIMESTAMP WITH TIME ZONE NULL DEFAULT NOW(),
			locked_until      TIMESTAMP WITH TIME ZONE NULL,
			last_status_code  INTEGER NULL,
			last_duration_ms  INTEGER NULL,
			created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due ON webhook_deliveries(next_attempt_at) WHERE status IN ('pending', 'retrying');
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint ON webhook_deliveries(endpoint_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant ON webhook_deliveries(tenant_id, created_at DESC);

		CREATE TABLE IF NOT EXISTS webhook_attempts (
			id            BIGSERIAL PRIMARY KEY,
			delivery_id   BIGINT NOT NULL REFERENCES webhook_deliveries(id) ON DELETE CASCADE,
			tenant_id     INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			attempt       INTEGER NOT NULL,
			status_code   INTEGER NULL,
			duration_ms   INTEGER NOT NULL DEFAULT 0,
			response      TEXT NOT NULL DEFAULT '',
			error         TEXT NOT NULL DEFAULT '',
			created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_attempts_delivery ON webhook_attempts(delivery_id, attempt);

		ALTER TABLE webhook_endpoints ENABLE ROW LEVEL SECURITY;
		ALTER TABLE webhook_endpoints FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON webhook_endpoints;
		CREATE POLICY tenant_isolation ON webhook_endpoints
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);

		ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;
		ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON webhook_deliveries;
		CREATE POLICY tenant_isolation ON webhook_deliveries
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);

		ALTER TABLE webhook_attempts ENABLE ROW LEVEL SECURITY;
		ALTER TABLE webhook_attempts FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON webhook_attempts;
		CREATE POLICY tenant_isolation ON webhook_attempts
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
