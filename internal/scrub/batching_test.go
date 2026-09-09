package scrub

import (
	"testing"

	"github.com/knadh/listmonk/models"
)

func makeEmails(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "user@example.com"
	}
	return out
}

func TestChunkEmails(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		size       int
		wantChunks int
		wantLast   int
	}{
		{"empty", 0, 30000, 0, 0},
		{"under one chunk", 100, 30000, 1, 100},
		{"exactly at cap", 30000, 30000, 1, 30000},
		{"one over cap", 30001, 30000, 2, 1},
		{"two full chunks", 60000, 30000, 2, 30000},
		{"several chunks with remainder", 35000, 30000, 2, 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkEmails(makeEmails(tt.total), tt.size)
			if len(chunks) != tt.wantChunks {
				t.Fatalf("len(chunks) = %d, want %d", len(chunks), tt.wantChunks)
			}
			if tt.wantChunks == 0 {
				return
			}

			total := 0
			for i, c := range chunks {
				if i < len(chunks)-1 && len(c) != tt.size {
					t.Errorf("chunk %d len = %d, want %d (full chunk)", i, len(c), tt.size)
				}
				total += len(c)
			}
			if total != tt.total {
				t.Errorf("sum of chunk lengths = %d, want %d", total, tt.total)
			}
			if got := len(chunks[len(chunks)-1]); got != tt.wantLast {
				t.Errorf("last chunk len = %d, want %d", got, tt.wantLast)
			}
		})
	}
}

func TestAggregateBatches(t *testing.T) {
	t.Run("not done until every chunk terminates", func(t *testing.T) {
		batches := []models.ScrubValidationBatch{
			{Status: models.ScrubBatchStatusCompleted, SubmittedCount: 10, ValidatedCount: 10, InvalidCount: 2},
			{Status: models.ScrubBatchStatusProcessing, SubmittedCount: 10, ValidatedCount: 4, InvalidCount: 1},
		}
		completed, submitted, validated, invalid, done := AggregateBatches(2, batches)
		if done {
			t.Error("done = true, want false (one chunk still processing)")
		}
		if completed != 1 {
			t.Errorf("completed = %d, want 1", completed)
		}
		if submitted != 20 || validated != 14 || invalid != 3 {
			t.Errorf("submitted/validated/invalid = %d/%d/%d, want 20/14/3", submitted, validated, invalid)
		}
	})

	t.Run("done once every chunk terminates, mixed outcomes", func(t *testing.T) {
		batches := []models.ScrubValidationBatch{
			{Status: models.ScrubBatchStatusCompleted, SubmittedCount: 10, ValidatedCount: 10, InvalidCount: 2},
			{Status: models.ScrubBatchStatusFailed, SubmittedCount: 10, ValidatedCount: 0, InvalidCount: 0},
		}
		completed, _, _, _, done := AggregateBatches(2, batches)
		if !done {
			t.Error("done = false, want true (both chunks terminal, even though one failed)")
		}
		if completed != 2 {
			t.Errorf("completed = %d, want 2", completed)
		}
	})

	t.Run("not done when a later chunk hasn't been inserted yet", func(t *testing.T) {
		// Only 1 of 3 planned chunks exists as a row so far (an early
		// chunk's webhook arrived before submitScrubListValidation
		// finished submitting the rest) -- must not look done just
		// because every *known* row happens to be terminal.
		batches := []models.ScrubValidationBatch{
			{Status: models.ScrubBatchStatusCompleted, SubmittedCount: 10, ValidatedCount: 10, InvalidCount: 0},
		}
		_, _, _, _, done := AggregateBatches(3, batches)
		if done {
			t.Error("done = true, want false (only 1 of 3 total_batches accounted for)")
		}
	})
}
