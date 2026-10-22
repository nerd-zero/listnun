package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MaxTrackingPixels caps the third-party pixels on one campaign; each one
// is an extra request when an email opens.
const MaxTrackingPixels = 10

// maxPixelURLLen bounds a pixel URL, well above what tracking services use.
const maxPixelURLLen = 2048

var (
	// rePixelURL accepts a plain http(s) URL: no whitespace, quotes or angle
	// brackets, which could break out of the src attribute it goes into.
	rePixelURL = regexp.MustCompile(`^https?://[^\s"'<>]+$`)

	// reHref finds quoted absolute http(s) link targets. Values that hold a
	// template expression ({{ }}) or whitespace aren't plain URLs and are
	// left alone.
	reHref = regexp.MustCompile(`(?i)(\bhref\s*=\s*)(["'])(https?://[^"'\s<>{}]+)(["'])`)

	// ErrTooManyPixels is returned when a campaign has more than
	// MaxTrackingPixels pixels.
	ErrTooManyPixels = fmt.Errorf("more than %d tracking pixels", MaxTrackingPixels)
)

// InvalidPixelError names a tracking pixel URL that was rejected.
type InvalidPixelError struct {
	URL string
}

func (e InvalidPixelError) Error() string {
	return fmt.Sprintf("invalid tracking pixel url %q", e.URL)
}

// CampaignTracking is what a campaign adds to its emails for tracking,
// set with switches in the editor so nobody has to write template tags.
// It applies to HTML emails only.
type CampaignTracking struct {
	// Links tracks every web link in the email, as if each had @TrackLink.
	Links bool `json:"links"`
	// Pixels are third-party tracking image URLs added to the end of the
	// email (analytics or ad platforms that count opens with an image).
	Pixels []string `json:"pixels"`
} // @name CampaignTracking

// Normalized trims the pixel URLs and drops empty and repeated ones.
func (t CampaignTracking) Normalized() CampaignTracking {
	out := CampaignTracking{Links: t.Links, Pixels: []string{}}
	seen := make(map[string]bool, len(t.Pixels))
	for _, p := range t.Pixels {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out.Pixels = append(out.Pixels, p)
	}
	return out
}

// Validate checks the pixel URLs; it returns ErrTooManyPixels or an
// InvalidPixelError.
func (t CampaignTracking) Validate() error {
	if len(t.Pixels) > MaxTrackingPixels {
		return ErrTooManyPixels
	}
	for _, p := range t.Pixels {
		if len(p) > maxPixelURLLen || !rePixelURL.MatchString(p) {
			return InvalidPixelError{URL: p}
		}
	}
	return nil
}

// Apply adds the tracking to an HTML body before it's compiled: every
// untracked web link gets the @TrackLink shorthand (which regTplFuncs then
// turns into a TrackLink call), and the pixels are appended as 1x1 images.
func (t CampaignTracking) Apply(body string) string {
	if t.Links {
		body = reHref.ReplaceAllStringFunc(body, func(m string) string {
			p := reHref.FindStringSubmatch(m)
			// Mismatched quotes (href="...') aren't one attribute value.
			if p[2] != p[4] || strings.HasSuffix(p[3], "@TrackLink") {
				return m
			}
			return p[1] + p[2] + p[3] + "@TrackLink" + p[4]
		})
	}

	if len(t.Pixels) > 0 {
		var b strings.Builder
		b.WriteString(body)
		for _, p := range t.Pixels {
			// Validate guarantees p can't break out of the attribute.
			fmt.Fprintf(&b, `<img src="%s" width="1" height="1" alt="" style="display:block;width:1px;height:1px;border:0;" />`, p)
		}
		body = b.String()
	}
	return body
}

// Value implements driver.Valuer, storing the tracking as JSONB.
func (t CampaignTracking) Value() (driver.Value, error) {
	if t.Pixels == nil {
		t.Pixels = []string{}
	}
	return json.Marshal(t)
}

// Scan implements sql.Scanner for the JSONB column.
func (t *CampaignTracking) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*t = CampaignTracking{Pixels: []string{}}
		return nil
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	}
	return errors.New("campaign tracking: unsupported type")
}
