package scrub

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// signWebhook builds the X-Webhook-Signature header value a real Scrub
// delivery would carry for payload, signed with apiKey at ts -- mirrors
// VerifyWebhook's own derivation so tests exercise the real algorithm
// rather than a mocked one.
func signWebhook(payload []byte, apiKey string, ts time.Time) string {
	rawSecret := sha256.Sum256([]byte(apiKey))
	secretHex := hex.EncodeToString(rawSecret[:])

	timestamp := fmt.Sprintf("%d", ts.Unix())
	toSign := fmt.Sprintf("%s.%s", timestamp, payload)
	h := hmac.New(sha256.New, []byte(secretHex))
	h.Write([]byte(toSign))
	sig := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("t=%s,v1=%s", timestamp, sig)
}

func TestVerifyWebhook(t *testing.T) {
	payload := []byte(`{"event_type":"events.progress"}`)
	apiKey := "sk_test_123"

	headers := http.Header{}
	headers.Set("X-Webhook-Signature", signWebhook(payload, apiKey, time.Now()))

	if err := VerifyWebhook(payload, headers, apiKey); err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
}

func TestVerifyWebhook_WrongKey(t *testing.T) {
	payload := []byte(`{"event_type":"events.progress"}`)

	headers := http.Header{}
	headers.Set("X-Webhook-Signature", signWebhook(payload, "sk_real", time.Now()))

	if err := VerifyWebhook(payload, headers, "sk_wrong"); err == nil {
		t.Fatal("expected verification to fail with the wrong API key")
	}
}

func TestVerifyWebhook_TamperedPayload(t *testing.T) {
	apiKey := "sk_test_123"
	original := []byte(`{"event_type":"events.progress"}`)
	tampered := []byte(`{"event_type":"events.completed"}`)

	headers := http.Header{}
	headers.Set("X-Webhook-Signature", signWebhook(original, apiKey, time.Now()))

	if err := VerifyWebhook(tampered, headers, apiKey); err == nil {
		t.Fatal("expected verification to fail for a tampered payload")
	}
}

func TestVerifyWebhook_StaleTimestamp(t *testing.T) {
	payload := []byte(`{"event_type":"events.progress"}`)
	apiKey := "sk_test_123"

	headers := http.Header{}
	headers.Set("X-Webhook-Signature", signWebhook(payload, apiKey, time.Now().Add(-1*time.Hour)))

	if err := VerifyWebhook(payload, headers, apiKey); err == nil {
		t.Fatal("expected verification to fail for a stale timestamp")
	}
}

func TestVerifyWebhook_MissingHeader(t *testing.T) {
	if err := VerifyWebhook([]byte(`{}`), http.Header{}, "sk_test"); err == nil {
		t.Fatal("expected verification to fail with no signature header")
	}
}

func TestValidateSingle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/validate/single" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "sk_test" {
			t.Errorf("X-API-Key = %q, want sk_test", got)
		}
		if got := r.URL.Query().Get("email"); got != "user@example.com" {
			t.Errorf("email query param = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Result{Email: "user@example.com", Status: StatusRisky, Reason: "low quality score"})
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	res, err := c.ValidateSingle(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("ValidateSingle: %v", err)
	}
	if res.Status != StatusRisky {
		t.Errorf("Status = %q, want %q", res.Status, StatusRisky)
	}
}

func TestValidateSingle_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_bad")
	if _, err := c.ValidateSingle(context.Background(), "user@example.com"); err == nil {
		t.Fatal("expected an error for a 401 response, got nil")
	}
}

func TestSubmitBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/validate/integration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "sk_test" {
			t.Errorf("X-API-Key = %q, want sk_test", got)
		}

		var body struct {
			IntegrationID string   `json:"integration_id"`
			BatchID       string   `json:"batch_id"`
			Emails        []string `json:"emails"`
			EventCallback string   `json:"event_callback"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.IntegrationID != "int_123" {
			t.Errorf("integration_id = %q, want int_123", body.IntegrationID)
		}
		if body.BatchID != "batch_123" {
			t.Errorf("batch_id = %q, want batch_123", body.BatchID)
		}
		if body.EventCallback != "https://listmonk.example/webhooks/scrub/batch?token=t" {
			t.Errorf("event_callback = %q", body.EventCallback)
		}
		if len(body.Emails) != 2 {
			t.Errorf("len(emails) = %d, want 2", len(body.Emails))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": map[string]any{
				"request_id":      "req_123",
				"integration_id":  101, // deliberately a bare number, per Scrub's documented example
				"batch_id":        "batch_123",
				"submitted_count": 2,
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	res, err := c.SubmitBatch(context.Background(), "int_123", "batch_123", "https://listmonk.example/webhooks/scrub/batch?token=t", []string{"a@example.com", "b@example.com"})
	if err != nil {
		t.Fatalf("SubmitBatch: %v", err)
	}
	if res.RequestID != "req_123" {
		t.Errorf("RequestID = %q, want req_123", res.RequestID)
	}
	if res.BatchID != "batch_123" {
		t.Errorf("BatchID = %q, want batch_123", res.BatchID)
	}
	if res.SubmittedCount != 2 {
		t.Errorf("SubmittedCount = %d, want 2", res.SubmittedCount)
	}
}

func TestSubmitBatch_OmitsOptionalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["batch_id"]; ok {
			t.Errorf("batch_id should be omitted when empty")
		}
		if _, ok := body["event_callback"]; ok {
			t.Errorf("event_callback should be omitted when empty")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": map[string]any{"submitted_count": 1}})
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	if _, err := c.SubmitBatch(context.Background(), "int_123", "", "", []string{"a@example.com"}); err != nil {
		t.Fatalf("SubmitBatch: %v", err)
	}
}

// TestGetHistory verifies GetHistory pages through /v1/history (which has
// no batch_id filter, only integration_id/invalid_only/limit/cursor) and
// matches batch_id client-side, ignoring records from other batches
// interleaved in the same history log.
func TestGetHistory(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("batch_id"); got != "" {
			t.Errorf("batch_id should never be sent as a query param (unsupported by the real API), got %q", got)
		}
		if got := r.URL.Query().Get("integration_id"); got != "int_123" {
			t.Errorf("integration_id = %q, want int_123", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("cursor") {
		case "":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"has_more":    true,
				"next_cursor": "page2",
				"results": []HistoryResult{
					{BatchID: "batch_999", Email: "unrelated@example.com", Status: HistoryStatusValid},
					{BatchID: "batch_123", Email: "a@example.com", Status: HistoryStatusValid},
				},
			})
		case "page2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"has_more":    false,
				"next_cursor": "",
				"results": []HistoryResult{
					{BatchID: "batch_123", Email: "bad@example.com", Status: HistoryStatusInvalid, ErrorCode: "domain_not_found"},
				},
			})
		default:
			t.Errorf("unexpected cursor: %s", r.URL.Query().Get("cursor"))
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	res, err := c.GetHistory(context.Background(), "int_123", "batch_123", 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (two pages)", calls)
	}
	if len(res) != 2 {
		t.Fatalf("len(res) = %d, want 2 (unrelated batch_999 record excluded)", len(res))
	}

	byEmail := map[string]HistoryResult{}
	for _, r := range res {
		byEmail[r.Email] = r
	}
	if _, ok := byEmail["unrelated@example.com"]; ok {
		t.Errorf("record from a different batch_id leaked into results")
	}
	if byEmail["a@example.com"].Status != HistoryStatusValid {
		t.Errorf("a@example.com status = %q, want %q", byEmail["a@example.com"].Status, HistoryStatusValid)
	}
	if byEmail["bad@example.com"].Status != HistoryStatusInvalid {
		t.Errorf("bad@example.com status = %q, want %q", byEmail["bad@example.com"].Status, HistoryStatusInvalid)
	}
	if byEmail["bad@example.com"].ErrorCode != "domain_not_found" {
		t.Errorf("bad@example.com error_code = %q, want domain_not_found", byEmail["bad@example.com"].ErrorCode)
	}
}

// TestGetHistory_StopsAtWantCount verifies GetHistory stops paging once
// it has collected wantCount matches, rather than always scanning
// everything has_more allows.
func TestGetHistory_StopsAtWantCount(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"has_more":    true,
			"next_cursor": "more",
			"results": []HistoryResult{
				{BatchID: "batch_123", Email: "a@example.com", Status: HistoryStatusValid},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	res, err := c.GetHistory(context.Background(), "int_123", "batch_123", 1)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should stop once wantCount is met)", calls)
	}
	if len(res) != 1 {
		t.Errorf("len(res) = %d, want 1", len(res))
	}
}
