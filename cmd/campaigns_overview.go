package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/core"
)

// campaignsOverviewLimits is how many scheduled, draft and sent campaigns
// the Campaigns overview shows; the full lists are a click away.
var campaignsOverviewLimits = core.CampaignsOverviewLimits{Scheduled: 10, Drafts: 3, Sent: 4}

// GetCampaignsOverview returns what the Campaigns page shows: running and
// paused campaigns, the next scheduled ones, the latest drafts and the
// latest sent, with how many campaigns are in each status.
//
//	@ID			getCampaignsOverview
//	@Summary	Get the campaigns overview
//	@Tags		campaigns
//	@Produce	json
//	@Success	200	{object}	models.CampaignsOverview
//	@Failure	500	{object}	echo.HTTPError
//	@Router		/api/campaigns/overview [get]
func (a *App) GetCampaignsOverview(c echo.Context) error {
	user := auth.GetUser(c)

	getAll := user.HasPerm(auth.PermCampaignsGetAll)
	var permittedLists []int
	if !getAll {
		getAll, permittedLists = user.GetPermittedLists(auth.PermTypeGet | auth.PermTypeManage)
	}

	out, err := a.core.GetCampaignsOverview(c.Request().Context(), tenantID(c), getAll, permittedLists, campaignsOverviewLimits)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// recordCampaignEditor notes who saved a campaign. A failure is logged,
// not returned: the save itself succeeded.
func (a *App) recordCampaignEditor(c echo.Context, campaignID int) {
	user := auth.GetUser(c)
	name := user.Name
	if name == "" {
		name = user.Username
	}
	if err := a.core.SetCampaignEditor(c.Request().Context(), tenantID(c), campaignID, user.ID, name); err != nil {
		a.log.Printf("error recording editor of campaign %d: %v", campaignID, err)
	}
}
