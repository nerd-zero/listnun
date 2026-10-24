package models

import (
	"encoding/json"
	"fmt"

	"github.com/nerd-zero/listnun/internal/segments"
	null "gopkg.in/volatiletech/null.v6"
)

// SegmentConditions is a saved segment's conditions, stored as JSON.
type SegmentConditions []segments.Condition

// Scan implements the sql.Scanner interface.
func (s *SegmentConditions) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = SegmentConditions{}
		return nil
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	}
	return fmt.Errorf("could not decode type %T -> %T", src, s)
}

// SavedSegment is a named segment a workspace saved.
type SavedSegment struct {
	ID            int               `db:"id" json:"id"`
	UUID          string            `db:"uuid" json:"uuid"`
	Name          string            `db:"name" json:"name"`
	Match         segments.Match    `db:"match" json:"match" swaggertype:"string"`
	Conditions    SegmentConditions `db:"conditions" json:"conditions"`
	CreatedByName string            `db:"created_by_name" json:"created_by_name"`
	CreatedAt     null.Time         `db:"created_at" json:"created_at" swaggertype:"string"`
	UpdatedAt     null.Time         `db:"updated_at" json:"updated_at" swaggertype:"string"`
} // @name SavedSegment

// Segment returns the segment to compile.
func (s SavedSegment) Segment() segments.Segment {
	return segments.Segment{Match: s.Match, Conditions: s.Conditions}
}
