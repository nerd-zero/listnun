package main

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/models"
)

// apiKeyReq is the body for creating a key or changing its scopes.
type apiKeyReq struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
} // @name APIKeyRequest

// apiKeyUser turns an authenticated API key into the request's user. It
// carries only the permissions the key's scopes grant.
func apiKeyUser(k models.APIKeyAuth) auth.User {
	u := auth.User{
		Username: k.Name,
		Name:     k.Name,
		TenantID: k.TenantID,
		Type:     auth.UserTypeAPI,
		Status:   auth.UserStatusEnabled,
		APIKeyID: k.ID,
	}

	perms := core.APIKeyPermissions(k.Scopes)
	u.UserRole.Name = "api-key"
	u.UserRole.Permissions = perms
	u.PermissionsMap = make(map[string]struct{}, len(perms))
	for _, p := range perms {
		u.PermissionsMap[p] = struct{}{}
	}
	return u
}

// GetAPIKeys lists the workspace's API keys.
//
//	@ID			listAPIKeys
//	@Summary		List API keys
//	@Tags			developers
//	@Produce		json
//	@Param			query	query		string	false	"Filter by name"
//	@Success		200		{object}	[]models.APIKey
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/developers/keys [get]
func (a *App) GetAPIKeys(c echo.Context) error {
	out, err := a.core.GetAPIKeys(c.Request().Context(), tenantID(c), strings.TrimSpace(c.QueryParam("query")))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateAPIKey creates an API key. The response is the only time the
// full secret is returned.
//
//	@ID			createAPIKey
//	@Summary		Create an API key
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		apiKeyReq	true	"Key name and scopes"
//	@Success		200		{object}	models.APIKeyWithSecret
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		409		{object}	echo.HTTPError
//	@Router			/api/developers/keys [post]
func (a *App) CreateAPIKey(c echo.Context) error {
	var req apiKeyReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if !core.ValidAPIKeyName(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.keys.invalidName"))
	}
	if err := a.validateAPIKeyScopes(req.Scopes); err != nil {
		return err
	}

	user := auth.GetUser(c)
	createdBy := user.Name
	if createdBy == "" {
		createdBy = user.Username
	}

	out, err := a.core.CreateAPIKey(c.Request().Context(), tenantID(c), req.Name, req.Scopes, user.ID, createdBy)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateAPIKey replaces an API key's scopes.
//
//	@ID			updateAPIKey
//	@Summary		Update an API key's scopes
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"Key ID"
//	@Param			body	body		apiKeyReq	true	"Scopes"
//	@Success		200		{object}	models.APIKey
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/keys/{id} [put]
func (a *App) UpdateAPIKey(c echo.Context) error {
	var req apiKeyReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := a.validateAPIKeyScopes(req.Scopes); err != nil {
		return err
	}

	out, err := a.core.UpdateAPIKeyScopes(c.Request().Context(), tenantID(c), getID(c), req.Scopes)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// RotateAPIKey issues a new secret for a key. The previous secret keeps
// working for 24 hours.
//
//	@ID			rotateAPIKey
//	@Summary		Rotate an API key
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Key ID"
//	@Success		200	{object}	models.APIKeyWithSecret
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/keys/{id}/rotate [post]
func (a *App) RotateAPIKey(c echo.Context) error {
	out, err := a.core.RotateAPIKey(c.Request().Context(), tenantID(c), getID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// RevokeAPIKey stops a key at once. Requests with it get 403.
//
//	@ID			revokeAPIKey
//	@Summary		Revoke an API key
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Key ID"
//	@Success		200	{object}	models.APIKey
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/keys/{id}/revoke [post]
func (a *App) RevokeAPIKey(c echo.Context) error {
	out, err := a.core.RevokeAPIKey(c.Request().Context(), tenantID(c), getID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteAPIKey deletes a key.
//
//	@ID			deleteAPIKey
//	@Summary		Delete an API key
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Key ID"
//	@Success		200	{object}	okResp
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/keys/{id} [delete]
func (a *App) DeleteAPIKey(c echo.Context) error {
	if err := a.core.DeleteAPIKey(c.Request().Context(), tenantID(c), getID(c)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateAPIKeyScopes(scopes []string) error {
	if len(scopes) == 0 || !core.ValidAPIKeyScopes(scopes) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.keys.invalidScopes"))
	}
	return nil
}
