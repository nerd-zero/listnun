// Package scrub is a thin client for Scrub's per-email validation APIs:
// POST /v1/validate/single (synchronous, single email -- used at the
// moment a subscriber is actually added) and the batch pair POST
// /v1/validate/integration + GET /v1/history (asynchronous -- used for
// CSV import, see internal/subimporter and cmd/scrub_batch.go). Distinct
// from the Listmonk-specific list-hygiene-job endpoints
// (/v1/integrations/{id}/lists/...) already called ad hoc from
// cmd/campaigns.go and cmd/settings.go, which trigger a list-wide re-scan
// rather than validating a caller-supplied set of emails.
//
// Client instances are built fresh per call from a tenant's stored
// settings (models.Settings.Scrub.URL/APIKey), the same ephemeral-client
// convention checkScrubJobOnCampaign/TestScrubSettings/ScrubList already
// use -- not a boot-wired singleton, since the URL/key are per-tenant.
package scrub

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Status values Scrub's POST /v1/validate/single returns, and the
// subscribers.scrub_status vocabulary derived from them. NOT the same
// vocabulary GET /v1/history uses (see HistoryStatusValid/Invalid below) --
// history only ever reports "valid"/"invalid", with no risky tier, so
// bulk-imported subscribers can never be flagged StatusRisky the way a
// single add/signup can (see cmd/scrub_batch.go's reconcileScrubBatch).
const (
	StatusDeliverable   = "deliverable"
	StatusUndeliverable = "undeliverable"
	StatusInvalidSyntax = "invalid_syntax"
	StatusRisky         = "risky"

	// StatusUncheckedError is a subscribers.scrub_status sentinel value
	// (not something Scrub itself ever returns): "Scrub was enabled and
	// reachable-attempt was made but errored". Distinct from an unset
	// scrub_status (never checked) so an outage is observable rather
	// than silently indistinguishable from "nothing risky found". Shared
	// between cmd/scrub_validation.go (single-email calls) and
	// internal/subimporter (bulk calls) so both fail the same way.
	StatusUncheckedError = "unchecked_error"
)

// Status values one GET /v1/history record's "status" field holds --
// Scrub's own confirmed vocabulary for this endpoint, distinct from (and
// coarser than) the subscribers.scrub_status values above.
const (
	HistoryStatusValid   = "valid"
	HistoryStatusInvalid = "invalid"
)

// Result is the response shape of POST /v1/validate/single.
type Result struct {
	Email  string `json:"email"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// BatchSubmission is the ack POST /v1/validate/integration returns --
// just an acknowledgment that Scrub accepted the batch for async
// processing, not a result. Actual per-email results arrive later, either
// via the event_callback webhook (progress/counts only) or by fetching
// GetHistory once the callback reports the batch as completed.
//
// integration_id is deliberately omitted here even though Scrub's real
// schema (IntegrationValidateResults) always returns it as a string --
// we already know our own integration_id (we sent it), so there's
// nothing to gain from round-tripping it through this response.
type BatchSubmission struct {
	RequestID      string `json:"request_id"`
	BatchID        string `json:"batch_id"`
	SubmittedCount int    `json:"submitted_count"`
}

// HistoryResult is one entry of GET /v1/history's results. Confirmed
// shape -- Status is only ever HistoryStatusValid/HistoryStatusInvalid,
// with ErrorCode ("domain_not_found" etc., empty when valid) as the only
// detail on why an invalid one failed; there is no risky tier here.
// BatchID is included on every record, even though /v1/history itself
// can't be *filtered* by batch_id server-side (only integration_id,
// invalid_only, limit, cursor are accepted query params) -- GetHistory
// pages through and matches on this field client-side.
type HistoryResult struct {
	RequestID string    `json:"request_id"`
	BatchID   string    `json:"batch_id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	ErrorCode string    `json:"error_code"`
	CreatedAt time.Time `json:"created_at"`
}

// WebhookEnvelope is the outer shape of every event_callback delivery
// POST /v1/validate/integration triggers. Data's shape depends on
// EventType -- callers decode it themselves (json.RawMessage) once they
// know which event this is; the only confirmed EventType so far is
// "events.progress", whose Data matches ProgressEvent below.
type WebhookEnvelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	CreatedAt     time.Time       `json:"created_at"`
	IntegrationID string          `json:"integration_id"`
	Data          json.RawMessage `json:"data"`
}

