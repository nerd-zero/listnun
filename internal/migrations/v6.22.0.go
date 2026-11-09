package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_22_0 adds the transactional message log: one tx_messages row per
// recipient of a POST /api/tx call, and tx_message_events for its
// timeline (accepted, rendered, sent, delivered, bounced, failed). The
// request payload and rendered body are kept so a message can be
// inspected and resent. Rows older than the retention window are pruned
// by the app.
func V6_22_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tx_messages (
			id              BIGSERIAL PRIMARY KEY,
			uuid            TEXT NOT NULL UNIQUE,
			tenant_id       INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			api_key_id      INTEGER NULL REFERENCES api_keys(id) ON DELETE SET NULL,
			sender          TEXT NOT NULL DEFAULT '',
			template_id     INTEGER NULL REFERENCES templates(id) ON DELETE SET NULL,
			template_name   TEXT NOT NULL DEFAULT '',
			subscriber_id   INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL,
			recipient       TEXT NOT NULL,
			subject         TEXT NOT NULL DEFAULT '',
			status          TEXT NOT NULL DEFAULT 'queued',
			bounce_type     TEXT NULL,
			error           TEXT NOT NULL DEFAULT '',
			messenger       TEXT NOT NULL DEFAULT '',
			source_ip       TEXT NOT NULL DEFAULT '',
			payload         JSONB NOT NULL DEFAULT '{}',
			body            TEXT NOT NULL DEFAULT '',
			created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_tx_messages_tenant_created ON tx_messages(tenant_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_tx_messages_recipient ON tx_messages(tenant_id, LOWER(recipient), created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_tx_messages_status ON tx_messages(tenant_id, status);

		CREATE TABLE IF NOT EXISTS tx_message_events (
			id          BIGSERIAL PRIMARY KEY,
			message_id  BIGINT NOT NULL REFERENCES tx_messages(id) ON DELETE CASCADE,
			tenant_id   INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			type        TEXT NOT NULL,
			detail      TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_tx_message_events_message ON tx_message_events(message_id, created_at);

		ALTER TABLE tx_messages ENABLE ROW LEVEL SECURITY;
		ALTER TABLE tx_messages FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON tx_messages;
		CREATE POLICY tenant_isolation ON tx_messages
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);

		ALTER TABLE tx_message_events ENABLE ROW LEVEL SECURITY;
		ALTER TABLE tx_message_events FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON tx_message_events;
		CREATE POLICY tenant_isolation ON tx_message_events
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`); err != nil {
		return err
	}

	return nil
}
