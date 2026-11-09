package models

import "testing"

func TestMergeHeaders(t *testing.T) {
	tpl := Headers{{"Reply-To": "support@acme.example"}, {"X-Kind": "receipt"}}
	req := Headers{{"X-Kind": "refund"}, {"X-Tag": "a"}, {"X-Tag": "b"}}

	h := MergeHeaders(tpl, req)

	if got := h.Get("Reply-To"); got != "support@acme.example" {
		t.Errorf("template header lost: %q", got)
	}
	if got := h.Values("X-Kind"); len(got) != 1 || got[0] != "refund" {
		t.Errorf("request should replace the template's header, got %v", got)
	}
	if got := h.Values("X-Tag"); len(got) != 2 {
		t.Errorf("repeated request headers should all be kept, got %v", got)
	}
}

func TestTxMessageRecipients(t *testing.T) {
	byEmail := TxMessage{SubscriberEmails: []string{"ana@example.com", "lee@example.net"}}
	got := byEmail.Recipients()
	if len(got) != 2 || got[0] != (TxRecipient{Label: "ana@example.com"}) || got[1].Label != "lee@example.net" {
		t.Errorf("emails: got %+v", got)
	}

	// IDs take over when both are set, matching how the send loop reads them.
	byID := TxMessage{SubscriberIDs: []int{7, 9}, SubscriberEmails: []string{"x@example.com"}}
	got = byID.Recipients()
	if len(got) != 2 || got[0] != (TxRecipient{Label: "#7", ID: 7}) || got[1] != (TxRecipient{Label: "#9", ID: 9}) {
		t.Errorf("ids: got %+v", got)
	}

	if got := (TxMessage{}).Recipients(); len(got) != 0 {
		t.Errorf("no recipients: got %+v", got)
	}
}
