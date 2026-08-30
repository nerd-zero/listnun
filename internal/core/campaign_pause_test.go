package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// TestSetCampaignPauseReason_RaceGuard verifies set-campaign-pause-reason's
// "WHERE status='paused' AND pause_reason IS NULL" guard: once a reason is
// set, a second auto-pause trigger racing the first must not overwrite it
// (or a manual pause's reason, which is always NULL and would otherwise
// get silently stomped by a concurrent auto-pause).
func TestSetCampaignPauseReason_RaceGuard(t *testing.T) {
	db, err := sqlx.Connect("postgres", adminDSN())
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping: test database not reachable: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var tenantID int
	if err := db.Get(&tenantID, `INSERT INTO tenants (slug, name) VALUES ($1, 'Pause Reason Test') RETURNING id`,
		fmt.Sprintf("pause-reason-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting test tenant: %v", err)
	}
	t.Cleanup(func() {
		db.MustExec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	var campID int
	if err := db.Get(&campID, `
		INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger)
		VALUES ($1, gen_random_uuid(), 'Pause reason test', 'subj', 'a@example.com', 'body', 'running', 'email')
		RETURNING id`, tenantID); err != nil {
		t.Fatalf("inserting test campaign: %v", err)
	}

	// Simulate the auto-pause sequence: UpdateCampaignStatus(paused) then
	// SetCampaignPauseReason -- exactly what pauseCampaignsForRiskySubscriberWith
	// (cmd/scrub_validation.go) and pipe.go's too-many-errors path both do.
	db.MustExec(`UPDATE campaigns SET status = 'paused' WHERE id = $1`, campID)

	res1, err := db.Exec(`UPDATE campaigns SET pause_reason = $2 WHERE id = $1 AND status = 'paused' AND pause_reason IS NULL`,
		campID, "scrub_risky_subscriber")
	if err != nil {
		t.Fatalf("first SetCampaignPauseReason: %v", err)
	}
	if n, _ := res1.RowsAffected(); n != 1 {
		t.Fatalf("first call: RowsAffected = %d, want 1 (should set the reason since it was NULL)", n)
	}

	// A second, racing auto-pause trigger for the same campaign must not
	// overwrite the first reason.
	res2, err := db.Exec(`UPDATE campaigns SET pause_reason = $2 WHERE id = $1 AND status = 'paused' AND pause_reason IS NULL`,
		campID, "too_many_errors")
	if err != nil {
		t.Fatalf("second SetCampaignPauseReason: %v", err)
	}
	if n, _ := res2.RowsAffected(); n != 0 {
		t.Fatalf("second call: RowsAffected = %d, want 0 (must not overwrite an already-set reason)", n)
	}

	var reason string
	if err := db.Get(&reason, `SELECT pause_reason FROM campaigns WHERE id = $1`, campID); err != nil {
		t.Fatalf("reading back pause_reason: %v", err)
	}
	if reason != "scrub_risky_subscriber" {
		t.Errorf("pause_reason = %q, want %q (the first writer's reason must win)", reason, "scrub_risky_subscriber")
	}
}

// TestGetRunningCampaignsByList verifies the query only returns campaigns
// that are actually 'running' and actually target one of the given lists
// -- not draft/paused/finished campaigns, and not campaigns on unrelated
// lists.
func TestGetRunningCampaignsByList(t *testing.T) {
	db, err := sqlx.Connect("postgres", adminDSN())
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping: test database not reachable: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var tenantID int
	if err := db.Get(&tenantID, `INSERT INTO tenants (slug, name) VALUES ($1, 'Running Campaigns Test') RETURNING id`,
		fmt.Sprintf("running-campaigns-test-%d", time.Now().UnixNano())); err != nil {
		t.Fatalf("inserting test tenant: %v", err)
	}
	t.Cleanup(func() {
		db.MustExec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	var targetList, otherList int
	if err := db.Get(&targetList, `INSERT INTO lists (tenant_id, uuid, name, type, optin) VALUES ($1, gen_random_uuid(), 'target', 'private', 'single') RETURNING id`, tenantID); err != nil {
		t.Fatalf("inserting target list: %v", err)
	}
	if err := db.Get(&otherList, `INSERT INTO lists (tenant_id, uuid, name, type, optin) VALUES ($1, gen_random_uuid(), 'other', 'private', 'single') RETURNING id`, tenantID); err != nil {
		t.Fatalf("inserting other list: %v", err)
	}

	newCampaign := func(status string) int {
		var id int
		if err := db.Get(&id, `
			INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger)
			VALUES ($1, gen_random_uuid(), $2, 'subj', 'a@example.com', 'body', $3, 'email')
			RETURNING id`, tenantID, status, status); err != nil {
			t.Fatalf("inserting %s campaign: %v", status, err)
		}
		return id
	}

	running := newCampaign("running")
	draft := newCampaign("draft")
	pausedAlready := newCampaign("paused")
	runningOtherList := newCampaign("running")

	db.MustExec(`INSERT INTO campaign_lists (campaign_id, tenant_id, list_id, list_name) VALUES ($1, $2, $3, 'target')`, running, tenantID, targetList)
	db.MustExec(`INSERT INTO campaign_lists (campaign_id, tenant_id, list_id, list_name) VALUES ($1, $2, $3, 'target')`, draft, tenantID, targetList)
	db.MustExec(`INSERT INTO campaign_lists (campaign_id, tenant_id, list_id, list_name) VALUES ($1, $2, $3, 'target')`, pausedAlready, tenantID, targetList)
	db.MustExec(`INSERT INTO campaign_lists (campaign_id, tenant_id, list_id, list_name) VALUES ($1, $2, $3, 'other')`, runningOtherList, tenantID, otherList)

	var got []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	if err := db.Select(&got, `
		SELECT DISTINCT campaigns.id, campaigns.name FROM campaigns
		JOIN campaign_lists ON campaign_lists.campaign_id = campaigns.id
		WHERE campaigns.tenant_id = $1 AND campaigns.status = 'running' AND campaign_lists.list_id = ANY($2::INT[])`,
		tenantID, pq.Array([]int{targetList})); err != nil {
		t.Fatalf("get-running-campaigns-by-list: %v", err)
	}

	if len(got) != 1 || got[0].ID != running {
		t.Errorf("got %+v, want exactly [{%d ...}] (only the running campaign on the target list)", got, running)
	}
}
