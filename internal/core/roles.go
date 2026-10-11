package core

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/auth"
)

// GetRoles retrieves all roles.
func (c *Core) GetRoles(ctx context.Context, tenantID int) ([]auth.Role, error) {
	out := []auth.Role{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetUserRoles).Select(&out, nil)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "role", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetRole retrieves a role.
func (c *Core) GetRole(ctx context.Context, tenantID int, id int) (auth.Role, error) {
	out := []auth.Role{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetUserRoles).Select(&out, id)
	})
	if err != nil {
		return auth.Role{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "role", "error", pqErrMsg(err)))
	}

	// Role does not exist.
	if len(out) == 0 {
		return auth.Role{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "role", "error", "role not found"))
	}

	return out[0], nil
}

// GetListRoles retrieves all list roles.
func (c *Core) GetListRoles(ctx context.Context, tenantID int) ([]auth.ListRole, error) {
	out := []auth.ListRole{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListRoles).Select(&out)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "role", "error", pqErrMsg(err)))
	}

	// Unmarshall the nested list permissions, if any.
	for n, r := range out {
		if r.ListsRaw == nil {
			continue
		}

		if err := json.Unmarshal(r.ListsRaw, &out[n].Lists); err != nil {
			c.log.Printf("error unmarshalling list permissions for role %d: %v", r.ID, err)
		}
	}

	return out, nil
}

// CreateRole creates a new role.
func (c *Core) CreateRole(ctx context.Context, tenantID int, r auth.Role) (auth.Role, error) {
	var out auth.Role

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateRole).Get(&out, r.Name, auth.RoleTypeUser, pq.Array(r.Permissions), tenantID)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// CreateListRole creates a new list role.
func (c *Core) CreateListRole(ctx context.Context, tenantID int, r auth.ListRole) (auth.ListRole, error) {
	var out auth.ListRole

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateRole).Get(&out, r.Name, auth.RoleTypeList, pq.Array([]string{}), tenantID)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	if err := c.UpsertListPermissions(ctx, tenantID, out.ID, r.Lists); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// UpsertListPermissions upserts permission for a role.
func (c *Core) UpsertListPermissions(ctx context.Context, tenantID int, roleID int, lp []auth.ListPermission) error {
	var (
		listIDs   = make([]int, 0, len(lp))
		listPerms = make([][]string, 0, len(lp))
	)
	for _, p := range lp {
		if len(p.Permissions) == 0 {
			continue
		}

		listIDs = append(listIDs, p.ID)

		// For the Postgres array unnesting query to work, all permissions arrays should
		// have equal number of entries. Add "" in case there's only one of either list:get or list:manage
		perms := make([]string, 2)
		copy(perms[:], p.Permissions[:])
		listPerms = append(listPerms, perms)
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpsertListPermissions).Exec(roleID, pq.Array(listIDs), pq.Array(listPerms), tenantID)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteListPermission deletes a list permission entry from a role.
func (c *Core) DeleteListPermission(ctx context.Context, tenantID int, roleID, listID int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteListPermission).Exec(roleID, listID)
		return err
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "users_role_id_fkey" {
			return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.cantDeleteRole"))
		}
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return nil
}

// UpdateUserRole updates a given role.
func (c *Core) UpdateUserRole(ctx context.Context, tenantID int, id int, r auth.Role) (auth.Role, error) {
	var out auth.Role

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.UpdateRole).Get(&out, id, r.Name, pq.Array(r.Permissions))
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{users.userRole}", "error", pqErrMsg(err)))
	}

	if out.ID == 0 {
		return out, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{users.userRole}"))
	}

	return out, nil
}

// UpdateListRole updates a given role.
func (c *Core) UpdateListRole(ctx context.Context, tenantID int, id int, r auth.ListRole) (auth.ListRole, error) {
	var out auth.ListRole

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.UpdateRole).Get(&out, id, r.Name, pq.Array([]string{}))
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{users.listRole}", "error", pqErrMsg(err)))
	}

	if out.ID == 0 {
		return out, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.notFound", "name", "{users.listRole}"))
	}

	if err := c.UpsertListPermissions(ctx, tenantID, out.ID, r.Lists); err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{users.listRole}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// DeleteRole deletes a given role.
func (c *Core) DeleteRole(ctx context.Context, tenantID int, id int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteRole).Exec(id)
		return err
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "users_role_id_fkey" {
			return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.cantDeleteRole"))
		}
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{users.role}", "error", pqErrMsg(err)))
	}

	return nil
}
