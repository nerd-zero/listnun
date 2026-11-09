package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
)

// twofaPolicyTTL bounds how stale the per-workspace "require two-step
// verification" flag can be; saving settings clears it right away.
const twofaPolicyTTL = 30 * time.Second

type twofaPolicyEntry struct {
	required bool
	at       time.Time
}

// twofaPolicy caches each workspace's security.require_2fa setting so the
// check below doesn't read settings on every API call.
type twofaPolicy struct {
	mu sync.Mutex
	m  map[int]twofaPolicyEntry
}

func (p *twofaPolicy) get(tenantID int, load func() (bool, error)) bool {
	p.mu.Lock()
	e, ok := p.m[tenantID]
	p.mu.Unlock()
	if ok && time.Since(e.at) < twofaPolicyTTL {
		return e.required
	}

	required, err := load()
	if err != nil {
		// Can't tell: don't lock people out over a read error.
		return false
	}
	p.mu.Lock()
	if p.m == nil {
		p.m = map[int]twofaPolicyEntry{}
	}
	p.m[tenantID] = twofaPolicyEntry{required: required, at: time.Now()}
	p.mu.Unlock()
	return required
}

func (p *twofaPolicy) forget(tenantID int) {
	p.mu.Lock()
	delete(p.m, tenantID)
	p.mu.Unlock()
}

// requireTwofa holds password users of workspaces that require two-step
// verification to their profile until they set it up: every other API call
// is refused with 403 and the X-Listnun-Error header set to twofa_required,
// which the admin app turns into a redirect to the profile page.
func (a *App) requireTwofa(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		u, ok := c.Get(auth.UserHTTPCtxKey).(auth.User)
		if !ok || !auth.NeedsTwofaSetup(u) || auth.TwofaSetupAllowed(c.Request().Method, c.Path()) {
			return next(c)
		}

		tID := tenantID(c)
		required := a.twofa.get(tID, func() (bool, error) {
			s, err := a.core.GetSettings(c.Request().Context(), tID)
			return s.SecurityRequire2FA, err
		})
		if !required {
			return next(c)
		}

		c.Response().Header().Set("X-Listnun-Error", auth.TwofaRequiredCode)
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("users.twofaRequired"))
	}
}
