package main

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/internal/utils"
	"github.com/nerd-zero/listnun/models"
	"github.com/pquerna/otp/totp"
	"gopkg.in/volatiletech/null.v6"
)

var (
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]+$`)
)

// GetUser retrieves a single user by ID.
//
//	@ID				getUser
//	@Summary		Get a user
//	@Tags			users
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	auth.User
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/users/{id} [get]
func (a *App) GetUser(c echo.Context) error {
	// Get the user from the DB.
	id := getID(c)
	out, err := a.core.GetUser(c.Request().Context(), tenantID(c), id, "", "")
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	out.Password = null.String{}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetUsers retrieves all users.
//
//	@ID				listUsers
//	@Summary		List users
//	@Tags			users
//	@Produce		json
//	@Success		200	{array}		auth.User
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/users [get]
func (a *App) GetUsers(c echo.Context) error {
	// Get all users from the DB.
	out, err := a.core.GetUsers(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	for n := range out {
		out[n].Password = null.String{}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateUser handles user creation.
//
//	@ID				createUser
//	@Summary		Create a user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		auth.User	true	"User to create"
//	@Success		200		{object}	auth.User
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/users [post]
func (a *App) CreateUser(c echo.Context) error {
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if u.Type != auth.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		if u.PasswordLogin {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	if u.Name == "" {
		u.Name = u.Username
	}

	// Create the user in the DB.
	user, err := a.core.CreateUser(c.Request().Context(), tenantID(c), u)
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	if user.Type != auth.UserTypeAPI {
		user.Password = null.String{}
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// UpdateUser handles user modification.
//
//	@ID				updateUser
//	@Summary		Update a user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"User ID"
//	@Param			body	body		auth.User	true	"Updated user"
//	@Success		200		{object}	auth.User
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/users/{id} [put]
func (a *App) UpdateUser(c echo.Context) error {
	// Incoming params.
	var u auth.User
	if err := c.Bind(&u); err != nil {
		return err
	}

	u.Username = strings.TrimSpace(u.Username)
	u.Name = strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email.String))

	// Validate fields.
	if !strHasLen(u.Username, 3, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}
	if !reUsername.MatchString(u.Username) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "username"))
	}

	// Get the user ID.
	id := getID(c)
	if u.Type != auth.UserTypeAPI {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}

		// Validate password if password login is enabled.
		if u.PasswordLogin && u.Password.String != "" {
			if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
			}

			if u.Password.String != "" {
				// If a password is sent, validate it before updating in the DB. If it's not set, leave the password in the DB untouched.
				if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
					return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			} else {
				// Get the user from the DB.
				user, err := a.core.GetUser(c.Request().Context(), tenantID(c), id, "", "")
				if err != nil {
					return err
				}

				// If password login is enabled, but there's no password in the DB and there's no incoming
				// password, throw an error.
				if !user.HasPassword {
					return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
				}
			}
		}

		u.Email = null.String{String: email, Valid: true}
	}

	// Default the name to username if not set.
	if u.Name == "" {
		u.Name = u.Username
	}

	// Update the user in the DB.
	user, err := a.core.UpdateUser(c.Request().Context(), tenantID(c), id, u)
	if err != nil {
		return err
	}

	// Blank out the password hash in the response.
	user.Password = null.String{}

	// If password was changed by admin, destroy all sessions for the given user.
	if u.Password.String != "" {
		if err := a.core.DeleteUserSessions(id, ""); err != nil {
			a.log.Printf("error destroying sessions on admin password change for user_id=%d: %v", id, err)
		}
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{user})
}

// DeleteUser handles the deletion of a single user by ID.
//
//	@ID				deleteUser
//	@Summary		Delete a user
//	@Tags			users
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/users/{id} [delete]
func (a *App) DeleteUser(c echo.Context) error {
	// Delete the user(s) from the DB.
	id := getID(c)
	if err := a.core.DeleteUsers(c.Request().Context(), tenantID(c), []int{id}); err != nil {
		return err
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteUsers handles user deletion, either a single one (ID in the URI), or a list.
//
//	@ID				deleteUsers
//	@Summary		Delete one or more users
//	@Tags			users
//	@Produce		json
//	@Param			id	query		[]int	true	"User ID(s)"	collectionFormat(multi)
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/users [delete]
func (a *App) DeleteUsers(c echo.Context) error {
	ids, err := getQueryInts("id", c.QueryParams())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidID"))
	}

	// Delete the user(s) from the DB.
	if err := a.core.DeleteUsers(c.Request().Context(), tenantID(c), ids); err != nil {
		return err
	}

	// Cache the API token for in-memory, off-DB /api/* request auth.
	if _, err := cacheUsers(a.core, a.auth); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetUserProfile fetches the uesr profile for the currently logged in user.
//
//	@ID			getUserProfile
//	@Summary	Get current user profile
//	@Tags		users
//	@Produce	json
//	@Success	200	{object}	User
//	@Router		/api/profile [get]
func (a *App) GetUserProfile(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Blank out the password hash in the response.
	user.Password.String = ""
	user.Password.Valid = false

	return c.JSON(http.StatusOK, okResp{user})
}

// profileReq is the body of a profile update: the editable user fields plus
// the current password, which is required when setting a new one.
type profileReq struct {
	auth.User
	CurrentPassword string `json:"current_password"`
} // @name ProfileUpdate

// UpdateUserProfile update's the current user's profile.
//
//	@ID			updateUserProfile
//	@Summary	Update current user profile
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		body	body		profileReq	true	"Profile fields; current_password is required with a new password"
//	@Success	200		{object}	User
//	@Failure	400		{object}	echo.HTTPError
//	@Failure	403		{object}	echo.HTTPError
//	@Router		/api/profile [put]
func (a *App) UpdateUserProfile(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Incoming params.
	var req profileReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	u := req.User
	u.PasswordLogin = user.PasswordLogin
	u.Name = strings.TrimSpace(u.Name)
	email := strings.TrimSpace(u.Email.String)

	// Validate fields.
	if user.PasswordLogin {
		if !utils.ValidateEmail(email) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email"))
		}
		u.Email = null.String{String: email, Valid: true}
	}

	if u.PasswordLogin && u.Password.String != "" {
		if !strHasLen(u.Password.String, 8, stdInputMaxLen) {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
		}

		// A new password needs the current one, so an unattended session
		// can't be used to lock the owner out.
		if _, err := a.core.LoginUser(c.Request().Context(), tenantID(c), user.Username, req.CurrentPassword); err != nil {
			return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("users.invalidPassword"))
		}
	}

	// Update the user in the DB.
	out, err := a.core.UpdateUserProfile(c.Request().Context(), tenantID(c), user.ID, u)
	if err != nil {
		return err
	}

	// If password was changed, destroy all existing sessions for the user except for the current one.
	if u.Password.String != "" {
		if err := a.core.DeleteUserSessions(user.ID, auth.GetSessionID(c)); err != nil {
			a.log.Printf("error destroying sessions after profile password change for user_id=%d: %v", user.ID, err)
		}
	}

	// Blank out the password hash in the response.
	out.Password = null.String{}

	return c.JSON(http.StatusOK, okResp{out})
}

// EnableTOTP enables TOTP 2FA for a user after verifying the code.
//
//	@ID				enableTotp
//	@Summary		Enable TOTP two-factor authentication
//	@Tags			users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		int		true	"User ID"
//	@Param			secret	formData	string	true	"TOTP secret"
//	@Param			code	formData	string	true	"TOTP verification code"
//	@Success		200
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/users/{id}/twofa [put]
func (a *App) EnableTOTP(c echo.Context) error {
	var (
		u      = c.Get(auth.UserHTTPCtxKey).(auth.User)
		secret = strings.TrimSpace(c.FormValue("secret"))
		code   = strings.TrimSpace(c.FormValue("code"))
	)

	if secret == "" || code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidFields"))
	}

	// If password login is disabled, can't enable TOTP.
	if !u.PasswordLogin {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("public.invalidFeature"))
	}

	// If TOTP is already enabled, don't allow re-enabling.
	if u.TwofaType == models.TwofaTypeTOTP {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.twoFAAlreadyEnabled"))
	}

	// Verify the TOTP code.
	valid := totp.Validate(code, secret)
	if !valid {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.invalidTOTPCode"))
	}

	// Enable TOTP in the DB.
	if err := a.core.SetTwoFA(c.Request().Context(), tenantID(c), u.ID, models.TwofaTypeTOTP, secret); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DisableTOTP disables TOTP 2FA for a user after verifying the password.
//
//	@ID				disableTotp
//	@Summary		Disable TOTP two-factor authentication
//	@Tags			users
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id			path		int		true	"User ID"
//	@Param			password	formData	string	true	"Current password"
//	@Success		200
//	@Failure		400			{object}	echo.HTTPError
//	@Failure		403			{object}	echo.HTTPError
//	@Router			/api/users/{id}/twofa [delete]
func (a *App) DisableTOTP(c echo.Context) error {
	var (
		u        = c.Get(auth.UserHTTPCtxKey).(auth.User)
		password = c.FormValue("password")
	)

	// TOTP isn't enabled.
	if u.TwofaType != models.TwofaTypeTOTP {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("users.twoFANotEnabled"))
	}

	// Validate password.
	if !strHasLen(password, 8, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "password"))
	}

	// Verify the password.
	if _, err := a.core.LoginUser(c.Request().Context(), tenantID(c), u.Username, password); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, a.i18n.T("users.invalidPassword"))
	}

	// Disable TOTP in the DB.
	if err := a.core.SetTwoFA(c.Request().Context(), tenantID(c), u.ID, models.TwofaTypeNone, ""); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// cacheUsers fetches (API) users and caches them in the auth module.
// It also returns a bool indicating whether there are any actual users in the DB at all,
// which if there aren't, the first time user setup needs to be run.
func cacheUsers(co *core.Core, a *auth.Auth) (bool, error) {
	users, err := co.GetAllUsersUnscoped()
	if err != nil {
		return false, err
	}

	hasUser := false
	apiUsers := make([]auth.User, 0, len(users))
	for _, u := range users {
		if u.Type == auth.UserTypeAPI && u.Status == auth.UserStatusEnabled {
			apiUsers = append(apiUsers, u)
		}

		if u.Type == auth.UserTypeUser {
			hasUser = true
		}
	}

	a.CacheAPIUsers(apiUsers)
	return hasUser, nil
}
