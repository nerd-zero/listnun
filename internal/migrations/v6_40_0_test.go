package migrations

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"
)

// The platform's existing servers get marked managed, a workspace's own
// don't, servers already marked keep their mark, and running it again
// changes nothing.
func TestV6_40_0_MarksPlatformSMTP(t *testing.T) {
	db := testDB(t)
	lo := log.New(io.Discard, "", 0)

	type server struct {
		Host    string `json:"host"`
		Managed *bool  `json:"managed_by_platform,omitempty"`
	}
	no := false
	tenant := func(servers []server) int {
		var id int
		if err := db.Get(&id, `INSERT INTO tenants (slug, name) VALUES ($1, 'Migration test') RETURNING id`,
			fmt.Sprintf("v6-40-0-%d", time.Now().UnixNano())); err != nil {
			t.Fatalf("inserting tenant: %v", err)
		}
		t.Cleanup(func() { db.MustExec(`DELETE FROM tenants WHERE id = $1`, id) })
		b, _ := json.Marshal(servers)
		db.MustExec(`INSERT INTO settings (tenant_id, key, value) VALUES ($1, 'smtp', $2::jsonb)
			ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value`, id, b)
		return id
	}
	read := func(id int) []server {
		var b []byte
		if err := db.Get(&b, `SELECT value FROM settings WHERE tenant_id = $1 AND key = 'smtp'`, id); err != nil {
			t.Fatalf("reading smtp: %v", err)
		}
		var out []server
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	platform := tenant([]server{{Host: " SMTP.postmarkapp.com"}})
	own := tenant([]server{{Host: "smtp.own.example"}, {Host: "smtp.backup.example"}})
	kept := tenant([]server{{Host: "smtp.postmarkapp.com", Managed: &no}})

	for range 2 {
		if err := V6_40_0(db, nil, nil, lo); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}

	if s := read(platform); s[0].Managed == nil || !*s[0].Managed {
		t.Errorf("platform server = %+v, want it marked managed", s)
	}
	for _, s := range read(own) {
		if s.Managed == nil || *s.Managed {
			t.Errorf("own server %+v, want managed_by_platform false", s)
		}
	}
	if s := read(own); len(s) != 2 || s[0].Host != "smtp.own.example" || s[1].Host != "smtp.backup.example" {
		t.Errorf("own servers = %+v, want both kept in order", s)
	}
	if s := read(kept); *s[0].Managed {
		t.Errorf("server already marked = %+v, want its mark kept", s)
	}
}
