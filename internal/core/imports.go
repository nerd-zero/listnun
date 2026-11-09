package core

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

// CountExistingEmails counts how many of emails (lower case) are already
// subscribers.
func (c *Core) CountExistingEmails(ctx context.Context, tenantID int, emails []string) (int, error) {
	var n int
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CountExistingEmails).Get(&n, pq.Array(emails))
	})
	if err != nil {
		c.log.Printf("error counting existing subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	return n, nil
}

// RecordImport saves a finished import for the Import page's history.
func (c *Core) RecordImport(ctx context.Context, r models.ImportRecord) error {
	return c.WithTenant(ctx, r.TenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.RecordImport).Exec(r.TenantID, r.Name, r.Mode, r.Status,
			r.Total, r.Imported, r.Added, r.Skipped, r.Failed)
		return err
	})
}

// GetRecentImports returns the latest n finished imports.
func (c *Core) GetRecentImports(ctx context.Context, tenantID, n int) ([]models.ImportRecord, error) {
	out := []models.ImportRecord{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetRecentImports).Select(&out, n)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{import.title}", "error", pqErrMsg(err)))
	}
	return out, nil
}
