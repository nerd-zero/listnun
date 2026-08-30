// Package scrub is a thin client for Scrub's generic per-email validation
// API (POST /v1/validate/single, POST /v1/validate/bulk) -- distinct from
// the Listmonk-specific list-hygiene-job endpoints (/listmonk/integrations/
// {id}/lists/...) already called ad hoc from cmd/campaigns.go and
// cmd/settings.go. Those trigger an async, list-wide re-scan; this
// package validates only the specific email(s) handed to it, synchronously,
// for use at the moment a subscriber is actually added.
//
// Client instances are built fresh per call from a tenant's stored
// settings (models.Settings.Scrub.URL/APIKey), the same ephemeral-client
// convention checkScrubJobOnCampaign/TestScrubSettings/ScrubList already
// use -- not a boot-wired singleton, since the URL/key are per-tenant.
package scrub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Status values Scrub's /v1/validate/* endpoints return.
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

// maxBulkPerRequest is Scrub's documented per-request cap for
// POST /v1/validate/bulk.
const maxBulkPerRequest = 30000

// Result is the response shape of POST /v1/validate/single.
type Result struct {
	Email  string `json:"email"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// BulkResult is the response shape of POST /v1/validate/bulk.
type BulkResult struct {
	Summary struct {
		Total     int `json:"total"`
		Processed int `json:"processed"`
		Valid     int `json:"valid"`
		Invalid   int `json:"invalid"`
		Errors    int `json:"errors"`
	} `json:"summary"`
	Results []struct {
		Email  string `json:"email"`
		Valid  bool   `json:"valid"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"results"`
}

// Client validates emails against a Scrub instance.
type Client struct {
	baseURL  string
	apiKey   string
	http     *http.Client // short timeout, single-email calls
	bulkHTTP *http.Client // longer timeout, up to 30k emails per call
}

// New builds a Client for one call -- baseURL/apiKey come from a tenant's
// models.Settings.Scrub, read fresh by the caller on every use (settings
// can change at runtime), not cached here.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 5 * time.Second},
		bulkHTTP: &http.Client{Timeout: 30 * time.Second},
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

// ValidateBulk validates a batch of emails synchronously, chunking at
// Scrub's documented 30,000-per-request cap if given more than that.
func (c *Client) ValidateBulk(ctx context.Context, emails []string) (BulkResult, error) {
	var merged BulkResult

	for start := 0; start < len(emails); start += maxBulkPerRequest {
		end := start + maxBulkPerRequest
		if end > len(emails) {
			end = len(emails)
		}

		part, err := c.validateBulkChunk(ctx, emails[start:end])
		if err != nil {
			return merged, err
		}
		merged.Summary.Total += part.Summary.Total
		merged.Summary.Processed += part.Summary.Processed
		merged.Summary.Valid += part.Summary.Valid
		merged.Summary.Invalid += part.Summary.Invalid
		merged.Summary.Errors += part.Summary.Errors
		merged.Results = append(merged.Results, part.Results...)
	}

	return merged, nil
}

func (c *Client) validateBulkChunk(ctx context.Context, emails []string) (BulkResult, error) {
	var out BulkResult

	body, err := json.Marshal(map[string]any{
		"emails":        emails,
		"response_mode": "all",
		"dedupe":        false,
	})
	if err != nil {
		return out, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/validate/bulk", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.bulkHTTP.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return out, fmt.Errorf("scrub: validate/bulk: status %d: %s", resp.StatusCode, string(respBody))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("scrub: validate/bulk: decode: %w", err)
	}
	return out, nil
}
