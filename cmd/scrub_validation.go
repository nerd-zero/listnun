package main

import (
	"context"
	"log"

	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/scrub"
	"github.com/knadh/listmonk/models"
)

// scrubValidationEnabled reports whether tenant s has Scrub's generic
// validate-on-add API usably configured. Mirrors cmd/admin.go's
// ScrubEnabled server-config expression -- kept in sync rather than
// re-derived, since a divergence here would silently disable/enable
// validation inconsistently with what the frontend believes is on.
func scrubValidationEnabled(s models.Settings) bool {
	return s.Scrub.Enabled && s.Scrub.URL != "" && s.Scrub.APIKey != ""
}

// pauseCampaignsForRiskySubscriber is the App-level convenience wrapper
// around pauseCampaignsForRiskySubscriberWith, used by the request-handler
// call sites (CreateSubscriber, processSubForm, ManageSubscriberLists).
func (a *App) pauseCampaignsForRiskySubscriber(ctx context.Context, tenantID int, listIDs []int) {
	pauseCampaignsForRiskySubscriberWith(ctx, a.core, a.manager, tenantID, listIDs, "scrub_risky_subscriber")
}

// pauseCampaignsForRiskySubscriberWith auto-pauses every running campaign
// targeting any of listIDs, recording why via reason (a
// campaigns.pause_reason value -- see queries/campaigns.sql's
// set-campaign-pause-reason and frontend campaigns.pauseReason.* i18n
// keys for the values in use: "scrub_risky_subscriber" -- a risky
// subscriber landed on the list, "scrub_job_started" -- a validation job
// was just triggered on the list via ScrubList). Errors are logged and
// swallowed -- callers treat this as a secondary side effect, never a
// reason to fail the request that triggered it. A package-level function
// (not an *App method) so cmd/tenant_importer.go's per-tenant
// OnRiskySubscriber callback can call it too, without needing a full *App
// reference.
func pauseCampaignsForRiskySubscriberWith(ctx context.Context, co *core.Core, mgr *manager.Manager, tenantID int, listIDs []int, reason string) {
	if len(listIDs) == 0 {
		return
	}

	camps, err := co.GetRunningCampaignsByList(ctx, tenantID, listIDs)
	if err != nil {
		log.Printf("error checking running campaigns for pause (tenant %d): %v", tenantID, err)
		return
	}

	for _, c := range camps {
		if _, err := co.UpdateCampaignStatus(ctx, tenantID, c.ID, models.CampaignStatusPaused); err != nil {
			log.Printf("error auto-pausing campaign %q (%d) (%s): %v", c.Name, c.ID, reason, err)
			continue
		}
		mgr.StopCampaign(c.ID)
		if err := co.SetCampaignPauseReason(ctx, tenantID, c.ID, reason); err != nil {
			log.Printf("error setting pause reason on campaign %q (%d): %v", c.Name, c.ID, err)
		}
		log.Printf("auto-paused campaign %q (%d): %s", c.Name, c.ID, reason)
	}
}

// validateEmailForAdd validates a single email against the tenant's
// configured Scrub instance, for use at the moment a subscriber is
// actually added (single add, public signup form). Returns the status to
// persist on the subscriber and whether the caller should reject the add
// outright (invalid_syntax/undeliverable). Fails open: a Scrub error
// yields (StatusUnchecked_error, false, nil) -- error is nil because
// "couldn't reach Scrub" isn't a request failure, matching every other
// Scrub touchpoint in this codebase (checkScrubJobOnCampaign,
// TestScrubSettings, ScrubList all fail open the same way).
func validateEmailForAdd(ctx context.Context, s models.Settings, email string) (status string, reject bool) {
	if !scrubValidationEnabled(s) {
		return "", false
	}

	c := scrub.New(s.Scrub.URL, s.Scrub.APIKey)
	res, err := c.ValidateSingle(ctx, email)
	if err != nil {
		return scrub.StatusUncheckedError, false
	}

	switch res.Status {
	case scrub.StatusInvalidSyntax, scrub.StatusUndeliverable:
		return res.Status, true
	case scrub.StatusRisky:
		return scrub.StatusRisky, false
	default:
		return scrub.StatusDeliverable, false
	}
}
