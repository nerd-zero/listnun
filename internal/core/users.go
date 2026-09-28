package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/utils"
	"gopkg.in/volatiletech/null.v6"
)

// HasUsers reports whether the given tenant has at least one non-API user.
// Used to decide whether that tenant needs the first-time-setup flow
// (cmd/auth.go's LoginPage) - a lightweight existence check instead of
// GetUsers' full role/list-role joins.
func (c *Core) HasUsers(ctx context.Context, tenantID int) (bool, error) {
	var out bool
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.HasUsers).Get(&out)
	})
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetUsers retrieves all users for the given tenant.
func (c *Core) GetUsers(ctx context.Context, tenantID int) ([]auth.User, error) {
	out := []auth.User{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetUsers).Select(&out)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields(out), nil
}

// GetAllUsersUnscoped retrieves every user across every tenant, bypassing
// tenant scoping entirely. This exists solely to populate the in-memory
// API-token cache (cmd/users.go's cacheUsers): an incoming API request's
// tenant is only known from its subdomain, and the token itself has to be
// matched against the full set of API users before internal/auth's
// tenantMismatch check can compare and reject it if it belongs to a
// different tenant - narrowing this query by tenant upfront would make that
// cross-tenant check unreachable. Same reasoning as GetUserUnscoped below.
func (c *Core) GetAllUsersUnscoped() ([]auth.User, error) {
	out := []auth.User{}
	if err := c.q.GetUsers.Select(&out); err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields(out), nil
}

// GetUser retrieves a specific user based on any one given identifier, scoped to tenantID.
func (c *Core) GetUser(ctx context.Context, tenantID int, id int, username, email string) (auth.User, error) {
	var out auth.User
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetUser).Get(&out, id, username, email)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusNotFound,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))

		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields([]auth.User{out})[0], nil
}

// GetUserUnscoped retrieves a user by ID without tenant filtering. Used only
// by the auth session/API-token lookup callback (cmd/init.go's
// auth.Callbacks.GetUser), which must find the user row regardless of which
// tenant it actually belongs to so internal/auth.tenantMismatch can
// explicitly compare it against the request's resolved tenant and reject
// with 403 - if this filtered by tenant via RLS instead, a session/token
// replayed against the wrong tenant's subdomain would silently look
// identical to "user not found", losing that distinct signal.
func (c *Core) GetUserUnscoped(id int) (auth.User, error) {
	var out auth.User
	if err := c.q.GetUser.Get(&out, id, "", ""); err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusNotFound,
				c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))

		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return c.setupUserFields([]auth.User{out})[0], nil
}

// CreateUser creates a new user.
func (c *Core) CreateUser(ctx context.Context, tenantID int, u auth.User) (auth.User, error) {
	var id int

	// If it's an API user, generate a random token for password
	// and set the e-mail to default.
	if u.Type == auth.UserTypeAPI {
		// Generate a random admin password.
		tk, err := utils.GenerateRandomString(32)
		if err != nil {
			return auth.User{}, err
		}

		u.Email = null.String{String: u.Username + "@api", Valid: true}
		u.PasswordLogin = false
		u.Password = null.String{String: tk, Valid: true}
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateUser).Get(&id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.UserRoleID, u.ListRoleID, u.Status, tenantID)
	})
	if err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	// Hide the password field in the response except for when the user type is an API token,
	// where the frontend shows the token on the UI just once.
	if u.Type != auth.UserTypeAPI {
		u.Password = null.String{Valid: false}
	}

	out, err := c.GetUser(ctx, tenantID, id, "", "")
	return out, err
}

// UpdateUser updates a given user.
func (c *Core) UpdateUser(ctx context.Context, tenantID int, id int, u auth.User) (auth.User, error) {
	listRoleID := 0
	if u.ListRoleID == nil {
		listRoleID = -1
	} else {
		listRoleID = *u.ListRoleID
	}

	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.UpdateUser).Exec(id, u.Username, u.PasswordLogin, u.Password, u.Email, u.Name, u.Type, u.UserRoleID, listRoleID, u.Status)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n == 0 {
		return auth.User{}, echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.needSuper"))
	}

	out, err := c.GetUser(ctx, tenantID, id, "", "")

	return out, err
}

