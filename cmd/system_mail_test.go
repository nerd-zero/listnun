package main

import (
	"context"
	"errors"
	"html/template"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/nerd-zero/listnun/internal/i18n"
	"github.com/nerd-zero/listnun/internal/manager"
	"github.com/nerd-zero/listnun/internal/notifs"
	"github.com/nerd-zero/listnun/internal/subimporter"
	"github.com/nerd-zero/listnun/models"
)

// The real system e-mail templates, with tenant 1's boot-time URLs in
// RootURL and LogoURL, as initNotifs sets them up.
func TestMain(m *testing.M) {
	lang, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		panic(err)
	}
	tpls, err := template.New("").
		Funcs(initTplFuncs(lang, &UrlConfig{RootURL: "https://tenant1.example", LogoURL: "https://tenant1.example/logo.png"})).
		ParseGlob("../static/email-templates/*.html")
	if err != nil {
		panic(err)
	}
	if err := notifs.Initialize(notifs.Opt{ContentType: models.CampaignContentTypeHTML}, tpls); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// fakeMailCore serves per-tenant settings and the lists an opt-in covers.
type fakeMailCore struct {
	settings map[int]models.Settings
	lists    []models.List
}

func (f *fakeMailCore) GetSettings(_ context.Context, tenantID int) (models.Settings, error) {
	s, ok := f.settings[tenantID]
	if !ok {
		return s, errors.New("no settings")
	}
	return s, nil
}

func (f *fakeMailCore) GetSubscriberLists(context.Context, int, int, string, []int, []string, string, string) ([]models.List, error) {
	return f.lists, nil
}

// fakeMsgr records what it's asked to send.
type fakeMsgr struct {
	name string
	mu   sync.Mutex
	sent []models.Message
}

func (f *fakeMsgr) Name() string { return f.name }
func (f *fakeMsgr) Push(m models.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}
func (f *fakeMsgr) Flush() error { return nil }
func (f *fakeMsgr) Close() error { return nil }

// fakeResolver hands out each tenant's own messenger, if it has one.
type fakeResolver map[int]manager.Messenger

func (r fakeResolver) GetMessenger(_ context.Context, tenantID int, _ string) (manager.Messenger, error) {
	if m, ok := r[tenantID]; ok {
		return m, nil
	}
	return nil, manager.ErrMessengerNotFound
}

func newTestSystemMail(co *fakeMailCore, r fakeResolver, fallback manager.Messenger) *systemMail {
	lang, _ := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	return &systemMail{
		core:     co,
		msgrs:    r,
		fallback: fallback,
		defaults: systemMailDefaults{FromEmail: "Tenant 1 <hello@tenant1.example>", RootURL: "https://tenant1.example", LogoURL: "https://tenant1.example/logo.png"},
		i18n:     lang,
		log:      log.New(io.Discard, "", 0),
	}
}

// Opt-in confirmations for every workspace went out through tenant 1's
// SMTP and From address, with links on tenant 1's domain where the
// workspace's subscriber doesn't exist, so double opt-in couldn't work.
func TestOptinNotify_AsTheWorkspace(t *testing.T) {
	own, platform := &fakeMsgr{name: "email"}, &fakeMsgr{name: "email"}
	co := &fakeMailCore{
		settings: map[int]models.Settings{
			2: {AppFromEmail: "Acme <news@acme.example>", AppRootURL: "https://acme.listnun.app/", PrivacyUnsubHeader: true},
		},
		lists: []models.List{{UUID: "list-uuid", Name: "Weekly", Type: models.ListTypePublic}},
	}
	sm := newTestSystemMail(co, fakeResolver{2: own}, platform)

	sub := models.Subscriber{Email: "reader@example.com"}
	sub.ID, sub.UUID = 9, "sub-uuid"
	n, err := sm.optinNotify(2, sub, nil)
	if err != nil || n != 1 {
		t.Fatalf("optinNotify = %d, %v; want 1 list, no error", n, err)
	}
	if len(platform.sent) != 0 || len(own.sent) != 1 {
		t.Fatalf("sent %d through the workspace's SMTP and %d through the server's; want 1 and 0", len(own.sent), len(platform.sent))
	}

	m := own.sent[0]
	body := string(m.Body)
	if m.From != "Acme <news@acme.example>" || m.TenantID != 2 {
		t.Errorf("From = %q, tenant %d; want the workspace's", m.From, m.TenantID)
	}
	if !strings.Contains(body, "https://acme.listnun.app/subscription/optin/sub-uuid?l=list-uuid") {
		t.Errorf("body has no opt-in link on the workspace's domain:\n%s", body)
	}
	if strings.Contains(body, "tenant1.example") {
		t.Errorf("body links to tenant 1:\n%s", body)
	}
	if got := m.Headers.Get("List-Unsubscribe"); got != "<https://acme.listnun.app/subscription/"+dummyUUID+"/sub-uuid>" {
		t.Errorf("List-Unsubscribe = %q, want the workspace's unsubscribe link", got)
	}
}

