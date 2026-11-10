package models

import (
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	APIKeyStatusActive  = "active"
	APIKeyStatusRevoked = "revoked"

	// APIKeyPrefix starts every generated secret so leaked keys are easy
	// to spot in logs and by secret scanners.
	APIKeyPrefix = "lnk_live_"
)

// APIKeyScopes maps each scope a key can be given to the permissions it
// grants. A key has no permissions beyond its scopes.
var APIKeyScopes = map[string][]string{
	"tx:send":            {"tx:send"},
	"tx:logs":            {"tx:logs"},
	"subscribers:read":   {"subscribers:get_all"},
	"subscribers:manage": {"subscribers:get_all", "subscribers:manage"},
	"subscribers:import": {"subscribers:import"},
	"lists:read":         {"lists:get_all"},
	"lists:manage":       {"lists:get_all", "lists:manage_all"},
	"campaigns:read":     {"campaigns:get_all", "campaigns:get_analytics"},
	"campaigns:manage":   {"campaigns:get_all", "campaigns:manage_all", "campaigns:send"},
	"templates:read":     {"templates:get"},
	"templates:manage":   {"templates:get", "templates:manage"},
	"media:read":         {"media:get"},
	"media:manage":       {"media:get", "media:manage"},
	"bounces:read":       {"bounces:get"},
	"bounces:manage":     {"bounces:get", "bounces:manage"},
	"keys:manage":        {"apikeys:get", "apikeys:manage"},
	"webhooks:manage":    {"webhooks:get", "webhooks:manage"},
}

// APIKeyScopeOrder is every scope in APIKeyScopes, in the order a key's
// scopes are stored and shown.
var APIKeyScopeOrder = []string{
	"tx:send", "tx:logs",
	"subscribers:read", "subscribers:manage", "subscribers:import",
	"lists:read", "lists:manage",
	"campaigns:read", "campaigns:manage",
	"templates:read", "templates:manage",
	"media:read", "media:manage",
	"bounces:read", "bounces:manage",
	"keys:manage", "webhooks:manage",
}

// APIKeyScopeRequires lists the scopes a scope can't work without, which
// are added to a key along with it. Adding someone to a list or importing
// into one needs manage access to that list, and a campaign is only
// visible with read access to its lists: without these, the engine
// silently drops the list IDs from the request.
var APIKeyScopeRequires = map[string][]string{
	"subscribers:manage": {"lists:manage"},
	"subscribers:import": {"lists:manage"},
	"lists:manage":       {"lists:read"},
	"campaigns:read":     {"lists:read"},
	"campaigns:manage":   {"campaigns:read"},
}

// APIKey is a named credential for the HTTP API. The secret itself is never
// returned after creation; SecretHint holds its last four characters.
type APIKey struct {
	Base

	Name          string         `db:"name" json:"name"`
	SecretHint    string         `db:"secret_hint" json:"secret_hint"`
	Scopes        pq.StringArray `db:"scopes" json:"scopes"`
	Status        string         `db:"status" json:"status"`
	CreatedBy     null.Int       `db:"created_by" json:"created_by"`
	CreatedByName string         `db:"created_by_name" json:"created_by_name"`
	PrevExpiresAt null.Time      `db:"prev_expires_at" json:"prev_expires_at"`
	LastUsedAt    null.Time      `db:"last_used_at" json:"last_used_at"`
	RevokedAt     null.Time      `db:"revoked_at" json:"revoked_at"`

	// LegacyUserID is the API user this key was migrated from. Until that
	// user is disabled, its old username:token still authenticates.
	LegacyUserID     null.Int `db:"legacy_user_id" json:"legacy_user_id"`
	LegacyUsername   string   `db:"legacy_username" json:"legacy_username"`
	LegacyUserStatus string   `db:"legacy_user_status" json:"legacy_user_status"`

	Total int `db:"total" json:"-"`
} // @name APIKey

// APIKeyWithSecret is returned once, when a key is created or rotated.
type APIKeyWithSecret struct {
	APIKey
	Secret string `json:"secret"`
} // @name APIKeyWithSecret

// APIKeyAuth is the subset of a key used to authenticate a request.
type APIKeyAuth struct {
	ID             int            `db:"id"`
	TenantID       int            `db:"tenant_id"`
	Name           string         `db:"name"`
	Scopes         pq.StringArray `db:"scopes"`
	SecretHash     string         `db:"secret_hash"`
	PrevSecretHash null.String    `db:"prev_secret_hash"`
}
