package main

import (
	"context"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/captcha"
	"github.com/nerd-zero/listnun/models"
)

// tenantSettingsTTL is how long a workspace's settings are cached for the
// request paths that read them (public pages, CORS, captcha). A save on
// this replica drops the entry at once; other replicas catch up within it.
const tenantSettingsTTL = 30 * time.Second

// settingsReader is what tenantSettings reads from (Core).
type settingsReader interface {
	GetSettings(ctx context.Context, tenantID int) (models.Settings, error)
}

// tenantSettings caches each workspace's own settings for request-time
// reads, plus the captcha client built from them. Request handlers used to
// read these from the boot-time config, which is tenant 1's, so every other
// workspace's public page, archive, privacy, captcha and trusted-URL
// settings were ignored.
type tenantSettings struct {
	core settingsReader
	now  func() time.Time

	mu       sync.Mutex
	cache    map[int]tenantSettingsEntry
	captchas map[int]tenantCaptcha
}

type tenantSettingsEntry struct {
	s       models.Settings
	captcha *captcha.Captcha
	fetched time.Time
}

// tenantCaptcha is a workspace's captcha client and the settings it was
// built from. It's kept across settings refreshes and only rebuilt when
// those change: an ALTCHA client signs its challenges with a key made when
// it's built, so rebuilding it would reject challenges already handed out.
type tenantCaptcha struct {
	opt captcha.Opt
	c   *captcha.Captcha
}

func newTenantSettings(co settingsReader) *tenantSettings {
	return &tenantSettings{core: co, now: time.Now, cache: map[int]tenantSettingsEntry{}, captchas: map[int]tenantCaptcha{}}
}

func (t *tenantSettings) entry(ctx context.Context, tenantID int) (tenantSettingsEntry, error) {
	t.mu.Lock()
	e, ok := t.cache[tenantID]
	t.mu.Unlock()
	if ok && t.now().Sub(e.fetched) < tenantSettingsTTL {
		return e, nil
	}

	s, err := t.core.GetSettings(ctx, tenantID)
	if err != nil {
		return tenantSettingsEntry{}, err
	}
	opt := captchaOpt(s)

	t.mu.Lock()
	defer t.mu.Unlock()
	tc, ok := t.captchas[tenantID]
	if !ok || tc.opt != opt {
		tc = tenantCaptcha{opt: opt, c: captcha.New(opt)}
		t.captchas[tenantID] = tc
	}
	e = tenantSettingsEntry{s: s, captcha: tc.c, fetched: t.now()}
	t.cache[tenantID] = e
	return e, nil
}

// Get returns tenantID's settings.
func (t *tenantSettings) Get(ctx context.Context, tenantID int) (models.Settings, error) {
	e, err := t.entry(ctx, tenantID)
	return e.s, err
}

// Captcha returns tenantID's captcha client, or a disabled one if its
// settings can't be read.
func (t *tenantSettings) Captcha(ctx context.Context, tenantID int) *captcha.Captcha {
	e, err := t.entry(ctx, tenantID)
	if err != nil {
		return captcha.New(captcha.Opt{})
	}
	return e.captcha
}

// invalidate drops tenantID's cached settings after a save.
func (t *tenantSettings) invalidate(tenantID int) {
	t.mu.Lock()
	delete(t.cache, tenantID)
	t.mu.Unlock()
}

// captchaOpt is a workspace's captcha settings as captcha.New takes them.
func captchaOpt(s models.Settings) captcha.Opt {
	var o captcha.Opt
	sc := s.SecurityCaptcha
	o.Altcha.Enabled, o.Altcha.Complexity = sc.Altcha.Enabled, sc.Altcha.Complexity
	o.HCaptcha.Enabled, o.HCaptcha.Key, o.HCaptcha.Secret = sc.HCaptcha.Enabled, sc.HCaptcha.Key, sc.HCaptcha.Secret
	o.Turnstile.Enabled, o.Turnstile.Key, o.Turnstile.Secret = sc.Turnstile.Enabled, sc.Turnstile.Key, sc.Turnstile.Secret
	o.Login = sc.Login
	return o
}

// ws returns the request workspace's own settings. If they can't be read,
// it logs and returns zero settings, which turn the optional public
// features they gate (archive, public subscription page, self-service
// export and wipe, captcha) off rather than falling back to another
// workspace's.
func (a *App) ws(c echo.Context) models.Settings {
	s, err := a.settings.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		a.log.Printf("error reading tenant %d's settings: %v", tenantID(c), err)
	}
	return s
}

// captchaFor returns the request workspace's captcha client.
func (a *App) captchaFor(c echo.Context) *captcha.Captcha {
	return a.settings.Captcha(c.Request().Context(), tenantID(c))
}

// lookupMap turns a list into a set.
func lookupMap(keys []string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}
