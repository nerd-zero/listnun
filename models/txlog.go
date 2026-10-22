package models

import (
	"encoding/json"
	"time"

	null "gopkg.in/volatiletech/null.v6"
)

// Transactional message statuses, in the order a message moves through them.
const (
	TxStatusQueued    = "queued"
	TxStatusSent      = "sent"
	TxStatusDelivered = "delivered"
	TxStatusBounced   = "bounced"
	TxStatusFailed    = "failed"
)

// Transactional message event types shown on a message's timeline.
const (
	TxEventAccepted  = "accepted"
	TxEventRendered  = "rendered"
	TxEventSent      = "sent"
	TxEventDelivered = "delivered"
	TxEventBounced   = "bounced"
	TxEventFailed    = "failed"
)

// TxMessageIDHeader carries a message's log ID on the outgoing email and
// in the POST /api/tx response (comma-separated for several recipients).
const TxMessageIDHeader = "Listnun-Message-Id"

// TxLogMessage is one recipient's transactional message in the log.
type TxLogMessage struct {
	ID               int64           `db:"id" json:"-"`
	UUID             string          `db:"uuid" json:"id"`
	APIKeyID         null.Int        `db:"api_key_id" json:"api_key_id"`
	Sender           string          `db:"sender" json:"sender"`
	TemplateID       null.Int        `db:"template_id" json:"template_id"`
	TemplateName     string          `db:"template_name" json:"template_name"`
	SubscriberID     null.Int        `db:"subscriber_id" json:"subscriber_id"`
	SubscriberStatus string          `db:"subscriber_status" json:"subscriber_status,omitempty"`
	Recipient        string          `db:"recipient" json:"recipient"`
	Subject          string          `db:"subject" json:"subject"`
	Status           string          `db:"status" json:"status"`
	BounceType       null.String     `db:"bounce_type" json:"bounce_type"`
	Error            string          `db:"error" json:"error"`
	Messenger        string          `db:"messenger" json:"messenger"`
	SourceIP         string          `db:"source_ip" json:"source_ip,omitempty"`
	Payload          json.RawMessage `db:"payload" json:"payload,omitempty"`
	Body             string          `db:"body" json:"body,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`

	Events []TxLogEvent `db:"-" json:"events,omitempty"`

	Total int `db:"total" json:"-"`
} // @name TxLogMessage

// TxLogEvent is one step on a message's timeline.
type TxLogEvent struct {
	Type      string    `db:"type" json:"type"`
	Detail    string    `db:"detail" json:"detail"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
} // @name TxLogEvent

// NewTxLogMessage is what's recorded when a message is accepted.
type NewTxLogMessage struct {
	UUID         string
	APIKeyID     int
	Sender       string
	TemplateID   int
	TemplateName string
	SubscriberID int
	Recipient    string
	Subject      string
	Status       string
	Error        string
	Messenger    string
	SourceIP     string
	Payload      json.RawMessage
	Body         string
	Events       []NewTxLogEvent
}

// NewTxLogEvent is an event recorded with a new message.
type NewTxLogEvent struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

// TxLogQuery filters the message log.
type TxLogQuery struct {
	Search     string
	Status     string
	TemplateID int
	APIKeyID   int
	From       null.Time
	To         null.Time
	Offset     int
	Limit      int
}

// TxLogPage is a page of the message log with per-status counts for the
// filter pills.
type TxLogPage struct {
	Results []TxLogMessage `json:"results"`
	Counts  map[string]int `json:"counts"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
} // @name TxLogPage

// TxStats summarises the last 24 hours of transactional email.
type TxStats struct {
	Sent      int    `db:"sent" json:"sent"`
	SentPrev  int    `db:"sent_prev" json:"sent_prev"`
	Delivered int    `db:"delivered" json:"delivered"`
	Bounced   int    `db:"bounced" json:"bounced"`
	Failed    int    `db:"failed" json:"failed"`
	TopError  string `db:"top_error" json:"top_error"`
} // @name TxStats
