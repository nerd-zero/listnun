package main

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/internal/replies"
	"github.com/nerd-zero/listnun/models"
	null "gopkg.in/volatiletech/null.v6"
)

// maxInboundBody bounds a reply webhook body. Postmark inlines attachments
// as base64 (up to 35 MB of message), and none of it is stored but the
// bodies below.
const maxInboundBody = 50 << 20

// maxReplyBody caps each stored reply body, so one huge HTML reply can't
// bloat campaign_replies; the full message still reaches the campaign's
// own inbox through the forward.
const maxReplyBody = 256 << 10

// RepliesWebhook receives replies to a tenant's reply-tracked campaigns
// from its Postmark server's inbound webhook, records each against its
// campaign and forwards it to the campaign's own Reply-To. Per tenant,
// like BounceWebhook: the tenant comes from the request host, and the
// request must carry that tenant's basic-auth credentials (its "replies"
// setup, pushed by the platform). A reply whose token is missing, forged
// or for another tenant, that didn't arrive at this tenant's inbound
// address, or whose campaign is gone, gets a 403 -- Postmark stops
// retrying on 403 -- and nothing is recorded.
//
//	@ID			postmarkRepliesWebhook
//	@Summary	Receive a reply to a campaign email (Postmark inbound)
//	@Tags		webhooks
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	okResp
//	@Failure	400	{object}	echo.HTTPError
//	@Failure	401	{object}	echo.HTTPError
//	@Failure	403	{object}	echo.HTTPError
//	@Failure	404	{object}	echo.HTTPError
//	@Router		/webhooks/replies/postmark [post]
func (a *App) RepliesWebhook(c echo.Context) error {
	tID := tenantID(c)
	cfg, addr, ok, err := a.replies.Config(tID)
	if err != nil {
		a.log.Printf("replies: error reading tenant %d's reply setup: %v", tID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error")
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "reply tracking is not set up")
	}
	if !replies.CheckBasicAuth(c.Request(), cfg.WebhookUsername, cfg.WebhookPassword) {
		c.Response().Header().Set(echo.HeaderWWWAuthenticate, `Basic realm="replies"`)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxInboundBody))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error reading body")
	}
	in, err := replies.ParsePostmarkInbound(body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if in.Header(replies.HeaderForwarded) != "" {
		a.log.Printf("replies: dropping forwarded reply %s that came back in", in.MessageID)
		return c.JSON(http.StatusOK, okResp{true})
	}

	// The reply must have arrived at this tenant's own inbound address,
	// carry a token signed with this tenant's secret, and name this
	// tenant.
	if replies.RecipientLocal(in.Recipient()) != strings.ToLower(addr.InboundLocal()) {
		a.log.Printf("replies: tenant %d rejecting inbound message %s sent to %q, not its inbound address", tID, in.MessageID, in.Recipient())
		return echo.NewHTTPError(http.StatusForbidden, "unknown reply address")
	}
	ref, err := addr.Codec().Decode(in.Token())
	if err != nil || ref.TenantID != tID {
		a.log.Printf("replies: tenant %d rejecting inbound message %s: bad or foreign reply token", tID, in.MessageID)
		return echo.NewHTTPError(http.StatusForbidden, "unknown reply address")
	}

	ctx := c.Request().Context()
	camp, err := a.core.GetReplyCampaign(ctx, ref.TenantID, ref.CampaignID)
	if errors.Is(err, core.ErrReplyCampaignNotFound) {
		a.log.Printf("replies: tenant %d campaign %d not found for inbound message %s", ref.TenantID, ref.CampaignID, in.MessageID)
		return echo.NewHTTPError(http.StatusForbidden, "unknown campaign")
	}
	if err != nil {
		a.log.Printf("replies: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error")
	}

	fromEmail, fromName := in.Sender()
	reply := models.CampaignReply{
		CampaignID:   camp.ID,
		SubscriberID: null.NewInt(ref.SubscriberID, ref.SubscriberID != 0),
		FromEmail:    fromEmail,
		FromName:     fromName,
		Subject:      in.Subject,
		TextBody:     replies.TruncateUTF8(in.TextBody, maxReplyBody),
		HTMLBody:     replies.TruncateUTF8(in.HTMLBody, maxReplyBody),
		StrippedText: replies.TruncateUTF8(in.StrippedTextReply, maxReplyBody),
		IsAuto:       in.IsAutoReply(),
	}
	replyID, inserted, err := a.core.InsertCampaignReply(ctx, ref.TenantID, reply, in.MessageID)
	if err != nil {
		a.log.Printf("replies: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error")
	}
	if !inserted {
		// Redelivered webhook: already recorded (and forwarded) once.
		return c.JSON(http.StatusOK, okResp{true})
	}

	// The reply is recorded; a failed forward is logged and leaves
	// forwarded_at empty rather than failing the webhook into a retry
	// that would only hit the duplicate check above.
	if to := replies.ForwardAddress(camp); to != "" {
		if err := a.manager.PushMessage(replies.ForwardMessage(camp, in, to)); err != nil {
			a.log.Printf("replies: error forwarding reply %d to %s: %v", replyID, to, err)
		} else if err := a.core.MarkCampaignReplyForwarded(ctx, ref.TenantID, replyID, to); err != nil {
			a.log.Printf("replies: %v", err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetCampaignReplies returns a page of a campaign's replies, newest first,
// with its reply counts (auto-replies counted separately).
//
//	@ID			getCampaignReplies
//	@Summary	Get a campaign's replies
//	@Tags		campaigns
//	@Produce	json
//	@Param		id			path		int	true	"Campaign ID"
//	@Param		page		query		int	false	"Page number"
//	@Param		per_page	query		int	false	"Results per page"
//	@Success	200	{object}	models.CampaignRepliesPage
//	@Failure	400	{object}	echo.HTTPError
//	@Router		/api/campaigns/{id}/replies [get]
func (a *App) GetCampaignReplies(c echo.Context) error {
	pg := a.pg.NewFromURL(c.Request().URL.Query())
	out, err := a.core.GetCampaignReplies(c.Request().Context(), tenantID(c), getID(c), pg.Offset, pg.Limit)
	if err != nil {
		return err
	}
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	return c.JSON(http.StatusOK, okResp{out})
}
