package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// TxLogRetention is how long transactional messages stay in the log.
const TxLogRetention = 30 * 24 * time.Hour

// crockford is the base32 alphabet used for message IDs: no I, L, O or U,
// so IDs survive being read aloud or retyped.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewTxMessageUUID returns a message ID like msg_01JAD5R9M3TB...: 10
// characters of millisecond timestamp, so IDs sort by time, then 16 of
// randomness.
func NewTxMessageUUID() string {
	var b strings.Builder
	b.WriteString("msg_")

	ms := uint64(time.Now().UnixMilli())
	ts := make([]byte, 10)
	for i := 9; i >= 0; i-- {
		ts[i] = crockford[ms&31]
		ms >>= 5
	}
	b.Write(ts)

	rnd := make([]byte, 16)
	if _, err := rand.Read(rnd); err != nil {
		// crypto/rand only fails if the OS entropy source is broken.
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	for _, r := range rnd {
		b.WriteByte(crockford[r&31])
	}
	return b.String()
}

// CreateTxMessage records an accepted message and its first events.
func (c *Core) CreateTxMessage(ctx context.Context, tenantID int, m models.NewTxLogMessage) (int64, error) {
	events, err := json.Marshal(m.Events)
	if err != nil {
		return 0, err
	}
	payload := m.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}

	var id int64
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateTxMessage).Get(&id,
			m.UUID, m.APIKeyID, m.Sender, m.TemplateID, m.TemplateName, m.SubscriberID,
			m.Recipient, m.Subject, m.Status, m.Error, m.Messenger, m.SourceIP,
			string(events), payload, m.Body, tenantID)
	})
	if err != nil {
		c.log.Printf("error recording tx message to %s: %v", m.Recipient, err)
		return 0, err
	}

	// A message that failed before it was queued still tells webhooks.
	if m.Status == models.TxStatusFailed {
		c.enqueueTxWebhook(tenantID, txStatusRow{
			ID: id, TenantID: tenantID, UUID: m.UUID, TemplateID: m.TemplateID,
			Recipient: m.Recipient, Status: m.Status, Error: m.Error,
		}, m.Error)
	}
	return id, nil
}

// SetTxMessageStatus moves a message to status and records event. A
// bounced or failed message isn't moved back to sent or delivered.
func (c *Core) SetTxMessageStatus(ctx context.Context, tenantID int, id int64, status, event, detail, bounceType string) {
	var (
		row     txStatusRow
		changed bool
	)
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		err := stmtx(tx, c.q.SetTxMessageStatus).Get(&row, id, status, detail, bounceType, event)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		changed = err == nil
		return err
	})
	if err != nil {
		c.log.Printf("error updating tx message %d to %s: %v", id, status, err)
		return
	}
	if changed {
		c.enqueueTxWebhook(row.TenantID, row, detail)
	}
}

// txStatusRow is a message as it is after a status change.
type txStatusRow struct {
	ID         int64  `db:"id"`
	TenantID   int    `db:"tenant_id"`
	UUID       string `db:"uuid"`
	TemplateID int    `db:"template_id"`
	Recipient  string `db:"recipient"`
	Status     string `db:"status"`
	BounceType string `db:"bounce_type"`
	Error      string `db:"error"`
}

// RecordTxSendResult records what the messenger did with a queued
// message. Email the mail server accepted is delivered (to the
// recipient's server); other messengers only report that they sent it.
func (c *Core) RecordTxSendResult(tenantID int, id int64, messenger string, sendErr error) {
	if id == 0 {
		return
	}
	ctx := context.Background()
	if sendErr != nil {
		c.SetTxMessageStatus(ctx, tenantID, id, models.TxStatusFailed, models.TxEventFailed, sendErr.Error(), "")
		return
	}

	c.SetTxMessageStatus(ctx, tenantID, id, models.TxStatusSent, models.TxEventSent, messenger, "")
	if messenger == "email" {
		c.SetTxMessageStatus(ctx, tenantID, id, models.TxStatusDelivered, models.TxEventDelivered, "", "")
	}
}

// markTxBounced attributes a bounce to the latest transactional message
// sent to its address in the last 72 hours, if there is one.
func (c *Core) markTxBounced(b models.Bounce) {
	if b.Email == "" {
		return
	}

	var id int64
	err := c.WithTenant(context.Background(), b.TenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetTxMessageForBounce).Get(&id, b.TenantID, b.Email)
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			c.log.Printf("error finding tx message for bounce (%s): %v", b.Email, err)
		}
		return
	}

	detail := bounceDetail(b.Meta)
	c.SetTxMessageStatus(context.Background(), b.TenantID, id, models.TxStatusBounced, models.TxEventBounced, detail, b.Type)
}

// bounceDetail pulls a readable reason out of a bounce's provider meta.
func bounceDetail(meta json.RawMessage) string {
	if len(meta) == 0 {
		return ""
	}
	// Bounce meta is whatever the provider sent, so it's read loosely.
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		return ""
	}
	for _, k := range []string{"diagnostic_code", "Details", "description", "Description", "reason", "error", "message"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// QueryTxMessages returns a page of the log and per-status counts.
func (c *Core) QueryTxMessages(ctx context.Context, tenantID int, q models.TxLogQuery) (models.TxLogPage, error) {
	out := models.TxLogPage{Results: []models.TxLogMessage{}, Counts: map[string]int{}}

	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.QueryTxMessages).Select(&out.Results,
			q.Search, q.Status, q.TemplateID, q.APIKeyID, q.From, q.To, q.Offset, q.Limit); err != nil {
			return err
		}

		var counts []struct {
			Status string `db:"status"`
			Count  int    `db:"count"`
		}
		if err := stmtx(tx, c.q.CountTxMessagesByStatus).Select(&counts,
			q.Search, q.TemplateID, q.APIKeyID, q.From, q.To); err != nil {
			return err
		}
		for _, r := range counts {
			out.Counts[r.Status] = r.Count
		}
		return nil
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.log.messages}", "error", pqErrMsg(err)))
	}

	if len(out.Results) > 0 {
		out.Total = out.Results[0].Total
	}
	return out, nil
}

// GetTxMessage returns one message with its events, payload and body.
func (c *Core) GetTxMessage(ctx context.Context, tenantID int, uuid string) (models.TxLogMessage, error) {
	var out models.TxLogMessage
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetTxMessage).Get(&out, uuid); err != nil {
			return err
		}
		out.Events = []models.TxLogEvent{}
		return stmtx(tx, c.q.GetTxMessageEvents).Select(&out.Events, out.ID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.log.message}"))
	}
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.log.message}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetTxStats returns the last 24 hours of transactional email.
func (c *Core) GetTxStats(ctx context.Context, tenantID int) (models.TxStats, error) {
	var out models.TxStats
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetTxStats).Get(&out)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.log.messages}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// PruneTxMessages deletes log entries older than TxLogRetention across
// all tenants and returns how many went.
func (c *Core) PruneTxMessages() (int64, error) {
	res, err := c.q.PruneTxMessages.Exec(fmt.Sprintf("%d seconds", int(TxLogRetention.Seconds())))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
