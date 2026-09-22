package core

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	CampaignAnalyticsViews   = "views"
	CampaignAnalyticsClicks  = "clicks"
	CampaignAnalyticsBounces = "bounces"

	campaignTplDefault = "default"
	campaignTplArchive = "archive"
)

// QueryCampaigns retrieves paginated campaigns optionally filtering them by the given arbitrary
// query expression. It also returns the total number of records in the DB.
func (c *Core) QueryCampaigns(ctx context.Context, tenantID int, searchStr string, statuses, tags []string, orderBy, order string, getAll bool, permittedLists []int, offset, limit int) (models.Campaigns, int, error) {
	queryStr, stmt := makeSearchQuery(searchStr, orderBy, order, c.q.QueryCampaigns, campQuerySortFields)

	if statuses == nil {
		statuses = []string{}
	}

	if tags == nil {
		tags = []string{}
	}

	// Unsafe to ignore scanning fields not present in models.Campaigns.
	var out models.Campaigns
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := tx.Select(&out, stmt, 0, pq.StringArray(statuses), pq.StringArray(tags), queryStr, getAll, pq.Array(permittedLists), offset, limit); err != nil {
			return err
		}

		for i := range out {
			// Replace null tags.
			if out[i].Tags == nil {
				out[i].Tags = []string{}
			}
		}

		// Lazy load stats.
		return out.LoadStats(stmtx(tx, c.q.GetCampaignStats))
	})
	if err != nil {
		c.log.Printf("error fetching campaigns: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// GetCampaign retrieves a campaign.
func (c *Core) GetCampaign(ctx context.Context, tenantID int, id int, uuid, archiveSlug string) (models.Campaign, error) {
	return c.getCampaign(ctx, tenantID, id, uuid, archiveSlug, campaignTplDefault)
}

// GetArchivedCampaign retrieves a campaign with the archive template body.
func (c *Core) GetArchivedCampaign(ctx context.Context, tenantID int, id int, uuid, archiveSlug string) (models.Campaign, error) {
	out, err := c.getCampaign(ctx, tenantID, id, uuid, archiveSlug, campaignTplArchive)
	if err != nil {
		return out, err
	}

	if !out.Archive {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return out, nil
}

// getCampaign retrieves a campaign. If typlType=default, then the campaign's
// template body is returned as "template_body". If tplType="archive",
// the archive template is returned.
func (c *Core) getCampaign(ctx context.Context, tenantID int, id int, uuid, archiveSlug string, tplType string) (models.Campaign, error) {
	// Unsafe to ignore scanning fields not present in models.Campaigns.
	var uu any
	if uuid != "" {
		uu = uuid
	}

	var out models.Campaigns
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		if err := stmtx(tx, c.q.GetCampaign).Select(&out, id, uu, archiveSlug, tplType); err != nil {
			return err
		}
		if len(out) == 0 {
			return nil
		}

		for i := 0; i < len(out); i++ {
			// Replace null tags.
			if out[i].Tags == nil {
				out[i].Tags = []string{}
			}
		}

		// Lazy load stats.
		return out.LoadStats(stmtx(tx, c.q.GetCampaignStats))
	})
	if err != nil {
		c.log.Printf("error fetching campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return out[0], nil
}

// GetCampaignForPreview retrieves a campaign with a template body. If the optional tplID is > 0
// that particular template is used, otherwise, the template saved on the campaign is.
func (c *Core) GetCampaignForPreview(ctx context.Context, tenantID int, id, tplID int) (models.Campaign, error) {
	var out models.Campaign
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetCampaignForPreview).Get(&out, id, tplID)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
		}

		c.log.Printf("error fetching campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetArchivedCampaigns retrieves campaigns with a template body.
func (c *Core) GetArchivedCampaigns(ctx context.Context, tenantID int, offset, limit int) (models.Campaigns, int, error) {
	var out models.Campaigns
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetArchivedCampaigns).Select(&out, offset, limit, campaignTplArchive)
	})
	if err != nil {
		c.log.Printf("error fetching public campaigns: %v", err)
		return models.Campaigns{}, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// CreateCampaign creates a new campaign.
func (c *Core) CreateCampaign(ctx context.Context, tenantID int, o models.Campaign, listIDs []int, mediaIDs []int) (models.Campaign, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Insert and read ID.
	var newID int
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateCampaign).Get(&newID,
			uu,
			o.Type,
			o.Name,
			o.Subject,
			o.FromEmail,
			o.Body,
			o.AltBody,
			o.ContentType,
			o.SendAt,
			o.Headers,
			o.Attribs,
			pq.StringArray(normalizeTags(o.Tags)),
			o.Messenger,
			o.TemplateID,
			pq.Array(listIDs),
			o.Archive,
			o.ArchiveSlug,
			o.ArchiveTemplateID,
			o.ArchiveMeta,
			pq.Array(mediaIDs),
			o.BodySource,
			tenantID,
		)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("campaigns.noSubs"))
		}

		c.log.Printf("error creating campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	out, err := c.GetCampaign(ctx, tenantID, newID, "", "")
	if err != nil {
		return models.Campaign{}, err
	}

	return out, nil
}

