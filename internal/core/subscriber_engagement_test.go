package core

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

// Only campaigns that started on a person's list while they were on it
// count as sent to them: not before they joined, not after they left.
func TestSubscriberEngagement_CountsCampaignsWhileOnTheList(t *testing.T) {
	pool := testPool(t)
	f := newListFixture(t, pool)
	ctx := context.Background()
	day := 24 * time.Hour

	qs, err := goyesql.ParseFile("../../queries/subscribers.sql")
	if err != nil {
		t.Fatalf("parsing queries/subscribers.sql: %v", err)
	}
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("building i18n: %v", err)
	}
	db := rlsPool(t, pool)
	prep := func(name string) *sqlx.Stmt {
		st, err := db.Preparex(qs[name].Query)
		if err != nil {
			t.Fatalf("prepare %s: %v", name, err)
		}
		t.Cleanup(func() { st.Close() })
		return st
	}
	c := &Core{db: db, i18n: lang, q: &models.Queries{
		GetSubscriberActivity:   prep("get-subscriber-activity"),
		GetSubscriberEngagement: prep("get-subscriber-engagement"),
	}}

	news := f.list("News", models.ListOptinSingle)
	left := f.list("Left", models.ListOptinSingle)
	other := f.list("Other", models.ListOptinSingle)
	sub := f.subscriber("ana@example.com", "enabled")
	f.subscribe(sub, news, "confirmed", 10*day, 10*day)
	f.subscribe(sub, left, "unsubscribed", 30*day, 15*day)

	started := func(name string, list int, ago time.Duration) int {
		var id int
		if err := pool.Get(&id, `
			INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger, started_at)
			VALUES ($1, gen_random_uuid(), $2, 's', 'a@example.com', 'b', 'finished', 'email', NOW() - MAKE_INTERVAL(secs => $3))
			RETURNING id`, f.tenant, name, ago.Seconds()); err != nil {
			t.Fatalf("inserting campaign: %v", err)
		}
		pool.MustExec(`INSERT INTO campaign_lists (tenant_id, campaign_id, list_id, list_name) VALUES ($1, $2, $3, 'x')`, f.tenant, id, list)
		return id
	}
	before := started("Before joining", news, 20*day)
	opened := started("While on News", news, 5*day)
	started("While on Left", left, 20*day)
	started("After leaving", left, 5*day)
	started("Another list", other, 5*day)
	pool.MustExec(`INSERT INTO campaign_views (tenant_id, campaign_id, subscriber_id) VALUES ($1, $2, $3), ($1, $2, $3)`, f.tenant, opened, sub)

	out, err := c.GetSubscriberActivity(ctx, f.tenant, sub)
	if err != nil {
		t.Fatalf("GetSubscriberActivity: %v", err)
	}
	e := out.Engagement
	if e.Campaigns != 2 || e.Opened != 1 || e.Clicked != 0 || !e.LastOpenedAt.Valid {
		t.Errorf("engagement = %+v, want 2 campaigns (one per list while on it), 1 opened, 0 clicked, a last open", e)
	}

	// Opening one from before they joined means it reached them anyway
	// (say, after a re-import), so opened never exceeds campaigns.
	pool.MustExec(`INSERT INTO campaign_views (tenant_id, campaign_id, subscriber_id) VALUES ($1, $2, $3)`, f.tenant, before, sub)
	out, err = c.GetSubscriberActivity(ctx, f.tenant, sub)
	if err != nil {
		t.Fatalf("GetSubscriberActivity: %v", err)
	}
	if e := out.Engagement; e.Campaigns != 3 || e.Opened != 2 {
		t.Errorf("after opening an earlier campaign: %+v, want 3 campaigns and 2 opened", e)
	}
}
