package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nerd-zero/listnun/models"
)

func TestSign(t *testing.T) {
	at := time.Unix(1792430895, 0)
	body := []byte(`{"type":"message.failed"}`)
	got := Sign("whsec_test", at, body)

	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte("1792430895." + string(body)))
	want := "t=1792430895,v1=" + hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("Sign = %q, want %q", got, want)
	}
}

func TestNextDelay(t *testing.T) {
	want := []time.Duration{4 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour, 6 * time.Hour, 6 * time.Hour, 6 * time.Hour}
	var total time.Duration
	for i, w := range want {
		d, ok := NextDelay(i + 1)
		if !ok || d != w {
			t.Errorf("after attempt %d: got %v %v, want %v", i+1, d, ok, w)
		}
		total += d
	}
	if _, ok := NextDelay(MaxAttempts); ok {
		t.Error("there should be no retry after the last attempt")
	}
	if total > 24*time.Hour {
		t.Errorf("retries span %v, more than a day", total)
	}
}

func TestPublicIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "192.168.0.1", "172.16.5.4", "169.254.169.254", "100.64.0.1", "0.0.0.0", "::1", "fe80::1", "fd00::1"} {
		if PublicIP(net.ParseIP(s)) {
			t.Errorf("%s should not be public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if !PublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
}

func TestClientRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	_, err := NewClient(false).Get(srv.URL)
	if err == nil || !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("expected the loopback test server to be refused, got %v", err)
	}
}

// fakeStore is an in-memory Store.
type fakeStore struct {
	jobs     []models.WebhookJob
	recorded []recorded
}

type recorded struct {
	res    Result
	status string
	next   *time.Time
}

func (f *fakeStore) ClaimWebhookDeliveries(int, time.Duration) ([]models.WebhookJob, error) {
	j := f.jobs
	f.jobs = nil
	return j, nil
}

func (f *fakeStore) RecordWebhookAttempt(_ models.WebhookJob, res Result, status string, next *time.Time) error {
	f.recorded = append(f.recorded, recorded{res, status, next})
	return nil
}

func TestWorkerSendsSignedRequestsAndSchedulesRetries(t *testing.T) {
	var gotSig, gotEvent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotSig = r.Header.Get(models.WebhookSignatureHeader)
		gotEvent = r.Header.Get(models.WebhookEventHeader)
		if strings.Contains(string(body), "fail") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"db connection refused"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	now := time.Unix(1792430895, 0)
	store := &fakeStore{jobs: []models.WebhookJob{
		{UUID: "whd_1", Event: "message.delivered", URL: srv.URL, Secret: "s", Payload: []byte(`{"ok":true}`)},
		{UUID: "whd_2", Event: "message.failed", URL: srv.URL, Secret: "s", Payload: []byte(`{"fail":true}`), Attempts: 1},
		{UUID: "whd_3", Event: "message.failed", URL: srv.URL, Secret: "s", Payload: []byte(`{"fail":true}`), Attempts: MaxAttempts - 1},
	}}
	w := New(store, true, log.New(io.Discard, "", 0))
	w.now = func() time.Time { return now }

	w.RunOnce(context.Background())

	if len(store.recorded) != 3 {
		t.Fatalf("recorded %d attempts, want 3", len(store.recorded))
	}
	if r := store.recorded[0]; r.status != models.WebhookSuccess || r.next != nil {
		t.Errorf("2xx should succeed: %+v", r)
	}
	if r := store.recorded[1]; r.status != models.WebhookRetrying || r.next == nil || !r.next.Equal(now.Add(15*time.Minute)) {
		t.Errorf("second failure should retry in 15 min: %+v", r)
	}
	if r := store.recorded[1]; r.res.StatusCode != 500 || !strings.Contains(r.res.Response, "db connection refused") {
		t.Errorf("the endpoint's reply should be kept: %+v", r.res)
	}
	if r := store.recorded[2]; r.status != models.WebhookFailed || r.next != nil {
		t.Errorf("the last attempt should fail for good: %+v", r)
	}
	if !strings.HasPrefix(gotSig, "t=1792430895,v1=") || gotEvent != "message.failed" {
		t.Errorf("unexpected headers: signature %q, event %q", gotSig, gotEvent)
	}
}
