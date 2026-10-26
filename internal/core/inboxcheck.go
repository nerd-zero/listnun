package core

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/cleanup"
	"github.com/nerd-zero/listnun/internal/inboxcheck"
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
