package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/internal/manager"
	"github.com/nerd-zero/listnun/internal/messenger/email"
	"github.com/nerd-zero/listnun/internal/notifs"
	"github.com/nerd-zero/listnun/models"
)

// systemMail sends a workspace's system e-mails (opt-in confirmations,
// password resets, data exports, campaign and import notifications) as
// that workspace: through its own SMTP servers, from its own From address,
// to its own notify addresses, with links on its own root URL. They all
// used to go out through tenant 1's SMTP and From address with tenant 1's
// links, so another workspace's opt-in links didn't work.
//
// Core needs the opt-in hook at construction and the hook needs Core, so
// boot (cmd/main.go) creates this first and fills in core, msgrs and
// fallback before anything is sent.
type systemMail struct {
	core  systemMailCore
	msgrs manager.MessengerResolver

	// fallback is the boot-time e-mail messenger (tenant 1's SMTP), used
	// for a workspace whose own SMTP can't be resolved, so a password reset
	// still arrives. Nil when tenant 1 has no e-mail messenger.
	fallback manager.Messenger

	// defaults are the boot-time (tenant 1) settings, for anything a
	// workspace hasn't set.
	defaults systemMailDefaults

	i18n *i18n.I18n
	log  *log.Logger
}

// systemMailCore is what systemMail reads from Core.
type systemMailCore interface {
	GetSettings(ctx context.Context, tenantID int) (models.Settings, error)
	GetSubscriberLists(ctx context.Context, tenantID int, subID int, uuid string, listIDs []int, listUUIDs []string, subStatus string, listType string) ([]models.List, error)
}

// systemMailDefaults are the boot-time values used where a workspace's
// own setting is empty.
type systemMailDefaults struct {
	FromEmail   string
	RootURL     string
	LogoURL     string
	UnsubHeader bool
}

// workspaceMail is what one workspace's system e-mail is sent with.
type workspaceMail struct {
	From        string
	Notify      []string
	RootURL     string
	LogoURL     string
	UnsubHeader bool
}

// errNoMessenger means neither the workspace nor the server has an e-mail
// messenger to send with.
var errNoMessenger = errors.New("no e-mail messenger to send system e-mail with")

// workspace returns tenantID's From address, notify addresses, root URL and
// logo. From and root URL fall back to the boot-time values when the
// workspace's own are empty; everything does when its settings can't be
// read.
func (s *systemMail) workspace(ctx context.Context, tenantID int) workspaceMail {
	w := workspaceMail{From: s.defaults.FromEmail, RootURL: s.defaults.RootURL, LogoURL: s.defaults.LogoURL, UnsubHeader: s.defaults.UnsubHeader}

	set, err := s.core.GetSettings(ctx, tenantID)
	if err != nil {
		s.log.Printf("system e-mail: error reading tenant %d's settings, using defaults: %v", tenantID, err)
		return w
	}
	if set.AppFromEmail != "" {
		w.From = set.AppFromEmail
	}
	if r := strings.TrimSuffix(set.AppRootURL, "/"); r != "" {
		w.RootURL = r
	}
	// The logo is the workspace's own or none: tenant 1's would put
	// another organisation's logo in this workspace's e-mail.
	w.LogoURL = set.AppLogoURL
	w.Notify = set.AppNotifyEmails
	w.UnsubHeader = set.PrivacyUnsubHeader
	return w
}

// messenger returns tenantID's own default e-mail messenger, else the
// boot-time one.
func (s *systemMail) messenger(ctx context.Context, tenantID int) (manager.Messenger, error) {
	m, err := s.msgrs.GetMessenger(ctx, tenantID, email.MessengerName)
	if err == nil {
		return m, nil
	}
	if s.fallback == nil {
		return nil, fmt.Errorf("%w: %v", errNoMessenger, err)
	}
	s.log.Printf("system e-mail: tenant %d has no e-mail messenger of its own (%v), sending through the server's", tenantID, err)
	return s.fallback, nil
}

