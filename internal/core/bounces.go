package core

import (
	"context"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

var bounceQuerySortFields = []string{"email", "campaign_name", "source", "created_at", "type"}

// QueryBounces retrieves paginated bounce entries based on the given params.
// It also returns the total number of bounce records in the DB.
func (c *Core) QueryBounces(ctx context.Context, tenantID int, f models.BounceFilter, orderBy, order string, offset, limit int) ([]models.Bounce, int, error) {
	if !strSliceContains(orderBy, bounceQuerySortFields) {
		orderBy = "created_at"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	out := []models.Bounce{}
	stmt := strings.ReplaceAll(c.q.QueryBounces, "%order%", orderBy+" "+order)
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return tx.Select(&out, stmt, 0, f.CampaignID, f.SubscriberID, f.Source, f.Type, f.Search, offset, limit)
	})
	if err != nil {
		c.log.Printf("error fetching bounces: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.bounce}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetBounceTypeCounts returns the number of bounces per type matching the
// filter. The filter's Type and SubscriberID are ignored.
func (c *Core) GetBounceTypeCounts(ctx context.Context, tenantID int, f models.BounceFilter) (models.BounceTypeCounts, error) {
	var out models.BounceTypeCounts
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return tx.Get(&out, c.q.GetBounceTypeCounts, f.CampaignID, f.Source, f.Search)
	})
	if err != nil {
		c.log.Printf("error fetching bounce counts: %v", err)
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.bounce}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetBounce retrieves bounce entries based on the given params.
func (c *Core) GetBounce(ctx context.Context, tenantID int, id int) (models.Bounce, error) {
	var out []models.Bounce
	stmt := strings.ReplaceAll(c.q.QueryBounces, "%order%", "id "+SortAsc)
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return tx.Select(&out, stmt, id, 0, 0, "", "", "", 0, 1)
	})
	if err != nil {
		c.log.Printf("error fetching bounces: %v", err)
		return models.Bounce{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.bounce}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Bounce{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.bounce}"))

	}

	return out[0], nil
}

// RecordBounce records a new bounce.
//
// This is called from bounce webhook/POP3-polling paths (internal/bounce)
// which have no request-scoped tenant to resolve - it does NOT go through
// WithTenant. queries/bounces.sql's record-bounce derives the inserted
// row's tenant_id from the resolved subscriber's own tenant_id instead of a
// session variable; see that query's comment.
//
// When b.TenantID is set (webhooks), the lookup is limited to that tenant
// and that tenant's own bounce.actions decide the count and action
// (blocklist, unsubscribe, delete). Otherwise the boot-time actions apply.
func (c *Core) RecordBounce(b models.Bounce) error {
	count, act, ok := c.bounceAction(b.TenantID, b.Type)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.invalidData")+": "+b.Type)
	}

	// Transactional email to addresses that aren't subscribers bounces too,
	// so the message log is updated before the subscriber lookup below.
	if b.CampaignUUID == "" {
		c.markTxBounced(b)
	}

	_, err := c.q.RecordBounce.Exec(b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta,
		b.CreatedAt,
		count,
		act,
		b.TenantID)

	if err != nil {
		// Ignore the error if it complained of no subscriber.
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "subscriber_id" {
			c.log.Printf("bounced subscriber (%s / %s) not found", b.SubscriberUUID, b.Email)
			return nil
		}

		c.log.Printf("error recording bounce: %v", err)
	}

	return err
}

// bounceAction returns the bounce count threshold and action for a bounce
// type. A tenant's own bounce.actions win; the boot-time constants (tenant
// 1's settings) are the fallback when there's no tenant, its settings can't
// be read, or it has no entry for the type.
func (c *Core) bounceAction(tenantID int, typ string) (int, string, bool) {
	if tenantID > 0 {
		if s, err := c.GetSettings(context.Background(), tenantID); err != nil {
			c.log.Printf("error reading bounce actions for tenant %d, using defaults: %v", tenantID, err)
		} else if a, ok := s.BounceActions[typ]; ok {
			return a.Count, a.Action, true
		}
	}

	a, ok := c.consts.BounceActions[typ]
	return a.Count, a.Action, ok
}

// BlocklistBouncedSubscribers blocklists all bounced subscribers.
func (c *Core) BlocklistBouncedSubscribers(ctx context.Context, tenantID int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.BlocklistBouncedSubscribers).Exec()
		return err
	})
	if err != nil {
		c.log.Printf("error blocklisting bounced subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
	}

	return nil
}

// DeleteBounce deletes a list.
func (c *Core) DeleteBounce(ctx context.Context, tenantID int, id int) error {
	return c.DeleteBounces(ctx, tenantID, []int{id}, false)
}

// DeleteBounces deletes multiple lists.
func (c *Core) DeleteBounces(ctx context.Context, tenantID int, ids []int, all bool) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteBounces).Exec(pq.Array(ids), all)
		return err
	})
	if err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return nil
}
