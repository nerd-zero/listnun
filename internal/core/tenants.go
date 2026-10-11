package core

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// GetTenantBySlug returns the tenant with the given slug. It is a plain,
// unscoped lookup - the `tenants` table itself carries no RLS policy
// (phase 2, v6.5.0) since it's the table that identifies tenants, not one
// scoped by tenant_id.
func (c *Core) GetTenantBySlug(slug string) (models.Tenant, error) {
	out := []models.Tenant{}
	if err := c.q.GetTenantBySlug.Select(&out, slug); err != nil {
		c.log.Printf("error fetching tenant: %v", err)
		return models.Tenant{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "tenant", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Tenant{}, ErrNotFound
	}

	return out[0], nil
}

// GetTenantByCustomDomain returns the tenant whose custom_domain exactly
// matches the given host (case-insensitively - see the query's own
// comment). Same unscoped-lookup reasoning as GetTenantBySlug: tenants
// itself has no RLS policy to bypass.
func (c *Core) GetTenantByCustomDomain(host string) (models.Tenant, error) {
	out := []models.Tenant{}
	if err := c.q.GetTenantByCustomDomain.Select(&out, host); err != nil {
		c.log.Printf("error fetching tenant by custom domain: %v", err)
		return models.Tenant{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "tenant", "error", pqErrMsg(err)))
	}

	if len(out) == 0 {
		return models.Tenant{}, ErrNotFound
	}

	return out[0], nil
}

// GetActiveTenantIDs returns the IDs of all active tenants. Reused by
// boot-time per-tenant loops (e.g. initTxTemplates) - the same underlying
// query internal/manager's scanCampaigns (phase 6) uses via cmd/manager_store.go.
func (c *Core) GetActiveTenantIDs() ([]int, error) {
	var out []int
	if err := c.q.GetActiveTenantIDs.Select(&out); err != nil {
		c.log.Printf("error fetching active tenant IDs: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "tenant", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetTenantOrganizationName returns the name of the organization the given
// tenant belongs to, or "" if it doesn't belong to one. Like
// GetTenantBySlug, this is a plain, unscoped lookup - organizations carry
// no RLS policy either (they're a purely cross-tenant, operator-managed
// concept - see docs/design/multi-tenancy.md's "Organizations" section),
// so this doesn't need WithTenant.
func (c *Core) GetTenantOrganizationName(tenantID int) (string, error) {
	var out sql.NullString
	if err := c.q.GetTenantOrganizationName.Get(&out, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		c.log.Printf("error fetching tenant organization name: %v", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "organization", "error", pqErrMsg(err)))
	}
	return out.String, nil
}
