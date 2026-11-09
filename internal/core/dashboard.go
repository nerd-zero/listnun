package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// GetDashboardCharts returns chart data points to render on the dashboard.
func (c *Core) GetDashboardCharts(ctx context.Context, tenantID int) (types.JSONText, error) {
	_ = c.refreshCache(matDashboardCharts, false)

	var out types.JSONText
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetDashboardCharts).Get(&out, tenantID)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard charts", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardCounts returns stats counts to show on the dashboard.
func (c *Core) GetDashboardCounts(ctx context.Context, tenantID int) (types.JSONText, error) {
	_ = c.refreshCache(matDashboardCounts, false)

	var out types.JSONText
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetDashboardCounts).Get(&out, tenantID)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetDashboardOverview returns live dashboard numbers for the last days.
func (c *Core) GetDashboardOverview(ctx context.Context, tenantID, days int) (models.DashboardOverview, error) {
	var (
		raw types.JSONText
		out models.DashboardOverview
	)
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetDashboardOverview).Get(&raw, days, tenantID)
	})
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	out.Totals()
	out.AsOf = time.Now()
	return out, nil
}
