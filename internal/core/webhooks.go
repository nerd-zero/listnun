package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/webhooks"
	"github.com/nerd-zero/listnun/models"
)

// WebhookRetention is how long finished deliveries are kept.
const WebhookRetention = 30 * 24 * time.Hour

// txEventByStatus maps a message log status to its webhook event.
var txEventByStatus = map[string]string{
	models.TxStatusSent:      models.WebhookEventSent,
	models.TxStatusDelivered: models.WebhookEventDelivered,
	models.TxStatusBounced:   models.WebhookEventBounced,
	models.TxStatusFailed:    models.WebhookEventFailed,
}

// ValidWebhookEvents reports whether every event is one endpoints can
// subscribe to.
func ValidWebhookEvents(events []string) bool {
	if len(events) == 0 {
		return false
	}
	for _, e := range events {
		if !slices.Contains(models.WebhookEvents, e) {
			return false
		}
	}
	return true
}

func newWebhookSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return models.WebhookSecretPrefix + hex.EncodeToString(b), nil
}

// enqueueTxWebhook queues the event for a message's new status to every
// endpoint subscribed to it. Failures are logged; they never block sending.
func (c *Core) enqueueTxWebhook(tenantID int, m txStatusRow, detail string) {
	event, ok := txEventByStatus[m.Status]
	if !ok {
		return
	}

	data := models.WebhookMessageData{
		MessageID:  m.UUID,
		TemplateID: m.TemplateID,
		Recipient:  m.Recipient,
		Status:     m.Status,
	}
	switch m.Status {
	case models.TxStatusBounced:
		data.Bounce = &models.WebhookBounce{Type: m.BounceType, Detail: detail}
	case models.TxStatusFailed:
		data.Error = m.Error
	}

	payload, err := json.Marshal(models.WebhookPayload{Type: event, CreatedAt: time.Now().UTC(), Data: data})
	if err != nil {
		c.log.Printf("error encoding webhook payload: %v", err)
		return
	}

	// Each endpoint's delivery ID is this prefix plus the endpoint ID.
	prefix := "whd_" + strings.TrimPrefix(NewTxMessageUUID(), "msg_") + "_"
	err = c.WithTenant(context.Background(), tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.EnqueueWebhookDeliveries).Exec(tenantID, event, payload, prefix, m.UUID)
		return err
	})
	if err != nil {
		c.log.Printf("error queueing %s webhooks for %s: %v", event, m.UUID, err)
	}
}

// GetWebhookEndpoints returns the tenant's endpoints with 24h stats.
func (c *Core) GetWebhookEndpoints(ctx context.Context, tenantID, id int) ([]models.WebhookEndpoint, error) {
	out := []models.WebhookEndpoint{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetWebhookEndpoints).Select(&out, id)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.webhooks.endpoints}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetWebhookEndpoint returns one endpoint.
func (c *Core) GetWebhookEndpoint(ctx context.Context, tenantID, id int) (models.WebhookEndpoint, error) {
	out, err := c.GetWebhookEndpoints(ctx, tenantID, id)
	if err != nil {
		return models.WebhookEndpoint{}, err
	}
	if len(out) == 0 {
		return models.WebhookEndpoint{}, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.endpoint}"))
	}
	return out[0], nil
}

// GetWebhookEndpointSecret returns an endpoint's signing secret.
func (c *Core) GetWebhookEndpointSecret(ctx context.Context, tenantID, id int) (string, error) {
	var out string
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetWebhookEndpointSecret).Get(&out, id)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.endpoint}"))
	}
	if err != nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.webhooks.endpoint}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// CreateWebhookEndpoint adds an endpoint with a new signing secret.
func (c *Core) CreateWebhookEndpoint(ctx context.Context, tenantID int, url string, events []string) (models.WebhookEndpoint, error) {
	secret, err := newWebhookSecret()
	if err != nil {
		return models.WebhookEndpoint{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var id int
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateWebhookEndpoint).Get(&id, url, pq.StringArray(events), secret, tenantID)
	})
	if err != nil {
		return models.WebhookEndpoint{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{developers.webhooks.endpoint}", "error", pqErrMsg(err)))
	}
	return c.GetWebhookEndpoint(ctx, tenantID, id)
}

// UpdateWebhookEndpoint changes an endpoint's URL, events or status.
func (c *Core) UpdateWebhookEndpoint(ctx context.Context, tenantID, id int, url string, events []string, status string) (models.WebhookEndpoint, error) {
	if err := c.execWebhookEndpoint(ctx, tenantID, c.q.UpdateWebhookEndpoint, "errorUpdating", id, url, pq.StringArray(events), status); err != nil {
		return models.WebhookEndpoint{}, err
	}
	return c.GetWebhookEndpoint(ctx, tenantID, id)
}

// RotateWebhookEndpointSecret replaces an endpoint's signing secret and
// returns the new one. Deliveries sign with it from the next attempt.
func (c *Core) RotateWebhookEndpointSecret(ctx context.Context, tenantID, id int) (string, error) {
	secret, err := newWebhookSecret()
	if err != nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := c.execWebhookEndpoint(ctx, tenantID, c.q.RotateWebhookEndpointSecret, "errorUpdating", id, secret); err != nil {
		return "", err
	}
	return secret, nil
}

// DeleteWebhookEndpoint removes an endpoint and its deliveries.
func (c *Core) DeleteWebhookEndpoint(ctx context.Context, tenantID, id int) error {
	return c.execWebhookEndpoint(ctx, tenantID, c.q.DeleteWebhookEndpoint, "errorDeleting", id)
}

