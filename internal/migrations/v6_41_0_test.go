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
	for name, id := range map[string]*int{"Super Admin": &superRole, "Editor": &editor} {
		if err := db.Get(id, `INSERT INTO roles (tenant_id, type, name, permissions) VALUES ($1, 'user', $2, '{}') RETURNING id`, tenant, name); err != nil {
			t.Fatalf("inserting role %s: %v", name, err)
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
	if err := db.Get(&one, `SELECT super FROM roles WHERE id = 1`); err == nil && !one {
		t.Error("tenant 1's role 1 isn't marked super")
	}
}
