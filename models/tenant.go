package models

import null "gopkg.in/volatiletech/null.v6"

// TenantCtxKey is the echo.Context key the tenant-resolution middleware
// (internal/tenant) stores the resolved *Tenant under, and internal/auth
// reads to cross-check against the authenticated user's TenantID. It lives
// here, not in internal/tenant itself, because internal/core already
// imports internal/auth - internal/auth importing internal/tenant (which
// must import internal/core for the DB lookup) would cycle.
const TenantCtxKey = "tenant"

// Tenant represents a single tenant ("listnun") in a multi-tenant
// listnun deployment - resolved per-request by subdomain and RLS-scoped.
// Not the same thing as an Organization (a separate, purely cross-tenant
// grouping construct, managed only via the Operator API): a tenant
// optionally belongs to one via OrganizationID, but tenants remain the
// unit every other table's tenant_id actually refers to.
type Tenant struct {
	Base

	OrganizationID null.Int    `db:"organization_id" json:"organization_id"`
	Slug           string      `db:"slug" json:"slug"`
	Name           string      `db:"name" json:"name"`
	Status         string      `db:"status" json:"status"`
	CustomDomain   null.String `db:"custom_domain" json:"custom_domain"`
}

// Organization is a purely cross-tenant grouping of tenants ("listnuns"),
// managed only via the Operator API - never RLS-scoped, never resolved
// per-request the way a Tenant is. Exists so one customer can run
// multiple tenants for different purposes (e.g. separate brands/
// departments) under one umbrella.
type Organization struct {
	Base

	Name string `db:"name" json:"name"`
} // @name Organization