// Campaign and import notifications went to tenant 1's notify addresses.
func TestNotifyAdmins_TheWorkspacesOwnAddresses(t *testing.T) {
	own := &fakeMsgr{name: "email"}
	co := &fakeMailCore{settings: map[int]models.Settings{
		2: {AppNotifyEmails: []string{"ops@acme.example"}, AppRootURL: "https://acme.listnun.app"},
		3: {},
	}}
	sm := newTestSystemMail(co, fakeResolver{2: own, 3: own}, nil)

	data := map[string]any{"ID": 5, "Name": "Launch", "Status": "finished", "Sent": 1, "ToSend": 1, "Reason": ""}
	if err := sm.notifyAdmins(context.Background(), 2, "Finished: Launch", notifs.TplCampaignStatus, data); err != nil {
		t.Fatalf("notifyAdmins: %v", err)
	}
	if len(own.sent) != 1 || own.sent[0].To[0] != "ops@acme.example" {
		t.Fatalf("sent %+v, want one notification to the workspace's ops@acme.example", own.sent)
	}
	if body := string(own.sent[0].Body); !strings.Contains(body, "https://acme.listnun.app/admin/campaigns/5") {
		t.Errorf("campaign link isn't on the workspace's domain:\n%s", body)
	}

	// A workspace with no notify addresses gets nothing, not tenant 1's.
	if err := sm.notifyAdmins(context.Background(), 3, "Finished: Launch", notifs.TplCampaignStatus, data); err != nil {
		t.Fatalf("notifyAdmins (no addresses): %v", err)
	}
	if len(own.sent) != 1 {
		t.Errorf("a workspace with no notify addresses sent %d notifications", len(own.sent)-1)
	}
}

// A workspace without SMTP of its own still gets its password resets,
// through the server's, but from its own address.
func TestSend_FallsBackToTheServersSMTP(t *testing.T) {
	platform := &fakeMsgr{name: "email"}
	co := &fakeMailCore{settings: map[int]models.Settings{4: {AppFromEmail: "Beta <hi@beta.example>"}}}
	sm := newTestSystemMail(co, fakeResolver{}, platform)

	err := sm.send(context.Background(), 4, systemMessage{
		To: []string{"admin@beta.example"}, Subject: "Reset", Template: notifs.TplForgotPassword,
		Data: struct {
			ResetURL string
			L        *i18n.I18n
		}{ResetURL: "https://beta.listnun.app/admin/reset?token=t", L: sm.i18n},
	})
	if err != nil || len(platform.sent) != 1 || platform.sent[0].From != "Beta <hi@beta.example>" {
		t.Fatalf("send = %v, sent %+v; want one message through the server's SMTP from the workspace's address", err, platform.sent)
	}

	// With neither, it's an error rather than a silent drop.
	sm.fallback = nil
	if err := sm.send(context.Background(), 4, systemMessage{To: []string{"a@b.c"}, Template: notifs.TplForgotPassword, Data: struct {
		ResetURL string
		L        *i18n.I18n
	}{L: sm.i18n}}); !errors.Is(err, errNoMessenger) {
		t.Errorf("err = %v, want errNoMessenger", err)
	}
}

// Renders for different workspaces at once don't mix their URLs, and
// rendering still works after the shared templates have been executed
// directly (html/template can't clone a set that has run).
func TestRender_PerWorkspaceAndConcurrent(t *testing.T) {
	if err := notifs.Tpls.ExecuteTemplate(io.Discard, "smtp-test", nil); err != nil {
		t.Fatalf("executing the shared templates: %v", err)
	}

	var wg sync.WaitGroup
	for _, root := range []string{"https://a.example", "https://b.example", "https://c.example"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_, body, err := render(workspaceMail{RootURL: root}, "s", notifs.TplImport, subimporter.ImportNotif{Name: "f.csv", Status: "finished"})
				if err != nil {
					t.Errorf("render: %v", err)
					return
				}
				if !strings.Contains(string(body), root+"/admin/subscribers/import") {
					t.Errorf("render for %s gave another workspace's link", root)
					return
				}
			}
		}()
	}
	wg.Wait()
}