// UpdateUserProfile updates the basic fields of a given uesr (name, email, password).
func (c *Core) UpdateUserProfile(ctx context.Context, tenantID int, id int, u auth.User) (auth.User, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.UpdateUserProfile).Exec(id, u.Name, u.Email, u.PasswordLogin, u.Password)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return auth.User{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	if n == 0 {
		return auth.User{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"))
	}

	return c.GetUser(ctx, tenantID, id, "", "")
}

// UpdateUserLogin updates a user's record post-login.
func (c *Core) UpdateUserLogin(ctx context.Context, tenantID int, id int, avatar string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.UpdateUserLogin).Exec(id, avatar)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	return nil
}

// SetTwoFA sets or clears the 2FA configuration for a user.
func (c *Core) SetTwoFA(ctx context.Context, tenantID int, id int, twofaType, twofaKey string) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.SetUserTwoFA).Exec(id, twofaType, twofaKey)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a given user ID, optionally
// excluding a specific session ID (to keep the current session alive).
//
// The sessions table is not tenant-scoped (no tenant_id column, no RLS
// policy) - a session is already tied to a specific user, whose own tenant
// is enforced elsewhere (internal/auth.tenantMismatch), so this doesn't
// need WithTenant.
func (c *Core) DeleteUserSessions(userID int, excludeID string) error {
	if _, err := c.q.DeleteUserSessions.Exec(strconv.Itoa(userID), excludeID); err != nil {
		return err
	}
	return nil
}

// DeleteUsers deletes a given user.
func (c *Core) DeleteUsers(ctx context.Context, tenantID int, ids []int) error {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.DeleteUsers).Exec(pq.Array(ids))
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}", "error", pqErrMsg(err)))
	}
	if n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, c.i18n.T("users.needSuper"))
	}

	return nil
}

// LoginUser attempts to log the given user_id in by matching the password.
//
// Scoped to tenantID via WithTenant/RLS: usernames aren't tenant-namespaced
// (username is still a global UNIQUE constraint, a pre-existing limitation
// tracked in docs/design/multi-tenancy.md), so without this scoping a
// correct username/password for tenant A would also successfully log in on
// tenant B's subdomain if a same-named account happened to exist there.
// Found while threading this file for #40 - not caught earlier because
// internal/auth.tenantMismatch only guards *replaying* an existing
// session/token against the wrong tenant, not the login call itself.
func (c *Core) LoginUser(ctx context.Context, tenantID int, username, password string) (auth.User, error) {
	var out auth.User
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.LoginUser).Get(&out, username, password)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return out, echo.NewHTTPError(http.StatusForbidden, c.i18n.T("users.invalidLogin"))
		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.users}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// setupUserFields prepares and sets up various user fields.
func (c *Core) setupUserFields(users []auth.User) []auth.User {
	for n, u := range users {
		u := u

		if u.Password.String != "" {
			u.HasPassword = true
			u.PasswordLogin = true
		}

		if u.Type == auth.UserTypeAPI {
			u.Email = null.String{}
		}

		u.UserRole.ID = u.UserRoleID
		u.UserRole.Name = u.UserRoleName
		u.UserRole.Permissions = u.UserRolePerms
		u.UserRoleID = 0

		// Prepare lookup maps.
		u.ListPermissionsMap = make(map[int]map[string]struct{})
		u.PermissionsMap = make(map[string]struct{})
		for _, p := range u.UserRolePerms {
			u.PermissionsMap[p] = struct{}{}
		}

		if u.ListRoleID != nil {
			// Unmarshall the raw list perms map.
			var listPerms []auth.ListPermission
			if u.ListsPermsRaw != nil {
				if err := json.Unmarshal(*u.ListsPermsRaw, &listPerms); err != nil {
					c.log.Printf("error unmarshalling list permissions for role %d: %v", u.ID, err)
				}
			}

			u.ListRole = &auth.ListRolePermissions{ID: *u.ListRoleID, Name: u.ListRoleName.String, Lists: listPerms}

			// Iterate each list in the list permissions and setup get/manage list IDs.
			for _, p := range listPerms {
				u.ListPermissionsMap[p.ID] = make(map[string]struct{})

				for _, perm := range p.Permissions {
					u.ListPermissionsMap[p.ID][perm] = struct{}{}

					// List IDs with get / manage permissions.
					if perm == auth.PermListGet {
						u.GetListIDs = append(u.GetListIDs, p.ID)
					}
					if perm == auth.PermListManage {
						u.ManageListIDs = append(u.ManageListIDs, p.ID)
					}
				}
			}
		}

		users[n] = u
	}

	return users
}
