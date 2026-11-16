package core

import (
	"errors"
	"testing"

	"github.com/lib/pq"
)

// Saving an address another subscriber has in different case hits the
// case-insensitive index, not the plain unique key, and came back as a 500
// instead of "e-mail already exists".
func TestIsEmailTaken(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{&pq.Error{Constraint: "subscribers_email_key"}, true},
		{&pq.Error{Constraint: "idx_subs_email"}, true},
		{&pq.Error{Constraint: "subscriber_lists_pkey"}, false},
		{errors.New("other"), false},
		{nil, false},
	} {
		if got := isEmailTaken(tc.err); got != tc.want {
			t.Errorf("isEmailTaken(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
