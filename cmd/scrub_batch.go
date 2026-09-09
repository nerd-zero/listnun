package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

// scrubBatchMaxEmails is Scrub's documented hard cap per POST
// /v1/validate/integration call (confirmed against Scrub's own
// MAX_BATCH_EMAILS in modules/validation/schemas.py, shared across
// bulk/CSV/integration requests alike). subimporter's CSV-import windows
// (commitBatchSize, 10,000) stay safely under this without needing to
// chunk; submitScrubListValidation below does chunk, since a list can
// have arbitrarily many subscribers.
const scrubBatchMaxEmails = 30000

// submitScrubBatch submits one batch of emails to Scrub's async
// POST /v1/validate/integration and records a scrub_validation_batches
// row so the /webhooks/scrub/batch callback below -- which carries no
// tenant context of its own, just a batch_id -- can later look up which
// tenant/lists/parent job the batch belongs to. jobID is "" for a
// standalone CSV-import batch (internal/subimporter's ScrubSubmitFunc,
// via cmd/tenant_importer.go) and a scrub_validation_jobs id for one
// chunk of a list-validate job (submitScrubListValidation below).
//
// Package-level (not an *App method) so cmd/tenant_importer.go's
// per-tenant ScrubSubmitFunc closure can call it without needing a full
// *App reference, same reasoning as pauseCampaignsForRiskySubscriberWith
// in cmd/scrub_validation.go.
//
// Returns ("", nil) when Scrub isn't configured for tenantID at all --
// deliberately not an error, since CSV import's caller (flushWindow)
// would otherwise log a spurious "error" on every commit-window for
// every tenant that simply doesn't use Scrub. Any other returned error is
// a genuine failure; callers that need "fail open, just log" (CSV import)
// get that from the caller side (subimporter's flushWindow already logs
// and continues on a returned error), while submitScrubListValidation
// below treats one being non-nil as reason to fail the whole job.
func submitScrubBatch(ctx context.Context, q *models.Queries, tenantID int, s models.Settings, listIDs []int, emails []string, jobID string) (batchID string, err error) {
	if !scrubValidationEnabled(s) || s.Scrub.IntegrationID == "" {
		return "", nil
	}

	root := strings.TrimRight(strings.TrimSpace(s.AppRootURL), "/")
	if root == "" {
		log.Printf("scrub: tenant %d has no app.root_url configured, skipping batch submission (%d emails will remain unchecked)", tenantID, len(emails))
		return "", nil
	}

	batchUUID, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("generating batch id: %w", err)
	}
	batchID = batchUUID.String()

	// job_id is a UUID column -- pass a real nil for a standalone
	// CSV-import batch (jobID == ""), not the empty string, which isn't
	// valid UUID syntax and would fail the insert outright.
	var jobIDArg any
	if jobID != "" {
		jobIDArg = jobID
	}
	if _, err := q.InsertScrubValidationBatch.Exec(batchID, tenantID, jobIDArg, pq.Array(listIDs), len(emails)); err != nil {
		return "", fmt.Errorf("recording validation batch: %w", err)
	}

	eventCallback := root + "/webhooks/scrub/batch"

	c := scrub.New(s.Scrub.URL, s.Scrub.APIKey)
	sub, err := c.SubmitBatch(ctx, s.Scrub.IntegrationID, batchID, eventCallback, emails)
	if err != nil {
		// Scrub never accepted this batch -- no webhook will ever arrive
		// for it, so the row would otherwise linger forever.
		if _, delErr := q.DeleteScrubValidationBatch.Exec(batchID); delErr != nil {
			log.Printf("scrub: error cleaning up unsubmitted batch %s: %v", batchID, delErr)
		}
		return "", fmt.Errorf("submitting validation batch: %w", err)
	}

	if _, err := q.UpdateScrubValidationBatchProgress.Exec(batchID, models.ScrubBatchStatusPending, sub.SubmittedCount, 0, 0); err != nil {
		log.Printf("scrub: error updating validation batch progress: %v", err)
	}

	return batchID, nil
}

