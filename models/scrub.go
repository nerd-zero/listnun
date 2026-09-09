package models

import (
	"time"

	"github.com/lib/pq"
)

// Scrub batch statuses, mirroring the "status" values Scrub's
// /webhooks/scrub/batch callback (events.progress) reports verbatim.
const (
	ScrubBatchStatusPending    = "pending"
	ScrubBatchStatusProcessing = "processing"
	ScrubBatchStatusCompleted  = "completed"
	ScrubBatchStatusFailed     = "failed"
)

// ScrubValidationBatch tracks one CSV-import batch submitted to Scrub's
// async POST /v1/validate/integration -- looked up by batch_id from the
// /webhooks/scrub/batch callback (see cmd/scrub_batch.go), since that
// request carries no tenant context of its own; the callback is
// authenticated by verifying Scrub's HMAC signature (internal/scrub's
// VerifyWebhook) against this tenant's own Settings.Scrub.APIKey, not a
// per-batch secret.
type ScrubValidationBatch struct {
	BatchID        string        `db:"batch_id" json:"batch_id"`
	TenantID       int           `db:"tenant_id" json:"tenant_id"`
	ListIDs        pq.Int64Array `db:"list_ids" json:"list_ids"`
	Status         string        `db:"status" json:"status"`
	SubmittedCount int           `db:"submitted_count" json:"submitted_count"`
	InvalidCount   int           `db:"invalid_count" json:"invalid_count"`
	CreatedAt      time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at" json:"updated_at"`
}
