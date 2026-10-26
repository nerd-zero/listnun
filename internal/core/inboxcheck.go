package core

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/cleanup"
	"github.com/nerd-zero/listnun/internal/inboxcheck"
	"github.com/nerd-zero/listnun/models"
)

// InboxCheckStats is what the inbox check's audience checks read for a
// campaign's lists (see internal/inboxcheck.Input).
type InboxCheckStats struct {
	Audience    int `db:"audience"`
	Inactive    int `db:"-"`
	Sent        int `db:"sent"`
	HardBounces int `db:"hard_bounces"`
	Complaints  int `db:"complaints"`
}

// GetInboxCheckStats counts a campaign's audience on listIDs, how much of
// it hasn't engaged in inboxcheck.InactiveDays (the inactive cleanup's
// rule), and the workspace's recent sends, hard bounces and complaints.
func (c *Core) GetInboxCheckStats(ctx context.Context, tenantID int, listIDs []int) (InboxCheckStats, error) {
	var out InboxCheckStats
	if len(listIDs) == 0 {
		return out, nil
	}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetInboxCheckStats).Get(&out, pq.Array(listIDs), inboxcheck.RecentDays); err != nil {
			return err
		}
		return stmtx(tx, c.q.CountInactiveSubscribers).Get(&out.Inactive,
			inboxcheck.InactiveDays, cleanup.ActivityEither, pq.Array(listIDs))
	})
	if err != nil {
		c.log.Printf("error fetching inbox check stats: %v", err)
		return InboxCheckStats{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// unsentCampaignsChecked is how many of a workspace's latest unsent
// campaigns the sending health looks through for missing unsubscribe
// links: the ones someone is likely to send next.
const unsentCampaignsChecked = 50

// SendingHealth is a workspace's recent sending reputation and what stops
// its campaigns sending, for the operator API (listnun's console).
type SendingHealth struct {
	WindowDays  int `db:"-" json:"window_days"`
	Sent        int `db:"sent" json:"sent"`
	HardBounces int `db:"hard_bounces" json:"hard_bounces"`
	Complaints  int `db:"complaints" json:"complaints"`
	inboxcheck.Rates
	// CampaignsWithoutUnsubscribe are unsent campaigns with no unsubscribe
	// link in their content or template, so they can't start or be
	// scheduled (see models.Campaign.HasUnsubscribeLink).
	CampaignsWithoutUnsubscribe []CampaignRef `db:"-" json:"campaigns_without_unsubscribe"`
}

// CampaignRef names a campaign.
type CampaignRef struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetSendingHealth reports tenantID's campaign sends, hard bounces and
// complaints over inboxcheck.RecentDays, rated the way the inbox check
// rates them, and its latest unsent campaigns without an unsubscribe link.
func (c *Core) GetSendingHealth(ctx context.Context, tenantID int) (SendingHealth, error) {
	out := SendingHealth{WindowDays: inboxcheck.RecentDays, CampaignsWithoutUnsubscribe: []CampaignRef{}}
	var camps []models.Campaign
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetSendingHealth).Get(&out, inboxcheck.RecentDays); err != nil {
			return err
		}
		return stmtx(tx, c.q.GetUnsentCampaignBodies).Select(&camps, unsentCampaignsChecked)
	})
	if err != nil {
		c.log.Printf("error fetching sending health: %v", err)
		return SendingHealth{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "sending health", "error", pqErrMsg(err)))
	}
	out.Rates = inboxcheck.RecentRates(out.Sent, out.HardBounces, out.Complaints)
	for _, camp := range camps {
		if !camp.HasUnsubscribeLink() {
			out.CampaignsWithoutUnsubscribe = append(out.CampaignsWithoutUnsubscribe,
				CampaignRef{ID: camp.ID, Name: camp.Name, Status: camp.Status})
		}
	}
	return out, nil
}
