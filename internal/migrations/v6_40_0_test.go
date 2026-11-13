package migrations

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	get := func(env, def string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return def
	}
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		get("LISTNUN_db__host", "127.0.0.1"), get("LISTNUN_db__port", "5435"),
		get("LISTNUN_db__user", "listnun-dev"), get("LISTNUN_db__password", "listnun-dev"),
		get("LISTNUN_db__database", "listnun-dev")))
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// Each workspace's Super Admin role gets marked, tenant 1's by its ID even
// if renamed, and running the migration again changes nothing.
func TestV6_40_0_MarksEachWorkspacesSuperAdmin(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)
	if err := V6_40_0(db, nil, nil, lo); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var tenant int
	if err := db.Get(&tenant, `INSERT INTO tenants (slug, name) VALUES ($1, 'Super test') RETURNING id`,
		fmt.Sprintf("super-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting tenant: %v", err)
	}
	t.Cleanup(func() { db.MustExec(`DELETE FROM tenants WHERE id = $1`, tenant) })

	var superRole, editor int
	for name, id := range map[string]*int{"Super Admin": &superRole, "Editor": &editor} {
		if err := db.Get(id, `INSERT INTO roles (tenant_id, type, name, permissions) VALUES ($1, 'user', $2, '{}') RETURNING id`, tenant, name); err != nil {
			t.Fatalf("inserting role %s: %v", name, err)
		}
	}

	for range 2 {
		if err := V6_40_0(db, nil, nil, lo); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	var got []int
	db.Select(&got, `SELECT id FROM roles WHERE tenant_id = $1 AND super ORDER BY id`, tenant)
	if len(got) != 1 || got[0] != superRole {
		t.Errorf("super roles = %v, want only the workspace's Super Admin %d", got, superRole)
	}

	var one bool
	if err := db.Get(&one, `SELECT super FROM roles WHERE id = 1`); err == nil && !one {
		t.Error("tenant 1's role 1 isn't marked super")
	}
}

// Every tenant_isolation policy is rewritten fail-closed, and running the
// migration again changes nothing.
func TestV6_41_0_FailClosedPolicies(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)
	for range 2 {
		if err := V6_41_0(db, nil, nil, lo); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	var total, closed int
	db.Get(&total, `SELECT COUNT(*) FROM pg_policies WHERE policyname = 'tenant_isolation'`)
	db.Get(&closed, `SELECT COUNT(*) FROM pg_policies WHERE policyname = 'tenant_isolation'
		AND qual LIKE '%app.all_tenants%' AND qual NOT LIKE '%IS NULL%'`)
	if total == 0 || closed != total {
		t.Errorf("%d of %d policies are fail-closed", closed, total)
	}
}
