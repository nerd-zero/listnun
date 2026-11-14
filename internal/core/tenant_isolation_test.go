package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

// These tests run the shipped SQL as a role without BYPASSRLS, like the
// app's, and check each path only touches the tenant's own rows.

// isolationPool connects as a fresh NOSUPERUSER NOBYPASSRLS role that can
// read everything and write the rows these paths write.
func isolationPool(t *testing.T, admin *sqlx.DB) *sqlx.DB {
	t.Helper()
	role := fmt.Sprintf("isolation_test_%d", time.Now().UnixNano())
	const password = "isolation_test_pw"
	quoted := pq.QuoteIdentifier(role)

	admin.MustExec(`CREATE ROLE ` + quoted + ` LOGIN PASSWORD ` + pq.QuoteLiteral(password) + ` NOSUPERUSER NOBYPASSRLS`)
	admin.MustExec(`GRANT USAGE ON SCHEMA public TO ` + quoted)
	admin.MustExec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO ` + quoted)
	admin.MustExec(`GRANT INSERT, UPDATE ON subscriber_lists TO ` + quoted)
	admin.MustExec(`GRANT UPDATE (last_subscriber_id, updated_at) ON campaigns TO ` + quoted)
	admin.MustExec(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO ` + quoted)

	db, err := sqlx.Connect("postgres", dbDSN(role, password))
	if err != nil {
		t.Fatalf("connecting as test role: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		admin.MustExec(`REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ` + quoted)
		admin.MustExec(`REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM ` + quoted)
		admin.MustExec(`REVOKE USAGE ON SCHEMA public FROM ` + quoted)
		admin.MustExec(`DROP ROLE IF EXISTS ` + quoted)
	})
	return db.Unsafe()
}

// shippedQueries parses queries/<file>.sql.
func shippedQueries(t *testing.T, file string) goyesql.Queries {
	t.Helper()
	qs, err := goyesql.ParseFile("../../queries/" + file)
	if err != nil {
		t.Fatalf("parsing queries/%s: %v", file, err)
	}
	return qs
}

func testCoreForIsolation(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	subs := shippedQueries(t, "subscribers.sql")
	camps := shippedQueries(t, "campaigns.sql")
	prep := func(qs goyesql.Queries, name string) *sqlx.Stmt {
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
			AddSubscribersToLists:     prep(subs, "add-subscribers-to-lists"),
			QuerySubscribersCount:     subs["query-subscribers-count"].Query,
			QuerySubscribersForExport: subs["query-subscribers-for-export"].Query,
			ExportCampaignViews:       prep(camps, "export-campaign-views"),
			ExportCampaignLinkClicks:  prep(camps, "export-campaign-link-clicks"),
			NextCampaignSubscribers:   prep(camps, "next-campaign-subscribers"),
		},
		i18n: lang,
		log:  log.New(io.Discard, "", 0),
	}
}

// Subscriptions are only made between the tenant's own subscribers and lists.
func TestAddSubscriptions_OnlyWithinTheTenant(t *testing.T) {
	admin := testPool(t)
	c := testCoreForIsolation(t, isolationPool(t, admin))
	a, b := newListFixture(t, admin), newListFixture(t, admin)

	subA, listA := a.subscriber("a@example.com", "enabled"), a.list("A", models.ListOptinSingle)
	subB, listB := b.subscriber("b@example.com", "enabled"), b.list("B", models.ListOptinSingle)

	if err := c.AddSubscriptions(context.Background(), b.tenant, []int{subA, subB}, []int{listA, listB}, "confirmed"); err != nil {
		t.Fatalf("AddSubscriptions: %v", err)
	}

	var got []string
	admin.Select(&got, `SELECT subscriber_id || ':' || list_id FROM subscriber_lists WHERE list_id = ANY($1) ORDER BY 1`,
		pq.Array([]int{listA, listB}))
	if want := fmt.Sprintf("%d:%d", subB, listB); len(got) != 1 || got[0] != want {
		t.Errorf("subscriptions = %v, want only tenant B's own %s", got, want)
	}
}

