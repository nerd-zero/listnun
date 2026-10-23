package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// ErrReplyCampaignNotFound is returned by GetReplyCampaign when the
// campaign a reply token names doesn't exist in that tenant (deleted, or
// never existed).
var ErrReplyCampaignNotFound = errors.New("reply campaign not found")

// GetReplyCampaign returns the campaign an inbound reply answers, under
// tenantID's row-level security.
func (c *Core) GetReplyCampaign(ctx context.Context, tenantID, campaignID int) (models.ReplyCampaign, error) {
	var out models.ReplyCampaign
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetReplyCampaign).Get(&out, campaignID, tenantID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrReplyCampaignNotFound
	}
	if err != nil {
		return out, fmt.Errorf("get reply campaign: %w", err)
	}
	return out, nil
}

// InsertCampaignReply records r for tenantID. It returns the new row's ID
// and true, or 0 and false when the same provider message was already
// recorded (a redelivered webhook).
func (c *Core) InsertCampaignReply(ctx context.Context, tenantID int, r models.CampaignReply, providerMessageID string) (int64, bool, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return 0, false, fmt.Errorf("generate reply uuid: %w", err)
	}

	var replyID int64
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.InsertCampaignReply).Get(&replyID,
			id.String(), tenantID, r.CampaignID, r.SubscriberID.Int, r.FromEmail, r.FromName,
			r.Subject, r.TextBody, r.HTMLBody, r.StrippedText, r.IsAuto, providerMessageID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("insert campaign reply: %w", err)
	}
	return replyID, true, nil
}

// MarkCampaignReplyForwarded records that reply replyID was forwarded to
// address.
func (c *Core) MarkCampaignReplyForwarded(ctx context.Context, tenantID int, replyID int64, address string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.MarkCampaignReplyForwarded).Exec(replyID, address)
		return err
	})
	if err != nil {
		return fmt.Errorf("mark campaign reply forwarded: %w", err)
	}
	return nil
}

// GetCampaignReplies returns one page of campaignID's replies, newest
// first, with the campaign's reply counts.
func (c *Core) GetCampaignReplies(ctx context.Context, tenantID, campaignID, offset, limit int) (models.CampaignRepliesPage, error) {
	out := models.CampaignRepliesPage{Results: []models.CampaignReply{}}

	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetCampaignReplies).Select(&out.Results, campaignID, offset, limit); err != nil {
			return err
		}
		return stmtx(tx, c.q.GetCampaignReplyCounts).Get(&out.Counts, campaignID)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{campaigns.replies.title}", "error", pqErrMsg(err)))
	}

	if len(out.Results) > 0 {
		out.Total = out.Results[0].Total
	}
	return out, nil
}