// submitScrubListValidation is cmd/settings.go's ScrubList: fetches
// listID's own subscriber emails and submits them to Scrub directly
// (chunked at scrubBatchMaxEmails), tracked as one scrub_validation_jobs
// row so the UI can poll a single activeJobRequestId regardless of how
// many chunks it took. Unlike submitScrubBatch's fail-open CSV-import
// convention, a submission failure here fails the whole job immediately
// -- this is a user-initiated, foreground action the caller can just
// retry, rather than a background job left showing stuck partial
// progress forever.
func submitScrubListValidation(ctx context.Context, q *models.Queries, tenantID int, s models.Settings, listID int, emails []string) (jobID string, err error) {
	jobUUID, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("generating job id: %w", err)
	}
	jobID = jobUUID.String()

	chunks := scrub.ChunkEmails(emails, scrubBatchMaxEmails)

	if _, err := q.InsertScrubValidationJob.Exec(jobID, tenantID, pq.Array([]int{listID}), len(chunks), len(emails)); err != nil {
		return "", fmt.Errorf("recording validation job: %w", err)
	}

	for _, chunk := range chunks {
		if _, err := submitScrubBatch(ctx, q, tenantID, s, []int{listID}, chunk, jobID); err != nil {
			// Cascades to delete any chunks that succeeded before this
			// one failed (scrub_validation_batches.job_id FK, ON DELETE
			// CASCADE).
			if _, delErr := q.DeleteScrubValidationJob.Exec(jobID); delErr != nil {
				log.Printf("scrub: error cleaning up failed validation job %s: %v", jobID, delErr)
			}
			return "", fmt.Errorf("submitting batch: %w", err)
		}
	}

	return jobID, nil
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
		batch.BatchID, progress.Status, progress.SubmittedTotalCount, progress.ValidatedTotalCount, progress.InvalidTotalCount,
	); err != nil {
		a.log.Printf("scrub: error updating validation batch %s: %v", batch.BatchID, err)
	}

	// A job-linked chunk (list-validate) is aggregated into its parent
	// job rather than reconciled/deleted in isolation here -- the job,
	// not any one chunk, is the unit the UI polls and reconciliation
	// runs against once every chunk has terminated.
	if batch.JobID.Valid {
		a.updateScrubJobProgress(c.Request().Context(), batch.JobID.String)
		return c.JSON(http.StatusOK, okResp{true})
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

// updateScrubJobProgress aggregates every chunk's progress under jobID
// into its scrub_validation_jobs row, and triggers reconcileScrubJob once
// every chunk has reached a terminal status (completed or failed).
func (a *App) updateScrubJobProgress(ctx context.Context, jobID string) {
	var job models.ScrubValidationJob
	if err := a.queries.GetScrubValidationJob.Get(&job, jobID); err != nil {
		a.log.Printf("scrub: error loading validation job %s: %v", jobID, err)
		return
	}

	var batches []models.ScrubValidationBatch
	if err := a.queries.GetScrubValidationBatchesByJob.Select(&batches, jobID); err != nil {
		a.log.Printf("scrub: error listing batches for job %s: %v", jobID, err)
		return
	}

	completed, submitted, validated, invalid, done := scrub.AggregateBatches(job.TotalBatches, batches)
	status := models.ScrubBatchStatusProcessing
	if done {
		status = models.ScrubBatchStatusCompleted
	}

	if _, err := a.queries.UpdateScrubValidationJobProgress.Exec(jobID, completed, submitted, validated, invalid, status); err != nil {
		a.log.Printf("scrub: error updating validation job %s: %v", jobID, err)
	}
	if !done {
		return
	}

	go a.reconcileScrubJob(context.Background(), job)
}

// classifyHistoryResults splits Scrub GET /v1/history results into
// deliverable (valid) and undeliverable (invalid) email lists -- shared
// by reconcileScrubBatch (one Scrub batch_id) and reconcileScrubJob
// (merged across every chunk's batch_id).
func classifyHistoryResults(results []scrub.HistoryResult) (deliverable, undeliverable []string) {
	for _, r := range results {
		if r.Status == scrub.HistoryStatusValid {
			deliverable = append(deliverable, r.Email)
		} else {
			undeliverable = append(undeliverable, r.Email)
		}
	}
	return deliverable, undeliverable
}

// applyScrubResults tags subscribers.scrub_status for deliverable/
// undeliverable emails and unsubscribes the undeliverable ones from
// listIDs, non-destructively -- they were already inserted/already on
// the list by the time Scrub's verdict comes back. The convergence point
// both reconcileScrubBatch (CSV import) and reconcileScrubJob
// (list-validate) reach once they've each gathered their own full result
// set.
//
// Confirmed against Scrub's real /v1/history response shape: each record
// is only ever HistoryStatusValid or HistoryStatusInvalid (with an
// optional error_code on invalid ones) -- there is no risky tier here,
// unlike POST /v1/validate/single's synchronous response. So neither
// caller can ever tag scrub_status=risky or trigger
// pauseCampaignsForRiskySubscriberWith -- that auto-pause-on-risky
// behavior only exists for single add/signup validation
// (cmd/scrub_validation.go's validateEmailForAdd). Valid records map to
// StatusDeliverable; invalid ones to StatusUndeliverable (the closer of
// the two existing buckets -- error_code isn't a documented enum, so
// there's no reliable way to split invalid_syntax out).
func (a *App) applyScrubResults(ctx context.Context, tenantID int, listIDs []int, deliverable, undeliverable []string) {
	if len(deliverable) > 0 {
		if _, err := a.queries.SetSubscribersScrubStatusByEmail.Exec(tenantID, scrub.StatusDeliverable, pq.Array(deliverable)); err != nil {
			a.log.Printf("scrub: error setting scrub status (tenant %d): %v", tenantID, err)
		}
	}
	if len(undeliverable) == 0 {
		return
	}
	if _, err := a.queries.SetSubscribersScrubStatusByEmail.Exec(tenantID, scrub.StatusUndeliverable, pq.Array(undeliverable)); err != nil {
		a.log.Printf("scrub: error setting scrub status (tenant %d): %v", tenantID, err)
	}
	if len(listIDs) == 0 {
		return
	}
	subs, err := a.core.GetSubscribersByEmail(ctx, tenantID, undeliverable)
	if err != nil {
		a.log.Printf("scrub: error resolving invalid subscribers (tenant %d): %v", tenantID, err)
		return
	}
	if len(subs) == 0 {
		return
	}
	subIDs := make([]int, len(subs))
	for i, sub := range subs {
		subIDs[i] = sub.ID
	}
	if err := a.core.UnsubscribeLists(ctx, tenantID, subIDs, listIDs, nil); err != nil {
		a.log.Printf("scrub: error unsubscribing invalid subscribers (tenant %d): %v", tenantID, err)
	}
}

// reconcileScrubBatch runs once a standalone (job-less, i.e. CSV-import)
// batch reaches status "completed". The callback payload itself only
// carries cumulative counts, not per-email detail, so this fetches the
// batch's full record set via GetHistory (paging /v1/history and matching
// batch_id client-side, since that endpoint has no server-side batch
// filter) and tags subscribers accordingly -- see applyScrubResults for
// what happens with the result.
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

	deliverable, undeliverable := classifyHistoryResults(results)
	a.applyScrubResults(ctx, batch.TenantID, listIDs, deliverable, undeliverable)
	for _, listID := range listIDs {
		if _, err := a.queries.UpdateListScrubResult.Exec(listID, len(deliverable), len(undeliverable)); err != nil {
			a.log.Printf("scrub: error updating list %d scrub result for batch %s: %v", listID, batch.BatchID, err)
		}
	}
}

