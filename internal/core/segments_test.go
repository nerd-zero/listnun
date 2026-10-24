package core

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/internal/segments"
	"github.com/nerd-zero/listnun/models"
)

// testCoreForSegments builds a *Core with the subscriber count query and
// the segment statements from queries/*.sql.
func testCoreForSegments(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	load := func(file string) goyesql.Queries {
		qs, err := goyesql.ParseFile("../../queries/" + file)
		if err != nil {
			t.Fatalf("parsing queries/%s: %v", file, err)
		}
		return qs
	}
	subs, segs := load("subscribers.sql"), load("segments.sql")
	prep := func(q string) *sqlx.Stmt {
		stmt, err := pool.Preparex(q)
		if err != nil {
			t.Fatalf("prepare: %v", err)
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
			QuerySubscribersCount: subs["query-subscribers-count"].Query,
			GetSegments:           prep(segs["get-segments"].Query),
			CreateSegment:         prep(segs["create-segment"].Query),
			UpdateSegment:         prep(segs["update-segment"].Query),
			DeleteSegment:         prep(segs["delete-segment"].Query),
		},
		i18n: lang,
	}
}

func TestSegments_EveryConditionRunsAndCounts(t *testing.T) {
	pool := testPool(t)
	c := testCoreForSegments(t, rlsPool(t, pool))
	f := newListFixture(t, pool)
	ctx := context.Background()
	day := 24 * time.Hour

	news := f.list("News", models.ListOptinDouble)
	harare := f.subscriber("ana@example.com", "enabled")
	pool.MustExec(`UPDATE subscribers SET attribs = '{"city": "Harare"}', created_at = NOW() - INTERVAL '10 days' WHERE id = $1`, harare)
	f.subscribe(harare, news, "confirmed", 10*day, 10*day)
	waiting := f.subscriber("tendai@example.com", "enabled")
	pool.MustExec(`UPDATE subscribers SET created_at = NOW() - INTERVAL '200 days' WHERE id = $1`, waiting)
	f.subscribe(waiting, news, "unconfirmed", 200*day, 200*day)
	f.subscriber("blocked@example.com", "blocklisted")

	camp := f.campaign("Digest", "finished", news, day)
	pool.MustExec(`INSERT INTO campaign_views (tenant_id, campaign_id, subscriber_id) VALUES ($1, $2, $3)`, f.tenant, camp, harare)

	list := strconv.Itoa(news)
	for _, tc := range []struct {
		name string
		cond segments.Condition
		want int
	}{
		{"attribute is", segments.Condition{Field: segments.FieldAttribute, Key: "city", Op: segments.OpIs, Value: "Harare"}, 1},
		{"attribute exists", segments.Condition{Field: segments.FieldAttribute, Key: "city", Op: segments.OpExists}, 1},
		{"email contains", segments.Condition{Field: segments.FieldEmail, Op: segments.OpContains, Value: "@example"}, 3},
		{"name is not", segments.Condition{Field: segments.FieldName, Op: segments.OpIsNot, Value: "ana@example.com"}, 2},
		{"joined in the last 30 days", segments.Condition{Field: segments.FieldJoined, Op: segments.OpInLast, Value: "30"}, 2},
		{"joined before", segments.Condition{Field: segments.FieldJoined, Op: segments.OpBefore, Value: time.Now().Add(-100 * day).Format(time.DateOnly)}, 1},
		{"on list", segments.Condition{Field: segments.FieldList, Op: segments.OpIs, Value: list}, 2},
		{"not on list", segments.Condition{Field: segments.FieldList, Op: segments.OpIsNot, Value: list}, 1},
		{"blocklisted", segments.Condition{Field: segments.FieldStatus, Op: segments.OpIs, Value: "blocklisted"}, 1},
		{"not confirmed yet", segments.Condition{Field: segments.FieldUnconfirmed, Op: segments.OpIs}, 1},
		{"opened in the last 90 days", segments.Condition{Field: segments.FieldLastOpened, Op: segments.OpInLast, Value: "90"}, 1},
		{"never opened", segments.Condition{Field: segments.FieldLastOpened, Op: segments.OpNever}, 2},
		{"not clicked in 90 days", segments.Condition{Field: segments.FieldLastClicked, Op: segments.OpNotInLast, Value: "90"}, 3},
	} {
		n, err := c.CountSegment(ctx, f.tenant, segments.Segment{Conditions: []segments.Condition{tc.cond}}, nil)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if n != tc.want {
			t.Errorf("%s: %d people, want %d", tc.name, n, tc.want)
		}
	}

	// Any of: Harare, or blocklisted.
	n, err := c.CountSegment(ctx, f.tenant, segments.Segment{Match: segments.MatchAny, Conditions: []segments.Condition{
		{Field: segments.FieldAttribute, Key: "city", Op: segments.OpIs, Value: "Harare"},
		{Field: segments.FieldStatus, Op: segments.OpIs, Value: "blocklisted"},
	}}, nil)
	if err != nil || n != 2 {
		t.Errorf("match any: %d people (err %v), want 2", n, err)
	}

	// An invalid segment is a 400, not a database error.
	_, err = c.CountSegment(ctx, f.tenant, segments.Segment{Conditions: []segments.Condition{{Field: "password", Op: segments.OpIs}}}, nil)
	var he *echo.HTTPError
	if !errors.As(err, &he) || he.Code != http.StatusBadRequest {
		t.Errorf("invalid segment: err = %v, want a 400", err)
	}
}

