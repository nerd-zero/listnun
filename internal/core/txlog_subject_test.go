package core

import "testing"

func TestTxLogSubject(t *testing.T) {
	cases := []struct {
		name, subject, tpl string
		failed             bool
		want               string
	}{
		{"rendered subject is kept", "Welcome Ana", "Welcome {{ .Subscriber.Name }}", false, "Welcome Ana"},
		{"template subject fills in when none was rendered", "", "Your receipt", false, "Your receipt"},
		{"failed before rendering hides raw tags", "", "Welcome {{ .Subscriber.Name }}", true, ""},
		{"failed with a plain subject keeps it", "", "Your receipt", true, "Your receipt"},
		{"failed with a raw override hides it too", "Order {{ .Tx.Data.id }}", "", true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TxLogSubject(c.subject, c.tpl, c.failed); got != c.want {
				t.Errorf("TxLogSubject(%q, %q, %v) = %q, want %q", c.subject, c.tpl, c.failed, got, c.want)
			}
		})
	}
}