// UpdateCampaign updates a campaign.
func (c *Core) UpdateCampaign(ctx context.Context, tenantID int, id int, o models.Campaign, listIDs []int, mediaIDs []int) (models.Campaign, error) {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateCampaign).Exec(id,
			o.Name,
			o.Subject,
			o.FromEmail,
			o.Body,
			o.AltBody,
			o.ContentType,
			o.SendAt,
			o.Headers,
			o.Attribs,
			pq.StringArray(normalizeTags(o.Tags)),
			o.Messenger,
			o.TemplateID,
			pq.Array(listIDs),
			o.Archive,
			o.ArchiveSlug,
			o.ArchiveTemplateID,
			o.ArchiveMeta,
			pq.Array(mediaIDs),
			o.BodySource,
			tenantID,
		)
		return err
	})
	if err != nil {
		c.log.Printf("error updating campaign: %v", err)
		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	out, err := c.GetCampaign(ctx, tenantID, id, "", "")
	if err != nil {
		return models.Campaign{}, err
	}

	return out, nil
}

// UpdateCampaignStatus updates a campaign's status, eg: draft to running.
func (c *Core) UpdateCampaignStatus(ctx context.Context, tenantID int, id int, status string) (models.Campaign, error) {
	cm, err := c.GetCampaign(ctx, tenantID, id, "", "")
	if err != nil {
		return models.Campaign{}, err
	}

	errMsg := ""
	switch status {
	case models.CampaignStatusDraft:
		if cm.Status != models.CampaignStatusScheduled {
			errMsg = c.i18n.T("campaigns.onlyScheduledAsDraft")
		}
	case models.CampaignStatusScheduled:
		if cm.Status != models.CampaignStatusDraft && cm.Status != models.CampaignStatusPaused {
			errMsg = c.i18n.T("campaigns.onlyDraftAsScheduled")
		}
		if !cm.SendAt.Valid {
			errMsg = c.i18n.T("campaigns.needsSendAt")
		}

	case models.CampaignStatusRunning:
		if cm.Status != models.CampaignStatusPaused && cm.Status != models.CampaignStatusDraft {
			errMsg = c.i18n.T("campaigns.onlyPausedDraft")
		}
	case models.CampaignStatusPaused:
		if cm.Status != models.CampaignStatusRunning {
			errMsg = c.i18n.T("campaigns.onlyActivePause")
		}
	case models.CampaignStatusCancelled:
		if cm.Status != models.CampaignStatusRunning && cm.Status != models.CampaignStatusPaused {
			errMsg = c.i18n.T("campaigns.onlyActiveCancel")
		}
	}

	if len(errMsg) > 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest, errMsg)
	}

	var n int64
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.UpdateCampaignStatus).Exec(cm.ID, status)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error updating campaign status: %v", err)

		return models.Campaign{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if n == 0 {
		return models.Campaign{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	cm.Status = status
	return cm, nil
}

// SetCampaignPauseReason records why a campaign was auto-paused (not
// user-triggered) -- a standalone follow-up to UpdateCampaignStatus
// rather than a param on it, since that's a shared statement with
// call sites (manual pause/cancel/finish) that have no reason to pass.
// Only takes effect if the campaign is currently paused with no reason
// already set -- see set-campaign-pause-reason's doc comment for why.
func (c *Core) SetCampaignPauseReason(ctx context.Context, tenantID int, id int, reason string) error {
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.SetCampaignPauseReason).Exec(id, reason)
		return err
	})
}