// Exports return only the tenant's own rows.
func TestExports_OnlyTheTenantsOwnRows(t *testing.T) {
	admin := testPool(t)
	c := testCoreForIsolation(t, isolationPool(t, admin))
	a, b := newListFixture(t, admin), newListFixture(t, admin)
	ctx := context.Background()

	subA := a.subscriber("export-a@example.com", "enabled")
	subB := b.subscriber("export-b@example.com", "enabled")

	next, err := c.ExportSubscribers(ctx, a.tenant, "", "subscribers.id > 0", nil, nil, "", 100)
	if err != nil {
		t.Fatalf("ExportSubscribers: %v", err)
	}
	var ids []int
	for {
		batch, err := next()
		if err != nil {
			t.Fatalf("export batch: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		for _, s := range batch {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) != 1 || ids[0] != subA {
		t.Errorf("exported subscribers %v, want only tenant A's %d", ids, subA)
	}

	campA := a.campaign("A", "finished", a.list("A", models.ListOptinSingle), time.Hour)
	campB := b.campaign("B", "finished", b.list("B", models.ListOptinSingle), time.Hour)
	admin.MustExec(`INSERT INTO campaign_views (tenant_id, campaign_id, subscriber_id) VALUES ($1, $2, $3), ($4, $5, $6)`,
		a.tenant, campA, subA, b.tenant, campB, subB)

	views, err := c.ExportCampaignViews(ctx, a.tenant, time.Now().Add(-time.Hour), 100)()
	if err != nil {
		t.Fatalf("ExportCampaignViews: %v", err)
	}
	if len(views) != 1 || views[0].CampaignID != campA {
		t.Errorf("exported views %+v, want only tenant A's campaign %d", views, campA)
	}
}

// The campaign manager runs without a tenant set; recipients still come
// only from the campaign's own tenant.
func TestNextCampaignSubscribers_OnlyTheCampaignsTenant(t *testing.T) {
	admin := testPool(t)
	c := testCoreForIsolation(t, isolationPool(t, admin))
	a, b := newListFixture(t, admin), newListFixture(t, admin)

	listA := a.list("A", models.ListOptinSingle)
	camp := a.campaign("A", "running", listA, time.Hour)
	own := a.subscriber("own@example.com", "enabled")
	other := b.subscriber("other@example.com", "enabled")
	a.subscribe(own, listA, "confirmed", time.Hour, time.Hour)
	// A stray row: tenant B's subscriber on tenant A's list.
	admin.MustExec(`INSERT INTO subscriber_lists (tenant_id, subscriber_id, list_id, status) VALUES ($1, $2, $3, 'confirmed')`,
		a.tenant, other, listA)

	var got []models.Subscriber
	if err := c.q.NextCampaignSubscribers.Select(&got, camp, "regular", 0, other+own, pq.Array([]int{listA}), 100); err != nil {
		t.Fatalf("next-campaign-subscribers: %v", err)
	}
	if len(got) != 1 || got[0].ID != own {
		t.Errorf("recipients = %+v, want only tenant A's subscriber %d", got, own)
	}
}

// New tenants start from install defaults for credentials and custom code,
// and from tenant 1 for general preferences.
func TestOperatorSeed_NoTenantOneSecrets(t *testing.T) {
	admin := testPool(t)
	seed := shippedQueries(t, "operator.sql")["operator-seed-tenant-settings"].Query

	tx := admin.MustBegin()
	defer tx.Rollback()

	tx.MustExec(`UPDATE settings SET value = '[{"enabled": true, "host": "smtp.tenant1", "password": "tenant-1-secret"}]' WHERE tenant_id = 1 AND key = 'smtp'`)
	tx.MustExec(`UPDATE settings SET value = '"alert(1)"' WHERE tenant_id = 1 AND key = 'appearance.admin.custom_js'`)
	tx.MustExec(`UPDATE settings SET value = '7' WHERE tenant_id = 1 AND key = 'app.concurrency'`)

	var tenant int
	if err := tx.Get(&tenant, `INSERT INTO tenants (slug, name) VALUES ($1, 'Seed test') RETURNING id`,
		fmt.Sprintf("seed-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting tenant: %v", err)
	}
	if _, err := tx.Exec(seed, tenant); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	val := func(key string) string {
		var v string
		if err := tx.Get(&v, `SELECT value::TEXT FROM settings WHERE tenant_id = $1 AND key = $2`, tenant, key); err != nil {
			t.Fatalf("reading %s: %v", key, err)
		}
		return v
	}
	if v := val("smtp"); v == "" || strings.Contains(v, "tenant-1-secret") {
		t.Errorf("smtp = %s, want the install default, not tenant 1's", v)
	}
	if v := val("appearance.admin.custom_js"); v != `""` {
		t.Errorf("admin custom JS = %s, want empty", v)
	}
	if v := val("app.concurrency"); v != "7" {
		t.Errorf("concurrency = %s, want tenant 1's 7 (preferences are still copied)", v)
	}
	var keys1, keysNew int
	tx.Get(&keys1, `SELECT COUNT(*) FROM settings WHERE tenant_id = 1 AND key NOT IN ('app.root_url', 'replies')`)
	tx.Get(&keysNew, `SELECT COUNT(*) FROM settings WHERE tenant_id = $1`, tenant)
	if keysNew < keys1 {
		t.Errorf("new tenant has %d settings, tenant 1 %d: some keys weren't seeded", keysNew, keys1)
	}
}
