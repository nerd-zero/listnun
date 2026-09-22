package main

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// store implements DataSource over the primary
// database.
type store struct {
	queries *models.Queries
	core    *core.Core
	media   *tenantMedia
}

type runningCamp struct {
	CampaignID       int    `db:"campaign_id"`
	CampaignType     string `db:"campaign_type"`
	LastSubscriberID int    `db:"last_subscriber_id"`
	MaxSubscriberID  int    `db:"max_subscriber_id"`
	ListID           int    `db:"list_id"`
}

func newManagerStore(q *models.Queries, c *core.Core, m *tenantMedia) *store {
	return &store{
		queries: q,
		core:    c,
		media:   m,
	}
}

// NextCampaigns retrieves active campaigns for the given tenant ready to be
// processed, excluding campaigns that are also being processed. currentIDs/
// sentCounts must only contain that tenant's campaigns (see
// queries/campaigns.sql's next-campaigns for why) - the caller
// (internal/manager) is responsible for that grouping, not this method.
// Additionally, it takes a map of campaignID:sentCount of campaigns that
// are being processed and updates them in the DB.
func (s *store) NextCampaigns(tenantID int, currentIDs []int64, sentCounts []int64) ([]*models.Campaign, error) {
	var out []*models.Campaign
	err := s.queries.NextCampaigns.Select(&out, pq.Int64Array(currentIDs), pq.Int64Array(sentCounts), tenantID)
	return out, err
}

// GetActiveTenantIDs returns the IDs of all active tenants, for
// scanCampaigns to iterate per tick.
func (s *store) GetActiveTenantIDs() ([]int, error) {
	var out []int
	err := s.queries.GetActiveTenantIDs.Select(&out)
	return out, err
}

// NextSubscribers retrieves a subset of subscribers of a given campaign.
// Since batches are processed sequentially, the retrieval is ordered by ID,
// and every batch takes the last ID of the last batch and fetches the next
// batch above that.
func (s *store) NextSubscribers(campID, limit int) ([]models.Subscriber, error) {
	var camps []runningCamp
	if err := s.queries.GetRunningCampaign.Select(&camps, campID); err != nil {
		return nil, err
	}

	var listIDs []int
	for _, c := range camps {
		listIDs = append(listIDs, c.ListID)
	}

	if len(listIDs) == 0 {
		return nil, nil
	}

	var out []models.Subscriber
	err := s.queries.NextCampaignSubscribers.Select(&out, camps[0].CampaignID, camps[0].CampaignType, camps[0].LastSubscriberID, camps[0].MaxSubscriberID, pq.Array(listIDs), limit)
	return out, err
}

// GetCampaign fetches a campaign from the database.
func (s *store) GetCampaign(campID int) (*models.Campaign, error) {
	var out = &models.Campaign{}
	err := s.queries.GetCampaign.Get(out, campID, nil, nil, "default")
	return out, err
}

// UpdateCampaignStatus updates a campaign's status.
func (s *store) UpdateCampaignStatus(campID int, status string) error {
	_, err := s.queries.UpdateCampaignStatus.Exec(campID, status)
	return err
}

// SetCampaignPauseReason records why a campaign was auto-paused. Only
// takes effect if the campaign is currently paused with no reason
// already set -- see set-campaign-pause-reason's doc comment.
func (s *store) SetCampaignPauseReason(campID int, reason string) error {
	_, err := s.queries.SetCampaignPauseReason.Exec(campID, reason)
	return err
}

// UpdateCampaignCounts updates a campaign's status.
func (s *store) UpdateCampaignCounts(campID int, toSend int, sent int, lastSubID int) error {
	_, err := s.queries.UpdateCampaignCounts.Exec(campID, toSend, sent, lastSubID)
	return err
}

// GetAttachment fetches a media attachment blob.
func (s *store) GetAttachment(ctx context.Context, tenantID int, mediaID int) (models.Attachment, error) {
	ms, _, err := s.media.Get(ctx, tenantID)
	if err != nil {
		return models.Attachment{}, err
	}

	m, err := s.core.GetMedia(ctx, tenantID, mediaID, "", "", ms)
	if err != nil {
		return models.Attachment{}, err
	}

	b, err := ms.GetBlob(m.URL)
	if err != nil {
		return models.Attachment{}, err
	}

	return models.Attachment{
		Name:    m.Filename,
		Content: b,
		Header:  manager.MakeAttachmentHeader(m.Filename, "base64", m.ContentType),
	}, nil
}

// GetInlineAttachmentByFilename fetches a media item by filename and returns
// it as an inline attachment along with the Content-ID value. The lookup is
// uniform across filesystem and S3 providers because both use the same media
// store interface; the first match for a given filename is returned.
func (s *store) GetInlineAttachmentByFilename(ctx context.Context, tenantID int, filename string) (models.Attachment, string, error) {
	ms, _, err := s.media.Get(ctx, tenantID)
	if err != nil {
		return models.Attachment{}, "", err
	}

	m, err := s.core.GetMedia(ctx, tenantID, 0, "", filename, ms)
	if err != nil {
		return models.Attachment{}, "", err
	}

	b, err := ms.GetBlob(m.URL)
	if err != nil {
		return models.Attachment{}, "", err
	}

	cid := manager.MakeContentID(m.Filename)
	return models.Attachment{
		Name:     m.Filename,
		Content:  b,
		Header:   manager.MakeInlineAttachmentHeader(m.Filename, "", m.ContentType, cid),
		IsInline: true,
	}, cid, nil
}

// CreateLink registers a URL with a UUID for tracking clicks and returns the UUID.
func (s *store) CreateLink(ctx context.Context, tenantID int, url string) (string, error) {
	// Create a new UUID for the URL. If the URL already exists in the DB
	// the UUID in the database is returned.
	uu, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	var out string
	if err := s.queries.CreateLink.Get(&out, uu, url, tenantID); err != nil {
		return "", err
	}

	return out, nil
}

// GetTenantRootURL returns the tenant's own app.root_url setting, trimmed
// of a trailing slash. Returns an empty string, no error, if the tenant
// has no app.root_url row (the caller falls back to the process-global
// default in that case).
func (s *store) GetTenantRootURL(tenantID int) (string, error) {
	var out string
	if err := s.queries.GetTenantRootURL.Get(&out, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSuffix(out, "/"), nil
}

// RecordBounce records a bounce event and returns the bounce count.
func (s *store) RecordBounce(b models.Bounce) (int64, int, error) {
	var res = struct {
		SubscriberID int64 `db:"subscriber_id"`
		Num          int   `db:"num"`
	}{}

	err := s.queries.UpdateCampaignStatus.Select(&res,
		b.SubscriberUUID,
		b.Email,
		b.CampaignUUID,
		b.Type,
		b.Source,
		b.Meta)

	return res.SubscriberID, res.Num, err
}

// BlocklistSubscriber blocklists a subscriber permanently.
func (s *store) BlocklistSubscriber(id int64) error {
	_, err := s.queries.BlocklistSubscribers.Exec(pq.Int64Array{id})
	return err
}

// DeleteSubscriber deletes a subscriber from the DB.
func (s *store) DeleteSubscriber(id int64) error {
	_, err := s.queries.DeleteSubscribers.Exec(pq.Int64Array{id})
	return err
}
