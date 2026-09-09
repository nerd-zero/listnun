package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/tmptokens"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const operatorSetupTokenTTL = 7 * 24 * time.Hour

var reTenantSlug = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// operatorQueries holds the prepared statements that run exclusively
// against the operator DB connection (see initOperatorDB) - a separate
// pool using a Postgres role with BYPASSRLS, distinct from the
// tenant-app pool. Never share these statements with the main *sqlx.DB.
type operatorQueries struct {
	CreateOrganization     *sqlx.Stmt `query:"operator-create-organization"`
	GetOrganization        *sqlx.Stmt `query:"operator-get-organization"`
	GetOrganizations       *sqlx.Stmt `query:"operator-get-organizations"`
	GetOrganizationTenants *sqlx.Stmt `query:"operator-get-organization-tenants"`
	DeleteOrganization     *sqlx.Stmt `query:"operator-delete-organization"`
	CreateTenant           *sqlx.Stmt `query:"operator-create-tenant"`
	SeedTenantSettings     *sqlx.Stmt `query:"operator-seed-tenant-settings"`
	SetTenantRootURL       *sqlx.Stmt `query:"operator-set-tenant-root-url"`
	SetTenantCustomDomain  *sqlx.Stmt `query:"operator-set-tenant-custom-domain"`
	SetTenantSMTP          *sqlx.Stmt `query:"operator-set-tenant-smtp"`
	SetTenantScrub         *sqlx.Stmt `query:"operator-set-tenant-scrub"`
	GetTenantScrubUsage    *sqlx.Stmt `query:"operator-get-tenant-scrub-usage"`
	GetTenant              *sqlx.Stmt `query:"operator-get-tenant"`
	GetTenants             *sqlx.Stmt `query:"operator-get-tenants"`
	UpdateTenantStatus     *sqlx.Stmt `query:"operator-update-tenant-status"`
	DeleteTenant           *sqlx.Stmt `query:"operator-delete-tenant"`
}

// operatorTenant is a tenant row augmented with cross-tenant counts, only
// obtainable via the BYPASSRLS operator connection.
type operatorTenant struct {
	models.Tenant
	UserCount       int `db:"user_count" json:"user_count"`
	SubscriberCount int `db:"subscriber_count" json:"subscriber_count"`
} // @name OperatorTenant

// operatorTenantScrubUsage is a tenant's subscriber validation status
// breakdown (subscribers.scrub_status), the same local data the
// tenant's own dashboard widget reads -- exposed cross-tenant here so
// the console can show per-instance usage without needing Scrub's
// account-scoped usage API (which has no per-integration breakdown).
type operatorTenantScrubUsage struct {
	Deliverable      int `db:"deliverable" json:"deliverable"`
	Undeliverable    int `db:"undeliverable" json:"undeliverable"`
	InvalidSyntax    int `db:"invalid_syntax" json:"invalid_syntax"`
	Risky            int `db:"risky" json:"risky"`
	UncheckedError   int `db:"unchecked_error" json:"unchecked_error"`
	Unchecked        int `db:"unchecked" json:"unchecked"`
	TotalSubscribers int `db:"total_subscribers" json:"total_subscribers"`
} // @name OperatorTenantScrubUsage

// operatorOrganization is an organization row augmented with a
// cross-tenant tenant count, only obtainable via the BYPASSRLS operator
// connection.
type operatorOrganization struct {
	models.Organization
	TenantCount int `db:"tenant_count" json:"tenant_count"`
} // @name OperatorOrganization

// operatorSetupToken is the payload stored against the one-time setup
// link token returned by CreateTenant (internal/tmptokens, in-memory,
// process-lifetime - same store already used for password-reset/2FA
// tokens, with the same accepted limitation that a restart invalidates
// pending links; see docs/design/multi-tenancy.md's Operator API
// section, "known gap, acceptable for v1").
type operatorSetupToken struct {
	TenantID int
	Email    string
}

// operatorStore bridges the operator DB connection and the normal
// tenant-app Core: tenant CRUD and cross-tenant counts go through the
// BYPASSRLS connection (operatorQueries below); creating the new
// tenant's initial role/user reuses Core.CreateRole/CreateUser on the
// normal pool via WithTenant(newTenantID, ...) - that's a same-tenant
// write using the newly-allocated tenant's own ID, so it needs no
// elevated privilege at all.
type operatorStore struct {
	db          *sqlx.DB
	q           *operatorQueries
	co          *core.Core
	permissions map[string]struct{}
}

