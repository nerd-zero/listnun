package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/cleanup"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/models"
	null "gopkg.in/volatiletech/null.v6"
)

// cleanupTick is how often the scheduler checks whether a workspace's
// inactive cleanup is due.
const cleanupTick = time.Minute

// initManagedCleanup reads Listnun's managed rules from config
// ([inactive_cleanup.managed]), falling back to cleanup.DefaultManaged.
func initManagedCleanup() cleanup.Rules {
	r := cleanup.DefaultManaged
	if ko.Exists("inactive_cleanup.managed") {
		if err := ko.UnmarshalWithConf("inactive_cleanup.managed", &r, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Printf("error reading inactive_cleanup.managed, using defaults: %v", err)
			return cleanup.DefaultManaged
		}
		if err := r.Validate(); err != nil {
			lo.Printf("invalid inactive_cleanup.managed, using defaults: %v", err)
			return cleanup.DefaultManaged
		}
	}
	return r
}

// runInactiveCleanups checks every active workspace each minute and runs
// its inactive cleanup when it's due.
func (a *App) runInactiveCleanups(ctx context.Context) {
	t := time.NewTicker(cleanupTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		ids, err := a.core.GetActiveTenantIDs()
		if err != nil {
			a.log.Printf("error listing workspaces for inactive cleanup: %v", err)
			continue
		}
		for _, id := range ids {
			a.maybeRunCleanup(ctx, id)
		}
	}
}

func (a *App) maybeRunCleanup(ctx context.Context, tenantID int) {
	s, err := a.core.GetSettings(ctx, tenantID)
	if err != nil || !s.InactiveCleanup.Enabled {
		return
	}
	ic := s.InactiveCleanup
	rules := ic.Effective(a.managedCleanup)

	var lastRun time.Time
	if last, ok, err := a.core.LastCleanupRun(ctx, tenantID); err == nil && ok {
		lastRun = last.StartedAt
	}
	due, _, err := cleanup.Due(rules.Schedule, time.Now(), lastRun)
	if err != nil || !due {
		return
	}

	// Without per-subscriber tracking every subscriber looks inactive.
	if !s.PrivacyIndividualTracking {
		if err := a.core.RecordCleanupSkip(tenantID, ic.Mode, rules.Action, cleanup.ErrNoTracking.Error()); err != nil {
			a.log.Printf("error recording skipped inactive cleanup: %v", err)
		}
		a.core.LogActivity(tenantID, models.ActivityWarn, "cleanup", "inactive cleanup skipped: "+cleanup.ErrNoTracking.Error())
		return
	}

	matched, done, err := a.core.RunInactiveCleanup(ctx, tenantID, ic.Mode, rules)
	switch {
	case errors.Is(err, core.ErrCleanupRunning):
		return
	case err != nil:
		a.core.LogActivity(tenantID, models.ActivityError, "cleanup",
			fmt.Sprintf("%s inactive cleanup failed after %d of %d subscribers: %v", ic.Mode, done, matched, err))
	default:
		a.core.LogActivity(tenantID, models.ActivityInfo, "cleanup",
			fmt.Sprintf("%s run found %d inactive subscribers, %s %d", ic.Mode, matched, cleanupActionVerb(rules.Action), done))
	}
}

func cleanupActionVerb(action string) string {
	switch action {
	case cleanup.ActionBlocklist:
		return "blocklisted"
	case cleanup.ActionDelete:
		return "deleted"
	}
	return "unsubscribed"
}

// cleanupStatus is what Settings › Inactive cleanup shows next to the form.
type cleanupStatus struct {
	// Managed are Listnun's rules, shown read-only in managed mode.
	Managed cleanup.Rules `json:"managed"`
	// Matching is how many subscribers the saved rules would act on now.
	Matching int `json:"matching"`
	// Tracking is false when per-subscriber tracking is off, which stops
	// the cleanup from running.
	Tracking bool               `json:"tracking"`
	NextRun  null.Time          `json:"next_run"`
	LastRun  *models.CleanupRun `json:"last_run"`
}

// GetCleanupStatus returns the managed rules, the next run and how many
// subscribers the saved rules match now.
//
//	@ID			getInactiveCleanupStatus
//	@Summary	Get inactive cleanup status
//	@Tags		settings
//	@Produce	json
//	@Success	200	{object}	cleanupStatus
//	@Failure	500	{object}	echo.HTTPError
//	@Router		/api/settings/inactive-cleanup [get]
func (a *App) GetCleanupStatus(c echo.Context) error {
	ctx := c.Request().Context()
	s, err := a.core.GetSettings(ctx, tenantID(c))
	if err != nil {
		return err
	}
	rules := s.InactiveCleanup.Effective(a.managedCleanup)

	out := cleanupStatus{Managed: a.managedCleanup, Tracking: s.PrivacyIndividualTracking}
	if rules.Validate() == nil {
		if out.Matching, err = a.core.CountInactive(ctx, tenantID(c), rules); err != nil {
			return err
		}
		if s.InactiveCleanup.Enabled {
			if _, next, err := cleanup.Due(rules.Schedule, time.Now(), time.Now()); err == nil {
				out.NextRun = null.TimeFrom(next)
			}
		}
	}
	if last, ok, err := a.core.LastCleanupRun(ctx, tenantID(c)); err == nil && ok {
		out.LastRun = &last
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// PreviewCleanup counts how many subscribers the given (unsaved) rules
// would act on now.
//
//	@ID			previewInactiveCleanup
//	@Summary	Preview inactive cleanup rules
//	@Tags		settings
//	@Accept		json
//	@Produce	json
//	@Param		rules	body		cleanup.Rules	true	"Rules to preview"
//	@Success	200		{object}	int
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/settings/inactive-cleanup/preview [post]
func (a *App) PreviewCleanup(c echo.Context) error {
	var r cleanup.Rules
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	n, err := a.core.CountInactive(c.Request().Context(), tenantID(c), r)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{n})
}
