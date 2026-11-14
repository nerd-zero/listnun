package migrations

import (
	"io"
	"log"
	"testing"
)

// Every tenant_isolation policy is rewritten fail-closed, and running the
// migration again changes nothing.
func TestV6_42_0_FailClosedPolicies(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)
	for range 2 {
		if err := V6_42_0(db, nil, nil, lo); err != nil {
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
