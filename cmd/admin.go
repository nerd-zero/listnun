package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	null "gopkg.in/volatiletech/null.v6"
)

type serverConfig struct {
	RootURL            string `json:"root_url"`
	FromEmail          string `json:"from_email"`
	PublicSubscription struct {
		Enabled          bool        `json:"enabled"`
		CaptchaEnabled   bool        `json:"captcha_enabled"`
		CaptchaProvider  null.String `json:"captcha_provider"`
		CaptchaKey       null.String `json:"captcha_key"`
		AltchaComplexity int         `json:"altcha_complexity"`
		RedirectURLs     []string    `json:"redirect_urls"`
	} `json:"public_subscription"`
	Privacy struct {
		DisableTracking    bool `json:"disable_tracking"`
		IndividualTracking bool `json:"individual_tracking"`
	} `json:"privacy"`
	MediaProvider    string          `json:"media_provider"`
	Messengers       []string        `json:"messengers"`
	Langs            []i18nLang      `json:"langs"`
	Lang             string          `json:"lang"`
	Permissions      json.RawMessage `json:"permissions"`
	Update           *AppUpdate      `json:"update"`
	NeedsRestart     bool            `json:"needs_restart"`
	HasLegacyUser    bool            `json:"has_legacy_user"`
	ScrubEnabled     bool            `json:"scrub_enabled"`
	Version          string          `json:"version"`
	OrganizationName string          `json:"organization_name,omitempty"`
	ConsoleURL       string          `json:"console_url,omitempty"`
} // @name ServerConfig

// GetServerConfig returns general server config.
//
//	@ID				getServerConfig
//	@Summary		Get server config
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	serverConfig
//	@Router			/api/config [get]
func (a *App) GetServerConfig(c echo.Context) error {
	rootURL := a.urlCfg.RootURL
	if a.cfg.MultiTenancyEnabled {
		// a.urlCfg.RootURL is a single boot-time value derived from
		// tenant 1's settings - every other tenant's admin SPA would
		// otherwise be told tenant 1's public URL for its own lists/
		// campaigns. Derive it from the actual request instead.
		rootURL = c.Scheme() + "://" + c.Request().Host
	}

	out := serverConfig{
		RootURL:       rootURL,
		FromEmail:     a.cfg.FromEmail,
		Lang:          a.cfg.Lang,
		Permissions:   a.cfg.PermissionsRaw,
		HasLegacyUser: a.cfg.HasLegacyUser,
		ConsoleURL:    a.cfg.ConsoleURL,
		Privacy: struct {
			DisableTracking    bool `json:"disable_tracking"`
			IndividualTracking bool `json:"individual_tracking"`
		}{
			DisableTracking:    a.cfg.Privacy.DisableTracking,
			IndividualTracking: a.cfg.Privacy.IndividualTracking,
		},
	}
	out.PublicSubscription.Enabled = a.cfg.EnablePublicSubPage
	for _, d := range a.cfg.Security.TrustedURLs {
		if d == "*" {
			continue
		}
		out.PublicSubscription.RedirectURLs = append(out.PublicSubscription.RedirectURLs, d)
	}

	// CAPTCHA.
	if w := a.captchaWidget(); w.Enabled {
		out.PublicSubscription.CaptchaEnabled = true
		out.PublicSubscription.CaptchaProvider = null.StringFrom(w.Provider)
		out.PublicSubscription.AltchaComplexity = w.Complexity
		if w.Key != "" {
			out.PublicSubscription.CaptchaKey = null.StringFrom(w.Key)
		}
	}

	if _, settings, err := a.media.Get(c.Request().Context(), tenantID(c)); err == nil {
		out.MediaProvider = settings.UploadProvider
	} else {
		out.MediaProvider = a.cfg.MediaUpload.Provider
	}

	// Organizations are an operator-managed, multi-tenancy-only concept -
	// single-tenant installs have no tenant to look one up for.
	if a.cfg.MultiTenancyEnabled {
		if name, err := a.core.GetTenantOrganizationName(tenantID(c)); err == nil {
			out.OrganizationName = name
		}
	}

	// Language list.
	langList, err := getI18nLangList(a.fs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList

	out.Messengers = make([]string, 0, len(a.messengers))
	for _, m := range a.messengers {
		out.Messengers = append(out.Messengers, m.Name())
	}

	if s, err := a.core.GetSettings(c.Request().Context(), tenantID(c)); err == nil {
		out.ScrubEnabled = s.Scrub.Enabled && s.Scrub.URL != "" && s.Scrub.APIKey != ""

		// a.cfg.FromEmail (out.FromEmail's default above) is a single
		// boot-time value derived from tenant 1's settings - every other
		// tenant's "New Campaign" form would otherwise be pre-filled with
		// tenant 1's From address. Use this tenant's own app.from_email
		// instead, same as RootURL above.
		if a.cfg.MultiTenancyEnabled && s.AppFromEmail != "" {
			out.FromEmail = s.AppFromEmail
		}
	}

	a.Lock()
	out.NeedsRestart = a.needsRestart
	out.Update = a.update
	a.Unlock()
	out.Version = versionString

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCharts returns chart data points to render ont he dashboard.
//
//	@ID				getDashboardCharts
//	@Summary		Get dashboard chart data
//	@Tags			dashboard
//	@Produce		json
//	@Success		200	{object}	object
//	@Router			/api/dashboard/charts [get]
func (a *App) GetDashboardCharts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := a.core.GetDashboardCharts(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetDashboardCounts returns stats counts to show on the dashboard.
//
//	@ID				getDashboardCounts
//	@Summary		Get dashboard counts
//	@Tags			dashboard
//	@Produce		json
//	@Success		200	{object}	object
//	@Router			/api/dashboard/counts [get]
func (a *App) GetDashboardCounts(c echo.Context) error {
	// Get the chart data from the DB.
	out, err := a.core.GetDashboardCounts(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// dashboardPeriods are the periods the dashboard can show, in days.
var dashboardPeriods = map[int]bool{7: true, 30: true, 90: true}

// GetDashboardOverview returns live dashboard numbers for a period: sends,
// bounces and failures per day, subscriber growth, queue and the counts
// behind "Needs attention".
//
//	@ID				getDashboardOverview
//	@Summary		Get the dashboard overview
//	@Tags			dashboard
//	@Produce		json
//	@Param			days	query		int	false	"7, 30 or 90 (default 30)"
//	@Success		200		{object}	models.DashboardOverview
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/dashboard/overview [get]
func (a *App) GetDashboardOverview(c echo.Context) error {
	days := 30
	if v := c.QueryParam("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || !dashboardPeriods[n] {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "days"))
		}
		days = n
	}

	out, err := a.core.GetDashboardOverview(c.Request().Context(), tenantID(c), days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// ReloadApp sends a reload signal to the app, causing a full restart.
//
//	@ID				reloadApp
//	@Summary		Reload the application
//	@Tags			settings
//	@Produce		json
//	@Success		200
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/admin/reload [post]
func (a *App) ReloadApp(c echo.Context) error {
	go func() {
		<-time.After(time.Millisecond * 500)

		// Send the reload signal to trigger the wait loop in main.
		a.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}