// EventTypeProgress is the only confirmed WebhookEnvelope.EventType.
const EventTypeProgress = "events.progress"

// ProgressEvent is WebhookEnvelope.Data's shape for EventTypeProgress.
// Status is one of models.ScrubBatchStatusPending/Processing/Completed/
// Failed -- those constants hold the same literal values Scrub sends
// here, kept at the models layer (like models.CampaignStatusPaused)
// rather than duplicated in this package.
type ProgressEvent struct {
	BatchID             string   `json:"batch_id"`
	Status              string   `json:"status"`
	SubmittedTotalCount int      `json:"submitted_total_count"`
	ValidatedTotalCount int      `json:"validated_total_count"`
	InvalidTotalCount   int      `json:"invalid_total_count"`
	LatestInvalidEmails []string `json:"latest_invalid_emails"`
}

// ErrWebhookVerification is returned by VerifyWebhook for any signature,
// header, or timestamp failure -- callers should respond 403 without
// distinguishing the exact cause.
var ErrWebhookVerification = errors.New("scrub webhook verification failed")

// webhookTolerance bounds how old/new a webhook's timestamp may be before
// it's rejected, guarding against replaying a captured payload.
const webhookTolerance = 5 * time.Minute

// VerifyWebhook checks payload against the X-Webhook-Signature header
// Scrub sends on every event_callback delivery. apiKey is the tenant's
// own models.Settings.Scrub.APIKey -- the same key SubmitBatch
// authenticates outbound calls with. Scrub derives its per-delivery
// signing secret as hex(sha256(apiKey)), so there is no separately
// configured webhook secret.
//
// Verbatim port of listnun's internal/scrubclient.VerifyWebhook (the
// platform's own, already-working implementation against the real Scrub
// service) -- reused here rather than re-derived so listmonk's receiver
// matches Scrub's actual signing scheme exactly, not a guess.
func VerifyWebhook(payload []byte, headers http.Header, apiKey string) error {
	sigHeader := headers.Get("X-Webhook-Signature")
	if sigHeader == "" {
		return fmt.Errorf("%w: missing signature header", ErrWebhookVerification)
	}
	if apiKey == "" {
		return fmt.Errorf("%w: scrub not configured", ErrWebhookVerification)
	}

	var timestamp, sig string
	for _, part := range strings.Split(sigHeader, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			timestamp = v
		case "v1":
			sig = v
		}
	}
	if timestamp == "" || sig == "" {
		return fmt.Errorf("%w: malformed signature header", ErrWebhookVerification)
	}

	timestampUnix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp", ErrWebhookVerification)
	}
	ts := time.Unix(timestampUnix, 0)
	if now := time.Now(); now.Sub(ts) > webhookTolerance || ts.Sub(now) > webhookTolerance {
		return fmt.Errorf("%w: timestamp outside tolerance", ErrWebhookVerification)
	}

	// The signing key is the *hex string* sha256(api_key).hexdigest(),
	// encoded to UTF-8 bytes -- not the raw 32-byte digest.
	rawSecret := sha256.Sum256([]byte(apiKey))
	secretHex := hex.EncodeToString(rawSecret[:])

	toSign := fmt.Sprintf("%s.%s", timestamp, payload)
	h := hmac.New(sha256.New, []byte(secretHex))
	h.Write([]byte(toSign))
	expected := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("%w: signature mismatch", ErrWebhookVerification)
	}
	return nil
}

// Client validates emails against a Scrub instance.
type Client struct {
	baseURL   string
	apiKey    string
	http      *http.Client // short timeout, single-email/submit calls
	batchHTTP *http.Client // longer timeout, GetHistory pagination
}

// New builds a Client for one call -- baseURL/apiKey come from a tenant's
// models.Settings.Scrub, read fresh by the caller on every use (settings
// can change at runtime), not cached here.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:    apiKey,
		http:      &http.Client{Timeout: 10 * time.Second},
		batchHTTP: &http.Client{Timeout: 30 * time.Second},
	}
}

