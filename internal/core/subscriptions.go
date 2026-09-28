package core

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

// GetSubscriptions retrieves the subscriptions for a subscriber.
func (c *Core) GetSubscriptions(ctx context.Context, tenantID int, subID int, subUUID string, allLists bool) ([]models.Subscription, error) {
	var out []models.Subscription
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetSubscriptions).Select(&out, subID, subUUID, allLists)
	})
	if err != nil {
		c.log.Printf("error getting subscriptions: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return out, err
}

// AddSubscriptions adds list subscriptions to subscribers.
func (c *Core) AddSubscriptions(ctx context.Context, tenantID int, subIDs, listIDs []int, status string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.AddSubscribersToLists).Exec(pq.Array(subIDs), pq.Array(listIDs), status, tenantID)
		return err
	})
	if err != nil {
		c.log.Printf("error adding subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// AddSubscriptionsByQuery adds list subscriptions to subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) AddSubscriptionsByQuery(ctx context.Context, tenantID int, searchStr, queryExp string, sourceListIDs, targetListIDs []int, status string, subStatus string) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return c.q.ExecSubQueryTpl(searchStr, queryExp, c.q.AddSubscribersToListsByQuery, sourceListIDs, tx, subStatus, pq.Array(targetListIDs), status, tenantID)
	})
	if err != nil {
		c.log.Printf("error adding subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscriptions delete list subscriptions from subscribers.
func (c *Core) DeleteSubscriptions(ctx context.Context, tenantID int, subIDs, listIDs []int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteSubscriptions).Exec(pq.Array(subIDs), pq.Array(listIDs))
		return err
	})
	if err != nil {
		c.log.Printf("error deleting subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))

	}

	return nil
}

// DeleteSubscriptionsByQuery deletes list subscriptions from subscribers by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteSubscriptionsByQuery(ctx context.Context, tenantID int, searchStr, queryExp string, sourceListIDs, targetListIDs []int, subStatus string) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return c.q.ExecSubQueryTpl(searchStr, queryExp, c.q.DeleteSubscriptionsByQuery, sourceListIDs, tx, subStatus, pq.Array(targetListIDs))
	})
	if err != nil {
		c.log.Printf("error deleting subscriptions by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// UnsubscribeLists sets list subscriptions to 'unsubscribed'.
func (c *Core) UnsubscribeLists(ctx context.Context, tenantID int, subIDs, listIDs []int, listUUIDs []string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UnsubscribeSubscribersFromLists).Exec(pq.Array(subIDs), pq.Array(listIDs), pq.StringArray(listUUIDs))
		return err
	})
	if err != nil {
		c.log.Printf("error unsubscribing from lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return nil
}

// UnsubscribeListsByQuery sets list subscriptions to 'unsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) UnsubscribeListsByQuery(ctx context.Context, tenantID int, searchStr, queryExp string, sourceListIDs, targetListIDs []int, subStatus string) error {
	if sourceListIDs == nil {
		sourceListIDs = []int{}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return c.q.ExecSubQueryTpl(searchStr, queryExp, c.q.UnsubscribeSubscribersFromListsByQuery, sourceListIDs, tx, subStatus, pq.Array(targetListIDs))
	})
	if err != nil {
		c.log.Printf("error unsubscribing from lists by query: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteUnconfirmedSubscriptions sets list subscriptions to 'unsubscribed' by a given arbitrary query expression.
// sourceListIDs is the list of list IDs to filter the subscriber query with.
func (c *Core) DeleteUnconfirmedSubscriptions(ctx context.Context, tenantID int, beforeDate time.Time) (int, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.DeleteUnconfirmedSubscriptions).Exec(beforeDate)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error deleting unconfirmed subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return int(n), nil
}
