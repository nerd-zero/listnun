package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gdgvda/cron"
	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx/types"
	koanfjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/notifs"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

const pwdMask = "•"

type aboutHost struct {
	OS       string `json:"os"`
	Machine  string `json:"arch"`
	Hostname string `json:"hostname"`
} // @name AboutHost

type aboutSystem struct {
	NumCPU  int    `json:"num_cpu"`
	AllocMB uint64 `json:"memory_alloc_mb"`
	OSMB    uint64 `json:"memory_from_os_mb"`
} // @name AboutSystem

type about struct {
	Version   string         `json:"version"`
	Build     string         `json:"build"`
	GoVersion string         `json:"go_version"`
	GoArch    string         `json:"go_arch"`
	Database  types.JSONText `json:"database"`
	System    aboutSystem    `json:"system"`
	Host      aboutHost      `json:"host"`
} // @name About

var (
	reAlphaNum = regexp.MustCompile(`[^a-z0-9\-]`)
)

// GetSettings returns settings from the DB.
//
//	@ID				getSettings
//	@Summary		Get application settings
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	models.Settings
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/settings [get]
func (a *App) GetSettings(c echo.Context) error {
	s, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	// Empty out passwords.
	for i := range s.SMTP {
		s.SMTP[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SMTP[i].Password))
	}
	for i := range s.BounceBoxes {
		s.BounceBoxes[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceBoxes[i].Password))
	}
	for i := range s.Messengers {
		s.Messengers[i].Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.Messengers[i].Password))
	}

	s.Scrub.APIKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.Scrub.APIKey))
	s.UploadS3AwsSecretAccessKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.UploadS3AwsSecretAccessKey))
	s.SendgridKey = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SendgridKey))
	s.BounceAzure.SharedSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceAzure.SharedSecret))
	s.BouncePostmark.Password = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BouncePostmark.Password))
	s.BounceForwardEmail.Key = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceForwardEmail.Key))
	s.BounceLettermint.Key = strings.Repeat(pwdMask, utf8.RuneCountInString(s.BounceLettermint.Key))
	s.SecurityCaptcha.HCaptcha.Secret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.SecurityCaptcha.HCaptcha.Secret))
	s.OIDC.ClientSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.OIDC.ClientSecret))

	return c.JSON(http.StatusOK, okResp{s})
}

