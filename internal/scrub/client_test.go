package scrub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestValidateBulk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/validate/bulk" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Emails       []string `json:"emails"`
			ResponseMode string   `json:"response_mode"`
			Dedupe       bool     `json:"dedupe"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.ResponseMode != "all" {
			t.Errorf("response_mode = %q, want all", body.ResponseMode)
		}
		if body.Dedupe {
			t.Errorf("dedupe = true, want false")
		}

		var res BulkResult
		res.Summary.Total = len(body.Emails)
		res.Summary.Processed = len(body.Emails)
		for _, e := range body.Emails {
			status := StatusDeliverable
			if strings.Contains(e, "risky") {
				status = StatusRisky
			} else if strings.Contains(e, "bad") {
				status = StatusUndeliverable
			}
			res.Results = append(res.Results, struct {
				Email  string `json:"email"`
				Valid  bool   `json:"valid"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			}{Email: e, Valid: status == StatusDeliverable, Status: status})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk_test")
	res, err := c.ValidateBulk(context.Background(), []string{"a@example.com", "risky@example.com", "bad@example.com"})
	if err != nil {
		t.Fatalf("ValidateBulk: %v", err)
	}
	if res.Summary.Total != 3 {
		t.Errorf("Summary.Total = %d, want 3", res.Summary.Total)
	}
	if len(res.Results) != 3 {
		t.Fatalf("len(Results) = %d, want 3", len(res.Results))
	}
	byEmail := map[string]string{}
	for _, r := range res.Results {
		byEmail[r.Email] = r.Status
	}
	if byEmail["risky@example.com"] != StatusRisky {
		t.Errorf("risky@example.com status = %q, want %q", byEmail["risky@example.com"], StatusRisky)
	}
	if byEmail["bad@example.com"] != StatusUndeliverable {
		t.Errorf("bad@example.com status = %q, want %q", byEmail["bad@example.com"], StatusUndeliverable)
	}
}

// TestValidateBulk_Chunking verifies a batch larger than Scrub's documented
// 30,000-per-request cap is split into multiple requests and merged back
// into one result, rather than sent in a single oversized call.
func TestValidateBulk_Chunking(t *testing.T) {
	const total = maxBulkPerRequest + 5

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Emails []string `json:"emails"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if len(body.Emails) > maxBulkPerRequest {
			t.Errorf("chunk size = %d, want <= %d", len(body.Emails), maxBulkPerRequest)
		}

		var res BulkResult
		res.Summary.Total = len(body.Emails)
		for _, e := range body.Emails {
			res.Results = append(res.Results, struct {
				Email  string `json:"email"`
				Valid  bool   `json:"valid"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			}{Email: e, Valid: true, Status: StatusDeliverable})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	emails := make([]string, total)
	for i := range emails {
		emails[i] = "user@example.com"
	}

	c := New(srv.URL, "sk_test")
	res, err := c.ValidateBulk(context.Background(), emails)
	if err != nil {
		t.Fatalf("ValidateBulk: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (one full chunk + one partial)", calls)
	}
	if res.Summary.Total != total {
		t.Errorf("merged Summary.Total = %d, want %d", res.Summary.Total, total)
	}
	if len(res.Results) != total {
		t.Errorf("merged len(Results) = %d, want %d", len(res.Results), total)
	}
}
