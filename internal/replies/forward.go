package replies

import (
	"crypto/subtle"
	"fmt"
	"html"
	"net/http"
	"net/textproto"
	"strings"
	"unicode/utf8"

	"github.com/nerd-zero/listnun/models"
)

// HeaderForwarded marks a reply listnun forwarded. A forward that comes
// back in (a campaign whose own Reply-To is another tracked address) is
// dropped instead of looping.
const HeaderForwarded = "X-Listnun-Reply-Forward"

// emailMessenger is the messenger a forward goes out through when the
// campaign used a non-email (postback) messenger -- matches cmd's
// emailMsgr.
const emailMessenger = "email"

// ForwardAddress is where a campaign's replies are forwarded: the
// campaign's own Reply-To header, or its From address when it has none.
// A templated Reply-To ({{ }}, different per subscriber) can't be
// resolved without the subscriber's original message, so it falls back
// to From too.
func ForwardAddress(camp models.ReplyCampaign) string {
	for _, set := range camp.Headers {
		for name, val := range set {
			if !strings.EqualFold(name, "Reply-To") {
				continue
			}
			if val = strings.TrimSpace(val); val != "" && !strings.Contains(val, "{{") {
				return val
			}
		}
	}
	return strings.TrimSpace(camp.FromEmail)
}

// ForwardMessage builds the forward of reply in to address to: sent from
// the campaign's From address through the campaign's messenger (the
// tenant's own mail server), with the replier as Reply-To so answering it
// goes straight back to them.
func ForwardMessage(camp models.ReplyCampaign, in PostmarkInbound, to string) models.Message {
	fromEmail, fromName := in.Sender()
	who := fromEmail
	if fromName != "" {
		who = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}
	note := fmt.Sprintf("Reply from %s to your campaign %q.", who, camp.Name)
	noteHTML := `<p style="font:13px sans-serif;color:#5c554b;margin:0 0 16px">` + html.EscapeString(note) + `</p>`

	body := noteHTML + in.HTMLBody
	if strings.TrimSpace(in.HTMLBody) == "" {
		body = noteHTML + `<pre style="white-space:pre-wrap;font:14px sans-serif">` + html.EscapeString(in.TextBody) + `</pre>`
	}

	h := textproto.MIMEHeader{}
	h.Set("Reply-To", fromEmail)
	h.Set(HeaderForwarded, "1")

	subject := in.Subject
	if strings.TrimSpace(subject) == "" {
		subject = "Re: " + camp.Subject
	}

	messenger := camp.Messenger
	if !strings.HasPrefix(messenger, emailMessenger) {
		// A postback (HTTP) messenger can't deliver an email reply.
		messenger = emailMessenger
	}

	return models.Message{
		TenantID:    camp.TenantID,
		From:        camp.FromEmail,
		To:          []string{to},
		Subject:     subject,
		ContentType: models.CampaignContentTypeHTML,
		Body:        []byte(body),
		AltBody:     []byte(note + "\n\n" + in.TextBody),
		Headers:     h,
		Messenger:   messenger,
	}
}

// CheckBasicAuth reports whether r carries exactly user/pass as HTTP basic
// auth, compared in constant time.
func CheckBasicAuth(r *http.Request, user, pass string) bool {
	gotUser, gotPass, ok := r.BasicAuth()
	if !ok || user == "" || pass == "" {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(gotUser), []byte(user)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(gotPass), []byte(pass)) == 1
	return userOK && passOK
}

// TruncateUTF8 cuts s to at most max bytes without splitting a rune.
func TruncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