// ClearCampaignPauseReason clears a campaign's pause reason -- called
// whenever its status changes to anything other than paused (resume,
// cancel, finish) so a stale reason doesn't linger.
func (c *Core) ClearCampaignPauseReason(ctx context.Context, tenantID int, id int) error {
	return c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.ClearCampaignPauseReason).Exec(id)
		return err
	})
}

// RunningCampaign is a minimal projection used by GetRunningCampaignsByList.
type RunningCampaign struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// GetRunningCampaignsByList returns every running campaign that targets
// any of the given list IDs -- backs the Scrub risky-subscriber
// auto-pause path. The opposite direction from CampaignHasLists (which
// answers "does this one campaign have these lists").
func (c *Core) GetRunningCampaignsByList(ctx context.Context, tenantID int, listIDs []int) ([]RunningCampaign, error) {
	var out []RunningCampaign
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetRunningCampaignsByList).Select(&out, tenantID, pq.Array(listIDs))
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// AutoPausedCampaign is a minimal projection used by GetAutoPausedCampaigns.
type AutoPausedCampaign struct {
	ID          int         `db:"id" json:"id"`
	Name        string      `db:"name" json:"name"`
	PauseReason null.String `db:"pause_reason" json:"pause_reason"`
}

// GetAutoPausedCampaigns returns currently auto-paused campaigns for the
// dashboard widget. Deliberately a live query, not routed through
// mat_dashboard_counts -- that view's refresh is a no-op under
// CacheSlowQueries, and this is the one signal whose entire purpose is
// prompt visibility right after the pause event.
func (c *Core) GetAutoPausedCampaigns(ctx context.Context, tenantID int) ([]AutoPausedCampaign, error) {
	var out []AutoPausedCampaign
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetAutoPausedCampaigns).Select(&out, tenantID)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// UpdateCampaignArchive updates a campaign's archive properties.
func (c *Core) UpdateCampaignArchive(ctx context.Context, tenantID int, id int, enabled bool, tplID int, meta models.JSON, archiveSlug string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateCampaignArchive).Exec(id, enabled, archiveSlug, tplID, meta)
		return err
	})
	if err != nil {
		c.log.Printf("error updating campaign: %v", err)

		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteCampaign deletes a campaign.
func (c *Core) DeleteCampaign(ctx context.Context, tenantID int, id int) error {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.DeleteCampaign).Exec(id)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error deleting campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))

	}

	if n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
	}

	return nil
}

// DeleteCampaigns deletes multiple campaigns by IDs or by query.
func (c *Core) DeleteCampaigns(ctx context.Context, tenantID int, ids []int, query string, hasAllPerm bool, permittedLists []int) error {
	var queryStr string

	if len(ids) > 0 {
		queryStr = ""
	} else {
		queryStr = makeSearchString(query)
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteCampaigns).Exec(pq.Array(ids), queryStr, hasAllPerm, pq.Array(permittedLists))
		return err
	})
	if err != nil {
		c.log.Printf("error deleting campaigns: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.campaigns}", "error", pqErrMsg(err)))
	}

	return nil
}

// CampaignHasLists checks if a campaign has any of the given list IDs.
func (c *Core) CampaignHasLists(ctx context.Context, tenantID int, id int, listIDs []int) (bool, error) {
	has := false
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CampaignHasLists).Get(&has, id, pq.Array(listIDs))
	})
	if err != nil {
		c.log.Printf("error checking campaign lists: %v", err)
		return false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return has, nil
}