// ValidateSingle validates one email address synchronously.
func (c *Client) ValidateSingle(ctx context.Context, email string) (Result, error) {
	var out Result

	u := fmt.Sprintf("%s/v1/validate/single?email=%s", c.baseURL, url.QueryEscape(email))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return out, fmt.Errorf("scrub: validate/single: status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("scrub: validate/single: decode: %w", err)
	}
	return out, nil
}

// SubmitBatch hands a batch of emails to Scrub for asynchronous
// validation. batchID and eventCallback are omitted from the request body
// when empty (both are documented as optional). Scrub's response is only
// an acknowledgment (BatchSubmission) -- actual results arrive later via
// eventCallback, or by calling GetHistory once that callback reports the
// batch as completed. Callers are expected to submit already-bounded
// windows (see internal/subimporter's commitBatchSize, 10,000) rather
// than relying on this method to chunk -- Scrub's own MAX_BATCH_EMAILS
// cap for this endpoint is 30,000 (confirmed against Scrub's own
// modules/validation/schemas.py -- shared across bulk/CSV/integration
// requests alike), safely above commitBatchSize.
//
// Endpoint and request/response shapes confirmed directly against
// Scrub's own source (modules/validation/router.py + schemas.py, mounted
// at /v1/validate per main.py) as well as listnun's already-working
// client: POST /v1/validate/integration -- not /v1/integrations/validate.
func (c *Client) SubmitBatch(ctx context.Context, integrationID, batchID, eventCallback string, emails []string) (BatchSubmission, error) {
	var out struct {
		Results BatchSubmission `json:"results"`
	}

	payload := map[string]any{
		"integration_id": integrationID,
		"emails":         emails,
	}
	if batchID != "" {
		payload["batch_id"] = batchID
	}
	if eventCallback != "" {
		payload["event_callback"] = eventCallback
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return out.Results, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/validate/integration", bytes.NewReader(body))
	if err != nil {
		return out.Results, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return out.Results, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return out.Results, fmt.Errorf("scrub: validate/integration: status %d: %s", resp.StatusCode, string(respBody))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out.Results, fmt.Errorf("scrub: validate/integration: decode: %w", err)
	}
	return out.Results, nil
}

// historyPageSize is the page size requested per GetHistory page.
const historyPageSize = 200

// historyMaxPages bounds how many pages GetHistory will scan looking for
// one batch's records. /v1/history has no batch_id *filter* -- only
// integration_id/invalid_only/limit/cursor are accepted query params --
// so scoping to one batch means paging through the tenant's whole history
// (newest first, per Scrub's documented ordering) and matching batch_id
// client-side. In practice a batch's records surface within the first
// page or two, since GetHistory is only ever called right after that
// batch's own "completed" callback; this cap just bounds the worst case
// on a very active integration rather than scanning forever.
const historyMaxPages = 200

// GetHistory fetches one batch's full per-email results by paging through
// Scrub's GET /v1/history (the same endpoint cmd/settings.go's
// GetScrubHistory proxies for the UI's audit log) and matching batch_id
// client-side on each returned record, stopping once wantCount matches
// are found (pass the batch's known submitted_count), history is
// exhausted, or historyMaxPages is reached.
func (c *Client) GetHistory(ctx context.Context, integrationID, batchID string, wantCount int) ([]HistoryResult, error) {
	var (
		matched []HistoryResult
		cursor  string
	)

	for page := 0; page < historyMaxPages; page++ {
		var resp struct {
			HasMore    bool            `json:"has_more"`
			NextCursor string          `json:"next_cursor"`
			Results    []HistoryResult `json:"results"`
		}

		q := url.Values{}
		q.Set("integration_id", integrationID)
		q.Set("limit", fmt.Sprintf("%d", historyPageSize))
		if cursor != "" {
			q.Set("cursor", cursor)
		}

		if err := c.historyRequest(ctx, q, &resp); err != nil {
			return matched, err
		}

		for _, r := range resp.Results {
			if r.BatchID == batchID {
				matched = append(matched, r)
			}
		}

		if wantCount > 0 && len(matched) >= wantCount {
			break
		}
		if !resp.HasMore || resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}

	return matched, nil
}

// historyRequest performs one GET /v1/history call and decodes into out.
func (c *Client) historyRequest(ctx context.Context, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/history?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.batchHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scrub: history: status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("scrub: history: decode: %w", err)
	}
	return nil
}
