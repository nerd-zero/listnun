package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var (
	allowedSubQueryTables = map[string]struct{}{
		"subscribers":      {},
		"lists":            {},
		"subscriber_lists": {},
		"campaigns":        {},
		"campaign_lists":   {},
		"campaign_views":   {},
		"links":            {},
		"link_clicks":      {},
		"bounces":          {},
	}
)

// GetSubscriber fetches a subscriber by one of the given params.
func (c *Core) GetSubscriber(ctx context.Context, tenantID int, id int, uuid, email string) (models.Subscriber, error) {
	var uu any
	if uuid != "" {
		uu = uuid
	}

	var out models.Subscribers
	loadListsFailed := false
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetSubscriber).Select(&out, id, uu, email); err != nil {
			return err
		}
		if len(out) == 0 {
			return nil
		}
		if err := out.LoadLists(stmtx(tx, c.q.GetSubscriberListsLazy)); err != nil {
			loadListsFailed = true
			return err
		}
		return nil
	})
	if err != nil && loadListsFailed {
		c.log.Printf("error loading subscriber lists: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	if err != nil {
		c.log.Printf("error fetching subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return models.Subscriber{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name",
				fmt.Sprintf("{globals.terms.subscriber} (%d: %s%s)", id, uuid, email)))
	}

	return out[0], nil
}

// HasSubscriberLists checks if the given subscribers have at least one of the given lists.
func (c *Core) HasSubscriberLists(ctx context.Context, tenantID int, subIDs []int, listIDs []int) (map[int]bool, error) {
	res := []struct {
		SubID int  `db:"subscriber_id"`
		Has   bool `db:"has"`
	}{}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.HasSubscriberLists).Select(&res, pq.Array(subIDs), pq.Array(listIDs))
	})
	if err != nil {
		c.log.Printf("error fetching subscriber: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	out := make(map[int]bool, len(res))
	for _, r := range res {
		out[r.SubID] = r.Has
	}

	return out, nil
}

// GetSubscribersByEmail fetches a subscriber by one of the given params.
func (c *Core) GetSubscribersByEmail(ctx context.Context, tenantID int, emails []string) (models.Subscribers, error) {
	var out models.Subscribers
	loadListsFailed := false

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetSubscribersByEmails).Select(&out, pq.Array(emails)); err != nil {
			return err
		}
		if len(out) == 0 {
			return nil
		}
		if err := out.LoadLists(stmtx(tx, c.q.GetSubscriberListsLazy)); err != nil {
			loadListsFailed = true
			return err
		}
		return nil
	})
	if err != nil && loadListsFailed {
		c.log.Printf("error loading subscriber lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	if err != nil {
		c.log.Printf("error fetching subscriber: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("campaigns.noKnownSubsToTest"))
	}

	return out, nil
}

// QuerySubscribers queries and returns paginated subscrribers based on the given params including the total count.
func (c *Core) QuerySubscribers(ctx context.Context, tenantID int, searchStr, queryExp string, listIDs []int, subStatus string, order, orderBy string, offset, limit int) (models.Subscribers, int, error) {
	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "subscribers.id"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	// Required for pq.Array()
	if listIDs == nil {
		listIDs = []int{}
	}

	// There's an arbitrary query condition.
	cond := "TRUE"
	if queryExp != "" {
		cond = queryExp
	}

	// stmt is the raw SQL query.
	stmt := strings.ReplaceAll(c.q.QuerySubscribers, "%query%", cond)
	stmt = strings.ReplaceAll(stmt, "%order%", orderBy+" "+order)

	// Validate the tables used in the query.
	if err := validateQueryTables(c.db, stmt, allowedSubQueryTables, pq.Array(listIDs), subStatus, searchStr, offset, limit); err != nil {
		c.log.Printf("error validating query tables: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", err.Error()))
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	total, err := c.getSubscriberCount(ctx, tenantID, searchStr, cond, subStatus, listIDs)
	if err != nil {
		c.log.Printf("error getting subscriber count: %v", err)
		return nil, 0, err
	}

	// No results.
	if total == 0 {
		return models.Subscribers{}, 0, nil
	}

	var out models.Subscribers
	err = c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := tx.Select(&out, stmt, pq.Array(listIDs), subStatus, searchStr, offset, limit); err != nil {
			return err
		}
		// Lazy load lists for each subscriber.
		return out.LoadLists(stmtx(tx, c.q.GetSubscriberListsLazy))
	})
	if err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return out, total, nil
}

