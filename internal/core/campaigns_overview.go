package core

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

// CampaignsOverviewLimits caps how many campaigns each section of the
// Campaigns overview loads. Running and paused campaigns are all loaded.
type CampaignsOverviewLimits struct {
	Scheduled int
	Drafts    int
	Sent      int
}

// GetCampaignsOverview returns the campaigns on the Campaigns overview and
// the number in each status, among those the user can see: all of them
// when getAll, else those on permittedLists.
func (c *Core) GetCampaignsOverview(ctx context.Context, tenantID int, getAll bool, permittedLists []int, lim CampaignsOverviewLimits) (models.CampaignsOverview, error) {
	out := models.CampaignsOverview{Campaigns: []models.CampaignOverviewItem{}, Counts: []models.CampaignStatusCount{}}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetCampaignsOverview).Select(&out.Campaigns, getAll, pq.Array(permittedLists),
			lim.Scheduled, lim.Drafts, lim.Sent); err != nil {
			return err
		}
		return stmtx(tx, c.q.GetCampaignStatusCounts).Select(&out.Counts, getAll, pq.Array(permittedLists))
	})
	if err != nil {
		c.log.Printf("error fetching campaigns overview: %v", err)
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaigns}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// SetCampaignEditor records the user who last edited a campaign.
func (c *Core) SetCampaignEditor(ctx context.Context, tenantID, campaignID, userID int, userName string) error {
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.SetCampaignEditor).Exec(campaignID, userID, userName)
		return err
	})
}
