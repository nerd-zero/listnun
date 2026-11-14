package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	txttpl "text/template"
	"time"

	null "gopkg.in/volatiletech/null.v6"
)

const (
	BaseTpl                    = "base"
	ContentTpl                 = "content"
	TemplateTypeCampaign       = "campaign"
	TemplateTypeCampaignVisual = "campaign_visual"
	TemplateTypeTx             = "tx"
)

// Template represents a reusable e-mail template.
type Template struct {
	Base

	Name string `db:"name" json:"name"`
	// Subject is only for type=tx.
	Subject    string      `db:"subject" json:"subject"`
	Type       string      `db:"type" json:"type"`
	Body       string      `db:"body" json:"body,omitempty"`
	BodySource null.String `db:"body_source" json:"body_source,omitempty"`
	IsDefault  bool        `db:"is_default" json:"is_default"`
	TenantID   int         `db:"tenant_id" json:"-"`
	// CampaignCount is how many campaigns use the template (read-only).
	CampaignCount int `db:"campaign_count" json:"campaign_count"`

	// Transactional templates only.
	SampleData json.RawMessage `db:"sample_data" json:"sample_data,omitempty"`
	AltBody    string          `db:"alt_body" json:"alt_body,omitempty"`
	Headers    Headers         `db:"headers" json:"headers,omitempty"`

	// Only relevant to tx (transactional) templates.
	SubjectTpl  *txttpl.Template   `json:"-"`
	Tpl         *template.Template `json:"-"`
	Attachments []Attachment       `json:"-"`
} // @name Template

// Compile compiles a template body and subject (only for tx templates) and
// caches the templat references to be executed later.
func (t *Template) Compile(f template.FuncMap) error {
	// The brand-logo block is defined first so every body can include it;
	// a body that defines its own replaces it.
	tpl, err := template.New(BaseTpl).Funcs(f).Parse(brandLogoDefine)
	if err != nil {
		return fmt.Errorf("error compiling brand-logo block: %v", err)
	}
	if tpl, err = tpl.Parse(t.Body); err != nil {
		return fmt.Errorf("error compiling transactional template: %v", err)
	}
	t.Tpl = tpl

	// If the subject line has a template string, compile it.
	if hasTplExpr(t.Subject) {
		subj := t.Subject

		subjTpl, err := txttpl.New(BaseTpl).Funcs(txttpl.FuncMap(f)).Parse(subj)
		if err != nil {
			return fmt.Errorf("error compiling subject: %v", err)
		}
		t.SubjectTpl = subjTpl
	}

	return nil
}

type CampaignStats struct {
	ID        int       `db:"id" json:"id"`
	Status    string    `db:"status" json:"status"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
	Started   null.Time `db:"started_at" json:"started_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
	Rate      int       `json:"rate"`
	NetRate   int       `json:"net_rate"`
}

type CampaignAnalyticsCount struct {
	CampaignID int       `db:"campaign_id" json:"campaign_id"`
	Count      int       `db:"count" json:"count"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
} // @name CampaignAnalyticsCount

// CampaignEngagement is a campaign's results, counted one way wherever
// they show (see get-campaign-engagement for each count's definition).
type CampaignEngagement struct {
	// Bounced is people with a bounce on the campaign.
	Bounced int `db:"bounced" json:"bounced"`
	// Delivered is sent less bounced: what rates are a share of.
	Delivered int `db:"delivered" json:"delivered"`
	// Opened is people who viewed or clicked.
	Opened int `db:"opened" json:"opened"`
	// Clicked is people who clicked.
	Clicked int `db:"clicked" json:"clicked"`
	// Anonymous is views and clicks recorded without a subscriber, which
	// Opened and Clicked may count more than once for one person.
	Anonymous int `db:"anonymous" json:"anonymous"`
	// Replied is people who replied, leaving out auto-replies.
	Replied int `db:"replied" json:"replied"`
} // @name CampaignEngagement

// CampaignEngagementCount is one campaign's CampaignEngagement.
type CampaignEngagementCount struct {
	CampaignID int `db:"campaign_id" json:"campaign_id"`
	CampaignEngagement
} // @name CampaignEngagementCount

// CampaignBounceTypeCount is the number of bounces of one type for a campaign.
type CampaignBounceTypeCount struct {
	CampaignID int    `db:"campaign_id" json:"campaign_id"`
	Type       string `db:"type" json:"type"`
	Count      int    `db:"count" json:"count"`
} // @name CampaignBounceTypeCount

type CampaignAnalyticsLink struct {
	URL   string `db:"url" json:"url"`
	Count int    `db:"count" json:"count"`
}

type CampaignViewExport struct {
	CampaignID     int       `db:"campaign_id"`
	CampaignUUID   string    `db:"campaign_uuid"`
	CampaignName   string    `db:"campaign_name"`
	SubscriberID   int       `db:"subscriber_id"`
	SubscriberUUID string    `db:"subscriber_uuid"`
	Email          string    `db:"email"`
	SubscriberName string    `db:"subscriber_name"`
	CreatedAt      time.Time `db:"created_at"`
}

type CampaignClickExport struct {
	CampaignID     int       `db:"campaign_id"`
	CampaignUUID   string    `db:"campaign_uuid"`
	CampaignName   string    `db:"campaign_name"`
	SubscriberID   int       `db:"subscriber_id"`
	SubscriberUUID string    `db:"subscriber_uuid"`
	Email          string    `db:"email"`
	SubscriberName string    `db:"subscriber_name"`
	URL            string    `db:"url"`
	CreatedAt      time.Time `db:"created_at"`
}
