package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

// Webhook events. The message.* events mirror the message log's statuses.
const (
	WebhookEventSent      = "message.sent"
	WebhookEventDelivered = "message.delivered"
	WebhookEventBounced   = "message.bounced"
	WebhookEventFailed    = "message.failed"
	// WebhookEventTest is sent by "Send test event"; endpoints always get it.
	WebhookEventTest = "webhook.test"
)

// WebhookEvents are the events an endpoint can subscribe to.
var WebhookEvents = []string{WebhookEventSent, WebhookEventDelivered, WebhookEventBounced, WebhookEventFailed}

// Webhook delivery statuses.
const (
	WebhookPending  = "pending"
	WebhookSuccess  = "success"
	WebhookRetrying = "retrying"
	WebhookFailed   = "failed"
)

// Webhook endpoint statuses.
const (
	WebhookEndpointActive   = "active"
	WebhookEndpointDisabled = "disabled"
)

const (
	// WebhookSignatureHeader carries t=<unix time>,v1=<hex HMAC-SHA256 of
	// "<t>.<body>" with the endpoint's secret>.
	WebhookSignatureHeader = "Listnun-Signature"
	WebhookEventHeader     = "Listnun-Event"
	WebhookDeliveryHeader  = "Listnun-Delivery-Id"

	// WebhookSecretPrefix starts every signing secret.
	WebhookSecretPrefix = "whsec_"
)

// WebhookEndpoint is a workspace URL that receives events. The secret is
// only returned when revealed.
type WebhookEndpoint struct {
	ID         int            `db:"id" json:"id"`
	URL        string         `db:"url" json:"url"`
	Events     pq.StringArray `db:"events" json:"events"`
	Status     string         `db:"status" json:"status"`
	SecretHint string         `db:"secret_hint" json:"secret_hint"`
	CreatedAt  time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at" json:"updated_at"`

	// Last 24 hours.
	Done24h  int `db:"done_24h" json:"done_24h"`
	OK24h    int `db:"ok_24h" json:"ok_24h"`
	MedianMS int `db:"median_ms" json:"median_ms"`
	Retrying int `db:"retrying" json:"retrying"`
} // @name WebhookEndpoint

// WebhookDelivery is one event sent (or being sent) to one endpoint.
type WebhookDelivery struct {
	ID             int64           `db:"id" json:"-"`
	UUID           string          `db:"uuid" json:"id"`
	EndpointID     int             `db:"endpoint_id" json:"endpoint_id"`
	EndpointURL    string          `db:"endpoint_url" json:"endpoint_url"`
	Event          string          `db:"event" json:"event"`
	MessageUUID    string          `db:"message_uuid" json:"message_id"`
	Payload        json.RawMessage `db:"payload" json:"payload,omitempty"`
	Status         string          `db:"status" json:"status"`
	Attempts       int             `db:"attempts" json:"attempts"`
	NextAttemptAt  null.Time       `db:"next_attempt_at" json:"next_attempt_at"`
	LastStatusCode null.Int        `db:"last_status_code" json:"last_status_code"`
	LastDurationMS null.Int        `db:"last_duration_ms" json:"last_duration_ms"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`

	History []WebhookAttempt `db:"-" json:"history,omitempty"`

	Total int `db:"total" json:"-"`
} // @name WebhookDelivery

// WebhookAttempt is one try at a delivery.
type WebhookAttempt struct {
	Attempt    int       `db:"attempt" json:"attempt"`
	StatusCode null.Int  `db:"status_code" json:"status_code"`
	DurationMS int       `db:"duration_ms" json:"duration_ms"`
	Response   string    `db:"response" json:"response"`
	Error      string    `db:"error" json:"error"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
} // @name WebhookAttempt

// WebhookJob is a claimed delivery, with what's needed to send it.
type WebhookJob struct {
	ID         int64           `db:"id"`
	UUID       string          `db:"uuid"`
	TenantID   int             `db:"tenant_id"`
	EndpointID int             `db:"endpoint_id"`
	Event      string          `db:"event"`
	Payload    json.RawMessage `db:"payload"`
	Attempts   int             `db:"attempts"`
	URL        string          `db:"url"`
	Secret     string          `db:"secret"`
}

// WebhookPayload is the JSON body POSTed to endpoints.
type WebhookPayload struct {
	Type      string             `json:"type"`
	CreatedAt time.Time          `json:"created_at"`
	Data      WebhookMessageData `json:"data"`
} // @name WebhookPayload

// WebhookMessageData describes the message an event is about.
type WebhookMessageData struct {
	MessageID  string         `json:"message_id"`
	TemplateID int            `json:"template_id,omitempty"`
	Recipient  string         `json:"recipient"`
	Status     string         `json:"status"`
	Bounce     *WebhookBounce `json:"bounce,omitempty"`
	Error      string         `json:"error,omitempty"`
} // @name WebhookMessageData

// WebhookBounce is the bounce on a message.bounced event.
type WebhookBounce struct {
	Type   string `json:"type"`
	Detail string `json:"detail,omitempty"`
} // @name WebhookBounce
