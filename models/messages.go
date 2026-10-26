package models

import (
	"bytes"
	"fmt"
	"html/template"
	"net/textproto"
	txttpl "text/template"
)

// Message is the message pushed to a Messenger.
type Message struct {
	// TenantID is set explicitly by whoever constructs the message (from
	// the campaign for campaign sends, from the authenticated request's
	// resolved tenant for transactional sends) rather than derived from
	// Campaign, which is nil for the transactional path.
	TenantID int

	From        string
	To          []string
	Subject     string
	ContentType string
	Body        []byte
	AltBody     []byte
	Headers     textproto.MIMEHeader
	Attachments []Attachment

	Subscriber Subscriber

	// Campaign is generally the same instance for a large number of subscribers.
	Campaign *Campaign

	// Messenger is the messenger backend to use: email|postback.
	Messenger string

	// TxMessageID is the message log row of a transactional message, so
	// the worker can record what happened to it. 0 for campaign messages.
	TxMessageID int64
}

// Attachment represents a file or blob attachment that can be
// sent along with a message by a Messenger.
type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte

	// IsInline marks the attachment as an inline (CID) part
	// to be wrapped in multipart/related.
	IsInline bool
}

// TxMessage subscriber modes.
const (
	TxSubModeDefault  = "default"
	TxSubModeFallback = "fallback"
	TxSubModeExternal = "external"
)

// TxMessage represents an e-mail campaign.
type TxMessage struct {
	SubscriberMode   string   `json:"subscriber_mode"`
	SubscriberEmails []string `json:"subscriber_emails"`
	SubscriberIDs    []int    `json:"subscriber_ids"`

	// Deprecated.
	SubscriberEmail string `json:"subscriber_email"`
	SubscriberID    int    `json:"subscriber_id"`

	TemplateID  int            `json:"template_id"`
	Data        map[string]any `json:"data"`
	FromEmail   string         `json:"from_email"`
	Headers     Headers        `json:"headers"`
	ContentType string         `json:"content_type"`
	Messenger   string         `json:"messenger"`
	Subject     string         `json:"subject"`
	AltBody     string         `json:"altbody"`

	// File attachments added from multi-part form data.
	Attachments []Attachment `json:"-"`

	Body       []byte             `json:"-"`
	Tpl        *template.Template `json:"-"`
	SubjectTpl *txttpl.Template   `json:"-"`

	// Brand is the workspace logo for the brand-logo block; nil renders
	// nothing.
	Brand *BrandBlock `json:"-"`
}

func (m *TxMessage) Render(sub Subscriber, tpl *Template, funcs txttpl.FuncMap) error {
	data := struct {
		Subscriber Subscriber
		Tx         *TxMessage
		Brand      *BrandBlock
	}{sub, m, m.Brand}

	// Render the body.
	b := bytes.Buffer{}
	if err := tpl.Tpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
		return err
	}
	m.Body = make([]byte, b.Len())
	copy(m.Body, b.Bytes())
	b.Reset()

	// Render alt body if it has any templating strings.
	if m.AltBody != "" && hasTplExpr(m.AltBody) {
		t, err := txttpl.New(BaseTpl).Funcs(funcs).Parse(m.AltBody)
		if err != nil {
			return fmt.Errorf("error compiling alt body: %v", err)
		}
		if err := t.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			return err
		}
		m.AltBody = b.String()
		b.Reset()
	}

	// Was a subject provided in the message?
	var (
		subjTpl *txttpl.Template
		subject = m.Subject
	)
	if subject != "" {
		if hasTplExpr(m.Subject) {
			// If the subject has a template string, render that.
			s, err := txttpl.New(BaseTpl).Funcs(funcs).Parse(m.Subject)
			if err != nil {
				return fmt.Errorf("error compiling subject: %v", err)
			}
			subjTpl = s
		}
	} else {
		// Use the subject from the template.
		subject = tpl.Subject
		subjTpl = tpl.SubjectTpl
	}

	// If the subject is also a template, render that.
	if subjTpl != nil {
		if err := subjTpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			return err
		}
		m.Subject = b.String()
		b.Reset()
	} else {
		m.Subject = subject
	}

	return nil
}

// MergeHeaders merges a template's default headers with a request's. A header
// the request sets replaces the template's value; repeated request headers
// are all kept.
func MergeHeaders(tplHeaders, reqHeaders Headers) textproto.MIMEHeader {
	out := make(textproto.MIMEHeader, len(tplHeaders)+len(reqHeaders)+1)
	for _, set := range tplHeaders {
		for hdr, val := range set {
			out.Set(hdr, val)
		}
	}

	replaced := map[string]bool{}
	for _, set := range reqHeaders {
		for hdr, val := range set {
			key := textproto.CanonicalMIMEHeaderKey(hdr)
			if !replaced[key] {
				out.Del(key)
				replaced[key] = true
			}
			out.Add(key, val)
		}
	}
	return out
}

// TxRecipient is one addressee of a tx request as the message log names it:
// the email, or "#<id>" with the ID when the request addressed subscriber IDs.
type TxRecipient struct {
	Label string
	ID    int
}

// Recipients lists the request's addressees in request order, before any
// subscriber lookup. IDs win when a request has both, as when sending.
func (m TxMessage) Recipients() []TxRecipient {
	if len(m.SubscriberIDs) > 0 {
		out := make([]TxRecipient, 0, len(m.SubscriberIDs))
		for _, id := range m.SubscriberIDs {
			out = append(out, TxRecipient{Label: fmt.Sprintf("#%d", id), ID: id})
		}
		return out
	}
	out := make([]TxRecipient, 0, len(m.SubscriberEmails))
	for _, e := range m.SubscriberEmails {
		out = append(out, TxRecipient{Label: e})
	}
	return out
}