// reconcileScrubJob runs once every chunk of a list-validate job
// (cmd/settings.go's ScrubList, via submitScrubListValidation) has
// reached a terminal status. Mirrors reconcileScrubBatch, merged across
// every chunk's Scrub-side batch_id before tagging/unsubscribing once for
// the whole job. A chunk that itself failed outright is skipped -- there's
// nothing to fetch for it, and its emails simply stay scrub_status=NULL,
// the same gap reconcileScrubBatch accepts for a fully-failed batch.
func (a *App) reconcileScrubJob(ctx context.Context, job models.ScrubValidationJob) {
	defer func() {
		if _, err := a.queries.DeleteScrubValidationJob.Exec(job.JobID); err != nil {
			a.log.Printf("scrub: error deleting validation job %s: %v", job.JobID, err)
		}
	}()

	s, err := a.core.GetSettings(ctx, job.TenantID)
	if err != nil {
		a.log.Printf("scrub: error getting settings to reconcile job %s (tenant %d): %v", job.JobID, job.TenantID, err)
		return
	}
	if !scrubValidationEnabled(s) || s.Scrub.IntegrationID == "" {
		return
	}

	var batches []models.ScrubValidationBatch
	if err := a.queries.GetScrubValidationBatchesByJob.Select(&batches, job.JobID); err != nil {
		a.log.Printf("scrub: error listing batches for job %s: %v", job.JobID, err)
		return
	}

	listIDs := make([]int, len(job.ListIDs))
	for i, id := range job.ListIDs {
		listIDs[i] = int(id)
	}

	c := scrub.New(s.Scrub.URL, s.Scrub.APIKey)
	var deliverable, undeliverable []string
	anyCompleted := false
	for _, b := range batches {
		if b.Status != models.ScrubBatchStatusCompleted {
			continue
		}
		anyCompleted = true
		results, err := c.GetHistory(ctx, s.Scrub.IntegrationID, b.BatchID, b.SubmittedCount)
		if err != nil {
			a.log.Printf("scrub: error fetching history for job %s batch %s: %v", job.JobID, b.BatchID, err)
			continue
		}
		d, u := classifyHistoryResults(results)
		deliverable = append(deliverable, d...)
		undeliverable = append(undeliverable, u...)
	}

	a.applyScrubResults(ctx, job.TenantID, listIDs, deliverable, undeliverable)

	// Only refresh the list's "last validated" summary if at least one
	// chunk actually completed -- a job where every chunk failed outright
	// has nothing genuine to report, and 0/0 counts would misleadingly
	// read as "validated, found nothing" rather than "never ran".
	if !anyCompleted {
		return
	}
	for _, listID := range listIDs {
		if _, err := a.queries.UpdateListScrubResult.Exec(listID, len(deliverable), len(undeliverable)); err != nil {
			a.log.Printf("scrub: error updating list %d scrub result for job %s: %v", listID, job.JobID, err)
		}
	}
}
