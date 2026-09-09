package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/scrub"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// submitScrubBatch submits one CSV-import commit-batch window's emails to
// Scrub's async POST /v1/validate/integration and records a
// scrub_validation_batches row so the /webhooks/scrub/batch callback
// below -- which carries no tenant context of its own, just a batch_id --
// can later look up which tenant/lists the batch belongs to.
//
// Package-level (not an *App method) so cmd/tenant_importer.go's
// per-tenant ScrubSubmitFunc closure can call it without needing a full
// *App reference, same reasoning as pauseCampaignsForRiskySubscriberWith
// in cmd/scrub_validation.go. Fails open -- logs and returns nil on any
// error -- matching every other Scrub touchpoint in this codebase.
func submitScrubBatch(ctx context.Context, q *models.Queries, tenantID int, s models.Settings, listIDs []int, emails []string) error {
	if !scrubValidationEnabled(s) || s.Scrub.IntegrationID == "" {
		return nil
	}

	root := strings.TrimRight(strings.TrimSpace(s.AppRootURL), "/")
	if root == "" {
		log.Printf("scrub: tenant %d has no app.root_url configured, skipping batch submission (%d emails will remain unchecked)", tenantID, len(emails))
		return nil
	}

	batchUUID, err := uuid.NewV4()
	if err != nil {
		log.Printf("scrub: error generating batch id: %v", err)
		return nil
	}
	batchID := batchUUID.String()

	if _, err := q.InsertScrubValidationBatch.Exec(batchID, tenantID, pq.Array(listIDs), len(emails)); err != nil {
		log.Printf("scrub: error recording validation batch: %v", err)
		return nil
	}

	eventCallback := root + "/webhooks/scrub/batch"

	c := scrub.New(s.Scrub.URL, s.Scrub.APIKey)
	sub, err := c.SubmitBatch(ctx, s.Scrub.IntegrationID, batchID, eventCallback, emails)
	if err != nil {
		log.Printf("scrub: error submitting validation batch: %v", err)
		return nil
	}

	if _, err := q.UpdateScrubValidationBatchProgress.Exec(batchID, models.ScrubBatchStatusPending, sub.SubmittedCount, 0); err != nil {
		log.Printf("scrub: error updating validation batch progress: %v", err)
	}

	return nil
}

