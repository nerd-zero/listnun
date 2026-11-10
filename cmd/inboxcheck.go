package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/inboxcheck"
	"github.com/nerd-zero/listnun/models"
)

// campaignListRef is one entry of a campaign's lists JSON (its id and
// name), as get-campaign-for-preview returns it.
type campaignListRef struct {
	ID int `json:"id"`
}

// GetCampaignInboxCheck reviews a campaign, as saved, for what commonly
// sends email to spam: its content, subject line, sender and audience
// (see internal/inboxcheck). The editor's Review step shows the result.
//
//	@ID			getCampaignInboxCheck
//	@Summary		Check a campaign for inbox placement problems
//	@Tags			campaigns
//	@Produce		json
//	@Param			id	path		int	true	"Campaign ID"
//	@Success		200	{object}	[]inboxcheck.Check
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/campaigns/{id}/inbox-check [get]
func (a *App) GetCampaignInboxCheck(c echo.Context) error {
	id := getID(c)
	if err := a.checkCampaignPerm(auth.PermTypeGet, id, c); err != nil {
		return err
	}

	ctx := c.Request().Context()
	camp, err := a.core.GetCampaignForPreview(ctx, tenantID(c), id, 0)
	if err != nil {
		return err
	}

	in := inboxcheck.Input{
		Subject:      camp.Subject,
		FromEmail:    camp.FromEmail,
		HasPlainText: camp.ContentType == models.CampaignContentTypePlain || strings.TrimSpace(camp.AltBody.String) != "",
	}
	if s, err := a.core.GetSettings(ctx, tenantID(c)); err == nil {
		in.WorkspaceFrom = s.AppFromEmail
	}

	// The email as a subscriber gets it, template included. Rendered for
	// the dummy subscriber, like the preview, so no views or clicks count.
	if camp.ContentType != models.CampaignContentTypePlain {
		if camp.ContentType == models.CampaignContentTypeVisual {
			camp.TemplateBody = ""
		}
		camp.UUID = dummySubscriber.UUID
		if err := camp.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
		}
		msg, err := a.manager.NewCampaignMessage(&camp, dummySubscriber)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.errorRendering", "error", err.Error()))
		}
		in.HTML = string(msg.Body())
	}

	var lists []campaignListRef
	if err := json.Unmarshal(camp.Lists, &lists); err != nil {
		a.log.Printf("error reading campaign %d lists: %v", id, err)
	}
	listIDs := make([]int, 0, len(lists))
	for _, l := range lists {
		if l.ID > 0 {
			listIDs = append(listIDs, l.ID)
		}
	}
	stats, err := a.core.GetInboxCheckStats(ctx, tenantID(c), camp.Type, listIDs)
	if err != nil {
		return err
	}
	in.Audience, in.Inactive = stats.Audience, stats.Inactive
	in.Sent, in.HardBounces, in.Complaints = stats.Sent, stats.HardBounces, stats.Complaints

	return c.JSON(http.StatusOK, okResp{inboxcheck.Run(in)})
}

// campaignAudience is how many subscribers a campaign goes to.
type campaignAudience struct {
	Count int `json:"count"`
}

// GetCampaignAudience counts the subscribers a campaign on the given lists
// would go to: each person once however many of the lists they're on,
// without unsubscriptions, blocklisted subscribers or (on double opt-in
// lists) unconfirmed ones, the same as sending counts them. Lists'
// subscriber counts can't be added up for this: they overlap and count
// every subscription status.
//
//	@ID			getCampaignAudience
//	@Summary		Count a campaign's audience
//	@Tags			campaigns
//	@Produce		json
//	@Param			list_id	query		[]int	false	"List IDs"	collectionFormat(multi)
//	@Param			type	query		string	false	"Campaign type: regular (default) or optin"
//	@Success		200		{object}	campaignAudience
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/campaigns/audience [get]
func (a *App) GetCampaignAudience(c echo.Context) error {
	listIDs, err := getQueryInts("list_id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}
	campType := c.QueryParam("type")
	if campType != models.CampaignTypeOptin {
		campType = models.CampaignTypeRegular
	}

	// Only lists the user can see count.
	user := auth.GetUser(c)
	listIDs = user.FilterListsByPerm(auth.PermTypeGet|auth.PermTypeManage, listIDs)

	n, err := a.core.CountCampaignAudience(c.Request().Context(), tenantID(c), campType, listIDs)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{campaignAudience{Count: n}})
}
