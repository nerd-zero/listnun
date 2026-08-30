package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/internal/scrub"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

const (
	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// subQueryReq is a "catch all" struct for reading various
// subscriber related requests.
type subQueryReq struct {
	Search             string `json:"search"`
	Query              string `json:"query"`
	ListIDs            []int  `json:"list_ids"`
	TargetListIDs      []int  `json:"target_list_ids"`
	SubscriberIDs      []int  `json:"ids"`
	Action             string `json:"action"`
	Status             string `json:"status"`
	SubscriptionStatus string `json:"subscription_status"`
	All                bool   `json:"all"`
} // @name SubscriberQueryReq

// subOptin contains the data that's passed to the double opt-in e-mail template.
type subOptin struct {
	models.Subscriber

	OptinURL string
	UnsubURL string
	Lists    []models.List
}

var (
	dummySubscriber = models.Subscriber{
		Email:   "demo@listmonk.app",
		Name:    "Demo Subscriber",
		UUID:    dummyUUID,
		Attribs: models.JSON{"city": "Bengaluru"},
	}
)

// GetSubscriber handles the retrieval of a single subscriber by ID.
//
//	@ID			getSubscriber
//	@Summary		Get subscriber
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int	true	"Subscriber ID"
//	@Success		200	{object}	models.Subscriber
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id} [get]
func (a *App) GetSubscriber(c echo.Context) error {
	user := auth.GetUser(c)

	// Check if the user has access to at least one of the lists on the subscriber.
	id := getID(c)
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, []int{id}); err != nil {
		return err
	}

	// Fetch the subscriber from the DB.
	out, err := a.core.GetSubscriber(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	maskRestrictedSubLists(user, &out)

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberActivity handles the retrieval of a subscriber's campaign views and link clicks.
//
//	@ID			getSubscriberActivity
//	@Summary		Get subscriber activity
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int	true	"Subscriber ID"
//	@Success		200	{object}	models.SubscriberActivity
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/activity [get]
func (a *App) GetSubscriberActivity(c echo.Context) error {
	user := auth.GetUser(c)

	// Check if the user has access to at least one of the lists on the subscriber.
	id := getID(c)
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, []int{id}); err != nil {
		return err
	}

	// Fetch the subscriber activity from the DB.
	out, err := a.core.GetSubscriberActivity(c.Request().Context(), tenantID(c), id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// QuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
//
//	@ID			listSubscribers
//	@Summary		Query subscribers
//	@Tags			subscribers
//	@Produce		json
//	@Param			query			query		string	false	"Raw SQL expression to filter subscribers"
//	@Param			search			query		string	false	"Search query"
//	@Param			list_id			query		[]int	false	"List IDs to filter by"	collectionFormat(multi)
//	@Param			subscription_status	query	string	false	"Subscription status filter"
//	@Param			order_by		query		string	false	"Column to order by"
//	@Param			order			query		string	false	"Sort order (asc, desc)"
//	@Param			page			query		int		false	"Page number"
//	@Param			per_page		query		int		false	"Results per page"
//	@Success		200				{object}	models.PageResults
//	@Failure		400				{object}	echo.HTTPError
//	@Router			/api/subscribers [get]
func (a *App) QuerySubscribers(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Filter list IDs by permission.
	listIDs, err := a.filterListQueryByPerm("list_id", c.QueryParams(), user)
	if err != nil {
		return err
	}

	// Does the user have the subscribers:sql_query permission?
	query := formatSQLExp(c.FormValue("query"))
	if query != "" {
		if !user.HasPerm(auth.PermSubscribersSqlQuery) {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
		}
	}

	var (
		searchStr = strings.TrimSpace(c.FormValue("search"))
		subStatus = c.FormValue("subscription_status")
		order     = c.FormValue("order")
		orderBy   = c.FormValue("order_by")
		pg        = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Query subscribers from the DB.
	res, total, err := a.core.QuerySubscribers(c.Request().Context(), tenantID(c), searchStr, query, listIDs, subStatus, order, orderBy, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	for i := range res {
		maskRestrictedSubLists(user, &res[i])
	}

	out := models.PageResults{
		Query:   query,
		Search:  searchStr,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ExportSubscribers handles querying subscribers based on an arbitrary SQL expression.
//
//	@ID			exportSubscribers
//	@Summary		Export subscribers as CSV
//	@Tags			subscribers
//	@Produce		text/csv
//	@Param			query				query	string	false	"Raw SQL expression to filter"
//	@Param			search				query	string	false	"Search query"
//	@Param			list_id				query	[]int	false	"List IDs"	collectionFormat(multi)
//	@Param			subscription_status	query	string	false	"Subscription status"
//	@Param			id					query	[]int	false	"Specific subscriber IDs"	collectionFormat(multi)
//	@Success		200	{string}	string	"CSV file download"
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/subscribers/export [get]
func (a *App) ExportSubscribers(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Filter list IDs by permission.
	listIDs, err := a.filterListQueryByPerm("list_id", c.QueryParams(), user)
	if err != nil {
		return err
	}

	// Export only specific subscriber IDs?
	subIDs, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Filter by subscription status
	subStatus := c.QueryParam("subscription_status")

	// Does the user have the subscribers:sql_query permission?
	var (
		searchStr = strings.TrimSpace(c.FormValue("search"))
		query     = formatSQLExp(c.FormValue("query"))
	)
	if query != "" {
		if !user.HasPerm(auth.PermSubscribersSqlQuery) {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
		}
	}

	// Get the batched export iterator.
	exp, err := a.core.ExportSubscribers(c.Request().Context(), tenantID(c), searchStr, query, subIDs, listIDs, subStatus, a.cfg.DBBatchSize)
	if err != nil {
		return err
	}

	var (
		hdr = c.Response().Header()
		wr  = csv.NewWriter(c.Response())
	)

	hdr.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	hdr.Set("Content-type", "text/csv")
	hdr.Set(echo.HeaderContentDisposition, "attachment; filename="+"subscribers.csv")
	hdr.Set("Content-Transfer-Encoding", "binary")
	hdr.Set("Cache-Control", "no-cache")
	wr.Write([]string{"uuid", "email", "name", "attributes", "status", "created_at", "updated_at"})

loop:
	// Iterate in batches until there are no more subscribers to export.
	for {
		out, err := exp()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			break
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				a.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}

		// Flush CSV to stream after each batch.
		wr.Flush()
	}

	return nil
}

// CreateSubscriber handles the creation of a new subscriber.
//
//	@ID			createSubscriber
//	@Summary		Create subscriber
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			subscriber	body		subimporter.SubReq	true	"Subscriber details"
//	@Success		200			{object}	models.Subscriber
//	@Failure		400			{object}	echo.HTTPError
//	@Failure		403			{object}	echo.HTTPError
//	@Router			/api/subscribers [post]
func (a *App) CreateSubscriber(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get and validate fields.
	var req subimporter.SubReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate fields.
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	req, err = imp.ValidateFields(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeManage, req.Lists)

	// Not a single permitted list?
	if len(req.Lists) > 0 && len(listIDs) == 0 {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", "lists"))
	}

	ctx := c.Request().Context()
	tID := tenantID(c)

	// Validate against Scrub, if configured for this tenant. invalid_syntax/
	// undeliverable are rejected outright; risky is inserted but flagged.
	scrubStatus := ""
	if s, err := a.core.GetSettings(ctx, tID); err == nil {
		status, reject := validateEmailForAdd(ctx, s, req.Email)
		if reject {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidEmail"))
		}
		scrubStatus = status
	}

	// Insert the subscriber into the DB.
	sub, _, err := a.core.InsertSubscriber(ctx, tID, req.Subscriber, listIDs, nil, req.PreconfirmSubs, false)
	if err != nil {
		return err
	}

	if scrubStatus != "" {
		if err := a.core.SetSubscriberScrubStatus(ctx, tID, sub.ID, scrubStatus); err != nil {
			a.log.Printf("error setting scrub status on subscriber %d: %v", sub.ID, err)
		}
		if scrubStatus == scrub.StatusRisky {
			a.pauseCampaignsForRiskySubscriber(ctx, tID, listIDs)
		}
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// UpdateSubscriber handles modification of a subscriber.
//
//	@ID			updateSubscriber
//	@Summary		Update subscriber
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int					true	"Subscriber ID"
//	@Param			subscriber	body		models.Subscriber	true	"Subscriber fields to update"
//	@Success		200			{object}	models.Subscriber
//	@Failure		400			{object}	echo.HTTPError
//	@Failure		403			{object}	echo.HTTPError
//	@Failure		404			{object}	echo.HTTPError
//	@Router			/api/subscribers/{id} [put]
func (a *App) UpdateSubscriber(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get and validate fields.
	req := struct {
		models.Subscriber
		Lists          []int `json:"lists"`
		PreconfirmSubs bool  `json:"preconfirm_subscriptions"`
	}{}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Sanitize and validate the email field.
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	if em, err := imp.SanitizeEmail(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req.Email = em
	}

	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidName"))
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeManage, req.Lists)

	// Not a single permitted list?
	if len(req.Lists) > 0 && len(listIDs) == 0 {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", "lists"))
	}

	// Update the subscriber in the DB.
	id := getID(c)

	// Get the user's permitted lists to pass to the update query so that lists on the subscribers
	// to which they don't have permissions are preserved/left as-is when deleteLists=true.
	allPerm, permittedLists := user.GetPermittedLists(auth.PermTypeManage)
	if allPerm {
		permittedLists = []int{}
	}

	out, _, err := a.core.UpdateSubscriberWithLists(c.Request().Context(), tenantID(c), id, req.Subscriber, listIDs, nil, req.PreconfirmSubs, true, false, permittedLists, false)
	if err != nil {
		return err
	}

	maskRestrictedSubLists(user, &out)

	return c.JSON(http.StatusOK, okResp{out})
}

// PatchSubscriber handles partially modifying a subscriber.
// Only fields present in the request body are updated.
//
//	@ID			patchSubscriber
//	@Summary		Partially update subscriber
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int					true	"Subscriber ID"
//	@Param			subscriber	body		models.Subscriber	true	"Fields to patch"
//	@Success		200			{object}	models.Subscriber
//	@Failure		400			{object}	echo.HTTPError
//	@Failure		403			{object}	echo.HTTPError
//	@Router			/api/subscribers/{id} [patch]
func (a *App) PatchSubscriber(c echo.Context) error {
	user := auth.GetUser(c)
	id := getID(c)

	// Fetch the sub subscriber from the DB.
	sub, err := a.core.GetSubscriber(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	// Prepopulate the incoming request struct with existing values.
	// Rather than tediously and conditionally checking each incoming field, we can simply
	// overwrite everything in the DB with the incoming fields+existing fields.
	req := struct {
		models.Subscriber
		Lists          *[]int `json:"lists"`
		PreconfirmSubs bool   `json:"preconfirm_subscriptions"`
	}{
		Subscriber: sub,
	}

	if err := c.Bind(&req); err != nil {
		return err
	}

	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	if em, err := imp.SanitizeEmail(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req.Email = em
	}

	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidName"))
	}

	// If lists were explicitly sent, replace the existing subscriptions.
	overwriteSubs := false
	var listIDs []int
	if req.Lists != nil {
		overwriteSubs = true
		listIDs = user.FilterListsByPerm(auth.PermTypeManage, *req.Lists)
		if len(*req.Lists) > 0 && len(listIDs) == 0 {
			return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", "lists"))
		}
	}

	allPerm, permittedLists := user.GetPermittedLists(auth.PermTypeManage)
	if allPerm {
		permittedLists = []int{}
	}

	out, _, err := a.core.UpdateSubscriberWithLists(c.Request().Context(), tenantID(c), id, req.Subscriber, listIDs, nil, req.PreconfirmSubs, overwriteSubs, false, permittedLists, false)
	if err != nil {
		return err
	}

	maskRestrictedSubLists(user, &out)

	return c.JSON(http.StatusOK, okResp{out})
}

// SubscriberSendOptin sends an optin confirmation e-mail to a subscriber.
//
//	@ID			sendSubscriberOptin
//	@Summary		Send opt-in email
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int		true	"Subscriber ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/optin [post]
func (a *App) SubscriberSendOptin(c echo.Context) error {
	user := auth.GetUser(c)

	// Fetch the subscriber.
	id := getID(c)
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, []int{id}); err != nil {
		return err
	}

	out, err := a.core.GetSubscriber(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	// Trigger the opt-in confirmation e-mail hook.
	if _, err := a.fnOptinNotify(out, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistSubscriber handles the blocklisting of a given subscriber.
//
//	@ID			blocklistSubscriber
//	@Summary		Blocklist subscriber
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int		true	"Subscriber ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/blocklist [put]
func (a *App) BlocklistSubscriber(c echo.Context) error {
	user := auth.GetUser(c)

	// Update the subscribers in the DB.
	id := getID(c)
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, []int{id}); err != nil {
		return err
	}

	if err := a.core.BlocklistSubscribers(c.Request().Context(), tenantID(c), []int{id}); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistSubscribers handles the blocklisting of one or more subscribers.
//
//	@ID			blocklistSubscribers
//	@Summary		Bulk blocklist subscribers
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			req	body		subQueryReq	true	"Subscriber IDs"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/blocklist [put]
func (a *App) BlocklistSubscribers(c echo.Context) error {
	user := auth.GetUser(c)

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", "ids"))
	}

	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, req.SubscriberIDs); err != nil {
		return err
	}

	// Update the subscribers in the DB.
	if err := a.core.BlocklistSubscribers(c.Request().Context(), tenantID(c), req.SubscriberIDs); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
//
//	@ID			manageSubscriberLists
//	@Summary		Manage subscriber list subscriptions
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			req	body		subQueryReq	true	"Action and target list IDs"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/lists [put]
func (a *App) ManageSubscriberLists(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Is it an /:id call?
	var (
		pID    = c.Param("id")
		subIDs []int
	)
	if pID != "" {
		id, _ := strconv.Atoi(pID)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}
		subIDs = append(subIDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.errorNoIDs"))
	}
	if len(subIDs) == 0 {
		subIDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.errorNoListsGiven"))
	}

	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, subIDs); err != nil {
		return err
	}

	// Filter lists against the current user's permitted lists.
	listIDs := user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, req.TargetListIDs)

	// User doesn't have the required list permissions.
	if len(listIDs) == 0 {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", "lists"))
	}

	// Run the action in the DB.
	ctx := c.Request().Context()
	tID := tenantID(c)

	var err error
	switch req.Action {
	case "add":
		err = a.core.AddSubscriptions(ctx, tID, subIDs, listIDs, req.Status)
	case "remove":
		err = a.core.DeleteSubscriptions(ctx, tID, subIDs, listIDs)
	case "unsubscribe":
		err = a.core.UnsubscribeLists(ctx, tID, subIDs, listIDs, nil)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	// Linking existing subscribers to a list can't itself make them risky --
	// they're already-known subscribers, so just check whether any of them
	// are already flagged risky and auto-pause accordingly, rather than
	// calling Scrub again for emails that haven't changed.
	if req.Action == "add" {
		if risky, err := a.core.GetRiskySubscriberIDs(ctx, tID, subIDs); err == nil && len(risky) > 0 {
			a.pauseCampaignsForRiskySubscriber(ctx, tID, listIDs)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscriber handles deletion of a single subscriber.
//
//	@ID			deleteSubscriber
//	@Summary		Delete subscriber
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int		true	"Subscriber ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id} [delete]
func (a *App) DeleteSubscriber(c echo.Context) error {
	user := auth.GetUser(c)

	// Delete the subscribers from the DB.
	id := getID(c)
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, []int{id}); err != nil {
		return err
	}

	if err := a.core.DeleteSubscribers(c.Request().Context(), tenantID(c), []int{id}, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscribers handles bulk deletion of one or more subscribers.
//
//	@ID			deleteSubscribers
//	@Summary		Bulk delete subscribers
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	query		[]int	true	"Subscriber IDs"	collectionFormat(multi)
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers [delete]
func (a *App) DeleteSubscribers(c echo.Context) error {
	user := auth.GetUser(c)

	// Multiple IDs.
	ids, err := parseStringIDs(c.Request().URL.Query()["id"])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}
	if len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", "ids"))
	}

	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, ids); err != nil {
		return err
	}

	// Delete the subscribers from the DB.
	if err := a.core.DeleteSubscribers(c.Request().Context(), tenantID(c), ids, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscribersByQuery bulk deletes based on an
// arbitrary SQL expression.
//
//	@ID			deleteSubscribersByQuery
//	@Summary		Delete subscribers by query
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			req	body		subQueryReq	true	"Query and options"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/subscribers/query/delete [post]
func (a *App) DeleteSubscribersByQuery(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	req.Search = strings.TrimSpace(req.Search)
	req.Query = formatSQLExp(req.Query)
	if req.All {
		// If the "all" flag is set, ignore any subquery that may be present.
		req.Search = ""
		req.Query = ""
	} else if req.Search == "" && req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}

	// Does the user have the subscribers:sql_query permission?
	if req.Query != "" {
		if !user.HasPerm(auth.PermSubscribersSqlQuery) {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
		}
	}

	// Filter list IDs against the current user's permitted lists.
	listIDs := user.GetPermittedListIDs(req.ListIDs)

	// Delete the subscribers from the DB.
	if err := a.core.DeleteSubscribersByQuery(c.Request().Context(), tenantID(c), req.Search, req.Query, listIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistSubscribersByQuery bulk blocklists subscribers
// based on an arbitrary SQL expression.
//
//	@ID			blocklistSubscribersByQuery
//	@Summary		Blocklist subscribers by query
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			req	body		subQueryReq	true	"Query and options"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/subscribers/query/blocklist [put]
func (a *App) BlocklistSubscribersByQuery(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	req.Search = strings.TrimSpace(req.Search)
	req.Query = formatSQLExp(req.Query)
	if req.All {
		// If the "all" flag is set, ignore any subquery that may be present.
		req.Search = ""
		req.Query = ""
	} else if req.Search == "" && req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "query"))
	}
	// Does the user have the subscribers:sql_query permission?
	if req.Query != "" {
		if !user.HasPerm(auth.PermSubscribersSqlQuery) {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
		}
	}

	// Filter list IDs against the current user's permitted lists.
	listIDs := user.GetPermittedListIDs(req.ListIDs)

	// Update the subscribers in the DB.
	if err := a.core.BlocklistSubscribersByQuery(c.Request().Context(), tenantID(c), req.Search, req.Query, listIDs, req.SubscriptionStatus); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ManageSubscriberListsByQuery bulk adds/removes/unsubscribes subscribers
// from one or more lists based on an arbitrary SQL expression.
//
//	@ID			manageSubscriberListsByQuery
//	@Summary		Manage subscriber lists by query
//	@Tags			subscribers
//	@Accept			json
//	@Produce		json
//	@Param			req	body		subQueryReq	true	"Query, action, and target lists"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/subscribers/query/lists [put]
func (a *App) ManageSubscriberListsByQuery(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.T("subscribers.errorNoListsGiven"))
	}

	req.Search = strings.TrimSpace(req.Search)
	req.Query = formatSQLExp(req.Query)

	// Does the user have the subscribers:sql_query permission?
	if req.Query != "" {
		if !user.HasPerm(auth.PermSubscribersSqlQuery) {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", auth.PermSubscribersSqlQuery))
		}
	}

	// Filter lists against the current user's permitted lists.
	sourceListIDs := user.GetPermittedListIDs(req.ListIDs)
	targetListIDs := user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, req.TargetListIDs)

	// Run the action in the DB.
	var err error
	switch req.Action {
	case "add":
		err = a.core.AddSubscriptionsByQuery(c.Request().Context(), tenantID(c), req.Search, req.Query, sourceListIDs, targetListIDs, req.Status, req.SubscriptionStatus)
	case "remove":
		err = a.core.DeleteSubscriptionsByQuery(c.Request().Context(), tenantID(c), req.Search, req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	case "unsubscribe":
		err = a.core.UnsubscribeListsByQuery(c.Request().Context(), tenantID(c), req.Search, req.Query, sourceListIDs, targetListIDs, req.SubscriptionStatus)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteSubscriberBounces deletes all the bounces on a subscriber.
//
//	@ID			deleteSubscriberBounces
//	@Summary		Delete subscriber bounces
//	@Tags			subscribers
//	@Produce		json
//	@Param			id	path		int		true	"Subscriber ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/bounces [delete]
func (a *App) DeleteSubscriberBounces(c echo.Context) error {
	// Delete the bounces from the DB.
	id := getID(c)
	if err := a.core.DeleteSubscriberBounces(c.Request().Context(), tenantID(c), id, ""); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in a.Constants.Privacy.
//
//	@ID			exportSubscriberData
//	@Summary		Export subscriber data (privacy)
//	@Tags			subscribers
//	@Produce		application/json
//	@Param			id	path		int		true	"Subscriber ID"
//	@Success		200	{object}	models.SubscriberExportProfile
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		403	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/export [get]
func (a *App) ExportSubscriberData(c echo.Context) error {
	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	id := getID(c)

	// Check if the user has access to at least one of the lists on the subscriber.
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), auth.GetUser(c), []int{id}); err != nil {
		return err
	}

	_, b, err := a.exportSubscriberData(c.Request().Context(), tenantID(c), id, "", a.cfg.Privacy.Exportable)
	if err != nil {
		a.log.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	// Set headers to force the browser to prompt for download.
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="data.json"`)
	return c.Blob(http.StatusOK, "application/json", b)
}

// exportSubscriberData collates the data of a subscriber including profile,
// subscriptions, campaign_views, link_clicks (if they're enabled in the config)
// and returns a formatted, indented JSON payload. Either takes a numeric id
// and an empty subUUID or takes 0 and a string subUUID.
func (a *App) exportSubscriberData(ctx context.Context, tenantID int, id int, subUUID string, exportables map[string]bool) (models.SubscriberExportProfile, []byte, error) {
	data, err := a.core.GetSubscriberProfileForExport(ctx, tenantID, id, subUUID)
	if err != nil {
		return data, nil, err
	}

	// Filter out the non-exportable items.
	if _, ok := exportables["profile"]; !ok {
		data.Profile = nil
	}
	if _, ok := exportables["subscriptions"]; !ok {
		data.Subscriptions = nil
	}
	if _, ok := exportables["campaign_views"]; !ok {
		data.CampaignViews = nil
	}
	if _, ok := exportables["link_clicks"]; !ok {
		data.LinkClicks = nil
	}

	// Marshal the data into an indented payload.
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		a.log.Printf("error marshalling subscriber export data: %v", err)
		return data, nil, err
	}

	return data, b, nil
}

// maskRestrictedSubLists replaces list names with "*Unknown" for lists
// the user doesn't have read access to. This appears on the subscriber
// details UI and prevents users without access to certain lists from seeing their names.
func maskRestrictedSubLists(user auth.User, sub *models.Subscriber) {
	if user.HasPerm(auth.PermListManageAll) || user.HasPerm(auth.PermListGetAll) {
		return
	}

	// Hacky JSON manipulation (for now).
	var lists []map[string]interface{}
	if err := json.Unmarshal(sub.Lists, &lists); err != nil || len(lists) == 0 {
		return
	}

	for i, l := range lists {
		id, _ := l["id"].(float64)
		if user.HasListPerm(auth.PermTypeGet, int(id)) != nil &&
			user.HasListPerm(auth.PermTypeManage, int(id)) != nil {
			lists[i]["name"] = "*Unknown"
			lists[i]["restricted"] = true
			delete(lists[i], "description")
		}
	}

	if b, err := json.Marshal(lists); err == nil {
		sub.Lists = b
	}
}

// hasSubPerm checks whether the current user has permission to access the given list
// of subscriber IDs.
func (a *App) hasSubPerm(ctx context.Context, tenantID int, u auth.User, subIDs []int) error {
	allPerm, listIDs := u.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)

	// User has blanket get_all|manage_all permission.
	if allPerm {
		return nil
	}

	// Check whether the subscribers have the list IDs permitted to the user.
	res, err := a.core.HasSubscriberLists(ctx, tenantID, subIDs, listIDs)
	if err != nil {
		return err
	}

	for id, has := range res {
		if !has {
			return echo.NewHTTPError(http.StatusForbidden, a.i18n.Ts("globals.messages.permissionDenied", "name", fmt.Sprintf("subscriber: %d", id)))
		}
	}

	return nil
}

// filterListQueryByPerm filters the list IDs in the query params and returns the list IDs to which the user has access.
func (a *App) filterListQueryByPerm(param string, qp url.Values, user auth.User) ([]int, error) {
	var listIDs []int

	// If there are incoming list query params, filter them by permission.
	if qp.Has(param) {
		ids, err := getQueryInts(param, qp)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
		}

		listIDs = ids
	}

	return user.GetPermittedListIDs(listIDs), nil
}

// formatSQLExp does basic sanitisation on arbitrary
// SQL query expressions coming from the frontend.
func formatSQLExp(q string) string {
	q = strings.TrimSpace(q)
	if len(q) == 0 {
		return ""
	}

	// Remove semicolon suffix.
	if q[len(q)-1] == ';' {
		q = q[:len(q)-1]
	}
	return q
}

// makeOptinNotifyHook returns an enclosed callback that sends optin confirmation e-mails.
// This is plugged into the 'core' package to send optin confirmations when a new subscriber is
// created via `core.CreateSubscriber()`.
func makeOptinNotifyHook(unsubHeader bool, u *UrlConfig, q *models.Queries, i *i18n.I18n) func(sub models.Subscriber, listIDs []int) (int, error) {
	return func(sub models.Subscriber, listIDs []int) (int, error) {
		// Fetch double opt-in lists from the given list IDs.
		// Get the list of subscription lists where the subscriber hasn't confirmed.
		var lists = []models.List{}
		if err := q.GetSubscriberLists.Select(&lists, sub.ID, nil, pq.Array(listIDs), nil, models.SubscriptionStatusUnconfirmed, models.ListOptinDouble); err != nil {
			lo.Printf("error fetching lists for opt-in: %s", err)
			return 0, err
		}

		// None.
		if len(lists) == 0 {
			return 0, nil
		}

		var (
			out      = subOptin{Subscriber: sub, Lists: lists}
			qListIDs = url.Values{}
		)

		// Construct the opt-in URL with list IDs.
		for _, l := range out.Lists {
			qListIDs.Add("l", l.UUID)
		}
		out.OptinURL = fmt.Sprintf(u.OptinURL, sub.UUID, qListIDs.Encode())
		out.UnsubURL = fmt.Sprintf(u.UnsubURL, dummyUUID, sub.UUID)

		// Unsub headers.
		hdr := textproto.MIMEHeader{}
		hdr.Set(models.EmailHeaderSubscriberUUID, sub.UUID)

		// Attach List-Unsubscribe headers?
		if unsubHeader {
			unsubURL := fmt.Sprintf(u.UnsubURL, dummyUUID, sub.UUID)
			hdr.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
			hdr.Set("List-Unsubscribe", `<`+unsubURL+`>`)
		}

		// Send the e-mail.
		if err := notifs.Notify([]string{sub.Email}, i.T("subscribers.optinSubject"), notifs.TplSubscriberOptin, out, hdr); err != nil {
			lo.Printf("error sending opt-in e-mail for subscriber %d (%s): %s", sub.ID, sub.UUID, err)
			return 0, err
		}

		return len(lists), nil
	}
}
