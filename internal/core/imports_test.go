package core

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

func testCoreForImports(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	qs, err := goyesql.ParseFile("../../queries/imports.sql")
	if err != nil {
		t.Fatalf("parsing queries/imports.sql: %v", err)
	}
	prep := func(name string) *sqlx.Stmt {
		stmt, err := pool.Preparex(qs[name].Query)
		if err != nil {
			t.Fatalf("prepare %s: %v", name, err)
		}
		t.Cleanup(func() { stmt.Close() })
		return stmt
	}
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("building i18n: %v", err)
	}
	return &Core{
		db: pool,
		q: &models.Queries{
			CountExistingEmails: prep("count-existing-emails"),
			RecordImport:        prep("record-import"),
			GetRecentImports:    prep("get-recent-imports"),
		},
		i18n: lang,
	}
}

// Another workspace's subscribers never count as "already here", and its
// imports never show in this workspace's history.
func TestImports_ScopedToTenant(t *testing.T) {
	pool := testPool(t)
	admin := rlsPool(t, pool)
	pool.MustExec(`GRANT INSERT ON imports TO ` + currentUser(t, admin))
	pool.MustExec(`GRANT USAGE ON SEQUENCE imports_id_seq TO ` + currentUser(t, admin))
	c := testCoreForImports(t, admin)
	ctx := context.Background()

	mine, theirs := newListFixture(t, pool), newListFixture(t, pool)
	mine.subscriber("ana@example.com", "enabled")
	theirs.subscriber("tendai@example.com", "enabled")

	n, err := c.CountExistingEmails(ctx, mine.tenant, []string{"ana@example.com", "tendai@example.com", "new@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("existing = %d, want 1 (only this workspace's)", n)
	}

	for _, r := range []models.ImportRecord{
		{TenantID: mine.tenant, Name: "old.csv", Mode: "subscribe", Status: "finished", Total: 3, Imported: 3, Added: 2},
		{TenantID: theirs.tenant, Name: "theirs.csv", Mode: "subscribe", Status: "finished"},
		{TenantID: mine.tenant, Name: "new.csv", Mode: "subscribe", Status: "finished", Total: 10, Imported: 9, Added: 9, Skipped: 1},
	} {
		if err := c.RecordImport(ctx, r); err != nil {
			t.Fatalf("RecordImport: %v", err)
		}
	}

	got, err := c.GetRecentImports(ctx, mine.tenant, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "new.csv" || got[1].Name != "old.csv" {
		t.Fatalf("history = %+v, want new.csv then old.csv", got)
	}
	if got[0].Added != 9 || got[0].Skipped != 1 {
		t.Errorf("new.csv = %+v, want 9 added, 1 skipped", got[0])
	}
}
