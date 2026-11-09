package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_34_0 adds campaign_replies: replies to campaign emails recorded by
// the inbound mail webhook for campaigns with reply tracking on
// (models.CampaignTracking.Replies). The switch itself lives in the
// existing campaigns.tracking JSONB, so campaigns need no change -- a
// missing "replies" key reads as off.
func V6_34_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS campaign_replies (
			id                   BIGSERIAL PRIMARY KEY,
			uuid                 uuid NOT NULL UNIQUE,
			tenant_id            INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
			campaign_id          INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id        INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL,
			from_email           TEXT NOT NULL,
			from_name            TEXT NOT NULL DEFAULT '',
			subject              TEXT NOT NULL DEFAULT '',
			text_body            TEXT NOT NULL DEFAULT '',
			html_body            TEXT NOT NULL DEFAULT '',
			stripped_text        TEXT NOT NULL DEFAULT '',
			is_auto              BOOLEAN NOT NULL DEFAULT false,
			provider_message_id  TEXT NOT NULL,
			forwarded_to         TEXT NOT NULL DEFAULT '',
			forwarded_at         TIMESTAMP WITH TIME ZONE NULL,
			received_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_campaign_replies_provider_msg ON campaign_replies(tenant_id, provider_message_id);
		CREATE INDEX IF NOT EXISTS idx_campaign_replies_campaign ON campaign_replies(campaign_id, received_at DESC);

		ALTER TABLE campaign_replies ENABLE ROW LEVEL SECURITY;
		ALTER TABLE campaign_replies FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON campaign_replies;
		CREATE POLICY tenant_isolation ON campaign_replies
			USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
				OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`)
	return err
}
