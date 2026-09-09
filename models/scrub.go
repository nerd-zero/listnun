package models

import (
	"time"

	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

// Scrub batch/job statuses, mirroring the "status" values Scrub's
// /webhooks/scrub/batch callback (events.progress) reports verbatim.
const (
	ScrubBatchStatusPending    = "pending"
	ScrubBatchStatusProcessing = "processing"
	ScrubBatchStatusCompleted  = "completed"
	ScrubBatchStatusFailed     = "failed"
)

// ScrubValidationBatch tracks one batch submitted to Scrub's async
// POST /v1/validate/integration -- looked up by batch_id from the
// /webhooks/scrub/batch callback (see cmd/scrub_batch.go), since that
// request carries no tenant context of its own; the callback is
// authenticated by verifying Scrub's HMAC signature (internal/scrub's
// VerifyWebhook) against this tenant's own Settings.Scrub.APIKey, not a
// per-batch secret.
//
// JobID is set when this batch is one chunk of a list-validate job
// (cmd/settings.go's ScrubList, chunked at Scrub's 30,000-email cap by
// submitScrubListValidation) and NULL for a standalone CSV-import batch,
// which never needs more than one batch per commit-window and has no
// aggregate "job" to report progress for.
type ScrubValidationBatch struct {
	BatchID        string        `db:"batch_id" json:"batch_id"`
	TenantID       int           `db:"tenant_id" json:"tenant_id"`
	JobID          null.String   `db:"job_id" json:"job_id,omitempty"`
	ListIDs        pq.Int64Array `db:"list_ids" json:"list_ids"`
	Status         string        `db:"status" json:"status"`
	SubmittedCount int           `db:"submitted_count" json:"submitted_count"`
	ValidatedCount int           `db:"validated_count" json:"validated_count"`
	InvalidCount   int           `db:"invalid_count" json:"invalid_count"`
	CreatedAt      time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at" json:"updated_at"`
}

// ListScrubStatus is one row of the lightweight get-list-scrub-status
// query GetScrubListStatus uses -- deliberately a standalone struct
// rather than three added fields on the widely-shared List model, since
// nothing outside that one handler needs them.
type ListScrubStatus struct {
	ID                    int       `db:"id"`
	ScrubLastValidatedAt  null.Time `db:"scrub_last_validated_at"`
	ScrubLastValidCount   null.Int  `db:"scrub_last_valid_count"`
	ScrubLastInvalidCount null.Int  `db:"scrub_last_invalid_count"`
}

// ScrubValidationJob tracks one list-validate action (cmd/settings.go's
// ScrubList) as a whole -- one job may fan out into multiple
// ScrubValidationBatch rows (chunked at Scrub's 30,000-email
// per-request cap), each with its own Scrub-side batch_id and its own
// event_callback progress. The job is "done" (CompletedBatches ==
// TotalBatches) once every chunk has reached a terminal status
// (completed or failed), independent of whether every individual chunk
// actually succeeded -- see cmd/scrub_batch.go's reconcileScrubJob.
type ScrubValidationJob struct {
	JobID            string        `db:"job_id" json:"job_id"`
	TenantID         int           `db:"tenant_id" json:"tenant_id"`
	ListIDs          pq.Int64Array `db:"list_ids" json:"list_ids"`
	TotalBatches     int           `db:"total_batches" json:"total_batches"`
	CompletedBatches int           `db:"completed_batches" json:"completed_batches"`
	SubmittedCount   int           `db:"submitted_count" json:"submitted_count"`
	ValidatedCount   int           `db:"validated_count" json:"validated_count"`
	InvalidCount     int           `db:"invalid_count" json:"invalid_count"`
	Status           string        `db:"status" json:"status"`
	CreatedAt        time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time     `db:"updated_at" json:"updated_at"`
}
