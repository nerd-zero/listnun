package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_16_0 adds validate-on-add/import email validation via Scrub, plus a
// visible reason for automatically-paused campaigns.
//
//   - subscribers.scrub_status: deliverable|undeliverable|invalid_syntax|
//     risky|unchecked_error, NULL for legacy rows or when Scrub isn't
//     configured for the tenant. unchecked_error is a distinct sentinel
//     from NULL so a Scrub outage is observable rather than silently
//     indistinguishable from "nothing risky found".
//   - subscribers.scrub_checked_at: when the above was last set.
//   - campaigns.pause_reason: NULL unless a campaign was auto-paused (by
//     this feature, or retrofitted onto the pre-existing "too many
//     errors" auto-pause in internal/manager/pipe.go) -- manual
//     pause/cancel/resume leave it NULL.
func V6_16_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS scrub_status TEXT NULL;
		ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS scrub_checked_at TIMESTAMP WITH TIME ZONE NULL;
		CREATE INDEX IF NOT EXISTS idx_subscribers_scrub_status ON subscribers(tenant_id, scrub_status);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS pause_reason TEXT NULL;
	`); err != nil {
		return err
	}

	// mat_dashboard_counts must be recreated (not just refreshed) to pick
	// up the new "scrub" JSON key -- matviews are defined once at CREATE
	// time and REFRESH doesn't change their column/JSON shape, only their
	// data. Same DROP+CREATE pattern v6.7.0 used when it last changed this
	// view's shape.
	if _, err := db.Exec(`
		DROP MATERIALIZED VIEW IF EXISTS mat_dashboard_counts;
		CREATE MATERIALIZED VIEW mat_dashboard_counts AS
			SELECT NOW() AS updated_at, t.id AS tenant_id,
				JSON_BUILD_OBJECT(
					'subscribers', JSON_BUILD_OBJECT(
						'total', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id),
						'blocklisted', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id AND status = 'blocklisted'),
						'orphans', (
							SELECT COUNT(subscribers.id) FROM subscribers
							LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
							WHERE subscribers.tenant_id = t.id AND subscriber_lists.subscriber_id IS NULL
						)
					),
					'lists', JSON_BUILD_OBJECT(
						'total', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id),
						'private', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND type='private'),
						'public', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND type='public'),
						'optin_single', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND optin='single'),
						'optin_double', (SELECT COUNT(*) FROM lists WHERE tenant_id = t.id AND optin='double')
					),
					'campaigns', JSON_BUILD_OBJECT(
						'total', (SELECT COUNT(*) FROM campaigns WHERE tenant_id = t.id),
						'by_status', (
							SELECT COALESCE(JSON_OBJECT_AGG (status, num), '{}'::JSON) FROM
							(SELECT status, COUNT(*) AS num FROM campaigns WHERE tenant_id = t.id GROUP BY status) r
						)
					),
					'messages', (SELECT COALESCE(SUM(sent), 0) FROM campaigns WHERE tenant_id = t.id),
					'scrub', JSON_BUILD_OBJECT(
						'risky_subscribers', (SELECT COUNT(*) FROM subscribers WHERE tenant_id = t.id AND scrub_status = 'risky')
					)
				) AS data
			FROM tenants t;
		CREATE UNIQUE INDEX mat_dashboard_stats_idx ON mat_dashboard_counts (tenant_id);
	`); err != nil {
		return err
	}

	return nil
}
