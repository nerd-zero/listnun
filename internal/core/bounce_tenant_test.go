package core

import (
	"testing"
	"time"

	"github.com/knadh/goyesql/v2"
	"github.com/nerd-zero/listnun/models"
)

// A bounce from the POP3 mailbox has no tenant. When the bounced address was
// a subscriber in two workspaces, record-bounce failed with "more than one
// row returned by a subquery" and the bounce was lost. It now uses the
// bounced campaign's workspace, and records nothing if it's still
// ambiguous rather than touching another workspace's subscriber.
func TestRecordBounce_AddressInTwoWorkspaces(t *testing.T) {
	admin := testPool(t)
	qs, err := goyesql.ParseFile("../../queries/bounces.sql")
	if err != nil {
		t.Fatalf("parsing queries/bounces.sql: %v", err)
	}
	recordBounce := qs["record-bounce"].Query
	a, b := newListFixture(t, admin), newListFixture(t, admin)

	const email = "shared@example.com"
	subA := a.subscriber(email, "enabled")
	b.subscriber(email, "enabled")
	camp := a.campaign("A", "finished", a.list("A", models.ListOptinSingle), time.Hour)
	var campUUID string
	admin.Get(&campUUID, `SELECT uuid::TEXT FROM campaigns WHERE id = $1`, camp)

	bounce := func(campaignUUID string) error {
		_, err := admin.Exec(recordBounce, "", email, campaignUUID, "hard", "pop3", "{}", time.Now(), 1, "none", 0)
		return err
	}

	if err := bounce(campUUID); err != nil {
		t.Fatalf("bounce with campaign: %v", err)
	}
	var got []int
	admin.Select(&got, `SELECT subscriber_id FROM bounces WHERE campaign_id = $1`, camp)
	if len(got) != 1 || got[0] != subA {
		t.Errorf("bounced subscribers = %v, want workspace A's %d", got, subA)
	}

	// No campaign: ambiguous, so nothing is recorded. The insert then fails
	// on its NOT NULL tenant, which RecordBounce treats as "not found".
	before := len(got)
	_ = bounce("")
	var n int
	admin.Get(&n, `SELECT COUNT(*) FROM bounces WHERE tenant_id = ANY(ARRAY[$1, $2]::INT[])`, a.tenant, b.tenant)
	if n != before {
		t.Errorf("bounces = %d after an ambiguous bounce, want %d (none added)", n, before)
	}
}
