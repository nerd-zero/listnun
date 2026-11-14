package migrations

import (
	"fmt"
	"io"
	"log"
	"testing"
	"time"
)

// Each workspace's Super Admin role gets marked, tenant 1's by its ID even
// if renamed, and running the migration again changes nothing.
func TestV6_41_0_MarksEachWorkspacesSuperAdmin(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)
	if err := V6_41_0(db, nil, nil, lo); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var tenant int
	if err := db.Get(&tenant, `INSERT INTO tenants (slug, name) VALUES ($1, 'Super test') RETURNING id`,
		fmt.Sprintf("super-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting tenant: %v", err)
	}
	t.Cleanup(func() { db.MustExec(`DELETE FROM tenants WHERE id = $1`, tenant) })

	var superRole, editor int
	for _, r := range []struct {
		name string
		id   *int
	}{{"Super Admin", &superRole}, {"Editor", &editor}} {
		if err := db.Get(r.id, `INSERT INTO roles (tenant_id, type, name, permissions) VALUES ($1, 'user', $2, '{}') RETURNING id`, tenant, r.name); err != nil {
			t.Fatalf("inserting role %s: %v", r.name, err)
		}
	}

	for range 2 {
		if err := V6_41_0(db, nil, nil, lo); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	var got []int
	db.Select(&got, `SELECT id FROM roles WHERE tenant_id = $1 AND super ORDER BY id`, tenant)
	if len(got) != 1 || got[0] != superRole {
		t.Errorf("super roles = %v, want only the workspace's Super Admin %d", got, superRole)
	}

	var one bool
	if err := db.Get(&one, `SELECT super FROM roles WHERE id = 1 AND tenant_id = 1`); err == nil && !one {
		t.Error("tenant 1's role 1 isn't marked super")
	}
}

// Role 1 is tenant 1's Super Admin only in tenant 1. In a database where
// role 1 belongs to another workspace (a fresh test database), that
// workspace's role 1 and its Super Admin were both marked, which broke the
// one-per-workspace index and failed the upgrade.
func TestV6_41_0_RoleOneElsewhereIsNotSuper(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)
	if err := V6_41_0(db, nil, nil, lo); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var tenant int
	if err := db.Get(&tenant, `INSERT INTO tenants (slug, name) VALUES ($1, 'Super test') RETURNING id`,
		fmt.Sprintf("super-role-one-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting tenant: %v", err)
	}
	t.Cleanup(func() { db.MustExec(`DELETE FROM tenants WHERE id = $1`, tenant) })

	res := db.MustExec(`INSERT INTO roles (id, tenant_id, type, name, permissions) VALUES (1, $1, 'user', 'Editor', '{}') ON CONFLICT DO NOTHING`, tenant)
	if n, _ := res.RowsAffected(); n == 0 {
		t.Skip("role 1 already exists in this database")
	}
	db.MustExec(`SELECT SETVAL('roles_id_seq', (SELECT MAX(id) FROM roles))`)
	var superRole int
	if err := db.Get(&superRole, `INSERT INTO roles (tenant_id, type, name, permissions) VALUES ($1, 'user', 'Super Admin', '{}') RETURNING id`, tenant); err != nil {
		t.Fatalf("inserting role: %v", err)
	}

	if err := V6_41_0(db, nil, nil, lo); err != nil {
		t.Fatalf("migration: %v", err)
	}
	var got []int
	db.Select(&got, `SELECT id FROM roles WHERE tenant_id = $1 AND super ORDER BY id`, tenant)
	if len(got) != 1 || got[0] != superRole {
		t.Errorf("super roles = %v, want only the workspace's Super Admin %d", got, superRole)
	}
}
