package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
)

// GetUserRoles retrieves roles.
//
//	@ID				listUserRoles
//	@Summary		List user roles
//	@Tags			roles
//	@Produce		json
//	@Success		200	{array}		auth.Role
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/roles/users [get]
func (a *App) GetUserRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetRoles(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GeListRoles retrieves roles.
//
//	@ID				listListRoles
//	@Summary		List list roles
//	@Tags			roles
//	@Produce		json
//	@Success		200	{array}		auth.ListRole
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/roles/lists [get]
func (a *App) GeListRoles(c echo.Context) error {
	// Get all roles.
	out, err := a.core.GetListRoles(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUserRole handles role creation.
//
//	@ID				createUserRole
//	@Summary		Create a user role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			body	body		auth.Role	true	"Role to create"
//	@Success		200		{object}	auth.Role
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/roles/users [post]
func (a *App) CreateUserRole(c echo.Context) error {
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateRole(c.Request().Context(), tenantID(c), r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateListRole handles role creation.
//
//	@ID				createListRole
//	@Summary		Create a list role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			body	body		auth.ListRole	true	"List role to create"
//	@Success		200		{object}	auth.ListRole
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/roles/lists [post]
func (a *App) CreateListRole(c echo.Context) error {
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Create the role in the DB.
	out, err := a.core.CreateListRole(c.Request().Context(), tenantID(c), r)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateUserRole handles role modification.
//
//	@ID				updateUserRole
//	@Summary		Update a user role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"Role ID"
//	@Param			body	body		auth.Role	true	"Updated role"
//	@Success		200		{object}	auth.Role
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/roles/users/{id} [put]
func (a *App) UpdateUserRole(c echo.Context) error {
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.Role
	if err := c.Bind(&r); err != nil {
		return err
	}
	if err := a.validateUserRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateUserRole(c.Request().Context(), tenantID(c), id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateListRole handles role modification.
//
//	@ID				updateListRole
//	@Summary		Update a list role
//	@Tags			roles
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Role ID"
//	@Param			body	body		auth.ListRole	true	"Updated list role"
//	@Success		200		{object}	auth.ListRole
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/roles/lists/{id} [put]
func (a *App) UpdateListRole(c echo.Context) error {
	// Get the role ID.
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Incoming params.
	var r auth.ListRole
	if err := c.Bind(&r); err != nil {
		return err
	}

	if err := a.validateListRole(r); err != nil {
		return err
	}

	// Validate.
	r.Name.String = strings.TrimSpace(r.Name.String)

	// Update the role in the DB.
	out, err := a.core.UpdateListRole(c.Request().Context(), tenantID(c), id, r)
	if err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteRole handles (user|list) role deletion.
//
//	@ID				deleteRole
//	@Summary		Delete a role
//	@Tags			roles
//	@Produce		json
//	@Param			id	path		int	true	"Role ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/roles/{id} [delete]
func (a *App) DeleteRole(c echo.Context) error {
	// Get the role ID.
	id := getID(c)

	// ID 1 is reserved for the Super Admin user role.
	if id == auth.SuperAdminRoleID {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the role from the DB.
	if err := a.core.DeleteRole(c.Request().Context(), tenantID(c), int(id)); err != nil {
		return err
	}

	// Cache API tokens for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateUserRole(r auth.Role) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, p := range r.Permissions {
		if _, ok := a.cfg.Permissions[p]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("permission: %s", p)))
		}
	}

	return nil
}

func (a *App) validateListRole(r auth.ListRole) error {
	if !strHasLen(r.Name.String, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}

	for _, l := range r.Lists {
		for _, p := range l.Permissions {
			if p != auth.PermListGet && p != auth.PermListManage {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("list permission: %s", p)))
			}
		}
	}

	return nil
}
