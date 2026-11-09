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

// testCoreForCampaignsOverview builds a *Core with the overview statements
// prepared from queries/campaigns.sql itself.
func testCoreForCampaignsOverview(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()

	qs, err := goyesql.ParseFile("../../queries/campaigns.sql")
	if err != nil {
		t.Fatalf("parsing queries/campaigns.sql: %v", err)
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
			SetCampaignEditor:       prep("set-campaign-editor"),
			GetCampaignsOverview:    prep("get-campaigns-overview"),
			GetCampaignStatusCounts: prep("get-campaign-status-counts"),
		},
		i18n: lang,
	}
}

// campaign inserts a campaign in status on list, updated `age` ago.
func (f *listFixture) campaign(name, status string, list int, age time.Duration) int {
	f.t.Helper()
	var id int
	if err := f.db.Get(&id, `
		INSERT INTO campaigns (tenant_id, uuid, name, subject, from_email, body, status, messenger, send_at, updated_at)
		VALUES ($1, gen_random_uuid(), $2, 'subj', 'a@example.com', 'body', $3, 'email', NOW() + MAKE_INTERVAL(secs => $4), NOW() - MAKE_INTERVAL(secs => $4))
		RETURNING id`, f.tenant, name, status, age.Seconds()); err != nil {
		f.t.Fatalf("inserting campaign: %v", err)
	}
	f.db.MustExec(`INSERT INTO campaign_lists (tenant_id, campaign_id, list_id, list_name) SELECT $1, $2, id, name FROM lists WHERE id = $3`,
		f.tenant, id, list)
	return id
}

func TestCampaignsOverview_SectionsLimitsAndPermissions(t *testing.T) {
	pool := testPool(t)
	c := testCoreForCampaignsOverview(t, rlsPool(t, pool))
	f := newListFixture(t, pool)
	ctx := context.Background()
	hour := time.Hour

	news := f.list("News", models.ListOptinSingle)
	secret := f.list("Secret", models.ListOptinSingle)

	running := f.campaign("Running", "running", news, hour)
	f.campaign("Paused", "paused", news, hour)
	// Four drafts; the overview shows the latest three.
	oldest := f.campaign("Draft 1", "draft", news, 4*hour)
	for i, age := range []time.Duration{3 * hour, 2 * hour, hour} {
		f.campaign("Draft "+string(rune('2'+i)), "draft", news, age)
	}
	f.campaign("Scheduled", "scheduled", news, hour)
	hidden := f.campaign("Secret draft", "draft", secret, 0)

	// One person on the scheduled campaign's list.
	f.subscribe(f.subscriber("p@example.com", "enabled"), news, "confirmed", hour, hour)

	if err := c.SetCampaignEditor(ctx, f.tenant, running, 7, "Tawanda"); err != nil {
		t.Fatalf("SetCampaignEditor: %v", err)
	}

	lim := CampaignsOverviewLimits{Scheduled: 10, Drafts: 3, Sent: 4}
	all, err := c.GetCampaignsOverview(ctx, f.tenant, true, nil, lim)
	if err != nil {
		t.Fatalf("GetCampaignsOverview: %v", err)
	}
	byID := map[int]models.CampaignOverviewItem{}
	byStatus := map[string]int{}
	for _, it := range all.Campaigns {
		byID[it.ID] = it
		byStatus[it.Status]++
	}
	if byStatus["running"] != 1 || byStatus["paused"] != 1 || byStatus["scheduled"] != 1 || byStatus["draft"] != 3 {
		t.Errorf("sections = %v, want 1 running, 1 paused, 1 scheduled, 3 drafts", byStatus)
	}
	if _, ok := byID[oldest]; ok {
		t.Errorf("the oldest of 5 drafts is shown, want only the latest 3")
	}
	if got := byID[running]; got.UpdatedByName != "Tawanda" || len(got.Lists) != 1 || got.Lists[0].Name != "News" {
		t.Errorf("running = %+v, want edited by Tawanda, on News", got)
	}
	for _, it := range all.Campaigns {
		if it.Status == "scheduled" && it.Audience != 1 {
			t.Errorf("scheduled audience = %d, want 1", it.Audience)
		}
	}
	counts := map[string]int{}
	for _, sc := range all.Counts {
		counts[sc.Status] = sc.Count
	}
	if counts["draft"] != 5 {
		t.Errorf("draft count = %d, want 5 (all drafts, not just those shown)", counts["draft"])
	}

	// A user who can see only News doesn't see the Secret draft or count it.
	limited, err := c.GetCampaignsOverview(ctx, f.tenant, false, []int{news}, lim)
	if err != nil {
		t.Fatalf("GetCampaignsOverview (limited): %v", err)
	}
	for _, it := range limited.Campaigns {
		if it.ID == hidden {
			t.Errorf("a campaign on a list the user can't see is shown")
		}
	}
	for _, sc := range limited.Counts {
		if sc.Status == "draft" && sc.Count != 4 {
			t.Errorf("limited draft count = %d, want 4 (not the Secret draft)", sc.Count)
		}
	}
}
