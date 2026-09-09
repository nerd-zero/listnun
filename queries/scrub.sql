-- scrub_validation_batches
-- name: insert-scrub-validation-batch
-- Records a CSV-import batch just submitted to Scrub's async
-- POST /v1/validate/integration, so the /webhooks/scrub/batch callback
-- (which only carries a batch_id) can look up its tenant/lists.
-- Authentication is via internal/scrub.VerifyWebhook against the
-- tenant's own Settings.Scrub.APIKey, not a value stored on this row.
INSERT INTO scrub_validation_batches (batch_id, tenant_id, list_ids, submitted_count)
VALUES ($1, $2, $3, $4);

-- name: get-scrub-validation-batch
-- Looked up by batch_id alone -- called from the webhook before any
-- tenant is known, relying on RLS being bypassed when app.current_tenant
-- isn't set (see schema.sql's tenant_isolation policy on this table).
SELECT * FROM scrub_validation_batches WHERE batch_id = $1;

-- name: update-scrub-validation-batch-progress
UPDATE scrub_validation_batches SET status = $2, submitted_count = $3, invalid_count = $4, updated_at = NOW()
WHERE batch_id = $1;

-- name: delete-scrub-validation-batch
DELETE FROM scrub_validation_batches WHERE batch_id = $1;
