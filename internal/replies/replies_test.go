package replies

import (
	"strings"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

func testCodec(t *testing.T) *Codec {
	t.Helper()
	c, err := NewCodec("0123456789abcdef-test-secret")
	if err != nil {
		t.Fatalf("NewCodec: %v", err)
	}
	return c
}

func TestCodecRoundTrip(t *testing.T) {
	c := testCodec(t)
	for _, r := range []Ref{
		{TenantID: 1, CampaignID: 1, SubscriberID: 0},
		{TenantID: 7, CampaignID: 1234, SubscriberID: 98765},
		{TenantID: 4242, CampaignID: 9_999_999, SubscriberID: 2_000_000_000},
	} {
		got, err := c.Decode(c.Encode(r))
		if err != nil || got != r {
			t.Errorf("Decode(Encode(%+v)) = %+v, %v", r, got, err)
		}
	}
}

// Mail servers and clients may change the case of an address's local
// part; the token must survive it.
func TestCodecDecodeIsCaseInsensitive(t *testing.T) {
	c := testCodec(t)
	r := Ref{TenantID: 3, CampaignID: 55, SubscriberID: 101}
	got, err := c.Decode(strings.ToUpper(c.Encode(r)))
	if err != nil || got != r {
		t.Fatalf("Decode(upper) = %+v, %v", got, err)
	}
}

func TestCodecRejectsForgedAndCorruptTokens(t *testing.T) {
	c := testCodec(t)
	valid := c.Encode(Ref{TenantID: 1, CampaignID: 2, SubscriberID: 3})

	other, _ := NewCodec("a-completely-different-secret")
	flipped := []byte(valid)
	if flipped[0] == 'a' {
		flipped[0] = 'b'
	} else {
		flipped[0] = 'a'
	}

	for name, tok := range map[string]string{
		"other secret":  other.Encode(Ref{TenantID: 1, CampaignID: 2, SubscriberID: 3}),
		"flipped byte":  string(flipped),
		"truncated":     valid[:len(valid)-2],
		"empty":         "",
		"not base32":    "!!!not-a-token!!!",
		"too short":     "aaaa",
		"zero tenant":   c.Encode(Ref{TenantID: 0, CampaignID: 2}),
		"zero campaign": c.Encode(Ref{TenantID: 1, CampaignID: 0}),
	} {
		if _, err := c.Decode(tok); err != ErrInvalidToken {
			t.Errorf("%s: Decode = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestNewCodecRejectsShortSecret(t *testing.T) {
	if _, err := NewCodec("short"); err == nil {
		t.Fatal("NewCodec accepted a 5-character secret")
	}
}

// An email local part is capped at 64 characters; Postmark's default
// inbound address already spends 33 of them on "<32-hex-hash>+".
func TestAddressFitsLocalPartLimit(t *testing.T) {
	a, err := NewAddresser(testCodec(t), "0123456789abcdef0123456789abcdef@inbound.postmarkapp.com")
	if err != nil {
		t.Fatalf("NewAddresser: %v", err)
	}
	addr := a.Address(Ref{TenantID: 99999, CampaignID: 99_999_999, SubscriberID: 2_000_000_000})
	local, domain, _ := strings.Cut(addr, "@")
	if len(local) > 64 {
		t.Errorf("local part %q is %d characters, over the 64 limit", local, len(local))
	}
	if domain != "inbound.postmarkapp.com" || !strings.HasPrefix(local, "0123456789abcdef0123456789abcdef+") {
		t.Errorf("address = %q, want <hash>+<token>@inbound.postmarkapp.com", addr)
	}
}

func TestNewAddresserRejectsBadInboundAddress(t *testing.T) {
	for _, in := range []string{"", "no-at-sign", "@domain", "local@", "a+b@domain"} {
		if _, err := NewAddresser(testCodec(t), in); err == nil {
			t.Errorf("NewAddresser(%q) accepted it", in)
		}
	}
}

const samplePostmarkInbound = `{
  "FromName": "Ada Lovelace",
  "From": "ada@example.com",
  "FromFull": {"Email": "ada@example.com", "Name": "Ada Lovelace", "MailboxHash": ""},
  "ToFull": [{"Email": "abc+tok123@inbound.postmarkapp.com", "Name": "", "MailboxHash": "tok123"}],
  "OriginalRecipient": "abc+tok123@inbound.postmarkapp.com",
  "Subject": "Re: October news",
  "MessageID": "73e6d360-66eb-11e1-8e72-a8904824019b",
  "MailboxHash": "tok123",
  "Date": "Fri, 1 Aug 2014 16:45:32 -04:00",
  "TextBody": "Thanks, this was great!\n\nOn Fri ...",
  "HtmlBody": "<p>Thanks, this was great!</p>",
  "StrippedTextReply": "Thanks, this was great!",
  "Headers": [{"Name": "X-Spam-Status", "Value": "No"}]
}`

func TestParsePostmarkInbound(t *testing.T) {
	in, err := ParsePostmarkInbound([]byte(samplePostmarkInbound))
	if err != nil {
		t.Fatalf("ParsePostmarkInbound: %v", err)
	}
	if in.Token() != "tok123" {
		t.Errorf("Token = %q, want tok123", in.Token())
	}
	if email, name := in.Sender(); email != "ada@example.com" || name != "Ada Lovelace" {
		t.Errorf("Sender = %q, %q", email, name)
	}
	if in.StrippedTextReply != "Thanks, this was great!" || in.HTMLBody == "" {
		t.Errorf("bodies not decoded: %+v", in)
	}
	if in.IsAutoReply() {
		t.Error("a person's reply was flagged as an auto-reply")
	}
}

func TestParsePostmarkInboundRequiresMessageID(t *testing.T) {
	if _, err := ParsePostmarkInbound([]byte(`{"Subject":"x"}`)); err == nil {
		t.Fatal("accepted a payload with no MessageID")
	}
}

func TestTokenFallsBackToRecipients(t *testing.T) {
	in := PostmarkInbound{ToFull: []PostmarkAddress{{Email: "other@x.com"}, {Email: "abc+t9@in.example", MailboxHash: "t9"}}}
	if in.Token() != "t9" {
		t.Errorf("Token = %q, want t9", in.Token())
	}
}

func TestIsAutoReply(t *testing.T) {
	cases := map[string]PostmarkInbound{
		"auto-submitted": {Headers: []PostmarkHeader{{Name: "Auto-Submitted", Value: "auto-replied"}}},
		"x-autoreply":    {Headers: []PostmarkHeader{{Name: "X-Autoreply", Value: "yes"}}},
		"precedence":     {Headers: []PostmarkHeader{{Name: "Precedence", Value: "auto_reply"}}},
		"exchange oof":   {Headers: []PostmarkHeader{{Name: "X-Auto-Response-Suppress", Value: "All, OOF"}}},
		"subject":        {Subject: "Automatic reply: October news"},
		"out of office":  {Subject: "Out of Office until Monday"},
	}
	for name, in := range cases {
		if !in.IsAutoReply() {
			t.Errorf("%s: not detected as an auto-reply", name)
		}
	}

	human := PostmarkInbound{
		Subject: "Re: Automatic reply settings question",
		Headers: []PostmarkHeader{{Name: "Auto-Submitted", Value: "no"}},
	}
	if human.IsAutoReply() {
		t.Error("Auto-Submitted: no with a normal Re: subject was flagged as an auto-reply")
	}
}

func TestNewTenantAddresser(t *testing.T) {
	ok := models.TenantReplies{
		InboundAddress:  "58f2c36439f79a2eacc56d3d29e8522d@inbound.postmarkapp.com",
		Secret:          "0123456789abcdef-test-secret",
		WebhookUsername: "postmark",
		WebhookPassword: "pw",
	}
	a, err := NewTenantAddresser(ok)
	if err != nil {
		t.Fatalf("NewTenantAddresser: %v", err)
	}
	if a.InboundLocal() != "58f2c36439f79a2eacc56d3d29e8522d" {
		t.Errorf("InboundLocal = %q", a.InboundLocal())
	}

	for name, mut := range map[string]func(*models.TenantReplies){
		"no webhook password": func(c *models.TenantReplies) { c.WebhookPassword = "" },
		"short secret":        func(c *models.TenantReplies) { c.Secret = "short" },
		"bad address":         func(c *models.TenantReplies) { c.InboundAddress = "nope" },
	} {
		c := ok
		mut(&c)
		if _, err := NewTenantAddresser(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestRecipientLocal(t *testing.T) {
	for in, want := range map[string]string{
		"58F2C364+tok@inbound.postmarkapp.com": "58f2c364",
		"abc@inbound.postmarkapp.com":          "abc",
		"":                                     "",
	} {
		if got := RecipientLocal(in); got != want {
			t.Errorf("RecipientLocal(%q) = %q, want %q", in, got, want)
		}
	}
}
