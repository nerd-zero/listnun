package models

import (
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	ListTypePrivate    = "private"
	ListTypePublic     = "public"
	ListOptinSingle    = "single"
	ListOptinDouble    = "double"
	ListStatusActive   = "active"
	ListStatusArchived = "archived"
)

// List represents a mailing list.
type List struct {
	Base

	UUID             string         `db:"uuid" json:"uuid"`
	Name             string         `db:"name" json:"name"`
	Type             string         `db:"type" json:"type"`
	Optin            string         `db:"optin" json:"optin"`
	Status           string         `db:"status" json:"status"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Description      string         `db:"description" json:"description"`
	SubscriberCount  int            `db:"subscriber_count" json:"subscriber_count"`
	SubscriberCounts StringIntMap   `db:"subscriber_statuses" json:"subscriber_statuses"`
	SubscriberID     int            `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus    string    `db:"subscription_status" json:"subscription_status,omitempty"`
	SubscriptionCreatedAt null.Time `db:"subscription_created_at" json:"subscription_created_at,omitempty"`
	SubscriptionUpdatedAt null.Time `db:"subscription_updated_at" json:"subscription_updated_at,omitempty"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
} // @name List

// ListGrowth is a list's joins and leaves over a recent window.
type ListGrowth struct {
	ListID int `db:"list_id" json:"list_id"`
	Joined int `db:"joined" json:"joined"`
	Left   int `db:"left" json:"left"`
} // @name ListGrowth

// ListActivityDay is one day of joins and leaves on a list.
type ListActivityDay struct {
	Day    null.Time `db:"day" json:"day" swaggertype:"string"`
	Joined int       `db:"joined" json:"joined"`
	Left   int       `db:"left" json:"left"`
} // @name ListActivityDay

// ListCampaign is a campaign sent, sending or scheduled to a list, with
// how many people opened and clicked it.
type ListCampaign struct {
	ID        int       `db:"id" json:"id"`
	UUID      string    `db:"uuid" json:"uuid"`
	Name      string    `db:"name" json:"name"`
	Status    string    `db:"status" json:"status"`
	StartedAt null.Time `db:"started_at" json:"started_at" swaggertype:"string"`
	SendAt    null.Time `db:"send_at" json:"send_at" swaggertype:"string"`
	Sent      int       `db:"sent" json:"sent"`
	Opened    int       `db:"opened" json:"opened"`
	Clicked   int       `db:"clicked" json:"clicked"`
} // @name ListCampaign

// ListActivity is a list's recent joins and leaves, day by day and in
// total, and the latest campaigns to it.
type ListActivity struct {
	Days      []ListActivityDay `json:"days"`
	Joined    int               `json:"joined"`
	Left      int               `json:"left"`
	Campaigns []ListCampaign    `json:"campaigns"`
} // @name ListActivity

// OptinPending counts the people waiting to confirm a double opt-in
// subscription, and how many of them are due a reminder.
type OptinPending struct {
	Total int `db:"total" json:"total"`
	Due   int `db:"due" json:"due"`
} // @name OptinPending

// ListsByUUID returns the lists among `lists` whose UUIDs are in `uuids`,
// or all of `lists` when `uuids` is empty or matches none of them, so a
// stale link still shows a usable form.
func ListsByUUID(lists []List, uuids []string) []List {
	if len(uuids) == 0 {
		return lists
	}
	want := make(map[string]bool, len(uuids))
	for _, u := range uuids {
		want[u] = true
	}
	out := make([]List, 0, len(uuids))
	for _, l := range lists {
		if want[l.UUID] {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return lists
	}
	return out
}
