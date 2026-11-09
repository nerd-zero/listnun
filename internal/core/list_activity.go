package core

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

// GetListsGrowth returns each list's joins and leaves since `since`. Lists
// with neither are left out.
func (c *Core) GetListsGrowth(ctx context.Context, tenantID int, since time.Time) ([]models.ListGrowth, error) {
	out := []models.ListGrowth{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListsGrowth).Select(&out, since)
	})
	if err != nil {
		c.log.Printf("error fetching list growth: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetListsSummary counts the active lists the user can see (all when
// getAll, else permittedIDs) and their subscriptions.
func (c *Core) GetListsSummary(ctx context.Context, tenantID int, getAll bool, permittedIDs []int) (models.ListsSummary, error) {
	var out models.ListsSummary
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListsSummary).Get(&out, getAll, pq.Array(permittedIDs))
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetListActivity returns a list's daily joins and leaves from the day of
// `since` to today, their totals, and its `numCampaigns` latest campaigns.
func (c *Core) GetListActivity(ctx context.Context, tenantID, listID int, since time.Time, numCampaigns int) (models.ListActivity, error) {
	out := models.ListActivity{Days: []models.ListActivityDay{}, Campaigns: []models.ListCampaign{}}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetListActivity).Select(&out.Days, listID, since); err != nil {
			return err
		}
		return stmtx(tx, c.q.GetListRecentCampaigns).Select(&out.Campaigns, listID, numCampaigns)
	})
	if err != nil {
		c.log.Printf("error fetching list activity: %v", err)
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	for _, d := range out.Days {
		out.Joined += d.Joined
		out.Left += d.Left
	}
	return out, nil
}

// CountOptinPending counts the people waiting to confirm a double opt-in
// subscription on listIDs (every list when nil), and how many of them
// weren't sent a confirmation since dueBefore.
func (c *Core) CountOptinPending(ctx context.Context, tenantID int, listIDs []int, dueBefore time.Time) (models.OptinPending, error) {
	var out models.OptinPending
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CountOptinPending).Get(&out, nullIntArray(listIDs), dueBefore)
	})
	if err != nil {
		c.log.Printf("error counting pending opt-ins: %v", err)
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetOptinDue returns up to `limit` people waiting to confirm a double
// opt-in subscription on listIDs (every list when nil) who weren't sent a
// confirmation since dueBefore.
func (c *Core) GetOptinDue(ctx context.Context, tenantID int, listIDs []int, dueBefore time.Time, limit int) ([]models.Subscriber, error) {
	out := []models.Subscriber{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetOptinPending).Select(&out, nullIntArray(listIDs), dueBefore, limit)
	})
	if err != nil {
		c.log.Printf("error fetching pending opt-ins: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// MarkOptinSent records that a subscriber was just sent an opt-in
// confirmation, so a bulk resend skips them for a while.
func (c *Core) MarkOptinSent(ctx context.Context, tenantID, subID int) error {
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.MarkOptinSent).Exec(subID)
		return err
	})
}

// nullIntArray is ids as a Postgres array, or NULL when there are none.
func nullIntArray(ids []int) any {
	if len(ids) == 0 {
		return nil
	}
	return pq.Array(ids)
}