// GetSubscriberLists returns a subscriber's lists based on the given conditions.
func (c *Core) GetSubscriberLists(ctx context.Context, tenantID int, subID int, uuid string, listIDs []int, listUUIDs []string, subStatus string, listType string) ([]models.List, error) {
	if listIDs == nil {
		listIDs = []int{}
	}
	if listUUIDs == nil {
		listUUIDs = []string{}
	}

	var uu any
	if uuid != "" {
		uu = uuid
	}

	// Fetch double opt-in lists from the given list IDs.
	// Get the list of subscription lists where the subscriber hasn't confirmed.
	out := []models.List{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetSubscriberLists).Select(&out, subID, uu, pq.Array(listIDs), pq.Array(listUUIDs), subStatus, listType)
	})
	if err != nil {
		c.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return nil, err
	}

	return out, nil
}

// GetSubscriberProfileForExport returns the subscriber's profile data as a JSON exportable.
// Get the subscriber's data. A single query that gets the profile, list subscriptions, campaign views,
// and link clicks. Names of private lists are replaced with "Private list".
func (c *Core) GetSubscriberProfileForExport(ctx context.Context, tenantID int, id int, uuid string) (models.SubscriberExportProfile, error) {
	var uu any
	if uuid != "" {
		uu = uuid
	}

	var out models.SubscriberExportProfile
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.ExportSubscriberData).Get(&out, id, uu)
	})
	if err != nil {
		c.log.Printf("error fetching subscriber export data: %v", err)

		return models.SubscriberExportProfile{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return out, nil
}

// GetSubscriberActivity returns the subscriber's campaign views and link clicks for the Activity tab.
func (c *Core) GetSubscriberActivity(ctx context.Context, tenantID int, id int) (models.SubscriberActivity, error) {
	var out models.SubscriberActivity
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetSubscriberActivity).Get(&out, id)
	})
	if err != nil {
		c.log.Printf("error fetching subscriber activity: %v", err)

		return models.SubscriberActivity{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "activity", "error", err.Error()))
	}

	return out, nil
}