// initOperatorDB connects to the same database as the main [db] config
// using a separate Postgres role with BYPASSRLS ([operator].db_user /
// db_password). Returns nil if the Operator API isn't configured
// (empty token or db_user) - it's off by default, an advanced feature
// most single-tenant installs will never enable.
func initOperatorDB(ko *koanf.Koanf) *sqlx.DB {
	if ko.String("operator.token") == "" || ko.String("operator.db_user") == "" {
		return nil
	}

	var c struct {
		Host    string `koanf:"host"`
		Port    int    `koanf:"port"`
		DBName  string `koanf:"database"`
		SSLMode string `koanf:"ssl_mode"`
		Params  string `koanf:"params"`
	}
	if err := ko.Unmarshal("db", &c); err != nil {
		lo.Fatalf("error loading db config for operator connection: %v", err)
	}

	fields := map[string]string{
		"host":     c.Host,
		"port":     strconv.Itoa(c.Port),
		"user":     ko.String("operator.db_user"),
		"password": ko.String("operator.db_password"),
		"dbname":   c.DBName,
		"sslmode":  c.SSLMode,
	}
	if c.Port == 0 {
		delete(fields, "port")
	}

	var parts []string
	for k, v := range fields {
		if v == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	if c.Params != "" {
		parts = append(parts, c.Params)
	}

	db, err := sqlx.Connect("postgres", strings.Join(parts, " "))
	if err != nil {
		lo.Fatalf("error connecting to operator DB: %v", err)
	}

	return db
}

// newOperatorStoreIfEnabled connects to the operator DB (if configured)
// and prepares its queries against the same parsed SQL map every other
// query set is prepared from (queries/operator.sql is loaded alongside
// the rest). Returns nil if the Operator API isn't configured.
func newOperatorStoreIfEnabled(qMap goyesql.Queries, co *core.Core, permissions map[string]struct{}) *operatorStore {
	db := initOperatorDB(ko)
	if db == nil {
		return nil
	}

	var q operatorQueries
	if err := goyesqlx.ScanToStruct(&q, qMap, db); err != nil {
		lo.Fatalf("error preparing operator SQL queries: %v", err)
	}

	lo.Println("operator API enabled")
	return &operatorStore{db: db, q: &q, co: co, permissions: permissions}
}

func (s *operatorStore) GetTenants() ([]operatorTenant, error) {
	out := []operatorTenant{}
	if err := s.q.GetTenants.Select(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *operatorStore) GetTenant(id int) (operatorTenant, error) {
	var out operatorTenant
	if err := s.q.GetTenant.Get(&out, id); err != nil {
		return out, err
	}
	return out, nil
}

func (s *operatorStore) GetTenantScrubUsage(id int) (operatorTenantScrubUsage, error) {
	var out operatorTenantScrubUsage
	if err := s.q.GetTenantScrubUsage.Get(&out, id); err != nil {
		return out, err
	}
	return out, nil
}

func (s *operatorStore) UpdateTenantStatus(id int, status string) (models.Tenant, error) {
	var out models.Tenant
	if err := s.q.UpdateTenantStatus.Get(&out, id, status); err != nil {
		return out, err
	}
	return out, nil
}

// SetTenantRootURL sets a tenant's app.root_url setting -- called both by
// CreateTenant (seeding the initial <slug>.root_domain value) and by
// SetTenantCustomDomain (keeping it in sync with tenants.custom_domain).
// Same upsert query either way, no RETURNING clause, so this method has no
// result to give back beyond the error.
func (s *operatorStore) SetTenantRootURL(id int, rootURL string) error {
	_, err := s.q.SetTenantRootURL.Exec(id, rootURL)
	return err
}

// SetTenantCustomDomain sets (customDomain != "") or clears (customDomain
// == "") tenants.custom_domain -- the exact Host internal/tenant.Middleware
// will now also accept for this tenant, alongside <slug>.root_domain -- and
// updates app.root_url to rootURL in the same transaction, so the two
// never drift apart (see UpdateOperatorTenantCustomDomain, the caller,
// for what rootURL should be in each case).
func (s *operatorStore) SetTenantCustomDomain(id int, customDomain, rootURL string) (models.Tenant, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return models.Tenant{}, err
	}
	defer tx.Rollback()

	var domain sql.NullString
	if customDomain != "" {
		domain = sql.NullString{String: customDomain, Valid: true}
	}

	var out models.Tenant
	if err := tx.Stmtx(s.q.SetTenantCustomDomain).Get(&out, id, domain); err != nil {
		return models.Tenant{}, err
	}
	if _, err := tx.Stmtx(s.q.SetTenantRootURL).Exec(id, rootURL); err != nil {
		return models.Tenant{}, err
	}

	return out, tx.Commit()
}

// DeleteTenant permanently deletes a tenant row. Every tenant-scoped table
// references tenants(id) ON DELETE CASCADE, so this cascades into deleting
// all of the tenant's subscribers, campaigns, users, settings, etc. along
// with it. Callers (DeleteOperatorTenant) are responsible for refusing to
// call this against tenant 1, the default tenant every install seeds data
// against.
func (s *operatorStore) DeleteTenant(id int) error {
	_, err := s.q.DeleteTenant.Exec(id)
	return err
}

func (s *operatorStore) CreateOrganization(name string) (models.Organization, error) {
	var out models.Organization
	if err := s.q.CreateOrganization.Get(&out, name); err != nil {
		return out, err
	}
	return out, nil
}

func (s *operatorStore) GetOrganizations() ([]operatorOrganization, error) {
	out := []operatorOrganization{}
	if err := s.q.GetOrganizations.Select(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *operatorStore) GetOrganization(id int) (operatorOrganization, error) {
	var out operatorOrganization
	if err := s.q.GetOrganization.Get(&out, id); err != nil {
		return out, err
	}
	return out, nil
}

// GetOrganizationTenants lists every tenant belonging to an organization.
func (s *operatorStore) GetOrganizationTenants(id int) ([]operatorTenant, error) {
	out := []operatorTenant{}
	if err := s.q.GetOrganizationTenants.Select(&out, id); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteOrganization removes an organization row. Tenants that belonged to
// it are not deleted - organization_id is ON DELETE SET NULL.
func (s *operatorStore) DeleteOrganization(id int) error {
	_, err := s.q.DeleteOrganization.Exec(id)
	return err
}

// CreateTenant creates a tenant row (optionally under an organization -
// organizationID <= 0 means none), then - via the normal tenant-app
// Core, not the operator connection - a "Super Admin" role with every
// permission and an initial admin user for it (PasswordLogin disabled,
// no password set yet). Returns the tenant and a one-time setup token
// the caller is responsible for delivering to the new admin (v1
// placeholder: returned directly in the API response - see the design
// doc's Operator API section for why e-mailing it isn't possible yet,
// a brand new tenant has no SMTP config of its own).
func (s *operatorStore) CreateTenant(ctx context.Context, slug, name, adminUsername, adminEmail, rootURL string, organizationID int) (models.Tenant, string, error) {
	var orgID sql.NullInt64
	if organizationID > 0 {
		orgID = sql.NullInt64{Int64: int64(organizationID), Valid: true}
	}

	var t models.Tenant
	if err := s.q.CreateTenant.Get(&t, slug, name, orgID); err != nil {
		return t, "", err
	}

	if _, err := s.q.SeedTenantSettings.Exec(t.ID); err != nil {
		return t, "", err
	}

	if rootURL != "" {
		if _, err := s.q.SetTenantRootURL.Exec(t.ID, rootURL); err != nil {
			return t, "", err
		}
	}

	r := auth.Role{
		Type: auth.RoleTypeUser,
		Name: null.NewString("Super Admin", true),
	}
	for p := range s.permissions {
		r.Permissions = append(r.Permissions, p)
	}

	newRole, err := s.co.CreateRole(ctx, t.ID, r)
	if err != nil {
		return t, "", err
	}

	u := auth.User{
		Type:          auth.UserTypeUser,
		HasPassword:   false,
		PasswordLogin: false,
		Username:      adminUsername,
		Name:          adminUsername,
		Email:         null.NewString(adminEmail, true),
		UserRoleID:    newRole.ID,
		Status:        auth.UserStatusEnabled,
	}
	if _, err := s.co.CreateUser(ctx, t.ID, u); err != nil {
		return t, "", err
	}

	token, err := generateRandomString(tmpAuthTokenLen)
	if err != nil {
		return t, "", err
	}
	tmptokens.Set(token, operatorSetupTokenTTL, operatorSetupToken{TenantID: t.ID, Email: adminEmail})

	return t, token, nil
}

// DeleteOperatorTenant permanently deletes a tenant and cascades into
// deleting all of its data (subscribers, campaigns, users, settings, etc.
// - every tenant-scoped table references tenants(id) ON DELETE CASCADE).
// Irreversible. Tenant 1, the default tenant every pre-multi-tenancy
// install and fresh schema.sql seeds data against, can never be deleted
// through this endpoint - doing so would wipe a single-tenant install's
// only data.
//
//	@ID			deleteOperatorTenant
//	@Summary		Delete a tenant (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. Permanently deletes the tenant and cascades into all of its data (subscribers, campaigns, users, settings, etc.) - irreversible. Tenant 1 (the default tenant) cannot be deleted through this endpoint.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Tenant ID"
//	@Success		200	{object}	okResp
//	@Failure		400	{object}	echo.HTTPError	"Cannot delete the default tenant (id=1)"
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/operator/tenants/{id} [delete]
func (a *App) DeleteOperatorTenant(c echo.Context) error {
	id := getID(c)
	if id == 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "the default tenant (id=1) cannot be deleted")
	}

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	if err := a.operator.DeleteTenant(id); err != nil {
		a.log.Printf("error deleting tenant: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting tenant")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// operatorSMTPEntry mirrors one entry of models.Settings' SMTP field
// (an anonymous struct there, so redeclared here for JSON marshaling).
// Used by SetTenantSMTP to accept a single SMTP server's config from an
// external provisioner (e.g. listnun, which owns the actual Postmark/
// SES/etc. API calls - this endpoint only ever writes whatever
// credentials it's given, listmonk has no provider-specific knowledge).
type operatorSMTPEntry struct {
	Name          string              `json:"name"`
	UUID          string              `json:"uuid"`
	Enabled       bool                `json:"enabled"`
	Host          string              `json:"host"`
	HelloHostname string              `json:"hello_hostname"`
	Port          int                 `json:"port"`
	AuthProtocol  string              `json:"auth_protocol"`
	Username      string              `json:"username"`
	Password      string              `json:"password"`
	EmailHeaders  []map[string]string `json:"email_headers"`
	MaxConns      int                 `json:"max_conns"`
	MaxMsgRetries int                 `json:"max_msg_retries"`
	MsgRetryDelay string              `json:"msg_retry_delay"`
	IdleTimeout   string              `json:"idle_timeout"`
	WaitTimeout   string              `json:"wait_timeout"`
	TLSType       string              `json:"tls_type"`
	TLSSkipVerify bool                `json:"tls_skip_verify"`
	FromAddresses []string            `json:"from_addresses"`
} // @name OperatorSMTPEntry

// SetTenantSMTP replaces a tenant's smtp setting (until now, tenant 1's
// placeholder example entries, copied in by
// operator-seed-tenant-settings) with a single real entry - the calling
// provisioner (e.g. listnun's create_postmark_server job) is responsible
// for actually creating the mail-provider server/credentials; this only
// writes whatever it's given into the tenant's settings.
func (s *operatorStore) SetTenantSMTP(tenantID int, entry operatorSMTPEntry) error {
	entry.UUID = uuid.Must(uuid.NewV4()).String()

	b, err := json.Marshal([]operatorSMTPEntry{entry})
	if err != nil {
		return err
	}

	_, err = s.q.SetTenantSMTP.Exec(tenantID, b)
	return err
}

// operatorScrubEntry mirrors models.Settings.Scrub for JSON marshaling
// from an external provisioner (listnun) that owns the actual Scrub
// integration -- this endpoint only ever writes whatever it's given,
// listmonk has no Scrub-provider knowledge of its own. ManagedByPlatform
// is always forced true by SetTenantScrub below, not taken from the
// caller, so a request body can't accidentally unlock a tenant.
type operatorScrubEntry struct {
	Enabled       bool   `json:"enabled"`
	URL           string `json:"url"`
	APIKey        string `json:"api_key"`
	IntegrationID string `json:"integration_id"`
} // @name OperatorScrubEntry

// SetTenantScrub replaces a tenant's scrub setting with a single real
// entry pushed by the calling provisioner (e.g. listnun's console-toggle
// action), always locking it (managed_by_platform: true) -- see
// models.Settings.Scrub's doc comment and internal/core.UpdateSettings/
// UpdateSettingsByKey, which refuse to let a tenant admin's own settings
// save change a locked row.
func (s *operatorStore) SetTenantScrub(tenantID int, entry operatorScrubEntry) error {
	b, err := json.Marshal(struct {
		operatorScrubEntry
		ManagedByPlatform bool `json:"managed_by_platform"`
	}{operatorScrubEntry: entry, ManagedByPlatform: true})
	if err != nil {
		return err
	}

	_, err = s.q.SetTenantScrub.Exec(tenantID, b)
	return err
}

// scrubIntegrationUsername is the reserved API username Scrub's own
// listnun/listmonk-provider integration authenticates as when it calls
// back into a tenant (GET /api/lists, /api/subscribers, blocklisting) --
// the reverse direction of SetTenantScrub above, which only pushes
// settings the other way (this tenant calling out to Scrub). Fixed
// rather than generated so CreateTenantScrubAPIUser can find (and
// rotate) any previous one by a simple username lookup.
const scrubIntegrationUsername = "scrub-integration"

// scrubIntegrationRoleName names the least-privilege role
// getOrCreateScrubIntegrationRole creates for scrubIntegrationUsername --
// just enough to read lists/subscribers and blocklist invalid ones, not
// the tenant's full Super Admin role (which would also hand over
// campaigns, users, and settings management to Scrub's API key).
const scrubIntegrationRoleName = "Scrub Integration"

var scrubIntegrationPermissions = []string{"lists:get_all", "subscribers:get_all", "subscribers:manage"}

// CreateTenantScrubAPIUser provisions the dedicated API user (and its
// supporting role, created on first use) that Scrub authenticates as
// when calling back into this tenant. Returns the username and a fresh
// one-time plaintext token -- see CreateUser's own doc comment on why
// the token can only ever be read at creation time.
//
// Create-or-rotate, not create-once: if scrubIntegrationUsername already
// exists (a previous call already ran, or the caller lost track of the
// token it was shown), the old user is deleted and a new one created
// rather than erroring. This is what listnun's periodic healing sweep
// (provisioning.HealScrubIntegrations) relies on to recover an
// integration whose credentials went bad for any reason -- there's no
// way to verify or recover an existing API user's plaintext token to
// confirm it still matches what Scrub has on file, so treating "already
// exists" as "needs a fresh one" is the only option that's actually
// self-healing.
func (s *operatorStore) CreateTenantScrubAPIUser(ctx context.Context, tenantID int) (string, string, error) {
	users, err := s.co.GetUsers(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	for _, u := range users {
		if u.Username == scrubIntegrationUsername {
			if err := s.co.DeleteUsers(ctx, tenantID, []int{u.ID}); err != nil {
				return "", "", err
			}
			break
		}
	}

	roleID, err := s.getOrCreateScrubIntegrationRole(ctx, tenantID)
	if err != nil {
		return "", "", err
	}

	out, err := s.co.CreateUser(ctx, tenantID, auth.User{
		Type:       auth.UserTypeAPI,
		Username:   scrubIntegrationUsername,
		Name:       scrubIntegrationRoleName,
		UserRoleID: roleID,
		Status:     auth.UserStatusEnabled,
	})
	if err != nil {
		return "", "", err
	}
	return out.Username, out.Password.String, nil
}

// getOrCreateScrubIntegrationRole returns the tenant's own
// scrubIntegrationRoleName role id, creating it on first use. Looked up
// by name rather than assumed to be auth.SuperAdminRoleID (id 1) --
// role ids come from one sequence shared across every tenant, so only
// tenant 1's own roles happen to land on the low numbers; see
// queries/users.sql's update-user query for the same reasoning applied
// to the Super Admin role.
func (s *operatorStore) getOrCreateScrubIntegrationRole(ctx context.Context, tenantID int) (int, error) {
	roles, err := s.co.GetRoles(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	for _, r := range roles {
		if r.Name.String == scrubIntegrationRoleName {
			return r.ID, nil
		}
	}

	newRole, err := s.co.CreateRole(ctx, tenantID, auth.Role{
		Type:        auth.RoleTypeUser,
		Name:        null.NewString(scrubIntegrationRoleName, true),
		Permissions: scrubIntegrationPermissions,
	})
	if err != nil {
		return 0, err
	}
	return newRole.ID, nil
}

// CreateSetupLink issues a fresh one-time setup token for an existing
// tenant admin, without creating a new tenant/user. Needed because
// internal/tmptokens is in-memory and process-lifetime only (see its own
// docs) - a setup link issued by CreateTenant is lost on every app
// restart, and there was previously no way to recover from that short of
// recreating the tenant from scratch. Verifies the user actually exists
// for this tenant/email first so this can't be used to probe for emails
// across tenants.
func (s *operatorStore) CreateSetupLink(ctx context.Context, tenantID int, email string) (string, error) {
	if _, err := s.co.GetUser(ctx, tenantID, 0, "", email); err != nil {
		return "", err
	}

	token, err := generateRandomString(tmpAuthTokenLen)
	if err != nil {
		return "", err
	}
	tmptokens.Set(token, operatorSetupTokenTTL, operatorSetupToken{TenantID: tenantID, Email: email})

	return token, nil
}

// operatorTenantReq is the request body for CreateTenant.
type operatorTenantReq struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	AdminUsername  string `json:"admin_username"`
	AdminEmail     string `json:"admin_email"`
	OrganizationID int    `json:"organization_id,omitempty"`
} // @name OperatorCreateTenantReq

// operatorCreateTenantResp is the response body for CreateOperatorTenant.
type operatorCreateTenantResp struct {
	Tenant     models.Tenant `json:"tenant"`
	SetupToken string        `json:"setup_token"`
	SetupURL   string        `json:"setup_url,omitempty"`
} // @name OperatorCreateTenantResp

// operatorSetupLinkReq is the request body for CreateOperatorSetupLink.
type operatorSetupLinkReq struct {
	AdminEmail string `json:"admin_email"`
} // @name OperatorCreateSetupLinkReq

// operatorSetupLinkResp is the response body for CreateOperatorSetupLink.
type operatorSetupLinkResp struct {
	SetupToken string `json:"setup_token"`
	SetupURL   string `json:"setup_url,omitempty"`
} // @name OperatorCreateSetupLinkResp

// operatorUpdateStatusReq is the request body for UpdateOperatorTenantStatus.
type operatorUpdateStatusReq struct {
	Status string `json:"status"`
} // @name OperatorUpdateTenantStatusReq

// operatorOrganizationReq is the request body for CreateOperatorOrganization.
type operatorOrganizationReq struct {
	Name string `json:"name"`
} // @name OperatorCreateOrganizationReq

// CreateOperatorOrganization creates an organization - a purely
// cross-tenant grouping construct that tenants can optionally belong to
// (see models.Organization).
//
//	@ID			createOperatorOrganization
//	@Summary		Create an organization (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. An organization is just a unique name that tenants can optionally be created under (via organization_id on POST /api/operator/tenants) - it groups multiple tenants ("listmonks") for the same customer under one umbrella.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			organization	body		operatorOrganizationReq	true	"Organization to create"
//	@Success		200	{object}	models.Organization
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		409	{object}	echo.HTTPError	"Name already in use"
//	@Router			/api/operator/organizations [post]
func (a *App) CreateOperatorOrganization(c echo.Context) error {
	var req operatorOrganizationReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	if !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid name")
	}

	out, err := a.operator.CreateOrganization(req.Name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "organizations_name_key" {
			return echo.NewHTTPError(http.StatusConflict, "an organization with this name already exists")
		}
		a.log.Printf("error creating organization: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating organization")
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ListOperatorOrganizations returns every organization with a
// cross-tenant tenant count.
//
//	@ID			listOperatorOrganizations
//	@Summary		List organizations (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	[]operatorOrganization
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/operator/organizations [get]
func (a *App) ListOperatorOrganizations(c echo.Context) error {
	out, err := a.operator.GetOrganizations()
	if err != nil {
		a.log.Printf("error fetching organizations: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching organizations")
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetOperatorOrganization returns a single organization (with a
// cross-tenant tenant count) plus the list of tenants belonging to it.
//
//	@ID			getOperatorOrganization
//	@Summary		Get an organization (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Organization ID"
//	@Success		200	{object}	operatorOrganizationResp
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/operator/organizations/{id} [get]
func (a *App) GetOperatorOrganization(c echo.Context) error {
	id := getID(c)

	org, err := a.operator.GetOrganization(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "organization not found")
	}

	tenants, err := a.operator.GetOrganizationTenants(id)
	if err != nil {
		a.log.Printf("error fetching organization tenants: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching organization tenants")
	}

	return c.JSON(http.StatusOK, okResp{operatorOrganizationResp{org, tenants}})
}

// operatorOrganizationResp is the response body for GetOperatorOrganization.
type operatorOrganizationResp struct {
	operatorOrganization
	Tenants []operatorTenant `json:"tenants"`
} // @name OperatorOrganizationResp

// DeleteOperatorOrganization deletes an organization row. Tenants that
// belonged to it are kept, just detached from it - organization_id is
// ON DELETE SET NULL (see docs/design/multi-tenancy.md's Organizations
// section).
//
//	@ID			deleteOperatorOrganization
//	@Summary		Delete an organization (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. Tenants belonging to this organization are not deleted - they're detached (organization_id set to NULL).
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Organization ID"
//	@Success		200	{object}	okResp
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/operator/organizations/{id} [delete]
func (a *App) DeleteOperatorOrganization(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetOrganization(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "organization not found")
	}

	if err := a.operator.DeleteOrganization(id); err != nil {
		a.log.Printf("error deleting organization: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting organization")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ListOperatorTenants returns every tenant with basic cross-tenant counts.
//
//	@ID			listOperatorTenants
//	@Summary		List tenants (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token, not a normal session/API-user token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	[]operatorTenant
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/operator/tenants [get]
func (a *App) ListOperatorTenants(c echo.Context) error {
	out, err := a.operator.GetTenants()
	if err != nil {
		a.log.Printf("error fetching tenants: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching tenants")
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetOperatorTenant returns a single tenant with basic cross-tenant counts.
//
//	@ID			getOperatorTenant
//	@Summary		Get a tenant (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token, not a normal session/API-user token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Tenant ID"
//	@Success		200	{object}	operatorTenant
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/operator/tenants/{id} [get]
func (a *App) GetOperatorTenant(c echo.Context) error {
	id := getID(c)
	out, err := a.operator.GetTenant(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetOperatorTenantScrubUsage returns a tenant's subscriber validation
// status breakdown -- the same subscribers.scrub_status data the
// tenant's own dashboard widget reads locally, exposed cross-tenant so
// a console can show per-instance usage. Scrub itself has no
// per-integration usage endpoint (only account-scoped /usage/* and raw
// paginated /history with no aggregate), so this is computed here
// instead of proxying to Scrub's API.
//
//	@ID			getOperatorTenantScrubUsage
//	@Summary		Get a tenant's Scrub validation usage (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Tenant ID"
//	@Success		200	{object}	operatorTenantScrubUsage
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Router			/api/operator/tenants/{id}/scrub/usage [get]
func (a *App) GetOperatorTenantScrubUsage(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	out, err := a.operator.GetTenantScrubUsage(id)
	if err != nil {
		a.log.Printf("error getting tenant scrub usage: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error getting tenant scrub usage")
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateOperatorTenant provisions a new tenant and its initial admin user.
//
//	@ID			createOperatorTenant
//	@Summary		Create a tenant (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token, not a normal session/API-user token. Creates the tenant plus a passwordless initial admin user; the returned setup_url/setup_token is a one-time link the tenant's actual admin uses to set their own password - the operator never sets or sees it. If [operator].postmark_account_token is set, also auto-provisions a dedicated Postmark server and wires its SMTP credentials into the new tenant's settings.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tenant	body		operatorTenantReq	true	"Tenant to create"
//	@Success		200	{object}	operatorCreateTenantResp
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		409	{object}	echo.HTTPError	"Slug already in use"
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/operator/tenants [post]
func (a *App) CreateOperatorTenant(c echo.Context) error {
	var req operatorTenantReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if !reTenantSlug.MatchString(req.Slug) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slug: use lowercase letters, numbers, hyphens")
	}
	if !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid name")
	}
	if !strHasLen(req.AdminUsername, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid admin_username")
	}
	if !utils.ValidateEmail(req.AdminEmail) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid admin_email")
	}
	if req.OrganizationID > 0 {
		if _, err := a.operator.GetOrganization(req.OrganizationID); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "organization not found")
		}
	}

	tenant, token, err := a.operator.CreateTenant(c.Request().Context(), req.Slug, req.Name, req.AdminUsername, req.AdminEmail, a.tenantRootURL(req.Slug), req.OrganizationID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "tenants_slug_key" {
			return echo.NewHTTPError(http.StatusConflict, "a tenant with this slug already exists")
		}
		a.log.Printf("error creating tenant: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating tenant")
	}

	return c.JSON(http.StatusOK, okResp{operatorCreateTenantResp{tenant, token, a.operatorSetupURL(tenant.Slug, tenant.CustomDomain, token)}})
}

// operatorSetupURL builds the one-time setup link for a tenant's admin,
// shared by CreateOperatorTenant and CreateOperatorSetupLink. Prefers the
// tenant's custom domain over its <slug>.root_domain subdomain once one is
// set - CreateOperatorSetupLink's whole reason to exist is reissuing a
// link after the original's token was lost (e.g. an app restart), and a
// tenant that's set up a custom domain since then would otherwise keep
// getting sent a link for a subdomain that no longer matches
// app.root_url/their OIDC redirect_uri, even though internal/tenant's
// custom-domain resolution path already accepts the subdomain too (see
// docs/design/multi-tenancy.md's Organizations section) - it's just not
// the address they've told their team/DNS to expect anymore. Empty if
// neither a custom domain nor app.root_domain is configured (or
// app.root_url has no scheme).
func (a *App) operatorSetupURL(tenantSlug string, customDomain null.String, token string) string {
	if customDomain.Valid && customDomain.String != "" {
		return "https://" + customDomain.String + "/admin/operator-setup?token=" + token
	}
	root := a.tenantRootURL(tenantSlug)
	if root == "" {
		return ""
	}
	return root + "/admin/operator-setup?token=" + token
}

// tenantRootURL computes a new tenant's own root URL from its slug and
// app.root_domain - used both for the operator setup link and to seed
// the new tenant's own app.root_url setting (CreateTenant must NOT
// blindly copy tenant 1's app.root_url the way it does every other
// setting: found live when a copied-verbatim tenant-1 URL showed up in
// another tenant's Settings page, and would have silently broken that
// tenant's OIDC redirect_uri too, since internal/auth reads
// settings.AppRootURL directly rather than deriving it from the
// request like cmd/public.go's tplRenderer does). Returns "" if
// app.root_domain isn't configured or app.root_url has no scheme.
func (a *App) tenantRootURL(tenantSlug string) string {
	if a.cfg.RootDomain == "" {
		return ""
	}
	u, err := url.Parse(a.urlCfg.RootURL)
	if err != nil || u.Scheme == "" {
		return ""
	}
	return u.Scheme + "://" + tenantSlug + "." + a.cfg.RootDomain
}

// CreateOperatorSetupLink issues a fresh one-time setup link for an
// existing tenant's admin. Needed because setup tokens live only in
// internal/tmptokens' in-memory store - restarting the app invalidates
// every pending link issued by CreateOperatorTenant, and until this
// endpoint existed the only recovery was recreating the tenant from
// scratch (which fails outright since the slug is already taken).
//
//	@ID			createOperatorSetupLink
//	@Summary		Reissue a tenant admin's setup link (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. Use when a prior setup_url from CreateOperatorTenant expired or was lost - e.g. every pending link is invalidated on app restart, since setup tokens are held in memory only.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"Tenant ID"
//	@Param			email	body		operatorSetupLinkReq	true	"Existing admin's email"
//	@Success		200	{object}	operatorSetupLinkResp
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant or admin email not found"
//	@Router			/api/operator/tenants/{id}/setup-link [post]
func (a *App) CreateOperatorSetupLink(c echo.Context) error {
	id := getID(c)

	var req operatorSetupLinkReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if !utils.ValidateEmail(req.AdminEmail) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid admin_email")
	}

	tenant, err := a.operator.GetTenant(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	token, err := a.operator.CreateSetupLink(c.Request().Context(), id, req.AdminEmail)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "no admin with that email found for this tenant")
	}

	return c.JSON(http.StatusOK, okResp{operatorSetupLinkResp{token, a.operatorSetupURL(tenant.Slug, tenant.CustomDomain, token)}})
}

// UpdateOperatorTenantStatus updates a tenant's status (active/suspended/disabled).
//
//	@ID			updateOperatorTenantStatus
//	@Summary		Update a tenant's status (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token, not a normal session/API-user token.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"Tenant ID"
//	@Param			status	body		operatorUpdateStatusReq	true	"New status"
//	@Success		200	{object}	models.Tenant
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/operator/tenants/{id}/status [put]
func (a *App) UpdateOperatorTenantStatus(c echo.Context) error {
	id := getID(c)

	var req operatorUpdateStatusReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Status != "active" && req.Status != "suspended" && req.Status != "disabled" {
		return echo.NewHTTPError(http.StatusBadRequest, "status must be one of active, suspended, disabled")
	}

	out, err := a.operator.UpdateTenantStatus(id, req.Status)
	if err != nil {
		a.log.Printf("error updating tenant status: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error updating tenant status")
	}

	return c.JSON(http.StatusOK, okResp{out})
}

type operatorSetCustomDomainReq struct {
	// CustomDomain is the bare host (e.g. "mail.acme.com"), no scheme --
	// empty clears it, reverting the tenant to <slug>.root_domain only.
	CustomDomain string `json:"custom_domain"`
}

// UpdateOperatorTenantCustomDomain godoc
//
//	@ID			updateOperatorTenantCustomDomain
//	@Summary		Set or clear a tenant's custom domain (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. Sets tenants.custom_domain -- the additional exact Host internal/tenant.Middleware will accept for this tenant, alongside <slug>.root_domain -- and updates app.root_url to match in the same transaction. Pass an empty custom_domain to clear it and revert root_url to <slug>.root_domain. The caller is responsible for having already verified domain ownership (e.g. via a Cloudflare Custom Hostname's DCV) before calling this; listmonk does no verification of its own.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id				path	int							true	"Tenant ID"
//	@Param			custom_domain	body	operatorSetCustomDomainReq	true	"Custom domain to set, or empty to clear"
//	@Success		200	{object}	okResp
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Failure		409	{object}	echo.HTTPError	"This custom domain is already in use by another tenant"
//	@Router			/api/operator/tenants/{id}/custom-domain [put]
func (a *App) UpdateOperatorTenantCustomDomain(c echo.Context) error {
	id := getID(c)

	t, err := a.operator.GetTenant(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	var req operatorSetCustomDomainReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	rootURL := a.tenantRootURL(t.Slug)
	if req.CustomDomain != "" {
		if strings.ContainsAny(req.CustomDomain, "/:") {
			return echo.NewHTTPError(http.StatusBadRequest, "custom_domain must be a bare host, no scheme or path")
		}
		rootURL = "https://" + req.CustomDomain
	}

	out, err := a.operator.SetTenantCustomDomain(id, req.CustomDomain, rootURL)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "tenants_custom_domain_key" {
			return echo.NewHTTPError(http.StatusConflict, "this custom domain is already in use by another tenant")
		}
		a.log.Printf("error updating tenant custom domain: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error updating tenant custom domain")
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// SetOperatorTenantSMTP replaces a tenant's SMTP settings with a single
// server entry.
//
//	@ID			setOperatorTenantSmtp
//	@Summary		Set a tenant's SMTP settings (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. listmonk has no mail-provider-specific logic - the caller (e.g. an external provisioner that owns the actual Postmark/SES/etc. API calls) is responsible for creating the server/credentials and passes them here as-is; this endpoint only writes them into the tenant's settings, replacing its placeholder SMTP examples.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Tenant ID"
//	@Param			smtp	body		operatorSMTPEntry	true	"SMTP server config"
//	@Success		200	{object}	okResp
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Router			/api/operator/tenants/{id}/smtp [put]
func (a *App) SetOperatorTenantSMTP(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	var req operatorSMTPEntry
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Host == "" || req.Username == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "host and username are required")
	}

	if err := a.operator.SetTenantSMTP(id, req); err != nil {
		a.log.Printf("error setting tenant SMTP: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error setting tenant SMTP")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// SetOperatorTenantScrub replaces a tenant's scrub setting with a
// platform-pushed, locked entry -- see operatorStore.SetTenantScrub.
func (a *App) SetOperatorTenantScrub(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	var req operatorScrubEntry
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Enabled && (req.URL == "" || req.APIKey == "" || req.IntegrationID == "") {
		return echo.NewHTTPError(http.StatusBadRequest, "url, api_key, and integration_id are required to enable")
	}

	if err := a.operator.SetTenantScrub(id, req); err != nil {
		a.log.Printf("error setting tenant scrub: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error setting tenant scrub")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// operatorScrubAPIUserResp is the response body for
// CreateOperatorTenantScrubAPIUser -- the token is shown exactly once
// here, same as the regular admin UI's own "create API user" flow.
type operatorScrubAPIUserResp struct {
	Username string `json:"username"`
	APIToken string `json:"api_token"`
} // @name OperatorScrubAPIUserResp

// CreateOperatorTenantScrubAPIUser provisions (or rotates, if one
// already exists) the dedicated API user Scrub's own
// listnun/listmonk-provider integration authenticates as when calling
// back into this tenant -- see operatorStore.CreateTenantScrubAPIUser,
// including why this rotates rather than erroring on a repeat call. The
// caller (listnun) is responsible for immediately pushing the returned
// credentials into Scrub's integration config; there is no way to
// retrieve the token again afterward.
//
//	@ID			createOperatorTenantScrubAPIUser
//	@Summary		Create or rotate the Scrub integration API user for a tenant (Operator API)
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Tenant ID"
//	@Success		200	{object}	operatorScrubAPIUserResp
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Router			/api/operator/tenants/{id}/scrub/api-user [post]
func (a *App) CreateOperatorTenantScrubAPIUser(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	username, token, err := a.operator.CreateTenantScrubAPIUser(c.Request().Context(), id)
	if err != nil {
		a.log.Printf("error creating tenant scrub API user: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating scrub API user")
	}

	return c.JSON(http.StatusOK, okResp{operatorScrubAPIUserResp{Username: username, APIToken: token}})
}

// OperatorSetupPage renders (GET) and processes (POST) the one-time
// setup link an operator-provisioned tenant's initial admin uses to set
// their password. Public - authorized entirely by possessing the
// one-time token, same trust model as the existing password-reset flow.
func (a *App) OperatorSetupPage(c echo.Context) error {
	token := strings.TrimSpace(c.QueryParam("token"))
	if c.Request().Method == http.MethodPost {
		token = strings.TrimSpace(c.FormValue("token"))
	}

	data, err := tmptokens.Check(token)
	if err != nil {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}
	payload, ok := data.(operatorSetupToken)
	if !ok {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	if c.Request().Method == http.MethodPost {
		return a.doOperatorSetup(c, token, payload)
	}

	return c.Render(http.StatusOK, "admin-operator-setup", resetPasswordTpl{
		Title: a.i18n.T("users.resetPassword"),
		Token: token,
		Email: payload.Email,
	})
}

func (a *App) doOperatorSetup(c echo.Context, token string, payload operatorSetupToken) error {
	var (
		password  = c.FormValue("password")
		password2 = c.FormValue("password2")
	)

	if !strHasLen(password, 8, stdInputMaxLen) {
		return c.Render(http.StatusBadRequest, "admin-operator-setup", resetPasswordTpl{
			Title: a.i18n.T("users.resetPassword"), Token: token, Email: payload.Email,
			Error: a.i18n.Ts("globals.messages.invalidFields", "name", "password"),
		})
	}
	if password != password2 {
		return c.Render(http.StatusBadRequest, "admin-operator-setup", resetPasswordTpl{
			Title: a.i18n.T("users.resetPassword"), Token: token, Email: payload.Email,
			Error: a.i18n.T("users.passwordMismatch"),
		})
	}

	// Consume the token so it can't be reused.
	if _, err := tmptokens.Get(token); err != nil {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	user, err := a.core.GetUser(c.Request().Context(), payload.TenantID, 0, "", payload.Email)
	if err != nil {
		return c.Render(http.StatusBadRequest, tplMessage, makeMsgTpl(a.i18n.T("users.resetPassword"), "", a.i18n.T("users.invalidResetLink")))
	}

	// UpdateUserProfile (not UpdateUser) - same as the analogous
	// forgot-password flow (cmd/auth.go's doResetPassword). UpdateUser is
	// for admin-editing-another-user and re-runs the last-Super-Admin
	// guard against role_id/status; GetUser zeroes UserRoleID after
	// copying it into UserRole.ID (setupUserFields, for JSON shaping), so
	// passing it straight back to UpdateUser sent role_id=0 - the query's
	// "unchanged" sentinel for a normal edit, but one the guard's OR
	// condition didn't recognize as "unchanged," so it always treated a
	// lone tenant admin's own password-setup as "reassigning away from
	// Super Admin" and rejected it. UpdateUserProfile only ever touches
	// name/email/password, sidestepping that guard entirely - correct,
	// since setting your own password isn't a role/status change.
	user.Password = null.NewString(password, true)
	user.PasswordLogin = true
	if _, err := a.core.UpdateUserProfile(c.Request().Context(), payload.TenantID, user.ID, user); err != nil {
		a.log.Printf("error completing operator setup for user_id=%d: %v", user.ID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Log the user in directly and land on the dashboard, rather than a
	// dead-end "Done" message with no way to continue - same UX as the
	// analogous forgot-password flow (cmd/auth.go's doResetPassword).
	if err := a.auth.SaveSession(user, "", c); err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, uriAdmin)
}
