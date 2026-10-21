package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

const (
	// ActivityRetention is how long activity log lines are kept.
	ActivityRetention = 30 * 24 * time.Hour

	// activityMaxLines caps how many lines one read returns; the Logs
	// screen shows the latest 500.
	activityMaxLines = 500
)

// LogActivity adds a line to a workspace's activity log. It never fails
// the caller: the log is a convenience, so errors are only printed.
func (c *Core) LogActivity(tenantID int, level, source, message string) {
	err := c.WithTenant(context.Background(), tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.InsertActivity).Exec(tenantID, level, source, message)
		return err
	})
	if err != nil {
		c.log.Printf("error writing activity log (%s: %s): %v", source, message, err)
	}
}

// GetActivity returns up to limit of the workspace's latest activity lines.
func (c *Core) GetActivity(ctx context.Context, tenantID, limit int) ([]models.ActivityEntry, error) {
	if limit < 1 || limit > activityMaxLines {
		limit = activityMaxLines
	}
	out := []models.ActivityEntry{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetActivity).Select(&out, limit)
	})
	if err != nil {
		c.log.Printf("error fetching activity log: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "logs", "error", pqErrMsg(err)))
	}
	return out, nil
}

// PruneActivity deletes activity log lines past their retention, across
// workspaces.
func (c *Core) PruneActivity() (int64, error) {
	res, err := c.q.PruneActivity.Exec(fmt.Sprintf("%d seconds", int(ActivityRetention.Seconds())))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
