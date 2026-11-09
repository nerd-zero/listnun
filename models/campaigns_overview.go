package models

import (
	"encoding/json"
	"fmt"

	null "gopkg.in/volatiletech/null.v6"
)

// CampaignListRef is a list a campaign goes to. ID is 0 when the list has
// since been deleted; the name is kept.
type CampaignListRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
} // @name CampaignListRef

// CampaignListRefs is a campaign's lists, scanned from a JSON array.
type CampaignListRefs []CampaignListRef

// Scan implements the sql.Scanner interface.
func (r *CampaignListRefs) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*r = CampaignListRefs{}
		return nil
	case []byte:
		return json.Unmarshal(v, r)
	case string:
		return json.Unmarshal([]byte(v), r)
	}
	return fmt.Errorf("could not decode type %T -> %T", src, r)
}

// CampaignOverviewItem is a campaign on the Campaigns overview, with what
// its section shows: progress for running and paused ones, the audience
// for scheduled ones, the editor for drafts and results for sent ones.
type CampaignOverviewItem struct {
	ID            int              `db:"id" json:"id"`
	UUID          string           `db:"uuid" json:"uuid"`
	Name          string           `db:"name" json:"name"`
	Status        string           `db:"status" json:"status"`
	PauseReason   null.String      `db:"pause_reason" json:"pause_reason" swaggertype:"string"`
	SendAt        null.Time        `db:"send_at" json:"send_at" swaggertype:"string"`
	StartedAt     null.Time        `db:"started_at" json:"started_at" swaggertype:"string"`
	UpdatedAt     null.Time        `db:"updated_at" json:"updated_at" swaggertype:"string"`
	UpdatedByName string           `db:"updated_by_name" json:"updated_by_name"`
	ToSend        int              `db:"to_send" json:"to_send"`
	Sent          int              `db:"sent" json:"sent"`
	Lists         CampaignListRefs `db:"lists" json:"lists"`
	Audience      int              `db:"audience" json:"audience"`
	Bounced       int              `db:"bounced" json:"bounced"`
	Opened        int              `db:"opened" json:"opened"`
	Clicked       int              `db:"clicked" json:"clicked"`
} // @name CampaignOverviewItem

// CampaignStatusCount is how many campaigns are in a status.
type CampaignStatusCount struct {
	Status string `db:"status" json:"status"`
	Count  int    `db:"count" json:"count"`
} // @name CampaignStatusCount

// CampaignsOverview is the Campaigns overview: the campaigns it shows and
// how many there are in each status.
type CampaignsOverview struct {
	Campaigns []CampaignOverviewItem `json:"campaigns"`
	Counts    []CampaignStatusCount  `json:"counts"`
} // @name CampaignsOverview
