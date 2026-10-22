package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/models"
	null "gopkg.in/volatiletech/null.v6"
)

// txLogPruneInterval is how often old message log entries are deleted.
const txLogPruneInterval = time.Hour

// QueryTxMessages returns a page of the transactional message log.
//
//	@ID			listTxMessages
//	@Summary		List transactional messages
//	@Tags			developers
//	@Produce		json
//	@Param			query		query		string	false	"Recipient or message ID"
//	@Param			status		query		string	false	"queued, sent, delivered, bounced or failed"
//	@Param			template_id	query		int		false	"Template ID"
//	@Param			api_key_id	query		int		false	"API key ID"
//	@Param			from		query		string	false	"From (RFC 3339)"
//	@Param			to			query		string	false	"To (RFC 3339)"
//	@Param			page		query		int		false	"Page"
//	@Param			per_page	query		int		false	"Results per page"
//	@Success		200			{object}	models.TxLogPage
//	@Failure		400			{object}	echo.HTTPError
//	@Router			/api/developers/messages [get]
func (a *App) QueryTxMessages(c echo.Context) error {
	pg := a.pg.NewFromURL(c.Request().URL.Query())

	q := models.TxLogQuery{
		Search: strings.TrimSpace(c.QueryParam("query")),
		Status: c.QueryParam("status"),
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}
	q.TemplateID, _ = strconv.Atoi(c.QueryParam("template_id"))
	q.APIKeyID, _ = strconv.Atoi(c.QueryParam("api_key_id"))

	var err error
	if q.From, err = parseTimeParam(c.QueryParam("from")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "from"))
	}
	if q.To, err = parseTimeParam(c.QueryParam("to")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "to"))
	}

	out, err := a.core.QueryTxMessages(c.Request().Context(), tenantID(c), q)
	if err != nil {
		return err
	}
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// GetTxMessage returns one message with its timeline, request payload and
// rendered body.
//
//	@ID			getTxMessage
//	@Summary		Get a transactional message
//	@Tags			developers
//	@Produce		json
//	@Param			uuid	path		string	true	"Message ID (msg_…)"
//	@Success		200		{object}	models.TxLogMessage
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/messages/{uuid} [get]
func (a *App) GetTxMessage(c echo.Context) error {
	out, err := a.core.GetTxMessage(c.Request().Context(), tenantID(c), c.Param("uuid"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// ResendTxMessage sends a logged message again with its original request.
// Attachments aren't kept in the log, so they aren't resent.
//
//	@ID			resendTxMessage
//	@Summary		Resend a transactional message
//	@Tags			developers
//	@Produce		json
//	@Param			uuid	path		string	true	"Message ID (msg_…)"
//	@Success		200		{object}	okResp
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/messages/{uuid}/resend [post]
func (a *App) ResendTxMessage(c echo.Context) error {
	orig, err := a.core.GetTxMessage(c.Request().Context(), tenantID(c), c.Param("uuid"))
	if err != nil {
		return err
	}

	var m models.TxMessage
	if err := json.Unmarshal(orig.Payload, &m); err != nil || m.TemplateID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.log.cantResend"))
	}

	user := auth.GetUser(c)
	origin := txOrigin{
		Sender:   orig.Sender,
		SourceIP: c.RealIP(),
		Accepted: a.i18n.Ts("developers.log.resentBy", "id", orig.UUID, "name", user.Username),
	}
	if orig.APIKeyID.Valid {
		origin.APIKeyID = int(orig.APIKeyID.Int)
	}

	ids, err := a.sendTx(c.Request().Context(), tenantID(c), m, origin)
	if len(ids) > 0 {
		c.Response().Header().Set(models.TxMessageIDHeader, strings.Join(ids, ","))
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{ids})
}

// GetTxStats returns the last 24 hours of transactional email.
//
//	@ID			getTxStats
//	@Summary		Transactional email stats, last 24 hours
//	@Tags			developers
//	@Produce		json
//	@Success		200	{object}	models.TxStats
//	@Router			/api/developers/stats [get]
func (a *App) GetTxStats(c echo.Context) error {
	out, err := a.core.GetTxStats(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// pruneTxLog deletes message log entries past their retention, forever.
func (a *App) pruneTxLog() {
	for {
		if n, err := a.core.PruneTxMessages(); err != nil {
			a.log.Printf("error pruning message log: %v", err)
		} else if n > 0 {
			a.log.Printf("pruned %d message log entries", n)
		}
		time.Sleep(txLogPruneInterval)
	}
}

// parseTimeParam parses an optional RFC 3339 query param.
func parseTimeParam(s string) (null.Time, error) {
	if s == "" {
		return null.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return null.Time{}, fmt.Errorf("invalid time %q: %w", s, err)
	}
	return null.TimeFrom(t), nil
}
