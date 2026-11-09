package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

// APIKeyRotationGrace is how long a rotated key's previous secret keeps
// working, so callers can deploy the new one without downtime.
const APIKeyRotationGrace = 24 * time.Hour

// apiKeySecretBytes is the random part of a secret: 16 bytes, 32 hex chars.
const apiKeySecretBytes = 16

// regexpAPIKeyName matches valid key names, which are also the basic auth
// username: lowercase letters, digits, dots, dashes and underscores.
var regexpAPIKeyName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,62}$`)

// ValidAPIKeyName reports whether name can be used for a key.
func ValidAPIKeyName(name string) bool {
	return regexpAPIKeyName.MatchString(name)
}

// ValidAPIKeyScopes reports whether every scope is a known one.
func ValidAPIKeyScopes(scopes []string) bool {
	for _, s := range scopes {
		if _, ok := models.APIKeyScopes[s]; !ok {
			return false
		}
	}
	return true
}

// APIKeyPermissions expands scopes into the permissions they grant.
func APIKeyPermissions(scopes []string) []string {
	var out []string
	for _, s := range scopes {
		for _, p := range models.APIKeyScopes[s] {
			if !slices.Contains(out, p) {
				out = append(out, p)
			}
		}
	}
	return out
}

func newAPIKeySecret() (secret, hash, hint string, err error) {
	b := make([]byte, apiKeySecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", err
	}
	secret = models.APIKeyPrefix + hex.EncodeToString(b)
	return secret, hashAPIKeySecret(secret), secret[len(secret)-4:], nil
}

// hashAPIKeySecret hashes a secret for storage. Secrets are 128 random
// bits, so a fast hash is enough; there's nothing to brute-force.
func hashAPIKeySecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// GetAPIKeys returns a tenant's keys, optionally filtered by name.
func (c *Core) GetAPIKeys(ctx context.Context, tenantID int, query string) ([]models.APIKey, error) {
	out := []models.APIKey{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetAPIKeys).Select(&out, query)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.keys.keys}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// GetAPIKey returns one key.
func (c *Core) GetAPIKey(ctx context.Context, tenantID, id int) (models.APIKey, error) {
	var out models.APIKey
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetAPIKey).Get(&out, id)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.keys.key}"))
	}
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{developers.keys.key}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// CreateAPIKey creates a key and returns it with its secret, which is
// never retrievable again.
func (c *Core) CreateAPIKey(ctx context.Context, tenantID int, name string, scopes []string, createdBy int, createdByName string) (models.APIKeyWithSecret, error) {
	secret, hash, hint, err := newAPIKeySecret()
	if err != nil {
		return models.APIKeyWithSecret{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var id int
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateAPIKey).Get(&id, name, hash, hint, pq.StringArray(scopes), createdBy, createdByName, tenantID)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return models.APIKeyWithSecret{}, echo.NewHTTPError(http.StatusConflict, c.i18n.T("developers.keys.nameTaken"))
		}
		return models.APIKeyWithSecret{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{developers.keys.key}", "error", pqErrMsg(err)))
	}

	k, err := c.GetAPIKey(ctx, tenantID, id)
	if err != nil {
		return models.APIKeyWithSecret{}, err
	}
	return models.APIKeyWithSecret{APIKey: k, Secret: secret}, nil
}

// UpdateAPIKeyScopes replaces an active key's scopes.
func (c *Core) UpdateAPIKeyScopes(ctx context.Context, tenantID, id int, scopes []string) (models.APIKey, error) {
	if err := c.execAPIKey(ctx, tenantID, c.q.UpdateAPIKeyScopes, "errorUpdating", id, pq.StringArray(scopes)); err != nil {
		return models.APIKey{}, err
	}
	return c.GetAPIKey(ctx, tenantID, id)
}

// RotateAPIKey issues a new secret. The old one keeps working for
// APIKeyRotationGrace.
func (c *Core) RotateAPIKey(ctx context.Context, tenantID, id int) (models.APIKeyWithSecret, error) {
	secret, hash, hint, err := newAPIKeySecret()
	if err != nil {
		return models.APIKeyWithSecret{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := c.execAPIKey(ctx, tenantID, c.q.RotateAPIKey, "errorUpdating", id, hash, hint, time.Now().Add(APIKeyRotationGrace)); err != nil {
		return models.APIKeyWithSecret{}, err
	}

	k, err := c.GetAPIKey(ctx, tenantID, id)
	if err != nil {
		return models.APIKeyWithSecret{}, err
	}
	return models.APIKeyWithSecret{APIKey: k, Secret: secret}, nil
}

// RevokeAPIKey stops a key at once, including any previous secret still
// in its rotation grace period.
func (c *Core) RevokeAPIKey(ctx context.Context, tenantID, id int) (models.APIKey, error) {
	if err := c.execAPIKey(ctx, tenantID, c.q.RevokeAPIKey, "errorUpdating", id); err != nil {
		return models.APIKey{}, err
	}
	return c.GetAPIKey(ctx, tenantID, id)
}

// DeleteAPIKey removes a key.
func (c *Core) DeleteAPIKey(ctx context.Context, tenantID, id int) error {
	return c.execAPIKey(ctx, tenantID, c.q.DeleteAPIKey, "errorDeleting", id)
}

// execAPIKey runs a write against one key and maps "no rows" to not found.
func (c *Core) execAPIKey(ctx context.Context, tenantID int, stmt *sqlx.Stmt, errKey string, args ...any) error {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, stmt).Exec(args...)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages."+errKey, "name", "{developers.keys.key}", "error", pqErrMsg(err)))
	}
	if n == 0 {
		return echo.NewHTTPError(http.StatusNotFound,
			c.i18n.Ts("globals.messages.notFound", "name", "{developers.keys.key}"))
	}
	return nil
}

// AuthenticateAPIKey checks a basic auth name and secret against the
// tenant's active keys, accepting a rotated key's previous secret during
// its grace period. On success it records the key as used.
func (c *Core) AuthenticateAPIKey(ctx context.Context, tenantID int, name, secret string) (models.APIKeyAuth, bool) {
	var k models.APIKeyAuth
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetAPIKeyAuth).Get(&k, tenantID, name)
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			c.log.Printf("error looking up API key %q: %v", name, err)
		}
		return models.APIKeyAuth{}, false
	}

	hash := []byte(hashAPIKeySecret(secret))
	ok := subtle.ConstantTimeCompare(hash, []byte(k.SecretHash)) == 1
	if !ok && k.PrevSecretHash.Valid {
		ok = subtle.ConstantTimeCompare(hash, []byte(k.PrevSecretHash.String)) == 1
	}
	if !ok {
		return models.APIKeyAuth{}, false
	}

	if err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.TouchAPIKey).Exec(k.ID)
		return err
	}); err != nil {
		c.log.Printf("error recording API key use %q: %v", name, err)
	}

	return k, true
}
