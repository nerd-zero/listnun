// Package segments turns a segment (a set of conditions such as "City is
// Harare" or "Joined in the last 90 days") into the SQL condition the
// subscriber query runs, so people can find and save groups of
// subscribers without writing SQL.
//
// Fields and operators come from fixed lists and every value is either
// parsed into a number or date or quoted as a SQL literal, so a segment
// can never inject SQL of its own.
package segments

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Match says whether people must meet all of a segment's conditions or
// any one of them.
type Match string

const (
	MatchAll Match = "all"
	MatchAny Match = "any"
)

// Field is what a condition tests.
type Field string

const (
	FieldAttribute   Field = "attribute"
	FieldEmail       Field = "email"
	FieldName        Field = "name"
	FieldJoined      Field = "joined"
	FieldList        Field = "list"
	FieldStatus      Field = "status"
	FieldUnconfirmed Field = "unconfirmed"
	FieldLastOpened  Field = "last_opened"
	FieldLastClicked Field = "last_clicked"
)

// Op is how a condition tests its field.
type Op string

const (
	OpIs        Op = "is"
	OpIsNot     Op = "is_not"
	OpContains  Op = "contains"
	OpExists    Op = "exists"
	OpInLast    Op = "in_last"
	OpNotInLast Op = "not_in_last"
	OpBefore    Op = "before"
	OpAfter     Op = "after"
	OpNever     Op = "never"
)

// maxDays caps "in the last N days" at ten years.
const maxDays = 3650

// maxConditions caps how many conditions a segment can have.
const maxConditions = 20

// Condition is one test in a segment. Key is the attribute name for
// FieldAttribute; Value is the operand, absent for ops that take none.
type Condition struct {
	Field Field  `json:"field"`
	Op    Op     `json:"op"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Segment is a group of people: those who match all (or any) of its
// conditions. No conditions means everyone.
type Segment struct {
	Match      Match       `json:"match"`
	Conditions []Condition `json:"conditions"`
}

// ErrInvalid is wrapped by every error Compile returns.
var ErrInvalid = errors.New("invalid segment")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Compile returns the SQL condition for s, to put in the subscriber
// query's WHERE clause, where the subscribers table is in scope. A
// segment with no conditions compiles to "TRUE".
func Compile(s Segment) (string, error) {
	if len(s.Conditions) == 0 {
		return "TRUE", nil
	}
	if len(s.Conditions) > maxConditions {
		return "", invalid("more than %d conditions", maxConditions)
	}

	join := " AND "
	switch s.Match {
	case MatchAll, "":
	case MatchAny:
		join = " OR "
	default:
		return "", invalid("unknown match %q", s.Match)
	}

	parts := make([]string, 0, len(s.Conditions))
	for i, c := range s.Conditions {
		sql, err := compileCondition(c)
		if err != nil {
			return "", fmt.Errorf("condition %d: %w", i+1, err)
		}
		parts = append(parts, "("+sql+")")
	}
	return "(" + strings.Join(parts, join) + ")", nil
}

func compileCondition(c Condition) (string, error) {
	switch c.Field {
	case FieldAttribute:
		if strings.TrimSpace(c.Key) == "" {
			return "", invalid("an attribute condition needs a key")
		}
		return text("subscribers.attribs->>"+pq.QuoteLiteral(c.Key), c)
	case FieldEmail:
		return text("subscribers.email", c)
	case FieldName:
		return text("subscribers.name", c)
	case FieldJoined:
		return when("subscribers.created_at", c)
	case FieldList:
		id, err := strconv.Atoi(c.Value)
		if err != nil || id < 1 {
			return "", invalid("list %q isn't a list ID", c.Value)
		}
		on := fmt.Sprintf("EXISTS (SELECT 1 FROM subscriber_lists sl WHERE sl.subscriber_id = subscribers.id AND sl.list_id = %d AND sl.status != 'unsubscribed')", id)
		switch c.Op {
		case OpIs:
			return on, nil
		case OpIsNot:
			return "NOT " + on, nil
		}
	case FieldStatus:
		if c.Value != "enabled" && c.Value != "blocklisted" {
			return "", invalid("status %q isn't enabled or blocklisted", c.Value)
		}
		switch c.Op {
		case OpIs:
			return "subscribers.status = " + pq.QuoteLiteral(c.Value), nil
		case OpIsNot:
			return "subscribers.status != " + pq.QuoteLiteral(c.Value), nil
		}
	case FieldUnconfirmed:
		if c.Op == OpIs {
			return "EXISTS (SELECT 1 FROM subscriber_lists sl JOIN lists l ON l.id = sl.list_id AND l.optin = 'double' " +
				"WHERE sl.subscriber_id = subscribers.id AND sl.status = 'unconfirmed')", nil
		}
	case FieldLastOpened:
		return activity("campaign_views", c)
	case FieldLastClicked:
		return activity("link_clicks", c)
	default:
		return "", invalid("unknown field %q", c.Field)
	}
	return "", invalid("%s can't use %q", c.Field, c.Op)
}

// text compiles a test on a text column.
func text(col string, c Condition) (string, error) {
	switch c.Op {
	case OpIs:
		return col + " = " + pq.QuoteLiteral(c.Value), nil
	case OpIsNot:
		return "COALESCE(" + col + ", '') != " + pq.QuoteLiteral(c.Value), nil
	case OpContains:
		return col + " ILIKE " + pq.QuoteLiteral("%"+escapeLike(c.Value)+"%"), nil
	case OpExists:
		return "COALESCE(" + col + ", '') != ''", nil
	}
	return "", invalid("%s can't use %q", c.Field, c.Op)
}

// when compiles a test on a timestamp column.
func when(col string, c Condition) (string, error) {
	switch c.Op {
	case OpInLast, OpNotInLast:
		days, err := parseDays(c.Value)
		if err != nil {
			return "", err
		}
		cmp := ">="
		if c.Op == OpNotInLast {
			cmp = "<"
		}
		return fmt.Sprintf("%s %s NOW() - INTERVAL '%d days'", col, cmp, days), nil
	case OpBefore, OpAfter:
		d, err := time.Parse(time.DateOnly, c.Value)
		if err != nil {
			return "", invalid("%q isn't a date (YYYY-MM-DD)", c.Value)
		}
		cmp := "<"
		if c.Op == OpAfter {
			cmp = ">="
		}
		return fmt.Sprintf("%s %s '%s'::DATE", col, cmp, d.Format(time.DateOnly)), nil
	}
	return "", invalid("%s can't use %q", c.Field, c.Op)
}

// activity compiles a test on when someone last opened or clicked, from
// table (campaign_views or link_clicks).
func activity(table string, c Condition) (string, error) {
	exists := "EXISTS (SELECT 1 FROM " + table + " a WHERE a.subscriber_id = subscribers.id"
	switch c.Op {
	case OpNever:
		return "NOT " + exists + ")", nil
	case OpInLast, OpNotInLast:
		days, err := parseDays(c.Value)
		if err != nil {
			return "", err
		}
		recent := fmt.Sprintf("%s AND a.created_at >= NOW() - INTERVAL '%d days')", exists, days)
		if c.Op == OpInLast {
			return recent, nil
		}
		return "NOT " + recent, nil
	}
	return "", invalid("%s can't use %q", c.Field, c.Op)
}

func parseDays(v string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 1 || n > maxDays {
		return 0, invalid("%q isn't a number of days from 1 to %d", v, maxDays)
	}
	return n, nil
}

// escapeLike escapes ILIKE's wildcards so a value matches literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