func (c *Core) execWebhookEndpoint(ctx context.Context, tenantID int, stmt *sqlx.Stmt, errKey string, args ...any) error {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, stmt).Exec(args...)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages."+errKey, "name", "{developers.webhooks.endpoint}", "error", pqErrMsg(err)))
	}
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.endpoint}"))
	}
	return nil
}

// SendWebhookTest queues a webhook.test event to one endpoint.
func (c *Core) SendWebhookTest(ctx context.Context, tenantID, id int) (string, error) {
	uuid := "whd_" + strings.TrimPrefix(NewTxMessageUUID(), "msg_")
	payload, err := json.Marshal(models.WebhookPayload{
		Type:      models.WebhookEventTest,
		CreatedAt: time.Now().UTC(),
		Data: models.WebhookMessageData{
			MessageID: "msg_test", TemplateID: 0, Recipient: "test@example.com", Status: models.TxStatusDelivered,
		},
	})
	if err != nil {
		return "", err
	}

	var rowID int64
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.EnqueueWebhookTest).Get(&rowID, id, models.WebhookEventTest, uuid, payload)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.endpoint}"))
	}
	if err != nil {
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{developers.webhooks.delivery}", "error", pqErrMsg(err)))
	}
	return uuid, nil
}

// QueryWebhookDeliveries returns a page of deliveries, newest first.
func (c *Core) QueryWebhookDeliveries(ctx context.Context, tenantID, endpointID int, status string, offset, limit int) ([]models.WebhookDelivery, int, error) {
	out := []models.WebhookDelivery{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.QueryWebhookDeliveries).Select(&out, endpointID, status, offset, limit)
	})
	if err != nil {
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.webhooks.deliveries}", "error", pqErrMsg(err)))
	}
	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}
	return out, total, nil
}

// GetWebhookDelivery returns a delivery with its payload and attempts.
func (c *Core) GetWebhookDelivery(ctx context.Context, tenantID int, uuid string) (models.WebhookDelivery, error) {
	var out models.WebhookDelivery
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetWebhookDelivery).Get(&out, uuid); err != nil {
			return err
		}
		out.History = []models.WebhookAttempt{}
		return stmtx(tx, c.q.GetWebhookAttempts).Select(&out.History, out.ID)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.delivery}"))
	}
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.webhooks.delivery}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// RetryWebhookDelivery sends a delivery again now.
func (c *Core) RetryWebhookDelivery(ctx context.Context, tenantID int, uuid string) error {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.RetryWebhookDelivery).Exec(uuid)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{developers.webhooks.delivery}", "error", pqErrMsg(err)))
	}
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.webhooks.delivery}"))
	}
	return nil
}

// ClaimWebhookDeliveries locks due deliveries across all tenants for the
// webhook worker (webhooks.Store).
func (c *Core) ClaimWebhookDeliveries(limit int, lease time.Duration) ([]models.WebhookJob, error) {
	var out []models.WebhookJob
	err := c.q.ClaimWebhookDeliveries.Select(&out, limit, fmt.Sprintf("%d", int(lease.Seconds())))
	return out, err
}

// RecordWebhookAttempt saves one try and the delivery's next state
// (webhooks.Store).
func (c *Core) RecordWebhookAttempt(job models.WebhookJob, res webhooks.Result, status string, next *time.Time) error {
	errMsg := ""
	if res.Err != nil {
		errMsg = res.Err.Error()
	}
	var nextAt sql.NullTime
	if next != nil {
		nextAt = sql.NullTime{Time: *next, Valid: true}
	}

	err := c.WithTenant(context.Background(), job.TenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.RecordWebhookAttempt).Exec(job.ID, job.TenantID, job.Attempts+1,
			res.StatusCode, int(res.Duration.Milliseconds()), res.Response, status, nextAt, errMsg)
		return err
	})
	if err != nil {
		return err
	}

	// Failures show up in the workspace's activity log (Settings › Logs).
	switch status {
	case models.WebhookRetrying:
		wait := ""
		if next != nil {
			wait = ", retrying in " + roughDuration(time.Until(*next))
		}
		c.LogActivity(job.TenantID, models.ActivityWarn, "webhook", fmt.Sprintf("%s %s%s", job.URL, webhookOutcome(res), wait))
	case models.WebhookFailed:
		c.LogActivity(job.TenantID, models.ActivityError, "webhook",
			fmt.Sprintf("%s %s, gave up after %d attempts", job.URL, webhookOutcome(res), job.Attempts+1))
	}
	return nil
}

// webhookOutcome describes a failed attempt for the activity log.
func webhookOutcome(res webhooks.Result) string {
	if res.Err != nil {
		return "failed: " + res.Err.Error()
	}
	return fmt.Sprintf("returned %d", res.StatusCode)
}

// PruneWebhookDeliveries deletes finished deliveries past their retention.
func (c *Core) PruneWebhookDeliveries() (int64, error) {
	res, err := c.q.PruneWebhookDeliveries.Exec(fmt.Sprintf("%d seconds", int(WebhookRetention.Seconds())))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// roughDuration reads a wait the way people say it: "4 min", "6 h".
func roughDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d min", max(1, int(d.Round(time.Minute).Minutes())))
	}
	return fmt.Sprintf("%d h", int(d.Round(time.Hour).Hours()))
}
