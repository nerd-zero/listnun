-- scrub_validation_jobs
-- name: insert-scrub-validation-job
-- Records a list-validate action (cmd/settings.go's ScrubList) before its
-- (possibly multiple, chunked) batches are submitted -- total_batches and
-- submitted_count are both known upfront since the full email list is
-- already in hand.
INSERT INTO scrub_validation_jobs (job_id, tenant_id, list_ids, total_batches, submitted_count)
VALUES ($1, $2, $3, $4, $5);

-- name: get-scrub-validation-job
-- Looked up by job_id -- cmd/settings.go's GetScrubListProgress polls this
-- (job_id is what the frontend calls "request_id"/activeJobRequestId).
SELECT * FROM scrub_validation_jobs WHERE job_id = $1;

-- name: get-active-scrub-job-for-list
-- Whether a list already has an in-flight validate job -- used both to
-- block a second concurrent ScrubList click on the same list, and by
-- cmd/campaigns.go's checkScrubJobOnCampaign to block starting a campaign
-- while one of its target lists is being validated.
SELECT * FROM scrub_validation_jobs WHERE tenant_id = $1 AND $2 = ANY(list_ids) AND status = 'processing' LIMIT 1;

-- name: get-active-scrub-jobs-for-tenant
-- Tenant-wide equivalent of the above, fetched once by
-- GetScrubListStatus and matched against every list in Go, rather than
-- one query per list.
SELECT * FROM scrub_validation_jobs WHERE tenant_id = $1 AND status = 'processing';

-- name: update-scrub-validation-job-progress
UPDATE scrub_validation_jobs
SET completed_batches = $2, submitted_count = $3, validated_count = $4, invalid_count = $5, status = $6, updated_at = NOW()
WHERE job_id = $1;

-- name: delete-scrub-validation-job
-- Cascades to every scrub_validation_batches row under this job_id (see
-- that table's job_id FK, ON DELETE CASCADE).
DELETE FROM scrub_validation_jobs WHERE job_id = $1;

-- scrub_validation_batches
-- name: insert-scrub-validation-batch
-- Records one batch submitted to Scrub's async POST /v1/validate/integration,
-- so the /webhooks/scrub/batch callback (which only carries a batch_id) can
-- look up its tenant/lists/parent job. Authentication is via
-- internal/scrub.VerifyWebhook against the tenant's own
-- Settings.Scrub.APIKey, not a value stored on this row. $3 (job_id) must
-- be passed as a real Go nil for a standalone CSV-import batch, not an
-- empty string -- job_id is a UUID column, and casting '' to uuid (e.g.
-- via a NULLIF($3, '') trick) fails outright since '' isn't valid UUID
-- syntax.
INSERT INTO scrub_validation_batches (batch_id, tenant_id, job_id, list_ids, submitted_count)
VALUES ($1, $2, $3, $4, $5);

-- name: get-scrub-validation-batch
-- Looked up by batch_id alone -- called from the webhook before any
-- tenant is known, relying on RLS being bypassed when app.current_tenant
-- isn't set (see schema.sql's tenant_isolation policy on this table).
SELECT * FROM scrub_validation_batches WHERE batch_id = $1;

-- name: get-scrub-validation-batches-by-job
-- Every chunk of one list-validate job -- used to aggregate progress
-- into the parent job row, and by reconcileScrubJob to fetch each
-- chunk's Scrub-side history once the whole job completes.
SELECT * FROM scrub_validation_batches WHERE job_id = $1;

-- name: update-scrub-validation-batch-progress
UPDATE scrub_validation_batches SET status = $2, submitted_count = $3, validated_count = $4, invalid_count = $5, updated_at = NOW()
WHERE batch_id = $1;

-- name: delete-scrub-validation-batch
DELETE FROM scrub_validation_batches WHERE batch_id = $1;
