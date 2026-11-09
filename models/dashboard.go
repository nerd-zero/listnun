package models

import "time"

// DashboardDay is one day of sending and growth on the dashboard.
type DashboardDay struct {
	Date           string `json:"date"`
	Delivered      int    `json:"delivered"`
	Bounced        int    `json:"bounced"`
	Failed         int    `json:"failed"`
	NewSubscribers int    `json:"new_subscribers"`
} // @name DashboardDay

// DashboardHotspot is the list with the most hard bounces this week.
type DashboardHotspot struct {
	ListID   int    `json:"list_id"`
	ListName string `json:"list_name"`
	Hard     int    `json:"hard"`
} // @name DashboardHotspot

// DashboardOverview is the live dashboard for a period.
type DashboardOverview struct {
	Days             []DashboardDay    `json:"days"`
	Subscribers      int               `json:"subscribers"`
	Unsubscribed     int               `json:"unsubscribed"`
	CampaignsStarted int               `json:"campaigns_started"`
	Running          int               `json:"running"`
	Queued           int               `json:"queued"`
	StaleDrafts      int               `json:"stale_drafts"`
	BounceHotspot    *DashboardHotspot `json:"bounce_hotspot"`

	// Filled in Go from Days.
	Sent       int       `json:"sent"`
	Bounced    int       `json:"bounced"`
	New        int       `json:"new"`
	BounceRate float64   `json:"bounce_rate"`
	AsOf       time.Time `json:"as_of"`
} // @name DashboardOverview

// Totals fills the period totals and bounce rate from Days.
func (o *DashboardOverview) Totals() {
	o.Sent, o.Bounced, o.New = 0, 0, 0
	for _, d := range o.Days {
		o.Sent += d.Delivered
		o.Bounced += d.Bounced
		o.New += d.NewSubscribers
	}
	o.BounceRate = 0
	if o.Sent > 0 {
		o.BounceRate = float64(o.Bounced) / float64(o.Sent)
	}
}