// ExportSubscribers returns an iterator function that provides lists of subscribers based
// on the given criteria in an exportable form. The iterator function returned can be called
// repeatedly until there are nil subscribers. It's an iterator because exports can be extremely
// large and may have to be fetched in batches from the DB and streamed somewhere.
//
// TODO(#40): ctx/tenantID are accepted for a consistent call-site shape
// with the rest of Core, but the iterator below does NOT yet enforce
// tenant scoping - it holds a single Preparex'd statement across
// potentially many calls over a long export, which doesn't fit
// WithTenant's short-transaction shape. Zero actual risk today (no way to
// create a second tenant with real data yet, Operator API is phase 9,
// unbuilt); revisit once there's a real multi-tenant dataset to protect.
func (c *Core) ExportSubscribers(ctx context.Context, tenantID int, searchStr, query string, subIDs, listIDs []int, subStatus string, batchSize int) (func() ([]models.SubscriberExport, error), error) {
	if subIDs == nil {
		subIDs = []int{}
	}
	if listIDs == nil {
		listIDs = []int{}
	}

	// There's an arbitrary query condition.
	cond := "TRUE"
	if query != "" {
		cond = query
	}

	stmt := strings.ReplaceAll(c.q.QuerySubscribersForExport, "%query%", cond)

	// Validate the tables used in the query.
	if err := validateQueryTables(c.db, stmt, allowedSubQueryTables,
		pq.Array(listIDs), 0, pq.Array(subIDs), subStatus, searchStr, batchSize); err != nil {
		c.log.Printf("error validating query tables: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", err.Error()))
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	if _, err := c.getSubscriberCount(ctx, tenantID, searchStr, cond, subStatus, listIDs); err != nil {
		c.log.Printf("error getting subscriber count: %v", err)
		return nil, err
	}

	// Prepare the actual query statement.
	tx, err := c.db.Preparex(stmt)
	if err != nil {
		c.log.Printf("error preparing subscriber query: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}

	id := 0
	return func() ([]models.SubscriberExport, error) {
		var out []models.SubscriberExport
		if err := tx.Select(&out, pq.Array(listIDs), id, pq.Array(subIDs), subStatus, searchStr, batchSize); err != nil {
			c.log.Printf("error exporting subscribers by query: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
		if len(out) == 0 {
			return nil, nil
		}

		id = out[len(out)-1].ID
		return out, nil
	}, nil
}

// InsertSubscriber inserts a subscriber and returns the ID. The first bool indicates if
// it was a new subscriber, and the second bool indicates if the subscriber was sent an optin confirmation.
// bool = optinSent?
func (c *Core) InsertSubscriber(ctx context.Context, tenantID int, sub models.Subscriber, listIDs []int, listUUIDs []string, preconfirm, assertOptin bool) (models.Subscriber, bool, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}
	sub.UUID = uu.String()

	subStatus := models.SubscriptionStatusUnconfirmed
	if preconfirm {
		subStatus = models.SubscriptionStatusConfirmed
	}
	if sub.Status == "" {
		sub.Status = auth.UserStatusEnabled
	}

	// For pq.Array()
	if listIDs == nil {
		listIDs = []int{}
	}
	if listUUIDs == nil {
		listUUIDs = []string{}
	}

	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.InsertSubscriber).Get(&sub.ID,
			sub.UUID,
			sub.Email,
			strings.TrimSpace(sub.Name),
			sub.Status,
			sub.Attribs,
			pq.Array(listIDs),
			pq.Array(listUUIDs),
			subStatus,
			tenantID)
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "subscribers_email_key" {
			return models.Subscriber{}, false, echo.NewHTTPError(http.StatusConflict, c.i18n.T("subscribers.emailExists"))
		} else {
			// return sub.Subscriber, errSubscriberExists
			c.log.Printf("error inserting subscriber: %v", err)
			return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
		}
	}

	// Fetch the subscriber's full data. If the subscriber already existed and wasn't
	// created, the id will be empty. Fetch the details by e-mail then.
	out, err := c.GetSubscriber(ctx, tenantID, sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, false, err
	}

	hasOptin := false
	if !preconfirm && c.consts.SendOptinConfirmation {
		// Send a confirmation e-mail (if there are any double opt-in lists).
		num, err := c.h.SendOptinConfirmation(out, listIDs)
		if assertOptin && err != nil {
			return out, hasOptin, err
		}

		hasOptin = num > 0
	}

	return out, hasOptin, nil
}

// SetSubscriberScrubStatus records the outcome of a validate-on-add Scrub
// check against a subscriber -- a standalone follow-up call rather than a
// param threaded through InsertSubscriber, which is shared by callers
// (CreateSubscriber, processSubForm) that would otherwise all need
// reshaping. Errors here are expected to be logged and swallowed by
// callers -- this is a secondary side effect of a subscriber add, not a
// reason to fail the add itself.
func (c *Core) SetSubscriberScrubStatus(ctx context.Context, tenantID int, subscriberID int, status string) error {
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.SetSubscriberScrubStatus).Exec(subscriberID, status, tenantID)
		return err
	})
}

// GetRiskySubscriberIDs returns which of the given subscriber IDs are
// currently flagged scrub_status='risky'. Used by ManageSubscriberLists'
// "add" action, which links existing subscribers to a list and so never
// calls Scrub itself -- it just needs to know if any of them are already
// known-risky before deciding whether to auto-pause a running campaign.
func (c *Core) GetRiskySubscriberIDs(ctx context.Context, tenantID int, subscriberIDs []int) ([]int, error) {
	var ids []int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetRiskySubscriberIDs).Select(&ids, tenantID, pq.Array(subscriberIDs))
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	return ids, nil
}

// UpdateSubscriber updates a subscriber's properties.
func (c *Core) UpdateSubscriber(ctx context.Context, tenantID int, id int, sub models.Subscriber) (models.Subscriber, error) {
	// Format raw JSON attributes.
	attribs := []byte("{}")
	if len(sub.Attribs) > 0 {
		if b, err := json.Marshal(sub.Attribs); err != nil {
			return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		} else {
			attribs = b
		}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateSubscriber).Exec(id,
			sub.Email,
			strings.TrimSpace(sub.Name),
			sub.Status,
			json.RawMessage(attribs),
		)
		return err
	})
	if err != nil {
		c.log.Printf("error updating subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	out, err := c.GetSubscriber(ctx, tenantID, sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, err
	}

	return out, nil
}

