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
	"tx:send":          {"tx:send"},
	"tx:logs":          {"tx:logs"},
	"subscribers:read": {"subscribers:get_all"},
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