// UpdateSettings updates and saves settings to the DB.
//
//	@ID				updateSettings
//	@Summary		Update application settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.Settings	true	"Settings payload"
//	@Success		200
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/settings [put]
func (a *App) UpdateSettings(c echo.Context) error {
	// Unmarshal and marshal the fields once to sanitize the settings blob.
	var set models.Settings
	if err := c.Bind(&set); err != nil {
		return err
	}

	// Get the existing settings.
	cur, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	// Validate and sanitize postback Messenger names along with SMTP names
	// (where each SMTP is also considered as a standalone messenger).
	// Duplicates are disallowed and "email" is a reserved name.
	names := map[string]bool{emailMsgr: true}

	// There should be at least one SMTP block that's enabled.
	has := false
	for i, s := range set.SMTP {
		if s.Enabled {
			has = true
		}

		// Sanitize and normalize the SMTP server name.
		name := reAlphaNum.ReplaceAllString(strings.ToLower(strings.TrimSpace(s.Name)), "-")
		if name != "" {
			if !strings.HasPrefix(name, "email-") {
				name = "email-" + name
			}

			if _, ok := names[name]; ok {
				return echo.NewHTTPError(http.StatusBadRequest,
					a.i18n.Ts("settings.duplicateMessengerName", "name", name))
			}

			names[name] = true
		}
		set.SMTP[i].Name = name

		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of SMTP blocks with the array in the DB.
		if s.UUID == "" {
			set.SMTP[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		// Ensure the HOST is trimmed of any whitespace.
		// This is a common mistake when copy-pasting SMTP settings.
		set.SMTP[i].Host = strings.TrimSpace(s.Host)

		// If there's no password coming in from the frontend, copy the existing
		// password by matching the UUID.
		if s.Password == "" {
			for _, c := range cur.SMTP {
				if s.UUID == c.UUID {
					set.SMTP[i].Password = c.Password
				}
			}
		}
	}
	if !has {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.errorNoSMTP"))
	}

	// Normalize `from_addresses``. Values are either an e-mail address
	// or an FQDN. Duplicate domains across server blocks are allowed
	// (they get round-robin'd while sending).
	for i, s := range set.SMTP {
		if !s.Enabled {
			continue
		}

		addrs := make([]string, 0, len(s.FromAddresses))
		for _, addr := range s.FromAddresses {
			if k := email.NormalizeAddr(addr); k != "" {
				addrs = append(addrs, k)
			}
		}
		set.SMTP[i].FromAddresses = addrs
	}

	// Always remove the trailing slash from the app root URL.
	set.AppRootURL = strings.TrimRight(set.AppRootURL, "/")

	// Bounce boxes.
	for i, s := range set.BounceBoxes {
		// Assign a UUID. The frontend only sends a password when the user explicitly
		// changes the password. In other cases, the existing password in the DB
		// is copied while updating the settings and the UUID is used to match
		// the incoming array of blocks with the array in the DB.
		if s.UUID == "" {
			set.BounceBoxes[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		// Ensure the HOST is trimmed of any whitespace.
		// This is a common mistake when copy-pasting SMTP settings.
		set.BounceBoxes[i].Host = strings.TrimSpace(s.Host)

		if d, _ := time.ParseDuration(s.ScanInterval); d.Minutes() < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.bounces.invalidScanInterval"))
		}

		// If there's no password coming in from the frontend, copy the existing
		// password by matching the UUID.
		if s.Password == "" {
			for _, c := range cur.BounceBoxes {
				if s.UUID == c.UUID {
					set.BounceBoxes[i].Password = c.Password
				}
			}
		}
	}

	for i, m := range set.Messengers {
		// UUID to keep track of password changes similar to the SMTP logic above.
		if m.UUID == "" {
			set.Messengers[i].UUID = uuid.Must(uuid.NewV4()).String()
		}

		if m.Password == "" {
			for _, c := range cur.Messengers {
				if m.UUID == c.UUID {
					set.Messengers[i].Password = c.Password
				}
			}
		}

		name := reAlphaNum.ReplaceAllString(strings.ToLower(m.Name), "")
		if _, ok := names[name]; ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("settings.duplicateMessengerName", "name", name))
		}
		if len(name) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.invalidMessengerName"))
		}

		set.Messengers[i].Name = name
		names[name] = true
	}

	// Scrub API key — preserve if masked or empty.
	if set.Scrub.APIKey == "" || strings.Contains(set.Scrub.APIKey, pwdMask) {
		set.Scrub.APIKey = cur.Scrub.APIKey
	}
	if set.Scrub.Enabled && set.Scrub.URL != "" {
		if u, err := url.ParseRequestURI(set.Scrub.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidData")+": invalid Scrub URL")
		}
	}

	// S3 password?
	if set.UploadS3AwsSecretAccessKey == "" {
		set.UploadS3AwsSecretAccessKey = cur.UploadS3AwsSecretAccessKey
	}
	if set.SendgridKey == "" {
		set.SendgridKey = cur.SendgridKey
	}
	if set.BounceAzure.SharedSecret == "" {
		set.BounceAzure.SharedSecret = cur.BounceAzure.SharedSecret
	}
	if set.BouncePostmark.Password == "" {
		set.BouncePostmark.Password = cur.BouncePostmark.Password
	}
	if set.BounceForwardEmail.Key == "" {
		set.BounceForwardEmail.Key = cur.BounceForwardEmail.Key
	}
	if set.BounceLettermint.Key == "" {
		set.BounceLettermint.Key = cur.BounceLettermint.Key
	}
	if set.SecurityCaptcha.HCaptcha.Secret == "" {
		set.SecurityCaptcha.HCaptcha.Secret = cur.SecurityCaptcha.HCaptcha.Secret
	}
	if set.OIDC.ClientSecret == "" {
		set.OIDC.ClientSecret = cur.OIDC.ClientSecret
	}

	// OIDC user auto-creation is enabled. Validate.
	if set.OIDC.AutoCreateUsers {
		if set.OIDC.DefaultUserRoleID.Int < auth.SuperAdminRoleID {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidFields", "name", a.i18n.T("settings.security.OIDCDefaultRole")))
		}
	}

	for n, v := range set.UploadExtensions {
		set.UploadExtensions[n] = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "."))
	}

	// Domain blocklist / allowlist.
	doms := make([]string, 0, len(set.DomainBlocklist))
	for _, d := range set.DomainBlocklist {
		if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainBlocklist = doms

	doms = make([]string, 0, len(set.DomainAllowlist))
	for _, d := range set.DomainAllowlist {
		if d = strings.TrimSpace(strings.ToLower(d)); d != "" {
			doms = append(doms, d)
		}
	}
	set.DomainAllowlist = doms

	// Validate and clean trusted URLs.
	urls := make([]string, 0, len(set.SecurityTrustedURLs))
	for _, d := range set.SecurityTrustedURLs {
		if d = strings.TrimSpace(d); d != "" {
			if d == "*" {
				urls = append(urls, d)
				continue
			}

			// Parse and validate the URL.
			u, err := url.Parse(d)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return echo.NewHTTPError(http.StatusBadRequest,
					a.i18n.Ts("globals.messages.invalidData")+": invalid trusted URL: "+d)
			}
			urls = append(urls, d)
		}
	}
	set.SecurityTrustedURLs = urls

	// Validate slow query caching cron.
	if set.CacheSlowQueries {
		if _, err := cron.ParseStandard(set.CacheSlowQueriesInterval); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidData")+": slow query cron: "+err.Error())
		}
	}

	// Update the settings in the DB.
	if err := a.core.UpdateSettings(c.Request().Context(), tenantID(c), set); err != nil {
		return err
	}

	// SMTP/upload/OIDC settings can be applied by invalidating that
	// tenant's cached resolver (cmd/tenant_messenger.go, cmd/tenant_media.go,
	// internal/auth's OIDC cache) instead of the full-process restart every
	// other settings group still needs - a full restart affects every
	// tenant on the process, not just the one saving.
	changes := diffSettingsGroups(cur, set)
	if changes.other {
		return a.handleSettingsRestart(c)
	}
	a.invalidateTenantCaches(tenantID(c), changes)

	return c.JSON(http.StatusOK, okResp{true})
}

