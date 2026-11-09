// Package webhooks delivers transactional email events to workspace
// endpoints: it signs each request, retries failures with backoff for up
// to a day, and refuses to call private or loopback addresses.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/nerd-zero/listnun/models"
)

const (
	// Timeout is how long an endpoint has to answer.
	Timeout = 10 * time.Second

	// MaxAttempts is how many times a delivery is tried before it fails.
	MaxAttempts = 8

	// maxResponseBytes is how much of an endpoint's reply is kept.
	maxResponseBytes = 2048

	// claimBatch and claimLease bound one worker pass: deliveries claimed
	// at once, and how long they stay locked to this worker.
	claimBatch = 50
	claimLease = 60 * time.Second

	userAgent = "listnun-webhooks/1"
)

// backoff is the wait before each retry: 4 min, 15 min, 1 h, 4 h, then
// every 6 h. Eight attempts span about 23 hours.
var backoff = []time.Duration{
	4 * time.Minute,
	15 * time.Minute,
	time.Hour,
	4 * time.Hour,
	6 * time.Hour,
}

// NextDelay is the wait after the given (1-based) failed attempt, and
// false once there are no attempts left.
func NextDelay(attempt int) (time.Duration, bool) {
	if attempt >= MaxAttempts {
		return 0, false
	}
	if attempt-1 < len(backoff) {
		return backoff[attempt-1], true
	}
	return backoff[len(backoff)-1], true
}

// Sign returns the Listnun-Signature header value for body sent at t:
// t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>">. Receivers
// recompute it with their secret and reject old timestamps.
func Sign(secret string, t time.Time, body []byte) string {
	ts := strconv.FormatInt(t.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

// Store is what the worker needs from the database.
type Store interface {
	ClaimWebhookDeliveries(limit int, lease time.Duration) ([]models.WebhookJob, error)
	RecordWebhookAttempt(job models.WebhookJob, res Result, status string, next *time.Time) error
}

// Result is one attempt's outcome.
type Result struct {
	StatusCode int
	Duration   time.Duration
	Response   string
	Err        error
}

// OK reports whether the endpoint accepted the delivery (2xx).
func (r Result) OK() bool {
	return r.Err == nil && r.StatusCode >= 200 && r.StatusCode < 300
}

// Worker sends due deliveries.
type Worker struct {
	store  Store
	client *http.Client
	log    *log.Logger
	now    func() time.Time
}

// New returns a worker. allowPrivate permits private and loopback
// addresses, for local development only.
func New(store Store, allowPrivate bool, l *log.Logger) *Worker {
	return &Worker{
		store:  store,
		client: NewClient(allowPrivate),
		log:    l,
		now:    time.Now,
	}
}

// Run sends due deliveries every interval until ctx is done.
func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		w.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce claims and sends one batch of due deliveries.
func (w *Worker) RunOnce(ctx context.Context) {
	jobs, err := w.store.ClaimWebhookDeliveries(claimBatch, claimLease)
	if err != nil {
		w.log.Printf("error claiming webhook deliveries: %v", err)
		return
	}
	for _, j := range jobs {
		res := w.Send(ctx, j)
		attempt := j.Attempts + 1

		status, next := models.WebhookSuccess, (*time.Time)(nil)
		if !res.OK() {
			if d, ok := NextDelay(attempt); ok {
				at := w.now().Add(d)
				status, next = models.WebhookRetrying, &at
			} else {
				status = models.WebhookFailed
			}
		}
		if err := w.store.RecordWebhookAttempt(j, res, status, next); err != nil {
			w.log.Printf("error recording webhook attempt %s: %v", j.UUID, err)
		}
	}
}

// Send makes one signed POST of a delivery.
func (w *Worker) Send(ctx context.Context, j models.WebhookJob) Result {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.URL, bytes.NewReader(j.Payload))
	if err != nil {
		return Result{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(models.WebhookEventHeader, j.Event)
	req.Header.Set(models.WebhookDeliveryHeader, j.UUID)
	req.Header.Set(models.WebhookSignatureHeader, Sign(j.Secret, w.now(), j.Payload))

	start := time.Now()
	resp, err := w.client.Do(req)
	dur := time.Since(start)
	if err != nil {
		return Result{Duration: dur, Err: err}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	return Result{StatusCode: resp.StatusCode, Duration: dur, Response: string(body)}
}

// ErrBlockedAddress is returned for endpoints that resolve to private,
// loopback, link-local or otherwise internal addresses.
var ErrBlockedAddress = errors.New("webhook address is not public")

// NewClient returns an HTTP client for webhook calls. Unless allowPrivate
// is set, it refuses to connect to non-public addresses, checked after DNS
// resolution so a public name pointing at an internal IP is caught too. It
// doesn't follow redirects.
func NewClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: Timeout}
	if !allowPrivate {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !PublicIP(ip) {
				return fmt.Errorf("%w: %s", ErrBlockedAddress, host)
			}
			return nil
		}
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   Timeout,
		ResponseHeaderTimeout: Timeout,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// PublicIP reports whether ip is a public unicast address.
func PublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// Carrier-grade NAT (100.64.0.0/10) and IPv4-mapped forms of the above.
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && v4[1]&0xc0 == 64 {
			return false
		}
		if v4[0] == 0 {
			return false
		}
	}
	return true
}
