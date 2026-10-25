package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// maxSlugTries is how many numbered slugs ("signup-2", "signup-3", ...)
// CreateForm tries before giving up.
const maxSlugTries = 50

var reSlugJunk = regexp.MustCompile(`[^a-z0-9]+`)

// FormSlug turns a form's name into the last part of its page address.
func FormSlug(name string) string {
	s := strings.Trim(reSlugJunk.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "form"
	}
	return s
}

// GetForms returns every form with its signups over the last 30 days.
func (c *Core) GetForms(ctx context.Context, tenantID int) ([]models.Form, error) {
	out := []models.Form{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetForms).Select(&out)
	})
	if err != nil {
		return nil, c.formErr("globals.messages.errorFetching", err)
	}
	return out, nil
}

// GetForm returns a form by its ID.
func (c *Core) GetForm(ctx context.Context, tenantID, id int) (models.Form, error) {
	return c.getForm(ctx, tenantID, c.q.GetForm, id)
}

// GetFormBySlug returns a form by its page address.
func (c *Core) GetFormBySlug(ctx context.Context, tenantID int, slug string) (models.Form, error) {
	return c.getForm(ctx, tenantID, c.q.GetFormBySlug, slug)
}

func (c *Core) getForm(ctx context.Context, tenantID int, stmt *sqlx.Stmt, arg any) (models.Form, error) {
	var out models.Form
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, stmt).Get(&out, arg)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{forms.form}"))
	}
	if err != nil {
		return out, c.formErr("globals.messages.errorFetching", err)
	}
	return out, nil
}

// CreateForm saves a new form, its page address made from its name.
func (c *Core) CreateForm(ctx context.Context, tenantID int, f models.Form) (models.Form, error) {
	var out models.Form
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		slug, err := c.freeFormSlug(tx, FormSlug(f.Name))
		if err != nil {
			return err
		}
		return stmtx(tx, c.q.CreateForm).Get(&out, tenantID, slug, f.Name, f.Status, f.Heading, f.Intro,
			f.ListIDs, f.Fields, f.ButtonLabel, f.SuccessMessage, f.Captcha)
	})
	if err != nil {
		return out, c.formErr("globals.messages.errorCreating", err)
	}
	return out, nil
}

// freeFormSlug returns slug, or slug-2, slug-3 ... when it's taken.
func (c *Core) freeFormSlug(tx *sqlx.Tx, slug string) (string, error) {
	try := slug
	for i := 2; i <= maxSlugTries+1; i++ {
		var taken bool
		if err := stmtx(tx, c.q.FormSlugExists).Get(&taken, try); err != nil {
			return "", err
		}
		if !taken {
			return try, nil
		}
		try = fmt.Sprintf("%s-%d", slug, i)
	}
	return "", fmt.Errorf("no free address for %q", slug)
}

// UpdateForm saves a form's changes. Until a form is first published its
// page address follows its name; after that it stays the same, so links to
// it keep working.
func (c *Core) UpdateForm(ctx context.Context, tenantID, id int, f models.Form) (models.Form, error) {
	var out models.Form
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		var old models.Form
		if err := stmtx(tx, c.q.GetForm).Get(&old, id); err != nil {
			return err
		}
		slug := old.Slug
		if old.Status == models.FormStatusDraft && old.Slug != FormSlug(f.Name) {
			var err error
			if slug, err = c.freeFormSlug(tx, FormSlug(f.Name)); err != nil {
				return err
			}
		}
		return stmtx(tx, c.q.UpdateForm).Get(&out, id, f.Name, f.Status, f.Heading, f.Intro,
			f.ListIDs, f.Fields, f.ButtonLabel, f.SuccessMessage, f.Captcha, slug)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{forms.form}"))
	}
	if err != nil {
		return out, c.formErr("globals.messages.errorUpdating", err)
	}
	return out, nil
}

// DeleteForm deletes a form and its figures.
func (c *Core) DeleteForm(ctx context.Context, tenantID, id int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteForm).Exec(id)
		return err
	})
	if err != nil {
		return c.formErr("globals.messages.errorDeleting", err)
	}
	return nil
}

// AddFormEvent records a view, signup or blocked submission. subID is the
// subscriber who signed up, or 0.
func (c *Core) AddFormEvent(ctx context.Context, tenantID, formID int, kind string, subID int) error {
	var sub sql.NullInt64
	if subID > 0 {
		sub = sql.NullInt64{Int64: int64(subID), Valid: true}
	}
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.AddFormEvent).Exec(tenantID, formID, kind, sub)
		return err
	})
}

// GetFormStats returns a form's figures since since.
func (c *Core) GetFormStats(ctx context.Context, tenantID, id int, since time.Time) (models.FormStats, error) {
	var out models.FormStats
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetFormStats).Get(&out, id, since)
	})
	if err != nil {
		return out, c.formErr("globals.messages.errorFetching", err)
	}
	return out, nil
}

func (c *Core) formErr(msg string, err error) error {
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return he
	}
	c.log.Printf("forms: %v", err)
	return echo.NewHTTPError(http.StatusInternalServerError,
		c.i18n.Ts(msg, "name", "{forms.form}", "error", pqErrMsg(err)))
}
