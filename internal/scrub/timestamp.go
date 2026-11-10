package scrub

import (
	"encoding/json"
	"fmt"
	"time"
)

// Timestamp is a time Scrub sends. Scrub writes some timestamps without a
// UTC offset ("2026-09-23T12:04:38.294577"), which time.Time's own JSON
// decoding rejects, failing the whole response. Those are read as UTC;
// RFC 3339 timestamps with an offset decode as usual. It encodes as
// RFC 3339, so the console's own responses always carry an offset.
type Timestamp struct {
	time.Time
}

// scrubLayoutNoOffset is how Scrub writes a timestamp without an offset.
const scrubLayoutNoOffset = "2006-01-02T15:04:05.999999999"

// UnmarshalJSON accepts an RFC 3339 string, one without an offset (read as
// UTC), or null (left zero).
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("scrub timestamp: %w", err)
	}
	if s == "" {
		return nil
	}
	if v, err := time.Parse(time.RFC3339Nano, s); err == nil {
		t.Time = v
		return nil
	}
	v, err := time.ParseInLocation(scrubLayoutNoOffset, s, time.UTC)
	if err != nil {
		return fmt.Errorf("scrub timestamp %q: %w", s, err)
	}
	t.Time = v
	return nil
}

// MarshalJSON writes the time as RFC 3339 (time.Time's own encoding).
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return t.Time.MarshalJSON()
}
