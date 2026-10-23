package models

import (
	"time"

	null "gopkg.in/volatiletech/null.v6"
)

// CampaignReply is one reply to a campaign email, recorded by the inbound
// mail webhook (cmd/replies.go) for a campaign with reply tracking on
// (CampaignTracking.Replies).
type CampaignReply struct {
	ID           int64    `db:"id" json:"id"`
	UUID         string   `db:"uuid" json:"uuid"`
	CampaignID   int      `db:"campaign_id" json:"campaign_id"`
	SubscriberID null.Int `db:"subscriber_id" json:"subscriber_id"`
	// SubscriberEmail/SubscriberName are joined from subscribers; empty
	// once the subscriber is deleted (FromEmail still says who replied).
	SubscriberEmail null.String `db:"subscriber_email" json:"subscriber_email"`
	SubscriberName  null.String `db:"subscriber_name" json:"subscriber_name"`
	FromEmail       string      `db:"from_email" json:"from_email"`
	FromName        string      `db:"from_name" json:"from_name"`
	Subject         string      `db:"subject" json:"subject"`
	TextBody        string      `db:"text_body" json:"text_body"`
	HTMLBody        string      `db:"html_body" json:"html_body"`
	// StrippedText is the reply without the quoted original message, when
	// the inbound provider could separate it.
	StrippedText string `db:"stripped_text" json:"stripped_text"`
	// IsAuto marks an automatic response (out-of-office, vacation). Kept,
	// but not counted as a reply.
	IsAuto      bool      `db:"is_auto" json:"is_auto"`
	ForwardedTo string    `db:"forwarded_to" json:"forwarded_to"`
	ForwardedAt null.Time `db:"forwarded_at" json:"forwarded_at"`
	ReceivedAt  time.Time `db:"received_at" json:"received_at"`

	Total int `db:"total" json:"-"`
} // @name CampaignReply

// CampaignReplyCounts summarises a campaign's replies. Replies and
// Repliers exclude automatic responses, which are counted in Auto.
type CampaignReplyCounts struct {
	Replies  int `db:"replies" json:"replies"`
	Repliers int `db:"repliers" json:"repliers"`
	Auto     int `db:"auto" json:"auto"`
} // @name CampaignReplyCounts

// CampaignRepliesPage is one page of a campaign's replies, newest first.
type CampaignRepliesPage struct {
	Results []CampaignReply     `json:"results"`
	Counts  CampaignReplyCounts `json:"counts"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
} // @name CampaignRepliesPage

// ReplyCampaign is what the inbound webhook needs to know about the
// campaign a reply answers: who to forward it to and through which
// messenger.
type ReplyCampaign struct {
	ID        int     `db:"id"`
	TenantID  int     `db:"tenant_id"`
	Name      string  `db:"name"`
	Subject   string  `db:"subject"`
	FromEmail string  `db:"from_email"`
	Headers   Headers `db:"headers"`
	Messenger string  `db:"messenger"`
}