// UpdateSubscriberWithLists updates a subscriber's properties.
// If deleteLists is set to true, all existing subscriptions are deleted and only
// the ones provided are added or retained.
func (c *Core) UpdateSubscriberWithLists(ctx context.Context, tenantID int, id int, sub models.Subscriber, listIDs []int, listUUIDs []string, preconfirm, deleteLists, assertOptin bool, permittedListIDs []int, allowResubscribe bool) (models.Subscriber, bool, error) {
	subStatus := models.SubscriptionStatusUnconfirmed
	if preconfirm {
		subStatus = models.SubscriptionStatusConfirmed
	}

	// Format raw JSON attributes.
	attribs := []byte("{}")
	if len(sub.Attribs) > 0 {
		if b, err := json.Marshal(sub.Attribs); err != nil {
			return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		} else {
			attribs = b
		}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateSubscriberWithLists).Exec(id,
			sub.Email,
			strings.TrimSpace(sub.Name),
			sub.Status,
			json.RawMessage(attribs),
			pq.Array(listIDs),
			pq.Array(listUUIDs),
			subStatus,
			deleteLists,
			pq.Array(permittedListIDs),
			allowResubscribe,
			tenantID)
		return err
	})
	if err != nil {
		c.log.Printf("error updating subscriber: %v", err)
		return models.Subscriber{}, false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	out, err := c.GetSubscriber(ctx, tenantID, sub.ID, "", sub.Email)
	if err != nil {
		return models.Subscriber{}, false, err
	}

	hasOptin := false
	if !preconfirm && c.consts.SendOptinConfirmation {
		// Send a confirmation e-mail (if there are any double opt-in lists).
		num, err := c.h.SendOptinConfirmation(out, listIDs)
		if assertOptin && err != nil {
			return out, hasOptin, err
		}
		hasOptin = num > 0
	}

	return out, hasOptin, nil
}

// BlocklistSubscribers blocklists the given list of subscribers.
func (c *Core) BlocklistSubscribers(ctx context.Context, tenantID int, subIDs []int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.BlocklistSubscribers).Exec(pq.Array(subIDs))
		return err
	})
	if err != nil {
		c.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
	}

	return nil
}

// BlocklistSubscribersByQuery blocklists the given list of subscribers.
func (c *Core) BlocklistSubscribersByQuery(ctx context.Context, tenantID int, searchStr, queryExp string, listIDs []int, subStatus string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return c.q.ExecSubQueryTpl(searchStr, sanitizeSQLExp(queryExp), c.q.BlocklistSubscribersByQuery, listIDs, tx, subStatus)
	})
	if err != nil {
		c.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscribers deletes the given list of subscribers.
func (c *Core) DeleteSubscribers(ctx context.Context, tenantID int, subIDs []int, subUUIDs []string) error {
	if subIDs == nil {
		subIDs = []int{}
	}
	if subUUIDs == nil {
		subUUIDs = []string{}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteSubscribers).Exec(pq.Array(subIDs), pq.Array(subUUIDs))
		return err
	})
	if err != nil {
		c.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscribersByQuery deletes subscribers by a given arbitrary query expression.
func (c *Core) DeleteSubscribersByQuery(ctx context.Context, tenantID int, searchStr, queryExp string, listIDs []int, subStatus string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return c.q.ExecSubQueryTpl(searchStr, sanitizeSQLExp(queryExp), c.q.DeleteSubscribersByQuery, listIDs, tx, subStatus)
	})
	if err != nil {
		c.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return err
}

