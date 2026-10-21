package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/cleanup"
	"github.com/nerd-zero/listnun/models"
)

// cleanupBatch is how many inactive subscribers one step acts on.
const cleanupBatch = 1000

// ErrCleanupRunning is returned when another run for the workspace holds
// the lock or started within the run window.
var ErrCleanupRunning = errors.New("an inactive cleanup run already started recently")

// CountInactive returns how many subscribers the rules would act on now.
func (c *Core) CountInactive(ctx context.Context, tenantID int, r cleanup.Rules) (int, error) {
	var n int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CountInactiveSubscribers).Get(&n, r.Days, r.Activity, pq.Array(nonNilInts(r.ListIDs)))
	})
	if err != nil {
		c.log.Printf("error counting inactive subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	return n, nil
}

// LastCleanupRun returns the workspace's latest run, if any.
func (c *Core) LastCleanupRun(ctx context.Context, tenantID int) (models.CleanupRun, bool, error) {
	var out models.CleanupRun
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetLastCleanupRun).Get(&out)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, false, nil
	}
	return out, err == nil, err
}

// RunInactiveCleanup applies the rules to every matching subscriber, in
// batches, and records the run. It returns how many matched and how many
// were acted on. Only one run per workspace can start per run window.
func (c *Core) RunInactiveCleanup(ctx context.Context, tenantID int, mode string, r cleanup.Rules) (int, int, error) {
	if err := r.Validate(); err != nil {
		return 0, 0, err
	}

	// Claim the run: an advisory lock serialises app instances, and a run
	// that started within the window makes this one a no-op.
	var runID int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext('inactive_cleanup'), $1)`, tenantID); err != nil {
			return err
		}
		var recent bool
		if err := tx.Get(&recent, `SELECT EXISTS (SELECT 1 FROM inactive_cleanup_runs
			WHERE started_at > NOW() - $1::INTERVAL)`, fmt.Sprintf("%d seconds", int(cleanup.RunWindow.Seconds()))); err != nil {
			return err
		}
		if recent {
			return ErrCleanupRunning
		}
		return stmtx(tx, c.q.InsertCleanupRun).Get(&runID, tenantID, mode, r.Action)
	})
	if err != nil {
		return 0, 0, err
	}

	matched, err := c.CountInactive(ctx, tenantID, r)
	done := 0
	for err == nil {
		var ids []int
		err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
			return stmtx(tx, c.q.GetInactiveSubscriberIDs).Select(&ids, r.Days, r.Activity, pq.Array(nonNilInts(r.ListIDs)), cleanupBatch)
		})
		if err != nil || len(ids) == 0 {
			break
		}
		if err = c.applyCleanupAction(ctx, tenantID, r, ids); err != nil {
			break
		}
		done += len(ids)
		if done > matched+cleanupBatch {
			// Acted subscribers stop matching; if they somehow don't, stop
			// rather than loop.
			err = errors.New("cleanup made no progress")
		}
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if ferr := c.WithTenant(context.Background(), tenantID, nil, func(tx *sqlx.Tx) error {
		_, e := stmtx(tx, c.q.FinishCleanupRun).Exec(runID, matched, done, errMsg)
		return e
	}); ferr != nil {
		c.log.Printf("error recording inactive cleanup run %d: %v", runID, ferr)
	}
	return matched, done, err
}

func (c *Core) applyCleanupAction(ctx context.Context, tenantID int, r cleanup.Rules, ids []int) error {
	switch r.Action {
	case cleanup.ActionUnsubscribe:
		return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
			_, err := stmtx(tx, c.q.UnsubscribeInactive).Exec(pq.Array(ids), pq.Array(nonNilInts(r.ListIDs)))
			return err
		})
	case cleanup.ActionBlocklist:
		return c.BlocklistSubscribers(ctx, tenantID, ids)
	case cleanup.ActionDelete:
		return c.DeleteSubscribers(ctx, tenantID, ids, nil)
	}
	return errors.New("unknown cleanup action " + r.Action)
}

// nonNilInts turns nil into an empty slice so it binds as '{}' (all
// lists), not NULL.
func nonNilInts(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

// RecordCleanupSkip records a scheduled run that couldn't happen (for
// example, tracking is off), so the schedule moves on and the screen can
// say why.
func (c *Core) RecordCleanupSkip(tenantID int, mode, action, reason string) error {
	return c.WithTenant(context.Background(), tenantID, nil, func(tx *sqlx.Tx) error {
		var id int
		if err := stmtx(tx, c.q.InsertCleanupRun).Get(&id, tenantID, mode, action); err != nil {
			return err
		}
		_, err := stmtx(tx, c.q.FinishCleanupRun).Exec(id, 0, 0, reason)
		return err
	})
}
