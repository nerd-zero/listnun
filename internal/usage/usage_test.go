package usage

import (
	"errors"
	"io"
	"log"
	"testing"
	"time"
)

type fakeStore struct {
	rows []Row
	err  error
}

func (f *fakeStore) AddUsage(rows []Row) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, rows...)
	return nil
}

func newTestMeter(store Store, now *time.Time) *Meter {
	m := New(store, log.New(io.Discard, "", 0))
	m.now = func() time.Time { return *now }
	return m
}

func sum(rows []Row, tenantID int, kind string, bucket time.Time) int64 {
	var n int64
	for _, r := range rows {
		if r.TenantID == tenantID && r.Kind == kind && r.Bucket.Equal(bucket) {
			n += r.Count
		}
	}
	return n
}

func TestMeter_AggregatesByTenantKindAndUTCHour(t *testing.T) {
	store := &fakeStore{}
	now := time.Date(2026, 10, 31, 23, 59, 0, 0, time.FixedZone("CAT", 2*60*60))
	m := newTestMeter(store, &now)

	m.Add(1, KindEmailSent, 1)
	m.Add(1, KindEmailSent, 2)
	m.Add(1, KindValidation, 500)
	m.Add(2, KindEmailSent, 1)
	now = now.Add(2 * time.Minute) // next hour
	m.Add(1, KindEmailSent, 4)

	if err := m.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	first := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	for _, tc := range []struct {
		tenant int
		kind   string
		bucket time.Time
		want   int64
	}{
		{1, KindEmailSent, first, 3},
		{1, KindValidation, first, 500},
		{2, KindEmailSent, first, 1},
		{1, KindEmailSent, second, 4},
	} {
		if got := sum(store.rows, tc.tenant, tc.kind, tc.bucket); got != tc.want {
			t.Errorf("tenant %d %s @ %s = %d, want %d", tc.tenant, tc.kind, tc.bucket, got, tc.want)
		}
	}
	if len(store.rows) != 4 {
		t.Errorf("wrote %d rows, want 4 (one per counter)", len(store.rows))
	}
}

func TestMeter_IgnoresUnscopedAndEmptyIncrements(t *testing.T) {
	store := &fakeStore{}
	now := time.Now()
	m := newTestMeter(store, &now)

	m.Add(0, KindEmailSent, 1)
	m.Add(1, KindEmailSent, 0)
	if err := m.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if len(store.rows) != 0 {
		t.Fatalf("wrote %+v, want nothing", store.rows)
	}

	var nilMeter *Meter
	nilMeter.Add(1, KindEmailSent, 1) // must not panic
}

func TestMeter_FailedFlushKeepsCountsForRetry(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	now := time.Date(2026, 10, 1, 10, 30, 0, 0, time.UTC)
	m := newTestMeter(store, &now)

	m.Add(1, KindEmailSent, 5)
	if err := m.Flush(); err == nil {
		t.Fatal("Flush succeeded against a failing store")
	}
	m.Add(1, KindEmailSent, 2)

	store.err = nil
	if err := m.Flush(); err != nil {
		t.Fatalf("retry Flush: %v", err)
	}
	if got := sum(store.rows, 1, KindEmailSent, now.Truncate(time.Hour)); got != 7 {
		t.Fatalf("count after retry = %d, want 7 (nothing lost, nothing doubled)", got)
	}
}

func TestMeter_CloseFlushesRemainder(t *testing.T) {
	store := &fakeStore{}
	now := time.Now()
	m := newTestMeter(store, &now)
	go m.Run(time.Hour)

	m.Add(3, KindValidation, 9)
	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(store.rows) != 1 || store.rows[0].Count != 9 {
		t.Fatalf("rows after Close = %+v, want the buffered 9", store.rows)
	}
}
