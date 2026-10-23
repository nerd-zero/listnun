package replies

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PostmarkInbound is the subset of Postmark's inbound webhook payload
// (https://postmarkapp.com/developer/webhooks/inbound-webhook) a reply
// needs.
type PostmarkInbound struct {
	From              string            `json:"From"`
	FromName          string            `json:"FromName"`
	FromFull          PostmarkAddress   `json:"FromFull"`
	ToFull            []PostmarkAddress `json:"ToFull"`
	OriginalRecipient string            `json:"OriginalRecipient"`
	Subject           string            `json:"Subject"`
	MessageID         string            `json:"MessageID"`
	ReplyTo           string            `json:"ReplyTo"`
	// MailboxHash is the plus-addressed part of the recipient -- the
	// reply token.
	MailboxHash       string           `json:"MailboxHash"`
	Date              string           `json:"Date"`
	TextBody          string           `json:"TextBody"`
	HTMLBody          string           `json:"HtmlBody"`
	StrippedTextReply string           `json:"StrippedTextReply"`
	Headers           []PostmarkHeader `json:"Headers"`
}

// PostmarkAddress is one parsed address in an inbound payload.
type PostmarkAddress struct {
	Email       string `json:"Email"`
	Name        string `json:"Name"`
	MailboxHash string `json:"MailboxHash"`
}

// PostmarkHeader is one raw header of the inbound message.
type PostmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// ParsePostmarkInbound decodes a Postmark inbound webhook body.
func ParsePostmarkInbound(body []byte) (PostmarkInbound, error) {
	var in PostmarkInbound
	if err := json.Unmarshal(body, &in); err != nil {
		return in, fmt.Errorf("decode postmark inbound payload: %w", err)
	}
	if in.MessageID == "" {
		return in, fmt.Errorf("postmark inbound payload has no MessageID")
	}
	return in, nil
}

// Token returns the reply token the message was addressed to: Postmark's
// top-level MailboxHash, or failing that the first recipient that has one
// (a reply that also Cc'd other addresses).
func (in PostmarkInbound) Token() string {
	if in.MailboxHash != "" {
		return in.MailboxHash
	}
	for _, to := range in.ToFull {
		if to.MailboxHash != "" {
			return to.MailboxHash
		}
	}
	return ""
}

// Sender returns the replier's address and name.
func (in PostmarkInbound) Sender() (email, name string) {
	email, name = in.FromFull.Email, in.FromFull.Name
	if email == "" {
		email = in.From
	}
	if name == "" {
		name = in.FromName
	}
	return strings.TrimSpace(email), strings.TrimSpace(name)
}

// Header returns the first value of header name (case-insensitive).
func (in PostmarkInbound) Header(name string) string {
	for _, h := range in.Headers {
		if strings.EqualFold(h.Name, name) {
			return strings.TrimSpace(h.Value)
		}
	}
	return ""
}

// autoReplySubjects are subject prefixes common auto-responders use when
// they don't set any of the standard headers IsAutoReply checks first.
var autoReplySubjects = []string{
	"auto:",
	"automatic reply",
	"autoreply",
	"auto-reply",
	"auto reply",
	"out of office",
	"out of the office",
	"abwesenheitsnotiz",
	"réponse automatique",
	"respuesta automática",
}

// IsAutoReply reports whether the message is an automatic response
// (out-of-office, vacation, auto-acknowledgement) rather than a person
// replying. Such replies are recorded but not counted.
func (in PostmarkInbound) IsAutoReply() bool {
	// RFC 3834: anything but "no" means machine-generated.
	if v := strings.ToLower(in.Header("Auto-Submitted")); v != "" && v != "no" {
		return true
	}
	if in.Header("X-Autoreply") != "" || in.Header("X-Autorespond") != "" || in.Header("X-Autoresponse") != "" {
		return true
	}
	switch strings.ToLower(in.Header("Precedence")) {
	case "auto_reply", "bulk", "junk":
		return true
	}
	// Exchange/Outlook out-of-office replies.
	if strings.Contains(strings.ToLower(in.Header("X-Auto-Response-Suppress")), "oof") {
		return true
	}
	subject := strings.ToLower(strings.TrimSpace(in.Subject))
	for _, p := range autoReplySubjects {
		if strings.HasPrefix(subject, p) {
			return true
		}
	}
	return false
}

// Recipient returns the address the reply was delivered to: Postmark's
// OriginalRecipient, or failing that the first recipient carrying the
// reply token.
func (in PostmarkInbound) Recipient() string {
	if in.OriginalRecipient != "" {
		return in.OriginalRecipient
	}
	for _, to := range in.ToFull {
		if to.MailboxHash != "" {
			return to.Email
		}
	}
	return ""
}
