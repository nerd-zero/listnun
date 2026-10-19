package core

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

func TestNewTxMessageUUID(t *testing.T) {
	re := regexp.MustCompile(`^msg_[0-9A-HJKMNP-TV-Z]{26}$`)

	a := NewTxMessageUUID()
	if !re.MatchString(a) {
		t.Fatalf("unexpected ID shape %q", a)
	}

	time.Sleep(2 * time.Millisecond)
	b := NewTxMessageUUID()
	if b <= a {
		t.Errorf("IDs should sort by time: %q then %q", a, b)
	}
}

func TestBounceDetail(t *testing.T) {
	cases := map[string]string{
		`{"diagnostic_code": "550 5.1.1 user unknown"}`: "550 5.1.1 user unknown",
		`{"Details": "smtp;550 mailbox full"}`:           "smtp;550 mailbox full",
		`{"other": 1}`:                                   "",
		`not json`:                                       "",
		``:                                               "",
	}
	for in, want := range cases {
		if got := bounceDetail(json.RawMessage(in)); got != want {
			t.Errorf("bounceDetail(%q) = %q, want %q", in, got, want)
		}
	}
}
