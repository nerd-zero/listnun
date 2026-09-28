package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/models"
)

// GetLists retrieves lists with additional metadata like subscriber counts.
//
//	@ID			listLists
//	@Summary		Get lists
//	@Tags			lists
//	@Produce		json
//	@Param			query		query		string	false	"Search query"
//	@Param			type		query		string	false	"List type"
//	@Param			optin		query		string	false	"Opt-in type"
//	@Param			status		query		string	false	"List status"
//	@Param			tag			query		[]string	false	"Tags"
//	@Param			order_by	query		string	false	"Order by field"
//	@Param			order		query		string	false	"Sort order (asc/desc)"
//	@Param			minimal		query		bool	false	"Return minimal list without subscriber counts"
//	@Param			page		query		int		false	"Page number"
//	@Param			per_page	query		int		false	"Results per page"
//	@Success		200	{object}	models.PageResults
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/lists [get]
func (a *App) GetLists(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Get the list IDs (or blanket permission) the user has access to.
	hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeGet)

	// Minimal query simply returns the list of all lists without JOIN subscriber counts. This is fast.
	minimal, _ := strconv.ParseBool(c.FormValue("minimal"))
	if minimal {
		status := c.FormValue("status")
		res, err := a.core.GetLists(c.Request().Context(), tenantID(c), "", status, hasAllPerm, permittedIDs)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		// Meta.
		total := len(res)
		out := models.PageResults{
			Results: res,
			Total:   total,
			Page:    1,
			PerPage: total,
		}

		return c.JSON(http.StatusOK, okResp{out})
	}

	// Full list query.
	var (
		query   = strings.TrimSpace(c.FormValue("query"))
		tags    = c.QueryParams()["tag"]
		orderBy = c.FormValue("order_by")
		typ     = c.FormValue("type")
		optin   = c.FormValue("optin")
		status  = c.FormValue("status")
		order   = c.FormValue("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)
	res, total, err := a.core.QueryLists(c.Request().Context(), tenantID(c), query, typ, optin, status, tags, orderBy, order, hasAllPerm, permittedIDs, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Query:   query,
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetList retrieves a single list by id.
// It's permission checked by the listPerm middleware.
//
//	@ID			getList
//	@Summary		Get a list
//	@Tags			lists
//	@Produce		json
//	@Param			id	path		int	true	"List ID"
//	@Success		200	{object}	models.List
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/lists/{id} [get]
func (a *App) GetList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check if the user has access to the list.
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeGet, id); err != nil {
		return err
	}

	// Get the list from the DB.
	out, err := a.core.GetList(c.Request().Context(), tenantID(c), id, "")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateList handles list creation.
//
//	@ID			createList
//	@Summary		Create a list
//	@Tags			lists
//	@Accept			json
//	@Produce		json
//	@Param			list	body		models.List	true	"List to create"
//	@Success		200	{object}	models.List
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/lists [post]
func (a *App) CreateList(c echo.Context) error {
	l := models.List{}
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	out, err := a.core.CreateList(c.Request().Context(), tenantID(c), l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateList handles list modification.
// It's permission checked by the listPerm middleware.
//
//	@ID			updateList
//	@Summary		Update a list
//	@Tags			lists
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"List ID"
//	@Param			list	body		models.List	true	"List fields to update"
//	@Success		200	{object}	models.List
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/lists/{id} [put]
func (a *App) UpdateList(c echo.Context) error {
	// Get the authenticated user.
	user := auth.GetUser(c)

	// Check if the user has access to the list.
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	// Incoming params.
	var l models.List
	if err := c.Bind(&l); err != nil {
		return err
	}

	// Validate.
	if !strHasLen(l.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("lists.invalidName"))
	}

	// Update the list in the DB.
	out, err := a.core.UpdateList(c.Request().Context(), tenantID(c), id, l)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteList deletes a single list by ID.
//
//	@ID			deleteList
//	@Summary		Delete a list
//	@Tags			lists
//	@Produce		json
//	@Param			id	path		int	true	"List ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/lists/{id} [delete]
func (a *App) DeleteList(c echo.Context) error {
	id := getID(c)

	// Check if the user has manage permission for the list.
	user := auth.GetUser(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	// Delete the list from the DB.
	// Pass getAll=true since we've already verified permissions above.
	if err := a.core.DeleteLists(c.Request().Context(), tenantID(c), []int{id}, "", true, nil); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteLists deletes multiple lists by IDs or by query.
//
//	@ID			deleteLists
//	@Summary		Delete lists (bulk)
//	@Tags			lists
//	@Produce		json
//	@Param			id		query		[]int	false	"List IDs"
//	@Param			query	query		string	false	"SQL-like filter query"
//	@Param			all		query		bool	false	"Delete all lists matching the query"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/lists [delete]
func (a *App) DeleteLists(c echo.Context) error {
	user := auth.GetUser(c)

	var (
		ids   []int
		query string
		all   bool
	)

	// Check for IDs in query params.
	if len(c.Request().URL.Query()["id"]) > 0 {
		var err error
		ids, err = parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.errorInvalidIDs", "error", err.Error()))
		}
	} else {
		// Check for query param.
		query = strings.TrimSpace(c.FormValue("query"))
		all = c.FormValue("all") == "true"
	}

	// Validate that either IDs or query is provided.
	if len(ids) == 0 && (query == "" && !all) {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.errorInvalidIDs", "error", "id or query required"))
	}

	// For ID deletion, check if the user has manage permission for the specific lists.
	if len(ids) > 0 {
		if err := user.HasListPerm(auth.PermTypeManage, ids...); err != nil {
			return err
		}

		// Delete the lists from the DB.
		// Pass getAll=true since we've already verified permissions above.
		if err := a.core.DeleteLists(c.Request().Context(), tenantID(c), ids, "", true, nil); err != nil {
			return err
		}
	} else {
		// For query deletion, get the list IDs the user has manage permission for.
		hasAllPerm, permittedIDs := user.GetPermittedLists(auth.PermTypeManage)

		// Delete the lists from the DB with permission filtering.
		if err := a.core.DeleteLists(c.Request().Context(), tenantID(c), nil, query, hasAllPerm, permittedIDs); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}
