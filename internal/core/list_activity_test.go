package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

// testCoreForListActivity builds a *Core with the list activity and opt-in
// statements prepared from queries/lists.sql itself, so the tests run the
// shipped SQL.
func testCoreForListActivity(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()

	qs, err := goyesql.ParseFile("../../queries/lists.sql")
	if err != nil {
		t.Fatalf("parsing queries/lists.sql: %v", err)
	}
	prep := func(name string) *sqlx.Stmt {
		q, ok := qs[name]
		if !ok {
			t.Fatalf("query %q not in queries/lists.sql", name)
		}
		stmt, err := pool.Preparex(q.Query)
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
			GetListsGrowth:         prep("get-lists-growth"),
			GetListActivity:        prep("get-list-activity"),
			GetListRecentCampaigns: prep("get-list-recent-campaigns"),
			GetOptinPending:        prep("get-optin-pending"),
			CountOptinPending:      prep("count-optin-pending"),
			MarkOptinSent:          prep("mark-optin-sent"),
		},
		i18n: lang,
	}
}

// rlsPool connects as a throwaway role that, unlike the usual superuser
// test role, is subject to row-level security, so the queries only see
// the tenant WithTenant sets.
func rlsPool(t *testing.T, admin *sqlx.DB) *sqlx.DB {
	t.Helper()
	role := fmt.Sprintf("list_activity_test_%d", time.Now().UnixNano())
	const password = "list_activity_test_pw"
	quoted := pq.QuoteIdentifier(role)

	admin.MustExec(`CREATE ROLE ` + quoted + ` LOGIN PASSWORD ` + pq.QuoteLiteral(password) + ` NOSUPERUSER NOBYPASSRLS`)
	admin.MustExec(`GRANT USAGE ON SCHEMA public TO ` + quoted)
	admin.MustExec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO ` + quoted)
	admin.MustExec(`GRANT UPDATE (optin_sent_at) ON subscribers TO ` + quoted)
	admin.MustExec(`GRANT UPDATE (updated_by, updated_by_name) ON campaigns TO ` + quoted)

	db, err := sqlx.Connect("postgres", dbDSN(role, password))
	if err != nil {
		t.Fatalf("connecting as test role: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		admin.MustExec(`REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ` + quoted)
		admin.MustExec(`REVOKE USAGE ON SCHEMA public FROM ` + quoted)
		admin.MustExec(`DROP ROLE IF EXISTS ` + quoted)
	})
	// Unsafe, like the app's pool (cmd/init.go): models don't map every
	// column a query returns.
	return db.Unsafe()
}

// listFixture inserts rows for one test tenant, deleted with it.
type listFixture struct {
	t      *testing.T
	db     *sqlx.DB
	tenant int
}

func newListFixture(t *testing.T, db *sqlx.DB) *listFixture {
	t.Helper()
	f := &listFixture{t: t, db: db}
	if err := db.Get(&f.tenant, `INSERT INTO tenants (slug, name) VALUES ($1, 'List activity test') RETURNING id`,
		fmt.Sprintf("list-activity-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting test tenant: %v", err)
	}
	t.Cleanup(func() { db.MustExec(`DELETE FROM tenants WHERE id = $1`, f.tenant) })
	return f
}

func (f *listFixture) list(name, optin string) int {
	f.t.Helper()
	var id int
	if err := f.db.Get(&id, `INSERT INTO lists (tenant_id, uuid, name, type, optin) VALUES ($1, gen_random_uuid(), $2, 'public', $3) RETURNING id`,
		f.tenant, name, optin); err != nil {
		f.t.Fatalf("inserting list: %v", err)
	}
	return id
}

func (f *listFixture) subscriber(email, status string) int {
	f.t.Helper()
	var id int
	if err := f.db.Get(&id, `INSERT INTO subscribers (tenant_id, uuid, email, name, status) VALUES ($1, gen_random_uuid(), $2, $2, $3) RETURNING id`,
		f.tenant, email, status); err != nil {
		f.t.Fatalf("inserting subscriber: %v", err)
	}
	return id
}

// subscribe adds a subscription created `added` ago and last changed
// `changed` ago.
func (f *listFixture) subscribe(sub, list int, status string, added, changed time.Duration) {
	f.t.Helper()
	now := time.Now()
	f.db.MustExec(`INSERT INTO subscriber_lists (tenant_id, subscriber_id, list_id, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		f.tenant, sub, list, status, now.Add(-added), now.Add(-changed))
}