// ScrubBatchWebhook receives Scrub's async validation-progress callback
// for a batch submitted by submitScrubBatch. Registered in the
// public/unauthenticated route group (cmd/handlers.go) since there's no
// tenant session to resolve from Scrub's own request -- authenticated
// instead by verifying Scrub's HMAC signature (internal/scrub.VerifyWebhook,
// a verbatim port of listnun's own already-working implementation)
// against the resolved tenant's Settings.Scrub.APIKey. batch_id (from the
// still-unverified payload) is used only to look up *which* tenant's key
// to check the signature against; the request is trusted only once that
// signature verifies, so a forged batch_id alone can't pass.
//
//	@ID			scrubBatchWebhook
//	@Summary	Receive a Scrub batch validation progress callback
//	@Tags		webhooks
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	okResp
//	@Failure	400	{object}	echo.HTTPError
//	@Failure	403	{object}	echo.HTTPError
//	@Failure	404	{object}	echo.HTTPError
//	@Router		/webhooks/scrub/batch [post]
func (a *App) ScrubBatchWebhook(c echo.Context) error {
	// Read the raw body and json.Unmarshal rather than c.Bind(), which
	// depends on Scrub sending a Content-Type Echo's binder recognizes --
	// same reasoning cmd/bounce.go's BounceWebhook already documents for
	// third-party webhook callers. The raw bytes are also exactly what
	// VerifyWebhook's signature covers -- re-marshaling a parsed struct
	// could reorder/reformat fields and break the signature check.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error reading body")
	}

	var envelope scrub.WebhookEnvelope
	if err := json.Unmarshal(rawReq, &envelope); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}
	if envelope.EventType != scrub.EventTypeProgress {
		// Unrecognized event type -- ack it anyway (200) so Scrub's
		// redelivery mechanism doesn't retry forever over something this
		// version doesn't know how to handle yet, matching listnun's own
		// scrubWebhook's best-effort convention.
		return c.JSON(http.StatusOK, okResp{true})
	}

	var progress scrub.ProgressEvent
	if err := json.Unmarshal(envelope.Data, &progress); err != nil || progress.BatchID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or missing batch_id")
	}

	var batch models.ScrubValidationBatch
	if err := a.queries.GetScrubValidationBatch.Get(&batch, progress.BatchID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "unknown batch")
		}
		a.log.Printf("scrub: error looking up validation batch %s: %v", progress.BatchID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error")
	}

	s, err := a.core.GetSettings(c.Request().Context(), batch.TenantID)
	if err != nil {
		a.log.Printf("scrub: error getting settings to verify batch %s (tenant %d): %v", batch.BatchID, batch.TenantID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error")
	}
	if err := scrub.VerifyWebhook(rawReq, c.Request().Header, s.Scrub.APIKey); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, "webhook verification failed")
	}

	if _, err := a.queries.UpdateScrubValidationBatchProgress.Exec(
		batch.BatchID, progress.Status, progress.SubmittedTotalCount, progress.InvalidTotalCount,
	); err != nil {
		a.log.Printf("scrub: error updating validation batch %s: %v", batch.BatchID, err)
	}

	switch progress.Status {
	case models.ScrubBatchStatusCompleted:
		// Scrub's own delivery client times out a callback attempt at 10s
		// and retries with backoff on failure (modules/webhook/service.py)
		// -- reconcileScrubBatch pages through GetHistory (an external
		// HTTP call to Scrub, potentially several pages), so running it
		// inline here risks the response itself timing out and triggering
		// a duplicate redelivery. Ack immediately instead and reconcile in
		// the background with a fresh context, same "fire-and-forget past
		// the request boundary" pattern cmd/tenant_importer.go already
		// uses for ScrubSubmitFunc.
		go a.reconcileScrubBatch(context.Background(), batch)
	case models.ScrubBatchStatusFailed:
		// Fails open: emails from this batch simply stay scrub_status=NULL
		// (never resolved), same "unchecked" semantics as a row that was
		// never validated at all. Known gap: since only counts, not the
		// submitted email list, are persisted per batch, a failed batch
		// can't be swept back to unchecked_error automatically. An admin
		// can still recover via a manual full-list Scrub rescan (ScrubList).
		a.log.Printf("scrub: validation batch %s failed for tenant %d", batch.BatchID, batch.TenantID)
		if _, err := a.queries.DeleteScrubValidationBatch.Exec(batch.BatchID); err != nil {
			a.log.Printf("scrub: error deleting validation batch %s: %v", batch.BatchID, err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// reconcileScrubBatch runs once a batch reaches status "completed". The
// callback payload itself only carries cumulative counts, not per-email
// detail, so this fetches the batch's full record set via GetHistory
// (paging /v1/history and matching batch_id client-side, since that
// endpoint has no server-side batch filter) and tags subscribers
// accordingly.
//
// Confirmed against Scrub's real /v1/history response shape: each record
// is only ever HistoryStatusValid or HistoryStatusInvalid (with an
// optional error_code on invalid ones) -- there is no risky tier here,
// unlike POST /v1/validate/single's synchronous response. So unlike the
// old (incorrect, never-real) synchronous ValidateBulk path this
// implementation replaces, bulk-imported subscribers can never be tagged
// scrub_status=risky or trigger pauseCampaignsForRiskySubscriberWith --
// that auto-pause-on-risky behavior now only exists for single add/signup
// validation (cmd/scrub_validation.go's validateEmailForAdd). Valid
// records map to StatusDeliverable; invalid ones to StatusUndeliverable
// (the closer of the two existing buckets -- error_code isn't a
// documented enum, so there's no reliable way to split invalid_syntax
// out). Invalid emails are unsubscribed from the batch's target lists,
// non-destructively -- they were already inserted by the time Scrub's
// verdict comes back, unlike the old pre-insert skip.
func (a *App) reconcileScrubBatch(ctx context.Context, batch models.ScrubValidationBatch) {
	// Runs on every exit path (including the early-return guards below) --
	// this batch is done either way, and leaving the row behind would
	// just accumulate dead rows for tenants that, say, disabled Scrub
	// between submission and this callback arriving.
	defer func() {
		if _, err := a.queries.DeleteScrubValidationBatch.Exec(batch.BatchID); err != nil {
			a.log.Printf("scrub: error deleting validation batch %s: %v", batch.BatchID, err)
		}
	}()

	s, err := a.core.GetSettings(ctx, batch.TenantID)
	if err != nil {
		a.log.Printf("scrub: error getting settings to reconcile batch %s (tenant %d): %v", batch.BatchID, batch.TenantID, err)
		return
	}
	if !scrubValidationEnabled(s) || s.Scrub.IntegrationID == "" {
		return
	}

	listIDs := make([]int, len(batch.ListIDs))
	for i, id := range batch.ListIDs {
		listIDs[i] = int(id)
	}

	c := scrub.New(s.Scrub.URL, s.Scrub.APIKey)
	results, err := c.GetHistory(ctx, s.Scrub.IntegrationID, batch.BatchID, batch.SubmittedCount)
	if err != nil {
		a.log.Printf("scrub: error fetching history for batch %s: %v", batch.BatchID, err)
		return
	}

	var deliverable, undeliverable []string
	for _, r := range results {
		if r.Status == scrub.HistoryStatusValid {
			deliverable = append(deliverable, r.Email)
		} else {
			undeliverable = append(undeliverable, r.Email)
		}
	}

	if len(deliverable) > 0 {
		if _, err := a.queries.SetSubscribersScrubStatusByEmail.Exec(batch.TenantID, scrub.StatusDeliverable, pq.Array(deliverable)); err != nil {
			a.log.Printf("scrub: error setting scrub status for batch %s: %v", batch.BatchID, err)
		}
	}
	if len(undeliverable) == 0 {
		return
	}
	if _, err := a.queries.SetSubscribersScrubStatusByEmail.Exec(batch.TenantID, scrub.StatusUndeliverable, pq.Array(undeliverable)); err != nil {
		a.log.Printf("scrub: error setting scrub status for batch %s: %v", batch.BatchID, err)
	}

	if len(listIDs) == 0 {
		return
	}
	subs, err := a.core.GetSubscribersByEmail(ctx, batch.TenantID, undeliverable)
	if err != nil {
		a.log.Printf("scrub: error resolving invalid subscribers for batch %s: %v", batch.BatchID, err)
		return
	}
	if len(subs) == 0 {
		return
	}
	subIDs := make([]int, len(subs))
	for i, sub := range subs {
		subIDs[i] = sub.ID
	}
	if err := a.core.UnsubscribeLists(ctx, batch.TenantID, subIDs, listIDs, nil); err != nil {
		a.log.Printf("scrub: error unsubscribing invalid subscribers for batch %s: %v", batch.BatchID, err)
	}
}
