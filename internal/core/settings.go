package core

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetSettings returns settings from the DB.
func (c *Core) GetSettings(ctx context.Context, tenantID int) (models.Settings, error) {
	var (
		b   types.JSONText
		out models.Settings
	)

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetSettings).Get(&b, tenantID)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	// Unmarshal the settings and filter out sensitive fields.
	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	return out, nil
}

// UpdateSettings updates settings.
func (c *Core) UpdateSettings(ctx context.Context, tenantID int, s models.Settings) error {
	// A platform-managed Scrub config is locked -- a tenant admin's own
	// full-object settings save must not be able to change any part of
	// it, no matter what the request body contained. This is the whole
	// sub-object, not field-by-field, since everything is locked together
	// once platform-managed (see models.Settings.Scrub's doc comment).
	if cur, err := c.GetSettings(ctx, tenantID); err == nil && cur.Scrub.ManagedByPlatform {
		s.Scrub = cur.Scrub
	}

	// Marshal settings.
	b, err := json.Marshal(s)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("settings.errorEncoding", "error", err.Error()))
	}

	// Update the settings in the DB.
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateSettings).Exec(b, tenantID)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	return nil
}

// UpdateSettingsByKey updates a single setting by key.
func (c *Core) UpdateSettingsByKey(ctx context.Context, tenantID int, key string, value json.RawMessage) error {
	// A raw PUT /api/settings/:key targeting "scrub" is a second,
	// independent write path into the same row UpdateSettings guards --
	// it must be locked too, or a tenant admin can trivially bypass the
	// UpdateSettings-only guard by calling this endpoint directly instead.
	if key == "scrub" {
		if cur, err := c.GetSettings(ctx, tenantID); err == nil && cur.Scrub.ManagedByPlatform {
			return echo.NewHTTPError(http.StatusForbidden,
				c.i18n.T("settings.scrub.managedByPlatform"))
		}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateSettingsByKey).Exec(key, value, tenantID)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.settings}", "error", pqErrMsg(err)))
	}

	return nil
}