// render executes tplName with the workspace's root URL and logo in the
// RootURL and LogoURL template functions, and returns the subject (from the
// template's <title data-i18n>, else subject) and body.
func render(w workspaceMail, subject, tplName string, data any) (string, []byte, error) {
	b, err := notifs.Render(template.FuncMap{
		"RootURL": func() string { return w.RootURL },
		"LogoURL": func() string { return w.LogoURL },
	}, tplName, data)
	if err != nil {
		return "", nil, fmt.Errorf("error compiling notification template '%s': %w", tplName, err)
	}
	sub, body := notifs.GetTplSubject(subject, b)
	return sub, body, nil
}

// systemMessage is one system e-mail to send.
type systemMessage struct {
	To          []string
	Subject     string
	Template    string
	Data        any
	Headers     textproto.MIMEHeader
	Attachments []models.Attachment
}

// send sends m as tenantID.
func (s *systemMail) send(ctx context.Context, tenantID int, m systemMessage) error {
	return s.sendAs(ctx, tenantID, s.workspace(ctx, tenantID), m)
}

func (s *systemMail) sendAs(ctx context.Context, tenantID int, w workspaceMail, m systemMessage) error {
	if len(m.To) == 0 {
		return nil
	}

	subject, body, err := render(w, m.Subject, m.Template, m.Data)
	if err != nil {
		s.log.Printf("system e-mail: %v", err)
		return err
	}

	msgr, err := s.messenger(ctx, tenantID)
	if err != nil {
		return err
	}
	if err := msgr.Push(models.Message{
		TenantID:    tenantID,
		Messenger:   email.MessengerName,
		ContentType: notifs.ContentType(),
		From:        w.From,
		To:          m.To,
		Subject:     subject,
		Body:        body,
		Headers:     m.Headers,
		Attachments: m.Attachments,
	}); err != nil {
		s.log.Printf("system e-mail: error sending '%s' for tenant %d: %v", subject, tenantID, err)
		return err
	}
	return nil
}

// notifyAdmins sends a notification to tenantID's own notify addresses
// (Settings › General), if it has any.
func (s *systemMail) notifyAdmins(ctx context.Context, tenantID int, subject, tplName string, data any) error {
	w := s.workspace(ctx, tenantID)
	return s.sendAs(ctx, tenantID, w, systemMessage{To: w.Notify, Subject: subject, Template: tplName, Data: data})
}

// optinNotify sends sub an opt-in confirmation for those of listIDs (all
// their lists when nil) that are double opt-in and still unconfirmed, as
// tenantID, and returns how many lists it asked them to confirm. It's
// Core's SendOptinConfirmation hook.
func (s *systemMail) optinNotify(tenantID int, sub models.Subscriber, listIDs []int) (int, error) {
	ctx := context.Background()

	lists, err := s.core.GetSubscriberLists(ctx, tenantID, sub.ID, "", listIDs, nil,
		models.SubscriptionStatusUnconfirmed, models.ListOptinDouble)
	if err != nil {
		s.log.Printf("error fetching lists for opt-in: %s", err)
		return 0, err
	}
	if len(lists) == 0 {
		return 0, nil
	}

	w := s.workspace(ctx, tenantID)
	out := subOptin{Subscriber: sub, Lists: lists}
	q := url.Values{}
	for _, l := range lists {
		q.Add("l", l.UUID)
	}
	out.OptinURL = fmt.Sprintf("%s/subscription/optin/%s?%s", w.RootURL, sub.UUID, q.Encode())
	out.UnsubURL = fmt.Sprintf("%s/subscription/%s/%s", w.RootURL, dummyUUID, sub.UUID)

	hdr := textproto.MIMEHeader{}
	hdr.Set(models.EmailHeaderSubscriberUUID, sub.UUID)
	if w.UnsubHeader {
		hdr.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
		hdr.Set("List-Unsubscribe", `<`+out.UnsubURL+`>`)
	}

	if err := s.sendAs(ctx, tenantID, w, systemMessage{
		To:       []string{sub.Email},
		Subject:  s.i18n.T("subscribers.optinSubject"),
		Template: notifs.TplSubscriberOptin,
		Data:     out,
		Headers:  hdr,
	}); err != nil {
		s.log.Printf("error sending opt-in e-mail for subscriber %d (%s): %s", sub.ID, sub.UUID, err)
		return 0, err
	}
	return len(lists), nil
}
