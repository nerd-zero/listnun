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
