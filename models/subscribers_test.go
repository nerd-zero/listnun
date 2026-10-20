package models

import (
	"reflect"
	"testing"

	null "gopkg.in/volatiletech/null.v6"
)

func TestSplitUnsubscribed(t *testing.T) {
	sub := func(id int, name, typ, status string) Subscription {
		s := Subscription{SubscriptionStatus: null.StringFrom(status)}
		s.ID, s.Name, s.Type = id, name, typ
		return s
	}
	subs := []Subscription{
		sub(1, "Weekly digest", ListTypePublic, SubscriptionStatusUnsubscribed),
		sub(2, "Product updates", ListTypePublic, SubscriptionStatusConfirmed),
		sub(3, "Internal", ListTypePrivate, SubscriptionStatusUnsubscribed),
		sub(4, "Events", ListTypePublic, SubscriptionStatusUnsubscribed),
	}

	left, still := SplitUnsubscribed(subs, []int{1, 3})

	// The private list is never named; list 4 was left earlier, not now.
	if want := []string{"Weekly digest"}; !reflect.DeepEqual(left, want) {
		t.Errorf("left = %v, want %v", left, want)
	}
	if want := []string{"Product updates"}; !reflect.DeepEqual(still, want) {
		t.Errorf("still = %v, want %v", still, want)
	}
}