func TestSegments_SaveUpdateDelete(t *testing.T) {
	pool := testPool(t)
	admin := rlsPool(t, pool)
	pool.MustExec(`GRANT INSERT, UPDATE, DELETE ON segments TO ` + currentUser(t, admin))
	pool.MustExec(`GRANT USAGE ON SEQUENCE segments_id_seq TO ` + currentUser(t, admin))
	c := testCoreForSegments(t, admin)
	f := newListFixture(t, pool)
	ctx := context.Background()

	saved, err := c.CreateSegment(ctx, f.tenant, models.SavedSegment{
		Name:       "Harare, joined recently",
		Conditions: models.SegmentConditions{{Field: segments.FieldAttribute, Key: "city", Op: segments.OpIs, Value: "Harare"}},
	}, "Tawanda")
	if err != nil {
		t.Fatalf("CreateSegment: %v", err)
	}
	if saved.Match != segments.MatchAll || saved.CreatedByName != "Tawanda" || len(saved.Conditions) != 1 {
		t.Errorf("saved = %+v, want match all, by Tawanda, 1 condition", saved)
	}

	// Conditions that don't compile aren't saved.
	if _, err := c.CreateSegment(ctx, f.tenant, models.SavedSegment{
		Name: "Bad", Conditions: models.SegmentConditions{{Field: segments.FieldJoined, Op: segments.OpInLast, Value: "soon"}},
	}, ""); err == nil {
		t.Error("saved a segment whose conditions don't compile")
	}

	updated, err := c.UpdateSegment(ctx, f.tenant, saved.ID, models.SavedSegment{Name: "Everyone in Harare", Match: segments.MatchAny, Conditions: saved.Conditions})
	if err != nil || updated.Name != "Everyone in Harare" || updated.Match != segments.MatchAny {
		t.Errorf("UpdateSegment = %+v, %v", updated, err)
	}

	if err := c.DeleteSegment(ctx, f.tenant, saved.ID); err != nil {
		t.Fatalf("DeleteSegment: %v", err)
	}
	if all, _ := c.GetSegments(ctx, f.tenant); len(all) != 0 {
		t.Errorf("segments after delete = %v, want none", all)
	}
}

// currentUser is the role db connects as, quoted.
func currentUser(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	var u string
	if err := db.Get(&u, `SELECT quote_ident(current_user)`); err != nil {
		t.Fatalf("current_user: %v", err)
	}
	return u
}
