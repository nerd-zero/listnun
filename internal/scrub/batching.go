package scrub

import "github.com/knadh/listmonk/models"

// ChunkEmails splits emails into groups of at most size, preserving
// order -- used by cmd/scrub_batch.go's submitScrubListValidation to stay
// under Scrub's per-request cap (POST /v1/validate/integration's
// documented MAX_BATCH_EMAILS) when a list is larger than it.
func ChunkEmails(emails []string, size int) [][]string {
	if len(emails) == 0 {
		return nil
	}
	var chunks [][]string
	for start := 0; start < len(emails); start += size {
		end := start + size
		if end > len(emails) {
			end = len(emails)
		}
		chunks = append(chunks, emails[start:end])
	}
	return chunks
}

// AggregateBatches sums progress across a scrub_validation_job's chunks
// and reports whether the job as a whole has reached a terminal state.
// done compares completed against totalBatches (not len(batches)) since
// chunks are submitted one at a time by submitScrubListValidation and an
// early chunk's webhook callback can plausibly arrive before a later
// chunk has even been inserted -- len(batches) alone would then
// undercount and could flip done=true prematurely.
func AggregateBatches(totalBatches int, batches []models.ScrubValidationBatch) (completed, submitted, validated, invalid int, done bool) {
	for _, b := range batches {
		if b.Status == models.ScrubBatchStatusCompleted || b.Status == models.ScrubBatchStatusFailed {
			completed++
		}
		submitted += b.SubmittedCount
		validated += b.ValidatedCount
		invalid += b.InvalidCount
	}
	return completed, submitted, validated, invalid, completed == totalBatches
}
