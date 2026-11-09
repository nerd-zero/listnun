package models

import "testing"

func TestListsByUUID(t *testing.T) {
	lists := []List{{UUID: "a", Name: "A"}, {UUID: "b", Name: "B"}}

	names := func(ls []List) string {
		s := ""
		for _, l := range ls {
			s += l.Name
		}
		return s
	}

	for _, tc := range []struct {
		name  string
		uuids []string
		want  string
	}{
		{"no filter shows every list", nil, "AB"},
		{"one list", []string{"b"}, "B"},
		{"a stale link falls back to every list", []string{"gone"}, "AB"},
		{"unknown UUIDs are ignored", []string{"gone", "a"}, "A"},
	} {
		if got := names(ListsByUUID(lists, tc.uuids)); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
