package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// activityPruneInterval is how often old activity log lines are deleted.
const activityPruneInterval = 6 * time.Hour

// GetActivityLog returns the workspace's recent activity (Settings › Logs).
//
//	@ID			getActivityLog
//	@Summary	Get the workspace's activity log
//	@Tags		settings
//	@Produce	json
//	@Param		limit	query		int	false	"Lines to return (max 500)"
//	@Success	200		{object}	[]models.ActivityEntry
//	@Failure	500		{object}	echo.HTTPError
//	@Router		/api/settings/logs [get]
func (a *App) GetActivityLog(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	out, err := a.core.GetActivity(c.Request().Context(), tenantID(c), limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// pruneActivity deletes activity log lines past their retention, forever.
func (a *App) pruneActivity() {
	for {
		if n, err := a.core.PruneActivity(); err != nil {
			a.log.Printf("error pruning activity log: %v", err)
		} else if n > 0 {
			a.log.Printf("pruned %d activity log lines", n)
		}
		time.Sleep(activityPruneInterval)
	}
}
