package core

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/models"
)

func testCoreForOptin(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	qs, err := goyesql.ParseFile("../../queries/subscribers.sql")
	if err != nil {
		t.Fatalf("parsing queries/subscribers.sql: %v", err)
	}
	stmt, err := pool.Preparex(qs["confirm-subscription-optin"].Query)
	if err != nil {
		t.Fatalf("prepare confirm-subscription-optin: %v", err)
	}
	t.Cleanup(func() { stmt.Close() })
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("building i18n: %v", err)
	}
	return &Core{
		db:   pool,
		q:    &models.Queries{ConfirmSubscriptionOptin: stmt},
		i18n: lang,
		log:  log.New(os.Stderr, "", 0),
	}
}

// Confirming an opt-in never confirms lists for a blocklisted subscriber.
// Regression: a blocklisted subscriber who opened an old confirmation link
// ended up blocklisted with confirmed subscriptions.
func TestConfirmOptin_SkipsBlocklistedSubscribers(t *testing.T) {
	pool := testPool(t)
	admin := rlsPool(t, pool)
	pool.MustExec(`GRANT UPDATE (status, meta, updated_at) ON subscriber_lists TO ` + currentUser(t, admin))
	c := testCoreForOptin(t, admin)
	ctx := context.Background()

	f := newListFixture(t, pool)
	news := f.list("News", models.ListOptinDouble)
	var listUUID string
	if err := pool.Get(&listUUID, `SELECT uuid FROM lists WHERE id = $1`, news); err != nil {
		t.Fatal(err)
	}

	status := func(sub int) string {
		t.Helper()
		var s string
		if err := pool.Get(&s, `SELECT status FROM subscriber_lists WHERE subscriber_id = $1 AND list_id = $2`, sub, news); err != nil {
			t.Fatal(err)
		}
		return s
	}
	uuidOf := func(sub int) string {
		t.Helper()
		var u string
		if err := pool.Get(&u, `SELECT uuid FROM subscribers WHERE id = $1`, sub); err != nil {
			t.Fatal(err)
		}
		return u
	}

	ana := f.subscriber("ana@example.com", "enabled")
	ben := f.subscriber("ben@example.com", "blocklisted")
	f.subscribe(ana, news, "unconfirmed", 0, 0)
	f.subscribe(ben, news, "unconfirmed", 0, 0)

	for _, sub := range []int{ana, ben} {
		if err := c.ConfirmOptionSubscription(ctx, f.tenant, uuidOf(sub), []string{listUUID}, nil); err != nil {
			t.Fatal(err)
		}
	}

	if got := status(ana); got != "confirmed" {
		t.Errorf("enabled subscriber: %s, want confirmed", got)
	}
	if got := status(ben); got != "unconfirmed" {
		t.Errorf("blocklisted subscriber: %s, want unconfirmed", got)
	}
}
