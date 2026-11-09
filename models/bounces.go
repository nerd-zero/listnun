package models

import (
	"encoding/json"
	"time"
)

const (
	BounceTypeHard      = "hard"
	BounceTypeSoft      = "soft"
	BounceTypeComplaint = "complaint"
)

// Bounce represents a single bounce event.
type Bounce struct {
	ID        int             `db:"id" json:"id"`
	Type      string          `db:"type" json:"type"`
	Source    string          `db:"source" json:"source"`
	Meta      json.RawMessage `db:"meta" json:"meta"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`

	// One of these should be provided.
	Email            string `db:"email" json:"email,omitempty"`
	SubscriberUUID   string `db:"subscriber_uuid" json:"subscriber_uuid,omitempty"`
	SubscriberID     int    `db:"subscriber_id" json:"subscriber_id,omitempty"`
	SubscriberStatus string `db:"subscriber_status" json:"subscriber_status"`

	CampaignUUID string           `db:"campaign_uuid" json:"campaign_uuid,omitempty"`
	Campaign     *json.RawMessage `db:"campaign" json:"campaign"`

	// TenantID scopes recording to one tenant's subscribers, campaigns and
	// bounce actions. Webhooks set it from the request's tenant. 0 (POP3
	// mailbox scanning, which has no tenant) keeps the old unscoped lookup.
	TenantID int `db:"-" json:"-"`

	// Pseudofield for getting the total number of bounces
	// in searches and queries.
	Total int `db:"total" json:"-"`
} // @name Bounce

// BounceFilter narrows a bounce query. Zero values match everything.
type BounceFilter struct {
	CampaignID   int
	SubscriberID int
	Source       string
	// Type is one of soft, hard or complaint.
	Type string
	// Search matches the subscriber e-mail or campaign name, case-insensitively.
	Search string
}

// BounceTypeCounts is the number of bounces per type for a filter, ignoring
// the filter's type.
type BounceTypeCounts struct {
	Total     int `db:"total" json:"total"`
	Hard      int `db:"hard" json:"hard"`
	Soft      int `db:"soft" json:"soft"`
	Complaint int `db:"complaint" json:"complaint"`
} // @name BounceTypeCounts
