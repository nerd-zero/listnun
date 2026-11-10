package scrub

import (
	"encoding/json"
	"testing"
	"time"
)

// Regression: Scrub sends created_at without a UTC offset, which failed
// decoding GET /v1/history, so reconciling a validated batch or list never
// set any subscriber's scrub_status.
func TestHistoryResultDecodesScrubTimestamps(t *testing.T) {
	body := `[
		{"request_id":"r1","email":"a@example.com","status":"valid","created_at":"2026-09-23T12:04:38.294577"},
		{"request_id":"r2","email":"b@example.com","status":"invalid","created_at":"2026-09-23T14:04:38+02:00"},
		{"request_id":"r3","email":"c@example.com","status":"valid","created_at":null}
	]`
	var got []HistoryResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode history results: %v", err)
	}
	want := time.Date(2026, 9, 23, 12, 4, 38, 294577000, time.UTC)
	if !got[0].CreatedAt.Equal(want) {
		t.Errorf("no-offset created_at = %v, want %v (read as UTC)", got[0].CreatedAt.Time, want)
	}
	if !got[1].CreatedAt.Equal(want.Truncate(time.Second)) {
		t.Errorf("offset created_at = %v, want %v", got[1].CreatedAt.Time, want.Truncate(time.Second))
	}
	if !got[2].CreatedAt.IsZero() {
		t.Errorf("null created_at = %v, want zero", got[2].CreatedAt.Time)
	}
}

func TestWebhookEnvelopeDecodesScrubTimestamp(t *testing.T) {
	var env WebhookEnvelope
	if err := json.Unmarshal([]byte(`{"event_id":"e","event_type":"events.progress","created_at":"2026-09-23T12:04:38.294577","data":{}}`), &env); err != nil {
		t.Fatalf("decode webhook envelope: %v", err)
	}
}

func TestTimestampRejectsGarbage(t *testing.T) {
	var ts Timestamp
	if err := json.Unmarshal([]byte(`"yesterday"`), &ts); err == nil {
		t.Error("decoding \"yesterday\" succeeded, want an error")
	}
}