func TestOptinResend_OnlyRemindsPeopleWhoAreDue(t *testing.T) {
	pool := testPool(t)
	c := testCoreForListActivity(t, rlsPool(t, pool))
	f := newListFixture(t, pool)
	ctx := context.Background()

	double := f.list("Double", models.ListOptinDouble)
	single := f.list("Single", models.ListOptinSingle)
	day := 24 * time.Hour

	due := f.subscriber("due@example.com", "enabled")
	f.subscribe(due, double, "unconfirmed", 3*day, 3*day)

	// Signed up an hour ago, so the signup's own confirmation is recent.
	justJoined := f.subscriber("new@example.com", "enabled")
	f.subscribe(justJoined, double, "unconfirmed", time.Hour, time.Hour)

	reminded := f.subscriber("reminded@example.com", "enabled")
	f.subscribe(reminded, double, "unconfirmed", 3*day, 3*day)
	pool.MustExec(`UPDATE subscribers SET optin_sent_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, reminded)

	// Not waiting on a double opt-in confirmation at all.
	f.subscribe(f.subscriber("single@example.com", "enabled"), single, "unconfirmed", 3*day, 3*day)
	f.subscribe(f.subscriber("confirmed@example.com", "enabled"), double, "confirmed", 3*day, 3*day)
	f.subscribe(f.subscriber("blocked@example.com", "blocklisted"), double, "unconfirmed", 3*day, 3*day)

	dueBefore := time.Now().Add(-day)
	n, err := c.CountOptinPending(ctx, f.tenant, nil, dueBefore)
	if err != nil {
		t.Fatalf("CountOptinPending: %v", err)
	}
	if n.Total != 3 || n.Due != 1 {
		t.Errorf("pending = %+v, want total 3 (due, just joined, reminded) and due 1", n)
	}

	subs, err := c.GetOptinDue(ctx, f.tenant, nil, dueBefore, 100)
	if err != nil {
		t.Fatalf("GetOptinDue: %v", err)
	}
	if len(subs) != 1 || subs[0].ID != due {
		t.Fatalf("due subscribers = %v, want only %d", subs, due)
	}

	// Once reminded, a second resend skips them.
	if err := c.MarkOptinSent(ctx, f.tenant, due); err != nil {
		t.Fatalf("MarkOptinSent: %v", err)
	}
	if n, _ := c.CountOptinPending(ctx, f.tenant, nil, dueBefore); n.Due != 0 {
		t.Errorf("due after reminding = %d, want 0", n.Due)
	}

	// Limited to the single opt-in list, nobody is waiting.
	if n, _ := c.CountOptinPending(ctx, f.tenant, []int{single}, dueBefore); n.Total != 0 {
		t.Errorf("pending on the single opt-in list = %d, want 0", n.Total)
	}
}

func TestListActivity_CountsJoinsLeavesAndOpens(t *testing.T) {
	pool := testPool(t)
	c := testCoreForListActivity(t, rlsPool(t, pool))
	f := newListFixture(t, pool)
	ctx := context.Background()
	day := 24 * time.Hour

	list := f.list("Weekly", models.ListOptinSingle)
	other := f.list("Other", models.ListOptinSingle)

	joined := f.subscriber("joined@example.com", "enabled")
	f.subscribe(joined, list, "confirmed", 2*day, 2*day)
	left := f.subscriber("left@example.com", "enabled")
	f.subscribe(left, list, "unsubscribed", 40*day, 5*day)
	f.subscribe(f.subscriber("old@example.com", "enabled"), list, "unsubscribed", 60*day, 40*day)
	f.subscribe(f.subscriber("elsewhere@example.com", "enabled"), other, "confirmed", day, day)

	since := time.Now().Add(-30 * day)
	growth, err := c.GetListsGrowth(ctx, f.tenant, since)
	if err != nil {
		t.Fatalf("GetListsGrowth: %v", err)
	}
	byList := map[int]models.ListGrowth{}
	for _, g := range growth {
		byList[g.ListID] = g
	}
	if g := byList[list]; g.Joined != 1 || g.Left != 1 {
		t.Errorf("growth = %+v, want 1 joined and 1 left (the 40-day-old leave is outside the window)", g)
	}

	// A sent campaign opened twice by the same person counts one opener.
	var camp int
	if err := pool.Get(&camp, `
		INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger, sent, started_at)
		VALUES ($1, gen_random_uuid(), 'Digest', 'subj', 'a@example.com', 'body', 'finished', 'email', 2, NOW() - INTERVAL '3 days')
		RETURNING id`, f.tenant); err != nil {
		t.Fatalf("inserting campaign: %v", err)
	}
	pool.MustExec(`INSERT INTO campaign_lists (tenant_id, campaign_id, list_id, list_name) VALUES ($1, $2, $3, 'Weekly')`, f.tenant, camp, list)
	pool.MustExec(`INSERT INTO campaign_views (tenant_id, campaign_id, subscriber_id) VALUES ($1, $2, $3), ($1, $2, $3)`, f.tenant, camp, joined)

	act, err := c.GetListActivity(ctx, f.tenant, list, since, 4)
	if err != nil {
		t.Fatalf("GetListActivity: %v", err)
	}
	if len(act.Days) != 31 || act.Joined != 1 || act.Left != 1 {
		t.Errorf("activity = %d days, %d joined, %d left; want 31 days, 1 joined, 1 left", len(act.Days), act.Joined, act.Left)
	}
	if len(act.Campaigns) != 1 || act.Campaigns[0].ID != camp || act.Campaigns[0].Opened != 1 {
		t.Errorf("campaigns = %+v, want the digest with 1 opener", act.Campaigns)
	}
}

// Regression: query-lists cut the page before sorting, so each page was
// an arbitrary slice sorted on its own, and "Most subscribers" or paging
// by name came out in the wrong order.
func TestQueryLists_SortsBeforePaging(t *testing.T) {
	pool := testPool(t)
	f := newListFixture(t, pool)
	ctx := context.Background()

	qs, err := goyesql.ParseFile("../../queries/lists.sql")
	if err != nil {
		t.Fatalf("parsing queries/lists.sql: %v", err)
	}
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("building i18n: %v", err)
	}
	// CacheSlowQueries skips the stats refresh, which the RLS role may
	// not run; the test refreshes them itself.
	c := &Core{db: rlsPool(t, pool), q: &models.Queries{QueryLists: qs["query-lists"].Query}, i18n: lang,
		consts: Constants{CacheSlowQueries: true}, log: log.New(os.Stderr, "", 0)}

	// Lists created in the opposite order to their names and sizes.
	sizes := map[string]int{"Delta": 1, "Charlie": 2, "Bravo": 3, "Alpha": 4}
	for _, name := range []string{"Delta", "Charlie", "Bravo", "Alpha"} {
		id := f.list(name, models.ListOptinSingle)
		for i := 0; i < sizes[name]; i++ {
			f.subscribe(f.subscriber(fmt.Sprintf("%s%d@example.com", name, i), "enabled"), id, "confirmed", time.Hour, time.Hour)
		}
	}
	pool.MustExec(`REFRESH MATERIALIZED VIEW mat_list_subscriber_stats`)

	names := func(orderBy, order string, offset int) string {
		t.Helper()
		ls, _, err := c.QueryLists(ctx, f.tenant, "", "", "", "", nil, orderBy, order, true, nil, offset, 2)
		if err != nil {
			t.Fatalf("QueryLists: %v", err)
		}
		out := ""
		for _, l := range ls {
			out += l.Name + " "
		}
		return out
	}

	if got := names("name", "asc", 0) + names("name", "asc", 2); got != "Alpha Bravo Charlie Delta " {
		t.Errorf("by name, two pages = %q", got)
	}
	if got := names("subscriber_count", "desc", 0); got != "Alpha Bravo " {
		t.Errorf("most subscribers, first page = %q, want Alpha Bravo", got)
	}
}
