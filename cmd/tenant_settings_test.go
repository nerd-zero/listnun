package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// fakeSettings serves per-tenant settings and counts reads.
type fakeSettings struct {
	byTenant map[int]models.Settings
	reads    int
}

func (f *fakeSettings) GetSettings(_ context.Context, tenantID int) (models.Settings, error) {
	f.reads++
	return f.byTenant[tenantID], nil
}

func TestTenantSettings_CachesAndInvalidates(t *testing.T) {
	fs := &fakeSettings{byTenant: map[int]models.Settings{2: {AppSiteName: "Acme"}}}
	ts := newTenantSettings(fs)
	now := time.Now()
	ts.now = func() time.Time { return now }
	ctx := context.Background()

	for range 3 {
		if s, _ := ts.Get(ctx, 2); s.AppSiteName != "Acme" {
			t.Fatalf("site name = %q", s.AppSiteName)
		}
	}
	if fs.reads != 1 {
		t.Errorf("read settings %d times, want once while cached", fs.reads)
	}

	fs.byTenant[2] = models.Settings{AppSiteName: "Acme Co"}
	ts.invalidate(2)
	if s, _ := ts.Get(ctx, 2); s.AppSiteName != "Acme Co" {
		t.Errorf("after a save, site name = %q, want the new one", s.AppSiteName)
	}

	now = now.Add(tenantSettingsTTL + time.Second)
	ts.Get(ctx, 2)
	if fs.reads != 3 {
		t.Errorf("reads = %d, want a fresh read once the entry expires", fs.reads)
	}
}

// An ALTCHA client signs challenges with a key made when it's built, so
// refreshing settings mustn't rebuild it (that would reject challenges on
// forms left open); changing the captcha settings must.
func TestTenantSettings_CaptchaKeptUntilItsSettingsChange(t *testing.T) {
	s := models.Settings{}
	s.SecurityCaptcha.Altcha.Enabled = true
	s.SecurityCaptcha.Altcha.Complexity = 1000
	fs := &fakeSettings{byTenant: map[int]models.Settings{2: s}}
	ts := newTenantSettings(fs)
	ctx := context.Background()

	first := ts.Captcha(ctx, 2)
	if !first.IsEnabled() {
		t.Fatal("tenant 2's ALTCHA isn't enabled")
	}
	ts.invalidate(2)
	if ts.Captcha(ctx, 2) != first {
		t.Error("refreshing unchanged settings rebuilt the captcha client")
	}

	s.SecurityCaptcha.Altcha.Enabled = false
	fs.byTenant[2] = s
	ts.invalidate(2)
	if c := ts.Captcha(ctx, 2); c == first || c.IsEnabled() {
		t.Error("turning the captcha off didn't take effect")
	}
}

// The archive and CORS were set up once from tenant 1's settings, for
// every workspace.
func TestArchiveAndCORS_PerWorkspace(t *testing.T) {
	on := models.Settings{EnablePublicArchive: true, SecurityTrustedURLs: []string{"https://acme.example/page"}}
	a := &App{
		settings: newTenantSettings(&fakeSettings{byTenant: map[int]models.Settings{2: on, 3: {}}}),
		log:      log.New(io.Discard, "", 0),
	}
	ok := func(c echo.Context) error { return c.String(http.StatusOK, "archive") }

	serve := func(tenantID int, origin string) *httptest.ResponseRecorder {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/archive", nil)
		if origin != "" {
			req.Header.Set(echo.HeaderOrigin, origin)
		}
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		tn := &models.Tenant{}
		tn.ID = tenantID
		c.Set(models.TenantCtxKey, tn)
		if err := a.tenantCORS(a.archiveOn(ok))(c); err != nil {
			e.HTTPErrorHandler(err, c)
		}
		return rec
	}

	if rec := serve(2, "https://acme.example"); rec.Code != http.StatusOK || rec.Header().Get(echo.HeaderAccessControlAllowOrigin) != "https://acme.example" {
		t.Errorf("tenant 2: %d, CORS %q; want its archive and its trusted origin allowed", rec.Code, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	}
	if rec := serve(3, "https://acme.example"); rec.Code != http.StatusNotFound || rec.Header().Get(echo.HeaderAccessControlAllowOrigin) != "" {
		t.Errorf("tenant 3: %d, CORS %q; want no archive and no CORS for another workspace's origin", rec.Code, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	}
}

// Saving a setting that's read live no longer restarts the process; a
// process-wide one still does, but only for tenant 1, whose settings the
// process runs on.
func TestSettingsRestart(t *testing.T) {
	cur := models.Settings{AppSiteName: "A", AppMessageRate: 10, UploadProvider: "filesystem"}

	live := cur
	live.AppSiteName, live.UploadProvider = "B", "s3"
	live.SecurityTrustedURLs = []string{"https://x.example"}
	if processWideChanged(cur, live) {
		t.Error("changing live settings counts as a process-wide change")
	}

	wide := cur
	wide.AppMessageRate = 50
	if !processWideChanged(cur, wide) {
		t.Error("changing the message rate isn't a process-wide change")
	}

	a := &App{cfg: &Config{}}
	a.cfg.MultiTenancyEnabled = true
	if a.restartsProcess(2) || !a.restartsProcess(1) {
		t.Error("multi-tenant: want restarts for tenant 1's changes only")
	}
	a.cfg.MultiTenancyEnabled = false
	if !a.restartsProcess(1) {
		t.Error("single-tenant: want restarts")
	}
}
