package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// dbDSN builds a Postgres DSN for the given role, using the same
// LISTNUN_db__* env vars the CI workflow (.github/workflows/tests.yml)
// already sets for host/port/dbname, falling back to this repo's local dev
// config.toml values so the test also runs against `make run`'s dev
// database without any extra setup.
func dbDSN(user, password string) string {
	get := func(env, def string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return def
	}
	port, _ := strconv.Atoi(get("LISTNUN_db__port", "5435"))
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		get("LISTNUN_db__host", "127.0.0.1"),
		port,
		user,
		password,
		get("LISTNUN_db__database", "listnun-dev"),
	)
}

func adminDSN() string {
	get := func(env, def string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return def
	}
	return dbDSN(get("LISTNUN_db__user", "listnun-dev"), get("LISTNUN_db__password", "listnun-dev"))
}

// TestWithTenant_ConcurrentIsolation is the concurrency spike flagged in
// docs/design/multi-tenancy-code-plan.md's phase 3: it proves that
// WithTenant's SET LOCAL-via-set_config pattern stays correctly isolated
// per-goroutine under Go's own pooled *sqlx.DB, run under `go test -race`.
//
// It deliberately does NOT run as the app's configured (often superuser,
// per this repo's dev setup and CI's default postgres image role) DB role,
// since superusers and table owners bypass RLS entirely regardless of
// policy - that would make this test pass for the wrong reason. Instead it
// creates a throwaway least-privilege role for the duration of the test,
// mirroring the manual verification done for phase 2 (v6.5.0).
func TestWithTenant_ConcurrentIsolation(t *testing.T) {
	adminDB, err := sqlx.Connect("postgres", adminDSN())
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	if err := adminDB.Ping(); err != nil {
		t.Skipf("skipping: test database not reachable: %v", err)
	}

	role := fmt.Sprintf("rls_spike_test_%d", time.Now().UnixNano())
	const rolePassword = "rls_spike_test_pw"
	quotedRole := pq.QuoteIdentifier(role)

	// Two throwaway tenants with one subscriber row each, isolated from
	// any real data. Cleaned up (along with the role) at the end.
	var tenantA, tenantB int
	if err := adminDB.Get(&tenantA, `INSERT INTO tenants (slug, name) VALUES ($1, 'RLS Spike A') RETURNING id`, role+"-a"); err != nil {
		t.Fatalf("inserting test tenant A: %v", err)
	}
	if err := adminDB.Get(&tenantB, `INSERT INTO tenants (slug, name) VALUES ($1, 'RLS Spike B') RETURNING id`, role+"-b"); err != nil {
		t.Fatalf("inserting test tenant B: %v", err)
	}
	emailA := role + "-a@example.com"
	emailB := role + "-b@example.com"
	if _, err := adminDB.Exec(`INSERT INTO subscribers (tenant_id, uuid, email, name) VALUES ($1, gen_random_uuid(), $2, 'RLS Spike A')`, tenantA, emailA); err != nil {
		t.Fatalf("inserting test subscriber A: %v", err)
	}
	if _, err := adminDB.Exec(`INSERT INTO subscribers (tenant_id, uuid, email, name) VALUES ($1, gen_random_uuid(), $2, 'RLS Spike B')`, tenantB, emailB); err != nil {
		t.Fatalf("inserting test subscriber B: %v", err)
	}

	t.Cleanup(func() {
		defer adminDB.Close()
		exec := func(query string, args ...any) {
			if _, err := adminDB.Exec(query, args...); err != nil {
				t.Errorf("cleanup: %q: %v", query, err)
			}
		}
		exec(`DELETE FROM subscribers WHERE tenant_id IN ($1, $2)`, tenantA, tenantB)
		exec(`DELETE FROM tenants WHERE id IN ($1, $2)`, tenantA, tenantB)
		exec(`REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ` + quotedRole)
		exec(`REVOKE USAGE ON SCHEMA public FROM ` + quotedRole)
		exec(`DROP ROLE IF EXISTS ` + quotedRole)
	})

	adminDB.MustExec(`CREATE ROLE ` + quotedRole + ` LOGIN PASSWORD ` + pq.QuoteLiteral(rolePassword) + ` NOSUPERUSER NOBYPASSRLS`)
	adminDB.MustExec(`GRANT USAGE ON SCHEMA public TO ` + quotedRole)
	adminDB.MustExec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO ` + quotedRole)

	roleDB, err := sqlx.Connect("postgres", dbDSN(role, rolePassword))
	if err != nil {
		t.Fatalf("connecting as test role: %v", err)
	}
	defer roleDB.Close()

	tc := New(&Opt{DB: roleDB}, &Hooks{})

	const goroutinesPerTenant = 20
	var wg sync.WaitGroup
	errCh := make(chan error, goroutinesPerTenant*2)

	run := func(tenantID int, wantEmail string) {
		defer wg.Done()
		err := tc.WithTenant(context.Background(), tenantID, nil, func(tx *sqlx.Tx) error {
			var emails []string
			if err := tx.Select(&emails, `SELECT email FROM subscribers ORDER BY email`); err != nil {
				return err
			}
			for _, e := range emails {
				if e != wantEmail {
					return fmt.Errorf("tenant %d saw unexpected row %q (cross-tenant leak)", tenantID, e)
				}
			}
			return nil
		})
		if err != nil {
			errCh <- err
		}
	}

	for i := 0; i < goroutinesPerTenant; i++ {
		wg.Add(2)
		go run(tenantA, emailA)
		go run(tenantB, emailB)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}
