package core

import (
	"context"
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

func TestFormSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Newsletter signup":      "newsletter-signup",
		"  Harare events! 2026 ": "harare-events-2026",
		"!!!":                    "form",
	} {
		if got := FormSlug(in); got != want {
			t.Errorf("FormSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func testCoreForForms(t *testing.T, pool *sqlx.DB) *Core {
	t.Helper()
	qs, err := goyesql.ParseFile("../../queries/forms.sql")
	if err != nil {
		t.Fatalf("parsing queries/forms.sql: %v", err)
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
			GetForms:       prep("get-forms"),
			GetForm:        prep("get-form"),
			GetFormBySlug:  prep("get-form-by-slug"),
			FormSlugExists: prep("form-slug-exists"),
			CreateForm:     prep("create-form"),
			UpdateForm:     prep("update-form"),
			DeleteForm:     prep("delete-form"),
			AddFormEvent:   prep("add-form-event"),
			GetFormStats:   prep("get-form-stats"),
		},
		i18n: lang,
		log:  log.New(os.Stderr, "", 0),
	}
}

// Two forms with the same name get different addresses, each workspace has
// its own addresses, and a form's figures count only its own events.
func TestForms_SlugsAndStats(t *testing.T) {
	pool := testPool(t)
	admin := rlsPool(t, pool)
	user := currentUser(t, admin)
	for _, q := range []string{
		`GRANT INSERT, UPDATE, DELETE ON forms TO ` + user,
		`GRANT INSERT ON form_events TO ` + user,
		`GRANT USAGE ON SEQUENCE forms_id_seq TO ` + user,
		`GRANT USAGE ON SEQUENCE form_events_id_seq TO ` + user,
	} {
		pool.MustExec(q)
	}
	c := testCoreForForms(t, admin)
	ctx := context.Background()

	mine, theirs := newListFixture(t, pool), newListFixture(t, pool)
	news := mine.list("News", models.ListOptinDouble)
	fields := models.FormFields{{Key: models.FormFieldEmail, Label: "Email", Required: true}}
	newForm := func(tenant int, name string) models.Form {
		t.Helper()
		f, err := c.CreateForm(ctx, tenant, models.Form{
			Name: name, Status: models.FormStatusLive, ListIDs: pq.Int64Array{int64(news)}, Fields: fields, Captcha: true,
		})
		if err != nil {
			t.Fatalf("CreateForm: %v", err)
		}
		return f
	}

	a, b := newForm(mine.tenant, "Newsletter signup"), newForm(mine.tenant, "Newsletter signup")
	other := newForm(theirs.tenant, "Newsletter signup")
	if a.Slug != "newsletter-signup" || b.Slug != "newsletter-signup-2" || other.Slug != "newsletter-signup" {
		t.Errorf("slugs = %q, %q, other workspace %q", a.Slug, b.Slug, other.Slug)
	}

	got, err := c.GetFormBySlug(ctx, mine.tenant, "newsletter-signup")
	if err != nil || got.ID != a.ID || len(got.Fields) != 1 {
		t.Fatalf("GetFormBySlug = %+v, %v", got, err)
	}

	// Ana confirmed, Tendai hasn't yet; one view, one bot.
	ana := mine.subscriber("ana@example.com", "enabled")
	mine.subscribe(ana, news, "confirmed", time.Hour, time.Hour)
	tendai := mine.subscriber("tendai@example.com", "enabled")
	mine.subscribe(tendai, news, "unconfirmed", time.Hour, time.Hour)
	for _, e := range []struct {
		kind string
		sub  int
	}{{models.FormEventView, 0}, {models.FormEventSignup, ana}, {models.FormEventSignup, tendai}, {models.FormEventBlocked, 0}} {
		if err := c.AddFormEvent(ctx, mine.tenant, a.ID, e.kind, e.sub); err != nil {
			t.Fatalf("AddFormEvent: %v", err)
		}
	}
	if err := c.AddFormEvent(ctx, mine.tenant, b.ID, models.FormEventSignup, ana); err != nil {
		t.Fatal(err)
	}

	stats, err := c.GetFormStats(ctx, mine.tenant, a.ID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if want := (models.FormStats{Views: 1, Signups: 2, Confirmed: 1, Blocked: 1}); stats != want {
		t.Errorf("stats = %+v, want %+v", stats, want)
	}

	forms, err := c.GetForms(ctx, mine.tenant)
	if err != nil || len(forms) != 2 || forms[0].Signups != 2 || forms[1].Signups != 1 {
		t.Errorf("GetForms = %+v, %v; want 2 forms with 2 and 1 signups", forms, err)
	}

	// Renaming a live form keeps its address.
	a.Name = "Weekly digest"
	upd, err := c.UpdateForm(ctx, mine.tenant, a.ID, a)
	if err != nil || upd.Slug != "newsletter-signup" || upd.Name != "Weekly digest" {
		t.Errorf("UpdateForm = %+v, %v", upd, err)
	}

	// A draft's address follows its name until it's published.
	d, err := c.CreateForm(ctx, mine.tenant, models.Form{Name: "New signup form", Status: models.FormStatusDraft, ListIDs: pq.Int64Array{}, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	d.Name, d.Status = "Harare events", models.FormStatusLive
	if d, err = c.UpdateForm(ctx, mine.tenant, d.ID, d); err != nil || d.Slug != "harare-events" {
		t.Errorf("publishing renamed draft: slug %q, %v", d.Slug, err)
	}
	d.Name = "Bulawayo events"
	if d, err = c.UpdateForm(ctx, mine.tenant, d.ID, d); err != nil || d.Slug != "harare-events" {
		t.Errorf("renaming live form: slug %q, %v", d.Slug, err)
	}
}
