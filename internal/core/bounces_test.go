package core

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

func testCoreForBounces(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	qs, err := goyesql.ParseFile("../../queries/bounces.sql")
	if err != nil {
		t.Fatalf("parsing queries/bounces.sql: %v", err)
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
			DeleteBounces:               prep("delete-bounces"),
			BlocklistBouncedSubscribers: prep("blocklist-bounced-subscribers"),
		},
		i18n: lang,
		log:  log.New(os.Stderr, "", 0),
	}
}

// "Delete all" and "Blocklist all" on the Bounces page act on the list as
// filtered, never on bounces the person can't see. Regression: with the
// Soft filter on, "Delete all 10" deleted every bounce in the workspace.
func TestBounces_AllActionsFollowTheFilter(t *testing.T) {
	pool := testPool(t)
	admin := rlsPool(t, pool)
	user := currentUser(t, admin)
	for _, q := range []string{
		`GRANT DELETE ON bounces TO ` + user,
		`GRANT UPDATE (status, updated_at) ON subscribers TO ` + user,
		`GRANT UPDATE (status, updated_at) ON subscriber_lists TO ` + user,
	} {
		pool.MustExec(q)
	}
	c := testCoreForBounces(t, admin)
	ctx := context.Background()

	mine, theirs := newListFixture(t, pool), newListFixture(t, pool)
	news := mine.list("News", models.ListOptinSingle)
	bounce := func(f *listFixture, sub int, typ string) {
		t.Helper()
		pool.MustExec(`INSERT INTO bounces (tenant_id, subscriber_id, type, source) VALUES ($1, $2, $3, 'api')`, f.tenant, sub, typ)
	}
	ana, ben, cara := mine.subscriber("ana@example.com", "enabled"), mine.subscriber("ben@example.com", "enabled"), mine.subscriber("cara@example.com", "enabled")
	mine.subscribe(ben, news, "confirmed", 0, 0)
	bounce(mine, ana, "hard")
	bounce(mine, ana, "soft")
	bounce(mine, ben, "soft")
	bounce(mine, cara, "complaint")
	dan := theirs.subscriber("dan@example.com", "enabled")
	bounce(theirs, dan, "soft")

	types := func(f *listFixture) []string {
		t.Helper()
		var out []string
		if err := pool.Select(&out, `SELECT type::TEXT FROM bounces WHERE tenant_id = $1 ORDER BY type`, f.tenant); err != nil {
			t.Fatal(err)
		}
		sort.Strings(out)
		return out
	}
	status := func(sub int) string {
		t.Helper()
		var s string
		if err := pool.Get(&s, `SELECT status FROM subscribers WHERE id = $1`, sub); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// Blocklisting all hard bounces blocklists Ana only, not Ben (soft).
	if err := c.BlocklistBouncedSubscribers(ctx, mine.tenant, models.BounceFilter{Type: "hard"}); err != nil {
		t.Fatal(err)
	}
	if status(ana) != "blocklisted" || status(ben) != "enabled" || status(cara) != "enabled" {
		t.Errorf("after blocklisting hard bounces: ana %s, ben %s, cara %s; want only ana blocklisted",
			status(ana), status(ben), status(cara))
	}

	// Deleting all soft bounces leaves the hard and complaint ones, and the
	// other workspace's soft bounce.
	if err := c.DeleteBounces(ctx, mine.tenant, nil, true, models.BounceFilter{Type: "soft"}); err != nil {
		t.Fatal(err)
	}
	if got := types(mine); len(got) != 2 || got[0] != "complaint" || got[1] != "hard" {
		t.Errorf("after deleting soft bounces: %v, want [complaint hard]", got)
	}

	// A search narrows "all" too.
	if err := c.DeleteBounces(ctx, mine.tenant, nil, true, models.BounceFilter{Search: "cara@"}); err != nil {
		t.Fatal(err)
	}
	if got := types(mine); len(got) != 1 || got[0] != "hard" {
		t.Errorf("after deleting cara's bounces: %v, want [hard]", got)
	}

	// No filter deletes the rest of this workspace's bounces, and only them.
	if err := c.DeleteBounces(ctx, mine.tenant, nil, true, models.BounceFilter{}); err != nil {
		t.Fatal(err)
	}
	if got := types(mine); len(got) != 0 {
		t.Errorf("after deleting all: %v, want none", got)
	}
	if got := types(theirs); len(got) != 1 || status(dan) != "enabled" {
		t.Errorf("other workspace: bounces %v, dan %s; want its soft bounce untouched", got, status(dan))
	}
}

// A bounce for an address that isn't a subscriber (a transactional email,
// or a subscriber already deleted) is skipped, not an error. Regression: the
// insert failed on tenant_id's NOT NULL check before subscriber_id's, so
// every such Postmark webhook logged "error recording bounce" twice.
func TestRecordBounce_SkipsUnknownSubscriber(t *testing.T) {
	pool := testPool(t)
	qs, err := goyesql.ParseFile("../../queries/bounces.sql")
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := pool.Preparex(qs["record-bounce"].Query)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stmt.Close() })
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatal(err)
	}
	c := &Core{
		db:   pool,
		q:    &models.Queries{RecordBounce: stmt},
		i18n: lang,
		log:  log.New(os.Stderr, "", 0),
	}
	c.consts.BounceActions = map[string]struct {
		Count  int
		Action string
	}{"hard": {Count: 1, Action: "blocklist"}}

	// A campaign UUID skips the transactional message lookup.
	err = c.RecordBounce(models.Bounce{
		Email:        "nobody-here@example.com",
		CampaignUUID: "00000000-0000-0000-0000-000000000000",
		Type:         "hard",
		Source:       "postmark",
		Meta:         json.RawMessage(`{}`),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("RecordBounce: %v, want nil", err)
	}
}
