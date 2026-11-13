package migrations

import (
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// V6_41_0 makes row-level security fail-closed. The tenant_isolation
// policies let every row through when no tenant was set, so any query that
// forgot to scope itself saw every tenant's data. Now a session sees rows
// only for the tenant it set (app.current_tenant, core.WithTenant) or, on
// the system pool, all of them (app.all_tenants, core.System).
//
// It also revokes set_config from PUBLIC: a user-written SQL expression
// (the subscriber query filter) could otherwise change app.current_tenant
// mid-query. The app sets it with SET LOCAL, a statement an expression
// can't run. Revoking needs a superuser; without one it logs a warning.
func V6_41_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Swapping a policy takes an exclusive lock on its table, one table at
	// a time, so with the app still serving requests (another replica,
	// say) it can deadlock or wait on a busy table. It's all one
	// statement, so a failed try changes nothing; try again.
	var err error
	for try := 1; try <= policyMigrationTries; try++ {
		if err = rewritePolicies(db); err == nil || !retryableLockErr(err) {
			break
		}
		lo.Printf("v6.41.0: rewriting policies failed (%v), try %d of %d", err, try, policyMigrationTries)
		time.Sleep(time.Duration(try) * time.Second)
	}
	if err != nil {
		return err
	}

	if _, err := db.Exec(`REVOKE EXECUTE ON FUNCTION pg_catalog.set_config(text, text, boolean) FROM PUBLIC`); err != nil {
		lo.Printf("WARNING: could not revoke EXECUTE on set_config from PUBLIC (%v). A superuser must run: "+
			"REVOKE EXECUTE ON FUNCTION pg_catalog.set_config(text, text, boolean) FROM PUBLIC;", err)
	}
	return nil
}

// policyMigrationTries is how many times V6_41_0 tries rewriting the
// policies when it loses a deadlock or times out waiting for a lock.
const policyMigrationTries = 5

// retryableLockErr is a deadlock (40P01) or lock timeout (55P03).
func retryableLockErr(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && (pqErr.Code == "40P01" || pqErr.Code == "55P03")
}

// rewritePolicies recreates every tenant_isolation policy fail-closed, in
// one transaction, waiting at most a few seconds for each table's lock.
func rewritePolicies(db *sqlx.DB) error {
	_, err := db.Exec(`
		SET lock_timeout = '5s';
		DO $$
		DECLARE t TEXT;
		BEGIN
		    FOR t IN SELECT tablename FROM pg_policies
		        WHERE policyname = 'tenant_isolation' AND schemaname = current_schema()
		    LOOP
		        EXECUTE format('DROP POLICY tenant_isolation ON %I', t);
		        EXECUTE format($p$CREATE POLICY tenant_isolation ON %I USING (
		            tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::INTEGER
		            OR current_setting('app.all_tenants', true) = 'on')$p$, t);
		    END LOOP;
		END $$;
		RESET lock_timeout;
	`)
	return err
}
