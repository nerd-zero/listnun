package segments

import (
	"errors"
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	for _, tc := range []struct {
		name string
		seg  Segment
		want string
	}{
		{"no conditions is everyone", Segment{}, "TRUE"},
		{
			"attribute is",
			Segment{Conditions: []Condition{{Field: FieldAttribute, Key: "city", Op: OpIs, Value: "Harare"}}},
			"((subscribers.attribs->>'city' = 'Harare'))",
		},
		{
			"all joins with AND",
			Segment{Match: MatchAll, Conditions: []Condition{
				{Field: FieldJoined, Op: OpInLast, Value: "90"},
				{Field: FieldList, Op: OpIs, Value: "3"},
			}},
			"((subscribers.created_at >= NOW() - INTERVAL '90 days') AND (EXISTS (SELECT 1 FROM subscriber_lists sl WHERE sl.subscriber_id = subscribers.id AND sl.list_id = 3 AND sl.status != 'unsubscribed')))",
		},
		{
			"any joins with OR",
			Segment{Match: MatchAny, Conditions: []Condition{
				{Field: FieldStatus, Op: OpIs, Value: "blocklisted"},
				{Field: FieldLastOpened, Op: OpNever},
			}},
			"((subscribers.status = 'blocklisted') OR (NOT EXISTS (SELECT 1 FROM campaign_views a WHERE a.subscriber_id = subscribers.id)))",
		},
		{
			"contains matches wildcards literally",
			Segment{Conditions: []Condition{{Field: FieldEmail, Op: OpContains, Value: "50%_off"}}},
			// pq.QuoteLiteral writes backslashes as an E'' escape string.
			`((subscribers.email ILIKE  E'%50\\%\\_off%'))`,
		},
		{
			"joined before a date",
			Segment{Conditions: []Condition{{Field: FieldJoined, Op: OpBefore, Value: "2026-01-31"}}},
			"((subscribers.created_at < '2026-01-31'::DATE))",
		},
	} {
		got, err := Compile(tc.seg)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

// Values and keys are quoted, so they can't end the literal and run SQL.
func TestCompile_QuotesInjection(t *testing.T) {
	got, err := Compile(Segment{Conditions: []Condition{
		{Field: FieldAttribute, Key: "x'); DROP TABLE subscribers; --", Op: OpIs, Value: "a' OR '1'='1"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := `((subscribers.attribs->>'x''); DROP TABLE subscribers; --' = 'a'' OR ''1''=''1'))`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestCompile_RejectsInvalid(t *testing.T) {
	for name, c := range map[string]Condition{
		"unknown field":             {Field: "password", Op: OpIs, Value: "x"},
		"op the field can't use":    {Field: FieldEmail, Op: OpInLast, Value: "3"},
		"attribute without key":     {Field: FieldAttribute, Op: OpIs, Value: "x"},
		"days that aren't a number": {Field: FieldJoined, Op: OpInLast, Value: "90; DROP"},
		"too many days":             {Field: FieldLastClicked, Op: OpInLast, Value: "99999"},
		"list that isn't an ID":     {Field: FieldList, Op: OpIs, Value: "1 OR 1=1"},
		"unknown status":            {Field: FieldStatus, Op: OpIs, Value: "admin"},
		"date that isn't a date":    {Field: FieldJoined, Op: OpAfter, Value: "yesterday"},
	} {
		if _, err := Compile(Segment{Conditions: []Condition{c}}); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}

	if _, err := Compile(Segment{Match: "some", Conditions: []Condition{{Field: FieldLastOpened, Op: OpNever}}}); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown match: err = %v, want ErrInvalid", err)
	}
	many := make([]Condition, maxConditions+1)
	for i := range many {
		many[i] = Condition{Field: FieldLastOpened, Op: OpNever}
	}
	if _, err := Compile(Segment{Conditions: many}); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Errorf("too many conditions: err = %v", err)
	}
}
