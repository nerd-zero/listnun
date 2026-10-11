package core

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
	null "gopkg.in/volatiletech/null.v6"
)

// GetTemplates retrieves all templates.
func (c *Core) GetTemplates(ctx context.Context, tenantID int, status string, noBody bool) ([]models.Template, error) {
	out := []models.Template{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetTemplates).Select(&out, 0, noBody, status)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetTemplate retrieves a given template.
func (c *Core) GetTemplate(ctx context.Context, tenantID int, id int, noBody bool) (models.Template, error) {
	var out []models.Template
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetTemplates).Select(&out, id, noBody, "")
	})
	if err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.templates}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Template{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	return out[0], nil
}

// CreateTemplate creates a new template.
func (c *Core) CreateTemplate(ctx context.Context, tenantID int, name, typ, subject string, body []byte, bodySource null.String) (models.Template, error) {
	var newID int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateTemplate).Get(&newID, name, typ, subject, body, bodySource, tenantID)
	})
	if err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return c.GetTemplate(ctx, tenantID, newID, false)
}

// UpdateTemplate updates a given template.
func (c *Core) UpdateTemplate(ctx context.Context, tenantID int, id int, name, subject string, body []byte, bodySource null.String) (models.Template, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.UpdateTemplate).Exec(id, name, subject, body, bodySource)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return models.Template{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	if n == 0 {
		return models.Template{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.template}"))
	}

	return c.GetTemplate(ctx, tenantID, id, false)
}

// SetDefaultTemplate sets a template as default.
func (c *Core) SetDefaultTemplate(ctx context.Context, tenantID int, id int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.SetDefaultTemplate).Exec(id)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteTemplate deletes a given template.
func (c *Core) DeleteTemplate(ctx context.Context, tenantID int, id int) error {
	var delID int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.DeleteTemplate).Get(&delID, id)
	})
	if err != nil && err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.template}", "error", pqErrMsg(err)))
	}
	if delID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("templates.cantDeleteDefault"))
	}

	return nil
}