// UnsubscribeByCampaign unsubscribes a given subscriber from lists in a given campaign.
func (c *Core) UnsubscribeByCampaign(ctx context.Context, tenantID int, subUUID, campUUID string, blocklist bool) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UnsubscribeByCampaign).Exec(campUUID, subUUID, blocklist)
		return err
	})
	if err != nil {
		c.log.Printf("error unsubscribing: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// ConfirmOptionSubscription confirms a subscriber's optin subscription.
func (c *Core) ConfirmOptionSubscription(ctx context.Context, tenantID int, subUUID string, listUUIDs []string, meta models.JSON) error {
	if meta == nil {
		meta = models.JSON{}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.ConfirmSubscriptionOptin).Exec(subUUID, pq.Array(listUUIDs), meta)
		return err
	})
	if err != nil {
		c.log.Printf("error confirming subscription: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteSubscriberBounces deletes the given list of subscribers.
func (c *Core) DeleteSubscriberBounces(ctx context.Context, tenantID int, id int, uuid string) error {
	var uu any
	if uuid != "" {
		uu = uuid
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteBouncesBySubscriber).Exec(id, uu)
		return err
	})
	if err != nil {
		c.log.Printf("error deleting bounces: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.bounces}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteOrphanSubscribers deletes orphan subscriber records (subscribers without lists).
func (c *Core) DeleteOrphanSubscribers(ctx context.Context, tenantID int) (int, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.DeleteOrphanSubscribers).Exec()
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error deleting orphan subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return int(n), nil
}

// DeleteBlocklistedSubscribers deletes blocklisted subscribers.
func (c *Core) DeleteBlocklistedSubscribers(ctx context.Context, tenantID int) (int, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.DeleteBlocklistedSubscribers).Exec()
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error deleting blocklisted subscribers: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return int(n), nil
}

func (c *Core) getSubscriberCount(ctx context.Context, tenantID int, searchStr, queryExp, subStatus string, listIDs []int) (int, error) {
	// If there's no condition, it's a "get all" call which can probably be optionally pulled from cache.
	if queryExp == "" {
		_ = c.refreshCache(matListSubStats, false)

		total := 0
		err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
			return stmtx(tx, c.q.QuerySubscribersCountAll).Get(&total, pq.Array(listIDs), subStatus, tenantID)
		})
		if err != nil {
			return 0, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}

		return total, nil
	}

	// Create a readonly transaction that just does COUNT() to obtain the count of results
	// and to ensure that the arbitrary query is indeed readonly.
	stmt := strings.ReplaceAll(c.q.QuerySubscribersCount, "%query%", queryExp)
	total := 0
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return tx.Get(&total, stmt, pq.Array(listIDs), subStatus, searchStr)
	})
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return total, nil
}

// validateQueryTables checks if the query accesses only allowed tables.
func validateQueryTables(db *sqlx.DB, query string, allowedTables map[string]struct{}, args ...any) error {
	// Get the EXPLAIN (FORMAT JSON) output.
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var plan string
	if err = tx.QueryRow("EXPLAIN (FORMAT JSON) "+query, args...).Scan(&plan); err != nil {
		return err
	}

	// Extract all relation names from the JSON plan.
	tables, err := getTablesFromQueryPlan(plan)
	if err != nil {
		return fmt.Errorf("error getting tables from query: %v", err)
	}

	// Validate against allowed tables.
	for _, table := range tables {
		if _, ok := allowedTables[table]; !ok {
			return fmt.Errorf("table '%s' is not allowed", table)
		}
	}

	return nil
}

// getTablesFromQueryPlan parses the EXPLAIN JSON to find all "Relation Name" entries.
func getTablesFromQueryPlan(explainJSON string) ([]string, error) {
	var plans []map[string]any
	if err := json.Unmarshal([]byte(explainJSON), &plans); err != nil {
		return nil, err
	}

	// Collect table names in `tables` recursively.
	tables := make(map[string]struct{})
	for _, plan := range plans {
		traverseQueryPlan(plan, tables)
	}

	result := make([]string, 0, len(tables))
	for table := range tables {
		result = append(result, table)
	}
	return result, nil
}

func traverseQueryPlan(node map[string]any, tables map[string]struct{}) {
	if relName, ok := node["Relation Name"].(string); ok {
		tables[relName] = struct{}{}
	}

	// Recursively check nested plans (e.g., subqueries, CTEs).
	for _, v := range node {
		switch v := v.(type) {
		case map[string]any:
			traverseQueryPlan(v, tables)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					traverseQueryPlan(m, tables)
				}
			}
		}
	}
}
