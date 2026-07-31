package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/zerodha/simplesessions/stores/postgres/v3"
	"github.com/zerodha/simplesessions/v3"
	"golang.org/x/oauth2"
)

type OIDCclaim struct {
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Sub               string `json:"sub"`
	Picture           string `json:"picture"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

type OIDCConfig struct {
	Enabled           bool   `json:"enabled"`
	ProviderURL       string `json:"provider_url"`
	RedirectURL       string `json:"redirect_url"`
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	AutoCreateUsers   bool   `json:"auto_create_users"`
	DefaultUserRoleID int    `json:"default_user_role_id"`
	DefaultListRoleID int    `json:"default_list_role_id"`
}

type BasicAuthConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	BasicAuth BasicAuthConfig
}

// Callbacks takes two callback functions required by simplesessions.
type Callbacks struct {
	SetCookie func(cookie *http.Cookie, w any) error
	GetCookie func(name string, r any) (*http.Cookie, error)
	GetUser   func(id int) (User, error)

	// GetOIDCConfig returns the given tenant's OIDC config (settings are
	// per-tenant since phase 5). Called lazily and cached per tenant - see
	// tenantOIDC/initOIDC. RedirectURL must already be resolved against
	// that tenant's own root URL by the callback's implementation; this
	// package has no notion of tenant settings beyond what's returned here.
	GetOIDCConfig func(tenantID int) (OIDCConfig, error)
}

// tenantOIDC holds one tenant's resolved OIDC provider/verifier/OAuth
// config, built lazily on first use and cached until evicted by
// InvalidateOIDC (called when that tenant saves OIDC settings - see
// cmd/settings.go's handleSettingsRestart) or the process restarts.
type tenantOIDC struct {
	cfg      OIDCConfig
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauthCfg oauth2.Config
}

type Auth struct {
	apiUsers map[string]User
	sync.RWMutex

	cfg       Config
	oidcCache map[int]*tenantOIDC
	sess      *simplesessions.Manager
	sessStore *postgres.Store
	cb        *Callbacks
	log       *log.Logger
}

var sessPruneInterval = time.Hour * 12

// New returns an initialize Auth instance.
func New(cfg Config, db *sql.DB, cb *Callbacks, lo *log.Logger) (*Auth, error) {
	a := &Auth{
		cfg: cfg,
		cb:  cb,
		log: lo,

		apiUsers:  map[string]User{},
		oidcCache: map[int]*tenantOIDC{},
	}

	// Initialize session manager.
	a.sess = simplesessions.New(simplesessions.Options{
		EnableAutoCreate: false,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			IsHTTPOnly: true,
			MaxAge:     time.Hour * 24 * 7,
		},
	})
	st, err := postgres.New(postgres.Opt{}, db)
	if err != nil {
		return nil, err
	}
	a.sessStore = st
	a.sess.UseStore(st)
	a.sess.SetCookieHooks(cb.GetCookie, cb.SetCookie)

	// Prune dead sessions from the DB periodically.
	go func() {
		if err := st.Prune(); err != nil {
			lo.Printf("error pruning login sessions: %v", err)
		}
		time.Sleep(sessPruneInterval)
	}()

	return a, nil
}

// CacheAPIUsers caches API users for authenticating requests. It wipes
// the existing cache every time and is meant for syncing all API users
// in the database in one shot.
func (o *Auth) CacheAPIUsers(users []User) {
	o.Lock()
	defer o.Unlock()

	o.apiUsers = map[string]User{}
	for _, u := range users {
		o.apiUsers[u.Username] = u
	}
}

// CacheAPIUser caches an API user for authenticating requests.
func (o *Auth) CacheAPIUser(u User) {
	o.Lock()
	o.apiUsers[u.Username] = u
	o.Unlock()
}

// GetAPIToken validates an API user+token.
func (o *Auth) GetAPIToken(user string, token string) (User, bool) {
	o.RLock()
	t, ok := o.apiUsers[user]
	o.RUnlock()

	if !ok || subtle.ConstantTimeCompare([]byte(t.Password.String), []byte(token)) != 1 {
		return User{}, false
	}

	return t, true
}

// initOIDC fetches the given tenant's OIDC config via the GetOIDCConfig
// callback and builds+caches its provider, verifier, and OAuth config.
// Must be called with o's lock held.
func (o *Auth) initOIDC(tenantID int) (*tenantOIDC, error) {
	if o.cb.GetOIDCConfig == nil {
		return nil, fmt.Errorf("OIDC is not configured")
	}

	cfg, err := o.cb.GetOIDCConfig(tenantID)
	if err != nil {
		return nil, fmt.Errorf("error fetching OIDC config: %v", err)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled")
	}

	provider, err := oidc.NewProvider(context.Background(), cfg.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("error initializing OIDC OAuth provider: %v", err)
	}

	t := &tenantOIDC{
		cfg:      cfg,
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{
			ClientID: cfg.ClientID,
		}),
		oauthCfg: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}
	o.oidcCache[tenantID] = t

	return t, nil
}

// getTenantOIDC returns the given tenant's cached OIDC state, initializing
// it if necessary.
func (o *Auth) getTenantOIDC(tenantID int) (*tenantOIDC, error) {
	o.Lock()
	defer o.Unlock()

	if t, ok := o.oidcCache[tenantID]; ok {
		return t, nil
	}
	return o.initOIDC(tenantID)
}

// InvalidateOIDC evicts the given tenant's cached OIDC provider/verifier/
// OAuth config, forcing it to be rebuilt (including a fresh IdP metadata
// discovery) on next use.
func (o *Auth) InvalidateOIDC(tenantID int) {
	o.Lock()
	delete(o.oidcCache, tenantID)
	o.Unlock()
}

// GetOIDCAuthURL returns the given tenant's OIDC provider auth URL to redirect to.
func (o *Auth) GetOIDCAuthURL(tenantID int, state, nonce string) string {
	t, err := o.getTenantOIDC(tenantID)
	if err != nil {
		o.log.Printf("error getting OAuth config: %v", err)
		return ""
	}
	return t.oauthCfg.AuthCodeURL(state, oidc.Nonce(nonce))
}

// ExchangeOIDCToken takes an OIDC authorization code (recieved via redirect from the OIDC provider),
// validates it, and returns an OIDC token for subsequent auth.
func (o *Auth) ExchangeOIDCToken(tenantID int, code, nonce string) (string, OIDCclaim, error) {
	t, err := o.getTenantOIDC(tenantID)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error getting OAuth config: %v", err))
	}

	tk, err := t.oauthCfg.Exchange(context.TODO(), code)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error exchanging token: %v", err))
	}

	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, "`id_token` missing.")
	}

	idTk, err := t.verifier.Verify(context.TODO(), rawIDTk)
	if err != nil {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("error verifying ID token: %v", err))
	}

	if idTk.Nonce != nonce {
		return "", OIDCclaim{}, echo.NewHTTPError(http.StatusUnauthorized, "nonce did not match")
	}

	var claims OIDCclaim
	if err := idTk.Claims(&claims); err != nil {
		return "", OIDCclaim{}, errors.New("error getting user from OIDC")
	}

	// If claims doesn't have the e-mail, attempt to fetch it from the userinfo endpoint.
	if claims.Email == "" {
		userInfo, err := t.provider.UserInfo(context.TODO(), oauth2.StaticTokenSource(tk))
		if err != nil {
			return "", OIDCclaim{}, errors.New("error fetching user info from OIDC")
		}

		// Parse the UserInfo claims into the claims struct
		if err := userInfo.Claims(&claims); err != nil {
			return "", OIDCclaim{}, errors.New("error parsing user info claims")
		}
	}

	return rawIDTk, claims, nil
}

// Middleware is the HTTP middleware used for wrapping HTTP handlers registered on the echo router.
// It authorizes token (BasicAuth/token) based and cookie based sessions and on successful auth,
// sets the authenticated User{} on the echo context on the key UserKey. On failure, it sets an Error{}
// instead on the same key.
func (o *Auth) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// It's an `Authorization` header request.
		hdr := strings.TrimSpace(c.Request().Header.Get("Authorization"))

		// If cookie is set, ignore BasicAuth. This is to preserve backwards compatibility
		// in v3 -> v4 upgrade where the user browser sessions would still have old
		// BasicAuth credentials, which no longer work in the new system which expects
		// session cookies instead, which causes a redirect loop despite loggin in and session
		// cookies being set.
		//
		// TODO: This should be removed in a future version.
		if c := strings.TrimSpace(c.Request().Header.Get("Cookie")); strings.Contains(c, "session=") {
			hdr = ""
		}

		if len(hdr) > 0 {
			key, token, err := parseAuthHeader(hdr)
			if err != nil {
				c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, err.Error()))
				return next(c)
			}

			// Validate the token.
			user, ok := o.GetAPIToken(key, token)
			if !ok {
				c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid API credentials"))
				return next(c)
			}
			if tenantMismatch(c, user) {
				c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
				return next(c)
			}

			// Set the user details on the handler context.
			c.Set(UserHTTPCtxKey, user)
			return next(c)
		}

		// Is it a cookie based session?
		sess, user, err := o.validateSession(c)
		if err != nil {
			c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}
		if tenantMismatch(c, user) {
			c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// Set the user details on the handler context.
		c.Set(UserHTTPCtxKey, user)
		c.Set(SessionKey, sess)
		return next(c)
	}
}

// tenantMismatch reports whether the resolved tenant (set by
// internal/tenant's middleware, which runs before this one) doesn't match
// the given user's own tenant - defense in depth against a session/token
// issued for one tenant being replayed against a different tenant's
// subdomain. Returns false (no mismatch) if no tenant was resolved onto
// the context at all, so this is a no-op wherever the tenant middleware
// isn't wired in (e.g. in tests that construct Auth directly).
func tenantMismatch(c echo.Context, user User) bool {
	t, ok := c.Get(models.TenantCtxKey).(*models.Tenant)
	return ok && user.TenantID != t.ID
}

// Perm is an HTTP handler middleware that checks if the authenticated user has the required permissions.
func (o *Auth) Perm(next echo.HandlerFunc, perms ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		u, ok := c.Get(UserHTTPCtxKey).(User)
		if !ok {
			c.Set(UserHTTPCtxKey, echo.NewHTTPError(http.StatusForbidden, "invalid session"))
			return next(c)
		}

		// If the current user is a Super Admin user, do no checks.
		if u.UserRole.ID == SuperAdminRoleID {
			return next(c)
		}

		// Check if the current handler's permission is in the user's permission map.
		var (
			has  = false
			perm = ""
		)
		for _, perm = range perms {
			if _, ok := u.PermissionsMap[perm]; ok {
				has = true
				break
			}
		}

		if !has {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("permission denied: %s", perm))
		}

		return next(c)
	}
}

// SaveSession creates and sets a session (post successful login/auth).
func (o *Auth) SaveSession(u User, oidcToken string, c echo.Context) error {
	sess, err := o.sess.NewSession(c, c)
	if err != nil {
		o.log.Printf("error creating login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	if err := sess.SetMulti(map[string]any{"user_id": u.ID, "oidc_token": oidcToken}); err != nil {
		o.log.Printf("error setting login session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating session")
	}

	return nil
}

// GetSessionID returns the current session ID from the echo context.
func GetSessionID(c echo.Context) string {
	sess, ok := c.Get(SessionKey).(*simplesessions.Session)
	if !ok || sess == nil {
		return ""
	}

	return sess.ID()
}

// validateSession checks if the cookie session is valid (in the DB) and returns the session and user details.
func (o *Auth) validateSession(c echo.Context) (*simplesessions.Session, User, error) {
	// Cookie session.
	sess, err := o.sess.Acquire(context.TODO(), c, c)
	if err != nil {
		return nil, User{}, echo.NewHTTPError(http.StatusForbidden, err.Error())
	}

	// Get the session variables.
	vars, err := sess.GetMulti("user_id", "oidc_token")
	if err != nil {
		return nil, User{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Validate the user ID in the session.
	userID, err := o.sessStore.Int(vars["user_id"], nil)
	if err != nil || userID < 1 {
		o.log.Printf("error fetching session user ID: %v", err)
		return nil, User{}, echo.NewHTTPError(http.StatusInternalServerError, "invalid session.")
	}

	// Fetch user details from the database.
	user, err := o.cb.GetUser(userID)
	if err != nil {
		o.log.Printf("error fetching session user: %v", err)
	}

	return sess, user, err
}

// GetUser retrieves and returns the User object from an authenticated
// HTTP handler request.
func GetUser(c echo.Context) User {
	return c.Get(UserHTTPCtxKey).(User)
}

// parseAuthHeader parses the Authorization header and returns the api_key and access_token.
func parseAuthHeader(h string) (string, string, error) {
	const authBasic = "Basic"
	const authToken = "token"

	var (
		pair  []string
		delim = ":"
	)

	if strings.HasPrefix(h, authToken) {
		// token api_key:access_token.
		pair = strings.SplitN(strings.Trim(h[len(authToken):], " "), delim, 2)
	} else if strings.HasPrefix(h, authBasic) {
		// HTTP BasicAuth. This is supported for backwards compatibility.
		payload, err := base64.StdEncoding.DecodeString(string(strings.Trim(h[len(authBasic):], " ")))
		if err != nil {
			return "", "", echo.NewHTTPError(http.StatusBadRequest, "invalid Base64 value in Basic Authorization header")
		}
		pair = strings.SplitN(string(payload), delim, 2)
	} else {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "unknown Authorization scheme")
	}

	if len(pair) < 2 {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "api_key:token missing")
	}

	if len(pair[0]) == 0 || len(pair[1]) == 0 {
		return "", "", echo.NewHTTPError(http.StatusBadRequest, "empty `api_key` or `token`")
	}

	return pair[0], pair[1], nil
}
