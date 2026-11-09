package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_39_0 adds saved signup forms and their events (views, signups, and
// submissions blocked as bots), which give each form its figures.
func V6_39_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS forms (
		    id               SERIAL PRIMARY KEY,
		    tenant_id        INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    uuid             uuid NOT NULL UNIQUE,
		    slug             TEXT NOT NULL,
		    name             TEXT NOT NULL,
		    status           TEXT NOT NULL DEFAULT 'draft',
		    heading          TEXT NOT NULL DEFAULT '',
		    intro            TEXT NOT NULL DEFAULT '',
		    list_ids         INTEGER[] NOT NULL DEFAULT '{}',
		    fields           JSONB NOT NULL DEFAULT '[]',
		    button_label     TEXT NOT NULL DEFAULT '',
		    success_message  TEXT NOT NULL DEFAULT '',
		    captcha          BOOLEAN NOT NULL DEFAULT TRUE,
		    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		    UNIQUE (tenant_id, slug)
		);
		CREATE TABLE IF NOT EXISTS form_events (
		    id             BIGSERIAL PRIMARY KEY,
		    tenant_id      INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    form_id        INTEGER NOT NULL REFERENCES forms(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    kind           TEXT NOT NULL,
		    subscriber_id  INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
		    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_form_events_form ON form_events(form_id, created_at);
		ALTER TABLE forms ENABLE ROW LEVEL SECURITY; ALTER TABLE forms FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON forms;
		CREATE POLICY tenant_isolation ON forms USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
		ALTER TABLE form_events ENABLE ROW LEVEL SECURITY; ALTER TABLE form_events FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON form_events;
		CREATE POLICY tenant_isolation ON form_events USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER OR NULLIF(current_setting('app.current_tenant', true), '') IS NULL);
	`)
	return err
}
