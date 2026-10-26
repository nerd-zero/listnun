package core

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

func testCoreForInboxCheck(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	prep := func(file, name string) *sqlx.Stmt {
		qs, err := goyesql.ParseFile("../../queries/" + file)
		if err != nil {
			t.Fatalf("parsing queries/%s: %v", file, err)
		}
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
			GetInboxCheckStats:       prep("inboxcheck.sql", "get-inbox-check-stats"),
			CountInactiveSubscribers: prep("cleanup.sql", "count-inactive-subscribers"),
		},
		i18n: lang,
		log:  log.New(os.Stderr, "", 0),
	}
}

// The audience counts only who the campaign actually goes to, and the
// rates only this workspace's campaign mail.
func TestGetInboxCheckStats(t *testing.T) {
	pool := testPool(t)
	c := testCoreForInboxCheck(t, rlsPool(t, pool))
	ctx := context.Background()
	day := 24 * time.Hour

	f, other := newListFixture(t, pool), newListFixture(t, pool)
	double := f.list("Double", models.ListOptinDouble)
	single := f.list("Single", models.ListOptinSingle)

	var camp int
	if err := pool.Get(&camp, `
		INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger, sent, started_at)
		VALUES ($1, gen_random_uuid(), 'Last week', 'subj', 'a@example.com', 'body', 'finished', 'email', 200, NOW() - INTERVAL '10 days')
		RETURNING id`, f.tenant); err != nil {
		t.Fatal(err)
	}
	pool.MustExec(`INSERT INTO campaign_lists (campaign_id, list_id, list_name, tenant_id) VALUES ($1, $2, 'Double', $4), ($1, $3, 'Single', $4)`,
		camp, double, single, f.tenant)

	// Ana opened last week's campaign; Ben hasn't opened anything.
	ana := f.subscriber("ana@example.com", "enabled")
	f.subscribe(ana, double, "confirmed", 120*day, 120*day)
	pool.MustExec(`INSERT INTO campaign_views (campaign_id, tenant_id, subscriber_id) VALUES ($1, $2, $3)`, camp, f.tenant, ana)
	ben := f.subscriber("ben@example.com", "enabled")
	f.subscribe(ben, single, "unconfirmed", 120*day, 120*day)
	// Not sent to: unconfirmed on a double opt-in list, unsubscribed, blocklisted.
	f.subscribe(f.subscriber("cara@example.com", "enabled"), double, "unconfirmed", day, day)
	f.subscribe(f.subscriber("dan@example.com", "enabled"), single, "unsubscribed", day, day)
	f.subscribe(f.subscriber("eve@example.com", "blocklisted"), single, "confirmed", day, day)

	bounce := func(tenant, sub int, campID any, typ string) {
		pool.MustExec(`INSERT INTO bounces (tenant_id, subscriber_id, campaign_id, type, source) VALUES ($1, $2, $3, $4, 'api')`,
			tenant, sub, campID, typ)
	}
	bounce(f.tenant, ben, camp, "hard")
	bounce(f.tenant, ben, camp, "hard")
	bounce(f.tenant, ben, nil, "hard") // a transactional email's
	bounce(f.tenant, ana, camp, "complaint")
	bounce(other.tenant, other.subscriber("zed@example.com", "enabled"), nil, "complaint")

	got, err := c.GetInboxCheckStats(ctx, f.tenant, []int{double, single})
	if err != nil {
		t.Fatalf("GetInboxCheckStats: %v", err)
	}
	want := InboxCheckStats{Audience: 2, Inactive: 1, Sent: 200, HardBounces: 2, Complaints: 1}
	if got != want {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
}
