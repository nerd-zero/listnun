package core

import (
	"errors"
	"strings"
	"testing"
)

func TestDescribeSendError(t *testing.T) {
	cases := []struct {
		err  string
		want string
	}{
		// What Postmark answered on dev with a stale server token.
		{"535 5.7.8 Error: authentication failed: ", "the SMTP server rejected the login"},
		{"dial tcp 127.0.0.1:1: connect: connection refused", "couldn't reach the SMTP server"},
		{"dial tcp: lookup smtp.nowhere.test: no such host", "couldn't reach the SMTP server"},
		{"x509: certificate signed by unknown authority", "TLS setup didn't match"},
		{"550 5.7.1 Sender signature not confirmed", "refused the message"},
		{"unknown messenger: email", "unknown messenger: email"},
	}
	for _, c := range cases {
		got := DescribeSendError(errors.New(c.err))
		if !strings.Contains(got, c.want) {
			t.Errorf("DescribeSendError(%q) = %q, want it to contain %q", c.err, got, c.want)
		}
	}

	// The raw error is kept, minus the trailing colon SMTP replies end with.
	if got := DescribeSendError(errors.New("535 5.7.8 Error: authentication failed: ")); !strings.HasSuffix(got, "(535 5.7.8 Error: authentication failed)") {
		t.Errorf("raw error not kept: %q", got)
	}
}