// GetRunningCampaignStats returns the progress stats of running campaigns.
func (c *Core) GetRunningCampaignStats(ctx context.Context, tenantID int) ([]models.CampaignStats, error) {
	out := []models.CampaignStats{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetCampaignStatus).Select(&out, models.CampaignStatusRunning)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		c.log.Printf("error fetching campaign stats: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	} else if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

func (c *Core) GetCampaignAnalyticsCounts(ctx context.Context, tenantID int, campIDs []int, typ, fromDate, toDate string) ([]models.CampaignAnalyticsCount, error) {
	// Pick campaign view counts or click counts.
	var q *sqlx.Stmt
	switch typ {
	case "views":
		q = c.q.GetCampaignViewCounts
	case "clicks":
		q = c.q.GetCampaignClickCounts
	case "bounces":
		q = c.q.GetCampaignBounceCounts
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("globals.messages.invalidData"))
	}

	if !strHasLen(fromDate, 10, 30) || !strHasLen(toDate, 10, 30) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("analytics.invalidDates"))
	}

	out := []models.CampaignAnalyticsCount{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, q).Select(&out, pq.Array(campIDs), fromDate, toDate)
	})
	if err != nil {
		c.log.Printf("error fetching campaign %s: %v", typ, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetCampaignAnalyticsLinks returns link click analytics for the given campaign IDs.
func (c *Core) GetCampaignAnalyticsLinks(ctx context.Context, tenantID int, campIDs []int, typ, fromDate, toDate string) ([]models.CampaignAnalyticsLink, error) {
	out := []models.CampaignAnalyticsLink{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetCampaignLinkCounts).Select(&out, pq.Array(campIDs), fromDate, toDate)
	})
	if err != nil {
		c.log.Printf("error fetching campaign %s: %v", typ, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// RegisterCampaignView registers a subscriber's view on a campaign.
func (c *Core) RegisterCampaignView(ctx context.Context, tenantID int, campUUID, subUUID string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.RegisterCampaignView).Exec(campUUID, subUUID, tenantID)
		return err
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "campaign_id" {
			return nil
		}

		c.log.Printf("error registering campaign view: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	return nil
}

// GetLinkURL returns the original URL for a link UUID without recording a click.
func (c *Core) GetLinkURL(ctx context.Context, tenantID int, linkUUID string) (string, error) {
	var url string
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetLinkURL).Get(&url, linkUUID)
	})
	if err != nil {
		c.log.Printf("error getting link URL: %s", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}
	return url, nil
}

// RegisterCampaignLinkClick registers a subscriber's link click on a campaign.
func (c *Core) RegisterCampaignLinkClick(ctx context.Context, tenantID int, linkUUID, campUUID, subUUID string) (string, error) {
	var url string
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.RegisterLinkClick).Get(&url, linkUUID, campUUID, subUUID, tenantID)
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "link_id" {
			return "", echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("public.invalidLink"))
		}

		c.log.Printf("error registering link click: %s", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return url, nil
}

// ExportCampaignViews returns an iterator with campaign views for streaming/exporting.
//
// TODO(#40): ctx/tenantID are accepted for a consistent call-site shape with
// the rest of Core, but the iterator below does NOT yet enforce tenant
// scoping - same deferred shape as ExportSubscribers (see subscribers.go),
// for the same reason (long-lived Preparex'd statement doesn't fit
// WithTenant's short-transaction shape; zero risk today, no real
// multi-tenant dataset exists yet).
func (c *Core) ExportCampaignViews(ctx context.Context, tenantID int, since time.Time, batchSize int) func() ([]models.CampaignViewExport, error) {
	offset := 0
	return func() ([]models.CampaignViewExport, error) {
		var out []models.CampaignViewExport
		if err := c.q.ExportCampaignViews.Select(&out, since, batchSize, offset); err != nil {
			c.log.Printf("error exporting campaign views: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
		}
		offset += len(out)
		return out, nil
	}
}

// ExportCampaignLinkClicks returns an iterator with campaign link click for streaming/exporting.
//
// TODO(#40): see ExportCampaignViews above - same deferred tenant scoping.
func (c *Core) ExportCampaignLinkClicks(ctx context.Context, tenantID int, since time.Time, batchSize int) func() ([]models.CampaignClickExport, error) {
	offset := 0
	return func() ([]models.CampaignClickExport, error) {
		var out []models.CampaignClickExport
		if err := c.q.ExportCampaignLinkClicks.Select(&out, since, batchSize, offset); err != nil {
			c.log.Printf("error exporting campaign link clicks: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError,
				c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.analytics}", "error", pqErrMsg(err)))
		}
		offset += len(out)
		return out, nil
	}
}

// DeleteCampaignViews deletes campaign views older than a given date.
func (c *Core) DeleteCampaignViews(ctx context.Context, tenantID int, before time.Time) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteCampaignViews).Exec(before)
		return err
	})
	if err != nil {
		c.log.Printf("error deleting campaign views: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return nil
}

// DeleteCampaignLinkClicks deletes campaign views older than a given date.
func (c *Core) DeleteCampaignLinkClicks(ctx context.Context, tenantID int, before time.Time) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteCampaignLinkClicks).Exec(before)
		return err
	})
	if err != nil {
		c.log.Printf("error deleting campaign link clicks: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.Ts("public.errorProcessingRequest"))
	}

	return nil
}
