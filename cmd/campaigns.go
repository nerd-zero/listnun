package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"gopkg.in/volatiletech/null.v6"
)

// campReq is a wrapper over the Campaign model for receiving
// campaign creation and update data from APIs.
type campReq struct {
	models.Campaign

	// This overrides Campaign.Lists to receive and
	// write a list of int IDs during creation and updation.
	// Campaign.Lists is JSONText for sending lists children
	// to the outside world.
	ListIDs []int `json:"lists"`

	MediaIDs []int `json:"media"`

	// This is only relevant to campaign test requests.
	SubscriberEmails pq.StringArray `json:"subscribers"`
} // @name CreateCampaignReq

// campContentReq wraps params coming from API requests for converting
// campaign content formats.
type campContentReq struct {
	models.Campaign
	From string `json:"from"`
	To   string `json:"to"`
} // @name UpdateCampaignContentReq

var (
	reFromAddress = regexp.MustCompile(`((.+?)\s)?<(.+?)@(.+?)>`)
	reSlug        = regexp.MustCompile(`[^\p{L}\p{M}\p{N}]`)
)

// GetCampaigns handles retrieval of campaigns.
//
//	@ID			listCampaigns
//	@Summary		Get campaigns
//	@Tags			campaigns
//	@Produce		json
//	@Param			query		query		string		false	"Search query"
//	@Param			status		query		[]string	false	"Campaign status filter"
//	@Param			tag			query		[]string	false	"Tags"
//	@Param			order_by	query		string		false	"Order by field"
//	@Param			order		query		string		false	"Sort order (asc/desc)"
//	@Param			no_body		query		bool		false	"Omit campaign body from response"
//	@Param			page		query		int			false	"Page number"
//	@Param			per_page	query		int			false	"Results per page"
//	@Success		200	{object}	models.PageResults
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/campaigns [get]
func (a *App) GetCampaigns(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var (
		hasAllPerm     = user.HasPerm(auth.PermCampaignsGetAll)
		permittedLists []int
	)

	if !hasAllPerm {
		// Either the user has campaigns:get_all permissions and can view all campaigns,
		// or the campaigns are filtered by the lists the user has get|manage access to.
		hasAllPerm, permittedLists = user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)
	}

	var (
		pg = a.pg.NewFromURL(c.Request().URL.Query())

		status    = c.QueryParams()["status"]
		tags      = c.QueryParams()["tag"]
		query     = strings.TrimSpace(c.FormValue("query"))
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		noBody, _ = strconv.ParseBool(c.QueryParam("no_body"))
	)

	// Query and retrieve campaigns from the DB.
	res, total, err := a.core.QueryCampaigns(c.Request().Context(), tenantID(c), query, status, tags, orderBy, order, hasAllPerm, permittedLists, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	// Remove the body from the response if requested.
	if noBody {
		for i := range res {
			res[i].Body = ""
			res[i].BodySource.Valid = false
		}
	}

	// Paginate the response.
	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.Campaign{}}})
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetCampaign handles retrieval of a single campaign.
//
//	@ID			getCampaign
//	@Summary		Get a campaign
//	@Tags			campaigns
//	@Produce		json
//	@Param			id		path		int		true	"Campaign ID"
//	@Param			no_body	query		bool	false	"Omit campaign body from response"
//	@Success		200	{object}	models.Campaign
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id} [get]
func (a *App) GetCampaign(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
		return err
	}

	// Get the campaign from the DB.
	out, err := a.core.GetCampaign(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	// Blank out the body if requested.
	noBody, _ := strconv.ParseBool(c.QueryParam("no_body"))
	if noBody {
		out.Body = ""
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// PreviewCampaign renders the HTML preview of a campaign body.
//
//	@ID			getCampaignPreview
//	@Summary		Preview a campaign (GET)
//	@Tags			campaigns
//	@Accept			mpfd
//	@Produce		html
//	@Param			id				path		int		true	"Campaign ID"
//	@Param			content_type	formData	string	false	"Content type override"
//	@Param			template_id		formData	int		false	"Template ID to use for preview"
//	@Success		200	{string}	string	"Rendered HTML or plain text"
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/preview [get]

//	@ID			previewCampaign
//	@Summary		Preview a campaign (POST)
//	@Tags			campaigns
//	@Accept			mpfd
//	@Produce		html
//	@Param			id				path		int		true	"Campaign ID"
//	@Param			content_type	formData	string	false	"Content type override"
//	@Param			template_id		formData	int		false	"Template ID to use for preview"
//	@Param			body			formData	string	false	"Campaign body override"
//	@Success		200	{string}	string	"Rendered HTML or plain text"
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/preview [post]
func (a *App) PreviewCampaign(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
		return err
	}

	var (
		isPost      = c.Request().Method == http.MethodPost
		contentType = c.FormValue("content_type")
		tplID, _    = strconv.Atoi(c.FormValue("template_id"))
	)
	// For visual content, template ID for previewing is irrelevant.
	if contentType == models.CampaignContentTypeVisual || tplID < 1 {
		tplID = 0
	}

	// Get the campaign from the DB for previewing with the `template_body` field.
	camp, err := a.core.GetCampaignForPreview(c.Request().Context(), tenantID(c), id, tplID)
	if err != nil {
		return err
	}

	// There's a body in the request to preview instead of the body in the DB.
	if isPost {
		camp.ContentType = contentType
		camp.Body = c.FormValue("body")

		// For visual campaigns, template body from the DB shouldn't be used.
		if contentType == models.CampaignContentTypeVisual {
			camp.TemplateBody = ""
		}
	}

	// Use a dummy campaign ID to prevent views and clicks from {{ TrackView }}
	// and {{ TrackLink }} being registered on preview.
	camp.UUID = dummySubscriber.UUID
	if err := camp.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
		a.log.Printf("error compiling template: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	// Render the message body.
	msg, err := a.manager.NewCampaignMessage(&camp, dummySubscriber)
	if err != nil {
		a.log.Printf("error rendering message: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	// Plaintext headers for plain body.
	if camp.ContentType == models.CampaignContentTypePlain {
		return c.String(http.StatusOK, string(msg.Body()))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// PreviewCampaignArchive renders the public campaign archives page.
//
//	@ID			previewCampaignArchive
//	@Summary		Preview a campaign as an archive page
//	@Tags			campaigns
//	@Accept			mpfd
//	@Produce		html
//	@Param			id				path		int		true	"Campaign ID"
//	@Param			template_id		formData	int		false	"Template ID to use for preview"
//	@Param			archive_meta	formData	string	false	"Archive meta JSON"
//	@Success		200	{string}	string	"Rendered HTML"
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/preview/archive [post]
func (a *App) PreviewCampaignArchive(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
		return err
	}

	// Fetch the campaign body from the DB.
	tplID, _ := strconv.Atoi(c.FormValue("template_id"))
	camp, err := a.core.GetCampaignForPreview(c.Request().Context(), tenantID(c), id, tplID)
	if err != nil {
		return err
	}

	camp.ArchiveMeta = json.RawMessage([]byte(c.FormValue("archive_meta")))

	// "Compile" the campaign template with appropriate data.
	res, err := a.compileArchiveCampaigns([]models.Campaign{camp})
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	// Render the campaign body.
	out := res[0].Campaign
	msg, err := a.manager.NewCampaignMessage(out, res[0].Subscriber)
	if err != nil {
		a.log.Printf("error rendering campaign: %v", err)
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingCampaign")))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// CampaignContent handles campaign content (body) format conversions.
//
//	@ID			setCampaignContent
//	@Summary		Convert campaign content format
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Campaign ID"
//	@Param			req		body		campContentReq		true	"Content conversion request"
//	@Success		200	{object}	string
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/content [post]
func (a *App) CampaignContent(c echo.Context) error {
	var camp campContentReq
	if err := c.Bind(&camp); err != nil {
		return err
	}

	// Convert formats, eg: markdown to HTML.
	out, err := camp.ConvertContent(camp.From, camp.To)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateCampaign handles campaign creation.
// Newly created campaigns are always drafts.
//
//	@ID			createCampaign
//	@Summary		Create a campaign
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			campaign	body		campReq	true	"Campaign to create"
//	@Success		200	{object}	models.Campaign
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/campaigns [post]
func (a *App) CreateCampaign(c echo.Context) error {
	var o campReq
	if err := c.Bind(&o); err != nil {
		return err
	}

	// Filter lists against the current user's permitted lists.
	user := auth.GetUser(c)
	o.ListIDs = user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, o.ListIDs)

	// If the campaign's 'opt-in', prepare a default message.
	switch o.Type {
	case models.CampaignTypeOptin:
		op, err := a.makeOptinCampaignMessage(c.Request().Context(), tenantID(c), o)
		if err != nil {
			return err
		}
		o = op
	case "":
		o.Type = models.CampaignTypeRegular
	}

	if o.Messenger == "" {
		o.Messenger = "email"
	}

	// Validate.
	if c, err := a.validateCampaignFields(c.Request().Context(), tenantID(c), o); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		o = c
	}

	if o.ArchiveTemplateID.Valid && o.ArchiveTemplateID.Int != 0 {
		o.ArchiveTemplateID = o.TemplateID
	}

	out, err := a.core.CreateCampaign(c.Request().Context(), tenantID(c), o.Campaign, o.ListIDs, o.MediaIDs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateCampaign handles campaign modification.
// Campaigns that are done cannot be modified.
//
//	@ID			updateCampaign
//	@Summary		Update a campaign
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int			true	"Campaign ID"
//	@Param			campaign	body		campReq		true	"Campaign fields to update"
//	@Success		200	{object}	models.Campaign
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id} [put]
func (a *App) UpdateCampaign(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeManage, id, c); err != nil {
		return err
	}

	// Retrieve the campaign from the DB.
	cm, err := a.core.GetCampaign(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	if !canEditCampaign(cm.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("campaigns.cantUpdate"))
	}

	// Clear attribs to avoid merging old and new values as json.Unmarshal in JSON.scan() merges maps,
	// merging values already in the DB and incoming values. If this is nil, then DB values remain
	// unchanged.
	cm.Attribs = nil

	// Read the incoming params into the existing campaign fields from the DB.
	// This allows updating of values that have been sent whereas fields
	// that are not in the request retain the old values.
	o := campReq{Campaign: cm}
	if err := c.Bind(&o); err != nil {
		return err
	}

	// Filter lists against the current user's permitted lists.
	user := auth.GetUser(c)
	o.ListIDs = user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, o.ListIDs)

	if c, err := a.validateCampaignFields(c.Request().Context(), tenantID(c), o); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		o = c
	}

	out, err := a.core.UpdateCampaign(c.Request().Context(), tenantID(c), id, o.Campaign, o.ListIDs, o.MediaIDs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateCampaignStatus handles campaign status modification.
//
//	@ID			updateCampaignStatus
//	@Summary		Update campaign status
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"Campaign ID"
//	@Param			req		body		object{status=string}			true	"New status"
//	@Success		200	{object}	models.Campaign
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/status [put]
func (a *App) UpdateCampaignStatus(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeManage, id, c); err != nil {
		return err
	}

	req := struct {
		Status string `json:"status"`
	}{}
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Block start/schedule when a Scrub validation job is active on any of the campaign's lists.
	if req.Status == models.CampaignStatusRunning || req.Status == models.CampaignStatusScheduled {
		if blocked, listName, err := a.checkScrubJobOnCampaign(c.Request().Context(), tenantID(c), id); err == nil && blocked {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("campaigns.scrubJobRunning", "list", listName))
		}
	}

	// Update the campaign status in the DB.
	out, err := a.core.UpdateCampaignStatus(c.Request().Context(), tenantID(c), id, req.Status)
	if err != nil {
		return err
	}

	// If the campaign is being stopped, send the signal to the manager to stop it in flight.
	if req.Status == models.CampaignStatusPaused || req.Status == models.CampaignStatusCancelled {
		a.manager.StopCampaign(id)
	}

	// A stale auto-pause reason (Scrub risky subscriber, too many errors)
	// shouldn't linger once the campaign moves to any other status --
	// only the auto-pause paths themselves ever set one.
	if req.Status != models.CampaignStatusPaused {
		if err := a.core.ClearCampaignPauseReason(c.Request().Context(), tenantID(c), id); err != nil {
			a.log.Printf("error clearing pause reason on campaign %d: %v", id, err)
		} else {
			out.PauseReason.Valid = false
		}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateCampaignArchive updates campaign archive settings.
//
//	@ID			updateCampaignArchive
//	@Summary		Update campaign archive settings
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int		true	"Campaign ID"
//	@Param			req		body		object{archive=bool,archive_template_id=int,archive_meta=object,archive_slug=string}	true	"Archive settings"
//	@Success		200	{object}	interface{}
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/archive [put]
func (a *App) UpdateCampaignArchive(c echo.Context) error {
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeManage, id, c); err != nil {
		return err
	}

	req := struct {
		Archive     bool        `json:"archive"`
		TemplateID  int         `json:"archive_template_id"`
		Meta        models.JSON `json:"archive_meta"`
		ArchiveSlug string      `json:"archive_slug"`
	}{}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.ArchiveSlug != "" {
		// Format the slug to be alpha-numeric-dash.
		s := strings.ToLower(req.ArchiveSlug)
		s = strings.TrimSpace(reSlug.ReplaceAllString(s, " "))
		s = regexpSpaces.ReplaceAllString(s, "-")
		req.ArchiveSlug = s
	}

	if err := a.core.UpdateCampaignArchive(c.Request().Context(), tenantID(c), id, req.Archive, req.TemplateID, req.Meta, req.ArchiveSlug); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{req})
}

// DeleteCampaign handles campaign deletion.
// Only scheduled campaigns that have not started yet can be deleted.
//
//	@ID			deleteCampaign
//	@Summary		Delete a campaign
//	@Tags			campaigns
//	@Produce		json
//	@Param			id	path		int	true	"Campaign ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id} [delete]
func (a *App) DeleteCampaign(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeManage, id, c); err != nil {
		return err
	}

	// Delete the campaign from the DB.
	if err := a.core.DeleteCampaign(c.Request().Context(), tenantID(c), id); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteCampaigns deletes multiple campaigns by IDs or by query.
//
//	@ID			deleteCampaigns
//	@Summary		Delete campaigns (bulk)
//	@Tags			campaigns
//	@Produce		json
//	@Param			id		query		[]int	false	"Campaign IDs"
//	@Param			query	query		string	false	"SQL-like filter query"
//	@Param			all		query		bool	false	"Delete all campaigns matching the query"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/campaigns [delete]
func (a *App) DeleteCampaigns(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	var (
		hasAllPerm     = user.HasPerm(auth.PermCampaignsManageAll)
		permittedLists []int
	)

	if !hasAllPerm {
		// Either the user has campaigns:manage_all permissions and can manage all campaigns,
		// or the campaigns are filtered by the lists the user has get|manage access to.
		hasAllPerm, permittedLists = user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)
	}

	var (
		ids   []int
		query string
		all   bool
	)

	// Check for IDs in query params.
	if len(c.Request().URL.Query()["id"]) > 0 {
		var err error
		ids, err = parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
	} else {
		// Check for query param.
		query = strings.TrimSpace(c.FormValue("query"))
		all = c.FormValue("all") == "true"
	}

	// Validate that either IDs or query is provided.
	if len(ids) == 0 && (query == "" && !all) {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", "id or query required"))
	}

	// Delete the campaigns from the DB.
	if err := a.core.DeleteCampaigns(c.Request().Context(), tenantID(c), ids, query, hasAllPerm, permittedLists); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetRunningCampaignStats returns stats of a given set of campaign IDs.
//
//	@ID			getRunningCampaignStats
//	@Summary		Get running campaign stats
//	@Tags			campaigns
//	@Produce		json
//	@Success		200	{object}	[]models.Campaign
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/campaigns/running/stats [get]
func (a *App) GetRunningCampaignStats(c echo.Context) error {
	// Get the running campaign stats from the DB.
	out, err := a.core.GetRunningCampaignStats(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Compute rate.
	for i, c := range out {
		if c.Started.Valid && c.UpdatedAt.Valid {
			diff := max(int(c.UpdatedAt.Time.Sub(c.Started.Time).Minutes()), 1)

			rate := c.Sent / diff
			if rate > c.Sent || rate > c.ToSend {
				rate = c.Sent
			}

			// Rate since the starting of the campaign.
			out[i].NetRate = rate

			// Realtime running rate over the last minute.
			out[i].Rate = a.manager.GetCampaignStats(c.ID).SendRate
		}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetAutoPausedCampaigns returns campaigns currently paused automatically
// (Scrub-flagged risky subscriber, too many send errors) rather than by a
// user -- backs the dashboard widget. Deliberately a live query, not
// routed through the cached dashboard-counts materialized view, since
// this is the one signal whose entire purpose is prompt visibility right
// after the pause event -- see core.GetAutoPausedCampaigns's doc comment.
//
//	@ID			getAutoPausedCampaigns
//	@Summary		Get currently auto-paused campaigns
//	@Tags			campaigns
//	@Produce		json
//	@Success		200	{object}	[]core.AutoPausedCampaign
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/campaigns/auto-paused [get]
func (a *App) GetAutoPausedCampaigns(c echo.Context) error {
	out, err := a.getAutoPausedCampaigns(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// getAutoPausedCampaigns is a thin, explicitly-typed wrapper around
// a.core.GetAutoPausedCampaigns -- swag's doc-comment type resolution
// needs core.AutoPausedCampaign used as a real Go identifier somewhere in
// this file, not just named in a comment string.
func (a *App) getAutoPausedCampaigns(ctx context.Context, tenantID int) ([]core.AutoPausedCampaign, error) {
	return a.core.GetAutoPausedCampaigns(ctx, tenantID)
}

// TestCampaign handles the sending of a campaign message to
// arbitrary subscribers for testing.
//
//	@ID			testCampaign
//	@Summary		Send a test campaign message
//	@Tags			campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"Campaign ID"
//	@Param			req		body		campReq		true	"Test campaign request with subscriber emails"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/test [post]
func (a *App) TestCampaign(c echo.Context) error {
	// Get the campaign ID.
	id := getID(c)

	// Check if the user has access to the campaign.
	if err := a.checkCampaignPerm(auth.PermTypeManage, id, c); err != nil {
		return err
	}

	// Get and validate fields.
	var req campReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate.
	if c, err := a.validateCampaignFields(c.Request().Context(), tenantID(c), req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req = c
	}
	if len(req.SubscriberEmails) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("campaigns.noSubsToTest"))
	}

	// Sanitize subscriber e-mails.
	for i := range req.SubscriberEmails {
		req.SubscriberEmails[i] = strings.ToLower(strings.TrimSpace(req.SubscriberEmails[i]))
	}

	// Get the subscribers from the DB by their e-mails.
	subs, err := a.core.GetSubscribersByEmail(c.Request().Context(), tenantID(c), req.SubscriberEmails)
	if err != nil {
		return err
	}

	// Check if the user has permission to access the subscribers.
	user := auth.GetUser(c)
	subIDs := make([]int, len(subs))
	for i, s := range subs {
		subIDs[i] = s.ID
	}
	if err := a.hasSubPerm(c.Request().Context(), tenantID(c), user, subIDs); err != nil {
		return err
	}

	// Get the campaign from the DB for previewing.
	tplID, _ := strconv.Atoi(c.FormValue("template_id"))
	camp, err := a.core.GetCampaignForPreview(c.Request().Context(), tenantID(c), id, tplID)
	if err != nil {
		return err
	}

	// Override certain values from the DB with incoming values.
	camp.Name = req.Name
	camp.Subject = req.Subject
	camp.FromEmail = req.FromEmail
	camp.Body = req.Body
	camp.AltBody = req.AltBody
	camp.Messenger = req.Messenger
	camp.ContentType = req.ContentType
	camp.Headers = req.Headers
	camp.TemplateID = req.TemplateID
	for _, id := range req.MediaIDs {
		if id > 0 {
			camp.MediaIDs = append(camp.MediaIDs, int64(id))
		}
	}

	// Send the test messages.
	for _, s := range subs {
		sub := s

		if err := a.sendTestMessage(sub, &camp); err != nil {
			a.log.Printf("error sending test message: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("campaigns.errorSendTest", "error", err.Error()))
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetCampaignViewAnalytics retrieves view/click analytics for a campaign.
//
//	@ID			getCampaignAnalytics
//	@Summary		Get campaign analytics
//	@Tags			campaigns
//	@Produce		json
//	@Param			type	path		string		true	"Analytics type (views, clicks, bounces, links)"
//	@Param			id		query		[]int		true	"Campaign IDs"
//	@Param			from	query		string		true	"Start date (YYYY-MM-DD)"
//	@Param			to		query		string		true	"End date (YYYY-MM-DD)"
//	@Success		200	{object}	[]models.CampaignAnalyticsCount
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/campaigns/analytics/{type} [get]
func (a *App) GetCampaignViewAnalytics(c echo.Context) error {
	ids, err := parseStringIDs(c.Request().URL.Query()["id"])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
	}

	if len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.missingFields", "name", "`id`"))
	}

	var (
		typ  = c.Param("type")
		from = c.QueryParams().Get("from")
		to   = c.QueryParams().Get("to")
	)
	if !strHasLen(from, 10, 30) || !strHasLen(to, 10, 30) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("analytics.invalidDates"))
	}

	// Campaign link stats.
	if typ == "links" {
		out, err := a.core.GetCampaignAnalyticsLinks(c.Request().Context(), tenantID(c), ids, typ, from, to)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Get the analytics numbers from the DB for the campaigns.
	out, err := a.core.GetCampaignAnalyticsCounts(c.Request().Context(), tenantID(c), ids, typ, from, to)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// sendTestMessage takes a campaign and a subscriber and sends out a sample campaign message.
func (a *App) sendTestMessage(sub models.Subscriber, camp *models.Campaign) error {
	if err := a.manager.LoadInlineImages(camp); err != nil {
		a.log.Printf("error loading inline images: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := camp.CompileTemplate(a.manager.TemplateFuncs(camp)); err != nil {
		a.log.Printf("error compiling template: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	// Create a sample campaign message.
	msg, err := a.manager.NewCampaignMessage(camp, sub)
	if err != nil {
		a.log.Printf("error rendering message: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, a.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	return a.manager.PushCampaignMessage(msg)
}

// validateCampaignFields validates incoming campaign field values.
func (a *App) validateCampaignFields(ctx context.Context, tenantID int, c campReq) (campReq, error) {
	if c.FromEmail == "" {
		// a.cfg.FromEmail is a single boot-time value derived from tenant
		// 1's settings (see cmd/init.go's initSettings) - falling back to
		// it here would send every other tenant's campaigns under tenant
		// 1's From address. Look up this tenant's own app.from_email
		// instead, only falling back to the global default if that fails.
		c.FromEmail = a.cfg.FromEmail
		if s, err := a.core.GetSettings(ctx, tenantID); err == nil && s.AppFromEmail != "" {
			c.FromEmail = s.AppFromEmail
		}
	} else if !reFromAddress.Match([]byte(c.FromEmail)) {
		imp, err := a.importers.Get(ctx, tenantID)
		if err != nil {
			return c, err
		}
		if _, err := imp.SanitizeEmail(c.FromEmail); err != nil {
			return c, errors.New(a.i18n.T("campaigns.fieldInvalidFromEmail"))
		}
	}

	if !strHasLen(c.Name, 1, stdInputMaxLen) {
		return c, errors.New(a.i18n.T("campaigns.fieldInvalidName"))
	}

	// Larger char limit for subject as it can contain {{ go templating }} logic.
	if !strHasLen(c.Subject, 1, 5000) {
		return c, errors.New(a.i18n.T("campaigns.fieldInvalidSubject"))
	}

	// If no content-type is specified, default to richtext.
	if c.ContentType != models.CampaignContentTypeRichtext &&
		c.ContentType != models.CampaignContentTypeHTML &&
		c.ContentType != models.CampaignContentTypePlain &&
		c.ContentType != models.CampaignContentTypeVisual &&
		c.ContentType != models.CampaignContentTypeMarkdown {
		c.ContentType = models.CampaignContentTypeRichtext
	}

	if c.ContentType != models.CampaignContentTypeVisual {
		c.BodySource.Valid = false
	}

	// If there's a "send_at" date, it should be in the future.
	if c.SendAt.Valid {
		if c.SendAt.Time.Before(time.Now()) {
			return c, errors.New(a.i18n.T("campaigns.fieldInvalidSendAt"))
		}
	}

	if len(c.ListIDs) == 0 {
		return c, errors.New(a.i18n.T("campaigns.fieldInvalidListIDs"))
	}

	if !a.manager.HasMessenger(c.Messenger) {
		// If it's a specific SMTP, but it's no longer available (removed/disabled), fall back to general email messenger.
		if strings.HasPrefix(c.Messenger, "email-") {
			c.Messenger = "email"
		} else {
			return c, errors.New(a.i18n.Ts("campaigns.fieldInvalidMessenger", "name", c.Messenger))
		}
	}

	camp := models.Campaign{Body: c.Body, TemplateBody: tplTag}
	if err := c.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
		return c, errors.New(a.i18n.Ts("campaigns.fieldInvalidBody", "error", err.Error()))
	}

	if len(c.Headers) == 0 {
		c.Headers = make([]map[string]string, 0)
	}

	// Validate and initialize attribs.
	if c.Attribs != nil {
		if _, err := json.Marshal(c.Attribs); err != nil {
			return c, errors.New(a.i18n.T("subscribers.invalidJSON"))
		}
	}

	if len(c.ArchiveMeta) == 0 {
		c.ArchiveMeta = json.RawMessage("{}")
	}

	if c.ArchiveSlug.String != "" {
		// Format the slug to be alpha-numeric-dash.
		s := strings.ToLower(c.ArchiveSlug.String)
		s = strings.TrimSpace(reSlug.ReplaceAllString(s, " "))
		s = regexpSpaces.ReplaceAllString(s, "-")

		c.ArchiveSlug = null.NewString(s, true)
	} else {
		// If there's no slug set, set it to NULL in the DB.
		c.ArchiveSlug.Valid = false
	}

	return c, nil
}

// makeOptinCampaignMessage makes a default opt-in campaign message body.
func (a *App) makeOptinCampaignMessage(ctx context.Context, tenantID int, o campReq) (campReq, error) {
	if len(o.ListIDs) == 0 {
		return o, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("campaigns.fieldInvalidListIDs"))
	}

	// Fetch double opt-in lists from the given list IDs from the DB.
	lists, err := a.core.GetListsByOptin(ctx, tenantID, o.ListIDs, models.ListOptinDouble)
	if err != nil {
		return o, err
	}

	// There are no double opt-in lists.
	if len(lists) == 0 {
		return o, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("campaigns.noOptinLists"))
	}

	// Construct the opt-in URL with list IDs.
	listIDs := url.Values{}
	for _, l := range lists {
		listIDs.Add("l", l.UUID)
	}
	// optinURLFunc := template.URL("{{ OptinURL }}?" + listIDs.Encode())
	optinURLAttr := template.HTMLAttr(fmt.Sprintf(`href="{{ OptinURL }}%s"`, listIDs.Encode()))

	// Prepare sample opt-in message for the campaign.
	var b bytes.Buffer

	if err := notifs.Tpls.ExecuteTemplate(&b, "optin-campaign", struct {
		Lists        []models.List
		OptinURLAttr template.HTMLAttr
	}{lists, optinURLAttr}); err != nil {
		a.log.Printf("error compiling 'optin-campaign' template: %v", err)
		return o, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	o.Body = b.String()
	return o, nil
}

// checkCampaignPerm checks if the user has get or manage access to the given campaign.
// Either the user has blanket get_all/manage_all permissions, or the campaign
// belongs to lists that the user has access to.
func (a *App) checkCampaignPerm(types auth.PermType, id int, c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	perm := auth.PermCampaignsGet
	if types&auth.PermTypeGet != 0 {
		// It's a get request and there's a blanket get all permission.
		if user.HasPerm(auth.PermCampaignsGetAll) {
			return nil
		}
	} else {
		// It's a manage request and there's a blanket manage_all permission.
		if user.HasPerm(auth.PermCampaignsManageAll) {
			return nil
		}

		perm = auth.PermCampaignsManage
	}

	// There are no *_all campaign permissions. Instead, check if the user access
	// blanket get_all/manage_all list permissions. If yes, then the user can access
	// all campaigns. If there are no *_all permissions, then ensure that the
	// campaign belongs to the lists that the user has access to.
	if hasAllPerm, permittedListIDs := user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage); !hasAllPerm {
		if ok, err := a.core.CampaignHasLists(c.Request().Context(), tenantID(c), id, permittedListIDs); err != nil {
			return err
		} else if !ok {
			return echo.NewHTTPError(http.StatusForbidden,
				a.i18n.Ts("globals.messages.permissionDenied", "name", perm))
		}
	}

	return nil
}

// canEditCampaign returns true if a campaign is in a status where updating
// its properties is allowed.
func canEditCampaign(status string) bool {
	return status == models.CampaignStatusDraft ||
		status == models.CampaignStatusPaused ||
		status == models.CampaignStatusScheduled
}

// checkScrubJobOnCampaign returns (true, listName) if any of the campaign's
// target lists has an active Scrub validation job. Returns (false, "") silently
// when Scrub is not configured or unreachable.
func (a *App) checkScrubJobOnCampaign(ctx context.Context, tenantID int, campaignID int) (bool, string, error) {
	s, err := a.core.GetSettings(ctx, tenantID)
	if err != nil || !s.Scrub.Enabled || s.Scrub.URL == "" || s.Scrub.APIKey == "" || s.Scrub.IntegrationID == 0 {
		return false, "", nil
	}

	// Get campaign with its lists.
	camp, err := a.core.GetCampaign(ctx, tenantID, campaignID, "", "")
	if err != nil {
		return false, "", err
	}

	// Parse campaign list IDs from the JSON array [{id, name}, ...].
	var campLists []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal([]byte(camp.Lists), &campLists); err != nil || len(campLists) == 0 {
		return false, "", nil
	}
	campListIDs := make(map[int]bool, len(campLists))
	for _, l := range campLists {
		campListIDs[l.ID] = true
	}

	// Query Scrub API for lists with active jobs.
	scrubURL := strings.TrimRight(strings.TrimSpace(s.Scrub.URL), "/")
	apiURL := fmt.Sprintf("%s/listmonk/integrations/%d/lists", scrubURL, s.Scrub.IntegrationID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return false, "", nil
	}
	httpReq.Header.Set("X-API-Key", s.Scrub.APIKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return false, "", nil // silently pass if Scrub is unreachable
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return false, "", nil // silently pass if Scrub is unreachable
	}

	var lists []struct {
		ID                 int     `json:"id"`
		Name               string  `json:"name"`
		ActiveJobRequestID *string `json:"active_job_request_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		return false, "", nil
	}

	for _, l := range lists {
		if l.ActiveJobRequestID != nil && campListIDs[l.ID] {
			return true, l.Name, nil
		}
	}
	return false, "", nil
}
