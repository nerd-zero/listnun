// Package notifs holds the system e-mail templates (opt-in confirmations,
// notifications, password resets and so on). It's initialized once in the
// main package; cmd/system_mail.go sends them per workspace.
package notifs

import (
	"bytes"
	"errors"
	"html/template"
	"regexp"
	"strings"
)

const (
	TplImport          = "import-status"
	TplCampaignStatus  = "campaign-status"
	TplSubscriberOptin = "subscriber-optin"
	TplSubscriberData  = "subscriber-data"
	TplForgotPassword  = "forgot-password"
)

type Opt struct {
	ContentType string
}

var (
	reTitle = regexp.MustCompile(`(?s)<title\s*data-i18n\s*>(.+?)</title>`)

	// Tpls are the system e-mail templates, with the boot-time (tenant 1)
	// template functions.
	Tpls *template.Template

	// pristine is a never-executed copy of Tpls for Render to clone:
	// html/template can't clone a template set once it has run.
	pristine *template.Template

	opt Opt
)

// Initialize sets the system e-mail templates. Sending is per workspace
// (cmd/system_mail.go); this package only holds the templates.
func Initialize(o Opt, tpls *template.Template) error {
	if Tpls != nil {
		return errors.New("notifs already initialized")
	}
	p, err := tpls.Clone()
	if err != nil {
		return err
	}
	Tpls, pristine, opt = tpls, p, o
	return nil
}

// ContentType is the system e-mail templates' content type (html or plain).
func ContentType() string {
	return opt.ContentType
}

// Render executes tplName with funcs replacing template functions of the
// same name (e.g. a workspace's own RootURL), on a fresh copy of the
// templates so concurrent renders for different workspaces don't mix.
func Render(funcs template.FuncMap, tplName string, data any) ([]byte, error) {
	t, err := pristine.Clone()
	if err != nil {
		return nil, err
	}
	t.Funcs(funcs)

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, tplName, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetTplSubject extracts any custom i18n subject rendered in the given rendered
// template body. If it's not found, the incoming subject and body are returned.
func GetTplSubject(subject string, body []byte) (string, []byte) {
	m := reTitle.FindSubmatch(body)
	if len(m) != 2 {
		return subject, body
	}

	return strings.TrimSpace(string(m[1])), reTitle.ReplaceAll(body, []byte(""))
}
