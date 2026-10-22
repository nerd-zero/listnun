package models

import "testing"

func TestDashboardTotals(t *testing.T) {
	o := DashboardOverview{Days: []DashboardDay{
		{Delivered: 900, Bounced: 6, NewSubscribers: 10},
		{Delivered: 100, Bounced: 4, NewSubscribers: 2},
	}}
	o.Totals()
	if o.Sent != 1000 || o.Bounced != 10 || o.New != 12 || o.BounceRate != 0.01 {
		t.Errorf("unexpected totals: %+v", o)
	}

	empty := DashboardOverview{}
	empty.Totals()
	if empty.BounceRate != 0 {
		t.Error("no sends should mean a zero bounce rate, not NaN")
	}
}
