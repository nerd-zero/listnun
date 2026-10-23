package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/models"
)

const (
	// listActivityWindow is how far back the Lists page counts joins and
	// leaves ("+412 in 30 days").
	listActivityWindow = 30 * 24 * time.Hour

	// listRecentCampaigns is how many campaigns a list's panel shows.
	listRecentCampaigns = 4

	// optinResendGap is how long a bulk opt-in resend skips someone who
	// was sent a confirmation, so repeat clicks don't email people twice
	// in a day.
	optinResendGap = 24 * time.Hour

	// optinResendBatch is how many people a bulk resend loads at a time.
	optinResendBatch = 500
)

// optinResendReq is the body of a bulk opt-in resend.
type optinResendReq struct {
	// ListIDs limits the resend to these lists. Empty means every list.
	ListIDs []int `json:"list_ids"`
}

// optinResendResp says how many people a bulk resend is emailing.
type optinResendResp struct {
	Queued int `json:"queued"`
}

// GetListsGrowth returns joins and leaves per list over the last 30 days,
// for the lists the user can see.
//
//	@ID			getListsGrowth
//	@Summary	Get list growth over the last 30 days
//	@Tags		lists
//	@Produce	json
//	@Success	200	{array}		models.ListGrowth
//	@Failure	500	{object}	echo.HTTPError
//	@Router		/api/lists/growth [get]
func (a *App) GetListsGrowth(c echo.Context) error {
	user := auth.GetUser(c)

	all, err := a.core.GetListsGrowth(c.Request().Context(), tenantID(c), time.Now().Add(-listActivityWindow))
	if err != nil {
		return err
	}

	out := make([]models.ListGrowth, 0, len(all))
	for _, g := range all {
		if user.HasListPerm(auth.PermTypeGet, g.ListID) == nil {
			out = append(out, g)
		}
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetListActivity returns a list's daily joins and leaves over the last
// 30 days and its latest campaigns.
//
//	@ID			getListActivity
//	@Summary	Get a list's recent activity
//	@Tags		lists
//	@Produce	json
//	@Param		id	path		int	true	"List ID"
//	@Success	200	{object}	models.ListActivity
//	@Failure	400	{object}	echo.HTTPError
//	@Router		/api/lists/{id}/activity [get]
func (a *App) GetListActivity(c echo.Context) error {
	user := auth.GetUser(c)
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeGet, id); err != nil {
		return err
	}

	out, err := a.core.GetListActivity(c.Request().Context(), tenantID(c), id,
		time.Now().Add(-listActivityWindow), listRecentCampaigns)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetOptinPending counts the people who haven't confirmed a double opt-in
// subscription, and how many of them a resend would email now.
//
//	@ID			getOptinPending
//	@Summary	Count unconfirmed double opt-in subscribers
//	@Tags		lists
//	@Produce	json
//	@Success	200	{object}	models.OptinPending
//	@Failure	500	{object}	echo.HTTPError
//	@Router		/api/lists/optin/pending [get]
func (a *App) GetOptinPending(c echo.Context) error {
	out, err := a.core.CountOptinPending(c.Request().Context(), tenantID(c), nil, time.Now().Add(-optinResendGap))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// ResendOptins emails an opt-in confirmation to everyone waiting to confirm
// a double opt-in subscription, skipping anyone sent one in the last day.
// The emails go out in the background; the response says how many.
//
//	@ID			resendOptins
//	@Summary	Resend opt-in confirmations
//	@Tags		lists
//	@Accept		json
//	@Produce	json
//	@Param		body	body		optinResendReq	false	"Lists to limit the resend to"
//	@Success	200		{object}	optinResendResp
//	@Failure	409		{object}	echo.HTTPError
//	@Router		/api/lists/optin/resend [post]
func (a *App) ResendOptins(c echo.Context) error {
	var req optinResendReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	tID := tenantID(c)
	dueBefore := time.Now().Add(-optinResendGap)
	n, err := a.core.CountOptinPending(c.Request().Context(), tID, req.ListIDs, dueBefore)
	if err != nil {
		return err
	}
	if n.Due == 0 {
		return c.JSON(http.StatusOK, okResp{optinResendResp{}})
	}

	if _, running := a.optinResends.LoadOrStore(tID, true); running {
		return echo.NewHTTPError(http.StatusConflict, a.i18n.T("lists.optinResend.running"))
	}
	go func() {
		defer a.optinResends.Delete(tID)
		a.resendOptins(context.Background(), tID, req.ListIDs, dueBefore)
	}()

	return c.JSON(http.StatusOK, okResp{optinResendResp{Queued: n.Due}})
}

// resendOptins sends the opt-in confirmations for ResendOptins, a batch at
// a time. Each person is marked as sent, so a failed send doesn't stop the
// rest and isn't retried in a loop: it's picked up by the next resend.
func (a *App) resendOptins(ctx context.Context, tID int, listIDs []int, dueBefore time.Time) {
	sent := 0
	for {
		subs, err := a.core.GetOptinDue(ctx, tID, listIDs, dueBefore, optinResendBatch)
		if err != nil || len(subs) == 0 {
			break
		}
		for _, sub := range subs {
			if err := a.core.MarkOptinSent(ctx, tID, sub.ID); err != nil {
				a.log.Printf("optin resend: error marking subscriber %d: %v", sub.ID, err)
				return
			}
			if _, err := a.fnOptinNotify(sub, listIDs); err != nil {
				a.log.Printf("optin resend: error sending to subscriber %d: %v", sub.ID, err)
				continue
			}
			sent++
		}
	}
	a.log.Printf("optin resend: sent %d confirmations for tenant %d", sent, tID)
}