// UpdateSettingsByKey updates a single setting key-value in the DB.
//
//	@ID				updateSettingByKey
//	@Summary		Update a single setting by key
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string			true	"Setting key"
//	@Param			body	body		interface{}		true	"Value (raw JSON)"
//	@Success		200
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/settings/{key} [put]
func (a *App) UpdateSettingsByKey(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	// Read the raw JSON body as the value.
	var b json.RawMessage
	if err := c.Bind(&b); err != nil {
		return err
	}

	// Update the value in the DB.
	if err := a.core.UpdateSettingsByKey(c.Request().Context(), tenantID(c), key, b); err != nil {
		return err
	}

	// Every settings row is keyed by one of models.Settings' top-level JSON
	// tags (schema.sql's `settings` seed rows, one row per key) - so the
	// key alone says which per-tenant cache to invalidate, same groups as
	// diffSettingsGroups. Anything outside those still needs the full
	// restart.
	switch {
	case key == "smtp":
		a.tenantMsgrs.Invalidate(tenantID(c))
	case strings.HasPrefix(key, "upload."):
		a.media.Invalidate(tenantID(c))
	case key == "security.oidc":
		a.auth.InvalidateOIDC(tenantID(c))
	default:
		return a.handleSettingsRestart(c)
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// tenantCacheChanges describes which process-lifetime per-tenant caches
// (cmd/tenant_messenger.go, cmd/tenant_media.go, internal/auth's OIDC
// cache) are stale after a settings save, and whether anything outside
// those three groups also changed.
type tenantCacheChanges struct {
	smtp  bool
	media bool
	oidc  bool
	other bool
}

// withoutTenantCachedGroups zeroes out the SMTP/upload/OIDC fields so the
// remainder can be compared to detect changes outside those groups.
func withoutTenantCachedGroups(s models.Settings) models.Settings {
	var zero models.Settings
	s.SMTP = nil
	s.OIDC = zero.OIDC
	s.UploadProvider = ""
	s.UploadExtensions = nil
	s.UploadFilesystemUploadPath = ""
	s.UploadFilesystemUploadURI = ""
	s.UploadS3URL = ""
	s.UploadS3PublicURL = ""
	s.UploadS3AwsAccessKeyID = ""
	s.UploadS3AwsDefaultRegion = ""
	s.UploadS3AwsSecretAccessKey = ""
	s.UploadS3Bucket = ""
	s.UploadS3BucketDomain = ""
	s.UploadS3BucketPath = ""
	s.UploadS3BucketType = ""
	s.UploadS3Expiry = ""
	return s
}

// diffSettingsGroups compares settings before and after a save to work out
// which per-tenant caches need invalidating. "other" is deliberately a
// diff of everything *except* the three known groups (rather than a list
// of every other field) so that any field added to models.Settings in the
// future safely falls into "other" (triggering the existing full-restart
// fallback) instead of silently being ignored by this optimization.
func diffSettingsGroups(cur, set models.Settings) tenantCacheChanges {
	return tenantCacheChanges{
		smtp: !reflect.DeepEqual(cur.SMTP, set.SMTP),
		media: cur.UploadProvider != set.UploadProvider ||
			!reflect.DeepEqual(cur.UploadExtensions, set.UploadExtensions) ||
			cur.UploadFilesystemUploadPath != set.UploadFilesystemUploadPath ||
			cur.UploadFilesystemUploadURI != set.UploadFilesystemUploadURI ||
			cur.UploadS3URL != set.UploadS3URL ||
			cur.UploadS3PublicURL != set.UploadS3PublicURL ||
			cur.UploadS3AwsAccessKeyID != set.UploadS3AwsAccessKeyID ||
			cur.UploadS3AwsDefaultRegion != set.UploadS3AwsDefaultRegion ||
			cur.UploadS3AwsSecretAccessKey != set.UploadS3AwsSecretAccessKey ||
			cur.UploadS3Bucket != set.UploadS3Bucket ||
			cur.UploadS3BucketDomain != set.UploadS3BucketDomain ||
			cur.UploadS3BucketPath != set.UploadS3BucketPath ||
			cur.UploadS3BucketType != set.UploadS3BucketType ||
			cur.UploadS3Expiry != set.UploadS3Expiry,
		oidc:  !reflect.DeepEqual(cur.OIDC, set.OIDC),
		other: !reflect.DeepEqual(withoutTenantCachedGroups(cur), withoutTenantCachedGroups(set)),
	}
}

// invalidateTenantCaches evicts the given tenant's cached SMTP messengers,
// media store, and/or OIDC config so the next request rebuilds them from
// this tenant's just-saved settings, in place of the full-process restart
// handleSettingsRestart still uses for changes outside these groups.
func (a *App) invalidateTenantCaches(tenantID int, c tenantCacheChanges) {
	if c.smtp {
		a.tenantMsgrs.Invalidate(tenantID)
	}
	if c.media {
		a.media.Invalidate(tenantID)
	}
	if c.oidc {
		a.auth.InvalidateOIDC(tenantID)
	}
}

// handleSettingsRestart checks for running campaigns and either triggers an
// immediate app restart or marks the app as needing a restart.
func (a *App) handleSettingsRestart(c echo.Context) error {
	// If there are any active campaigns, don't do an auto reload and
	// warn the user on the frontend.
	if a.manager.HasRunningCampaigns() {
		a.Lock()
		a.needsRestart = true
		a.Unlock()

		return c.JSON(http.StatusOK, okResp{struct {
			NeedsRestart bool `json:"needs_restart"`
		}{true}})
	}

	// No running campaigns. Reload the app.
	go func() {
		<-time.After(time.Millisecond * 500)
		a.chReload <- syscall.SIGHUP
	}()

	return c.JSON(http.StatusOK, okResp{true})
}

// GetLogs returns the log entries stored in the log buffer.
//
//	@ID				getLogs
//	@Summary		Get application logs
//	@Tags			settings
//	@Produce		json
//	@Success		200	{array}		string
//	@Router			/api/logs [get]
func (a *App) GetLogs(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{a.bufLog.Lines()})
}

// TestSMTPSettings tests an SMTP server connection by sending a test e-mail.
//
//	@ID				testSmtpSettings
//	@Summary		Test SMTP settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		interface{}		true	"SMTP config with target email"
//	@Success		200		{array}		string
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/settings/smtp/test [post]
func (a *App) TestSMTPSettings(c echo.Context) error {
	// Copy the raw JSON post body.
	reqBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading SMTP test: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	// Load the JSON into koanf to parse SMTP settings properly including timestrings.
	ko := koanf.New(".")
	if err := ko.Load(rawbytes.Provider(reqBody), koanfjson.Parser()); err != nil {
		a.log.Printf("error unmarshalling SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	req := email.Server{}
	if err := ko.UnmarshalWithConf("", &req, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		a.log.Printf("error scanning SMTP test request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	to := ko.String("email")
	if to == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.missingFields", "name", "email"))
	}

	// Initialize a new SMTP pool.
	req.MaxConns = 1
	req.IdleTimeout = time.Second * 2
	req.PoolWaitTimeout = time.Second * 2
	msgr, err := email.New("", req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorCreating", "name", "SMTP", "error", err.Error()))
	}

	// Render the test email template body.
	var b bytes.Buffer
	if err := notifs.Tpls.ExecuteTemplate(&b, "smtp-test", nil); err != nil {
		a.log.Printf("error compiling notification template '%s': %v", "smtp-test", err)
		return err
	}

	m := models.Message{}
	m.From = a.cfg.FromEmail
	m.To = []string{to}
	m.Subject = a.i18n.T("settings.smtp.testConnection")
	m.Body = b.Bytes()
	if err := msgr.Push(m); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{a.bufLog.Lines()})
}

// TestScrubSettings verifies that the Scrub URL and API key are reachable.
//
//	@ID				testScrubSettings
//	@Summary		Test Scrub integration settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			body	body		interface{}		true	"Scrub URL and API key"
//	@Success		200
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/settings/scrub/test [post]
func (a *App) TestScrubSettings(c echo.Context) error {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
	}

	req.URL = strings.TrimRight(strings.TrimSpace(req.URL), "/")
	if req.URL == "" || req.APIKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.missingFields", "name", "url / api_key"))
	}

	// Use the current stored key if the UI sent back a masked placeholder.
	if strings.Contains(req.APIKey, pwdMask) {
		s, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
		if err != nil {
			return err
		}
		req.APIKey = s.Scrub.APIKey
	}

	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, req.URL+"/usage/daily", nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorCreating", "name", "request", "error", err.Error()))
	}
	httpReq.Header.Set("X-API-Key", req.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.scrub.invalidKey"))
	}
	if resp.StatusCode >= 400 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", resp.Status))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetScrubListStatus proxies the Scrub integration lists endpoint, which returns
// each list along with its active job request_id and last validation result.
//
//	@ID			getScrubListStatus
//	@Summary	Get Scrub list validation status
//	@Tags		settings
//	@Produce	json
//	@Success	200	{object}	object
//	@Failure	400	{object}	echo.HTTPError
//	@Router		/api/lists/scrub [get]
func (a *App) GetScrubListStatus(c echo.Context) error {
	s, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	if !s.Scrub.Enabled || s.Scrub.URL == "" || s.Scrub.APIKey == "" || s.Scrub.IntegrationID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.scrub.notConfigured"))
	}

	scrubURL := strings.TrimRight(strings.TrimSpace(s.Scrub.URL), "/")
	apiURL := fmt.Sprintf("%s/v1/integrations/%s/lists", scrubURL, s.Scrub.IntegrationID)
	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("X-API-Key", s.Scrub.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return echo.NewHTTPError(http.StatusBadGateway,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", resp.Status))
	}

	var out interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// ScrubList triggers a Scrub email validation job on a subscriber list.
//
//	@ID			scrubList
//	@Summary	Trigger Scrub validation on a list
//	@Tags		settings
//	@Produce	json
//	@Param		id	path	int	true	"List ID"
//	@Success	200	{object}	object
//	@Failure	400	{object}	echo.HTTPError
//	@Router		/api/lists/{id}/scrub [post]
func (a *App) ScrubList(c echo.Context) error {
	id := getID(c)

	s, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	if !s.Scrub.Enabled || s.Scrub.URL == "" || s.Scrub.APIKey == "" || s.Scrub.IntegrationID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.scrub.notConfigured"))
	}

	scrubURL := strings.TrimRight(strings.TrimSpace(s.Scrub.URL), "/")
	apiURL := fmt.Sprintf("%s/v1/integrations/%s/lists/%d/validate", scrubURL, s.Scrub.IntegrationID, id)
	body := bytes.NewBufferString(`{"scan_mode":"full"}`)
	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, apiURL, body)
	if err != nil {
		return err
	}
	httpReq.Header.Set("X-API-Key", s.Scrub.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("settings.scrub.tooManyJobs"))
	}
	if resp.StatusCode >= 400 {
		return echo.NewHTTPError(http.StatusBadGateway,
			a.i18n.Ts("globals.messages.errorFetching", "name", "Scrub", "error", resp.Status))
	}

	var out interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Scrub gives no async signal back when a job actually starts (no
	// webhook/callback field on Settings.Scrub) -- this is the only hook
	// point, so pause inline right after a confirmed-successful trigger.
	pauseCampaignsForRiskySubscriberWith(c.Request().Context(), a.core, a.manager, tenantID(c), []int{id}, "scrub_job_started")

	return c.JSON(http.StatusOK, okResp{out})
}

// GetAboutInfo returns version, build, system, and host information about the app.
//
//	@ID				getAboutInfo
//	@Summary		Get application info
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	about
//	@Router			/api/about [get]
func (a *App) GetAboutInfo(c echo.Context) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	out := a.about
	out.System.AllocMB = mem.Alloc / 1024 / 1024
	out.System.OSMB = mem.Sys / 1024 / 1024

	return c.JSON(http.StatusOK, out)
}
