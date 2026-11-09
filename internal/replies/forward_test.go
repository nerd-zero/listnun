package replies

import (
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nerd-zero/listnun/models"
)

func TestForwardAddress(t *testing.T) {
	cases := map[string]struct {
		camp models.ReplyCampaign
		want string
	}{
		"campaign Reply-To wins": {
			models.ReplyCampaign{FromEmail: "News <news@fuser.tech>", Headers: models.Headers{{"reply-to": "support@fuser.tech"}}},
			"support@fuser.tech",
		},
		"no Reply-To falls back to From": {
			models.ReplyCampaign{FromEmail: "News <news@fuser.tech>", Headers: models.Headers{{"X-Other": "x"}}},
			"News <news@fuser.tech>",
		},
		"templated Reply-To falls back to From": {
			models.ReplyCampaign{FromEmail: "news@fuser.tech", Headers: models.Headers{{"Reply-To": "{{ .Subscriber.Attribs.rep }}"}}},
			"news@fuser.tech",
		},
	}
	for name, c := range cases {
		if got := ForwardAddress(c.camp); got != c.want {
			t.Errorf("%s: ForwardAddress = %q, want %q", name, got, c.want)
		}
	}
}

func TestForwardMessage(t *testing.T) {
	camp := models.ReplyCampaign{ID: 5, TenantID: 3, Name: "October news", Subject: "October news", FromEmail: "news@fuser.tech", Messenger: "email"}
	in := PostmarkInbound{
		FromFull:  PostmarkAddress{Email: "ada@example.com", Name: "Ada"},
		Subject:   "Re: October news",
		TextBody:  "Thanks <3",
		MessageID: "m1",
	}

	msg := ForwardMessage(camp, in, "support@fuser.tech")

	if msg.TenantID != 3 || msg.From != "news@fuser.tech" || len(msg.To) != 1 || msg.To[0] != "support@fuser.tech" {
		t.Errorf("addressing = tenant %d from %q to %v", msg.TenantID, msg.From, msg.To)
	}
	if msg.Headers.Get("Reply-To") != "ada@example.com" || msg.Headers.Get(HeaderForwarded) == "" {
		t.Errorf("headers = %v, want Reply-To the replier and the forward marker", msg.Headers)
	}
	if msg.Subject != "Re: October news" || msg.Messenger != "email" {
		t.Errorf("subject %q messenger %q", msg.Subject, msg.Messenger)
	}
	// A plain-text reply is HTML-escaped into the forward.
	if body := string(msg.Body); !strings.Contains(body, "Thanks &lt;3") || !strings.Contains(body, "Ada &lt;ada@example.com&gt;") {
		t.Errorf("body not escaped: %s", body)
	}

	// A postback messenger can't deliver email: the forward uses email.
	camp.Messenger = "postback"
	if m := ForwardMessage(camp, in, "x@y.z"); m.Messenger != "email" {
		t.Errorf("messenger = %q for a postback campaign, want email", m.Messenger)
	}
	// No subject on the reply: fall back to "Re: <campaign subject>".
	in.Subject = ""
	if m := ForwardMessage(camp, in, "x@y.z"); m.Subject != "Re: October news" {
		t.Errorf("subject = %q, want Re: October news", m.Subject)
	}
}

func TestCheckBasicAuth(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhooks/replies/postmark", nil)
	if CheckBasicAuth(req, "postmark", "s3cret") {
		t.Error("no credentials was authorized")
	}
	req.SetBasicAuth("postmark", "wrong")
	if CheckBasicAuth(req, "postmark", "s3cret") {
		t.Error("wrong password was authorized")
	}
	req.SetBasicAuth("postmark", "s3cret")
	if !CheckBasicAuth(req, "postmark", "s3cret") {
		t.Error("right credentials were rejected")
	}
	req.SetBasicAuth("", "")
	if CheckBasicAuth(req, "", "") {
		t.Error("empty configured credentials authorized an empty login")
	}
}

func TestTruncateUTF8KeepsRunesWhole(t *testing.T) {
	got := TruncateUTF8(strings.Repeat("é", 10), 5) // 2 bytes each
	if !utf8.ValidString(got) || len(got) != 4 {
		t.Errorf("TruncateUTF8 = %q (%d bytes), want 2 whole runes", got, len(got))
	}
	if TruncateUTF8("short", 100) != "short" {
		t.Error("a short string was changed")
	}
}
