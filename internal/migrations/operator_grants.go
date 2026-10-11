package migrations

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/lib/pq"
)

// ApplyOperatorGrants grants the configured [operator] BYPASSRLS role full
// read/write access (SELECT/INSERT/UPDATE/DELETE, plus sequence USAGE for
// SERIAL primary keys) on every table in the public schema, and sets
// default privileges so it keeps that access on tables created by future
// migrations. BYPASSRLS only skips row-level security policies - it does
// not grant table-level access, so a non-owner role still needs explicit
// GRANTs for cross-tenant reads *and* writes (cmd/operator.go's
// CreateTenant alone writes to tenants, settings, roles and users).
// Infra's pg_dump backups also run as this role and need every table.
//
// It grants ON ALL TABLES rather than a hand-maintained list so a new
// migration can never be missed: an earlier list-based version did not
// cover the scrub tables, and pg_dump failed with "permission denied for
// table scrub_validation_batches".
//
// The ALTER DEFAULT PRIVILEGES statements apply to objects created by the
// current role, i.e. the [db] user that runs migrations and owns the
// tables, which is exactly the role that creates future tables.
//
// It's a no-op when operator.db_user is unset (the default single-tenant
// case), and skips rather than fails if the role hasn't been provisioned in
// Postgres yet, since infra manages it out-of-band. All statements are
// idempotent; cmd/upgrade.go calls this on every --upgrade run and
// cmd/install.go after a fresh install, so already-deployed databases are
// corrected on their next boot.
func ApplyOperatorGrants(db *sqlx.DB, ko *koanf.Koanf, lo *log.Logger) error {
	role := ko.String("operator.db_user")
	if role == "" {
		return nil
	}

	var exists bool
	if err := db.Get(&exists, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role); err != nil {
		return err
	}
	if !exists {
		lo.Printf("operator.db_user %q is configured but the role does not exist in Postgres yet; skipping operator GRANTs (will retry on next boot)", role)
		return nil
	}

	r := pq.QuoteIdentifier(role)
	for _, q := range []string{
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`,
		`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO %s`,
	} {
		if _, err := db.Exec(fmt.Sprintf(q, r)); err != nil {
			return err
		}
	}

	return nil
}
