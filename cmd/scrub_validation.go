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
	pauseCampaignsForRiskySubscriberWith(ctx, a.core, a.manager, tenantID, listIDs)
}

// pauseCampaignsForRiskySubscriberWith auto-pauses every running campaign
// targeting any of listIDs, on the grounds that a Scrub-flagged risky
// subscriber just landed (or was linked) on one of those lists. Errors
// are logged and swallowed -- callers treat this as a secondary side
// effect of a subscriber add/link/import, never a reason to fail that
// request. A package-level function (not an *App method) so
// cmd/tenant_importer.go's per-tenant OnRiskySubscriber callback can call
// it too, without needing a full *App reference.
func pauseCampaignsForRiskySubscriberWith(ctx context.Context, co *core.Core, mgr *manager.Manager, tenantID int, listIDs []int) {
	if len(listIDs) == 0 {
		return
	}

	camps, err := co.GetRunningCampaignsByList(ctx, tenantID, listIDs)
	if err != nil {
		log.Printf("error checking running campaigns for risky subscriber pause (tenant %d): %v", tenantID, err)
		return
	}

	for _, c := range camps {
		if _, err := co.UpdateCampaignStatus(ctx, tenantID, c.ID, models.CampaignStatusPaused); err != nil {
			log.Printf("error auto-pausing campaign %q (%d) for risky subscriber: %v", c.Name, c.ID, err)
			continue
		}
		mgr.StopCampaign(c.ID)
		if err := co.SetCampaignPauseReason(ctx, tenantID, c.ID, "scrub_risky_subscriber"); err != nil {
			log.Printf("error setting pause reason on campaign %q (%d): %v", c.Name, c.ID, err)
		}
		log.Printf("auto-paused campaign %q (%d): Scrub flagged a risky subscriber on a target list", c.Name, c.ID)
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
