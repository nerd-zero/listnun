package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/segments"
	"github.com/nerd-zero/listnun/models"
)

// maxSegmentCounts caps how many segments one count request covers: the
// Subscribers sidebar's built-in and saved segments.
const maxSegmentCounts = 30

// segmentCountsReq is a request to count several segments at once.
type segmentCountsReq struct {
	Segments []segments.Segment `json:"segments"`
}

// segmentSQLResp is a segment's compiled SQL condition.
type segmentSQLResp struct {
	SQL string `json:"sql"`
}

// GetSegments returns the workspace's saved segments.
//
//	@ID			getSegments
//	@Summary	Get saved segments
//	@Tags		subscribers
//	@Produce	json
//	@Success	200	{array}		models.SavedSegment
//	@Router		/api/segments [get]
func (a *App) GetSegments(c echo.Context) error {
	out, err := a.core.GetSegments(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateSegment saves a segment.
//
//	@ID			createSegment
//	@Summary	Save a segment
//	@Tags		subscribers
//	@Accept		json
//	@Produce	json
//	@Param		body	body		models.SavedSegment	true	"Name, match and conditions"
//	@Success	200		{object}	models.SavedSegment
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/segments [post]
func (a *App) CreateSegment(c echo.Context) error {
	var req models.SavedSegment
	if err := c.Bind(&req); err != nil {
		return err
	}
	user := auth.GetUser(c)
	name := user.Name
	if name == "" {
		name = user.Username
	}
	out, err := a.core.CreateSegment(c.Request().Context(), tenantID(c), req, name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateSegment renames a saved segment and replaces its conditions.
//
//	@ID			updateSegment
//	@Summary	Update a saved segment
//	@Tags		subscribers
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int					true	"Segment ID"
//	@Param		body	body		models.SavedSegment	true	"Name, match and conditions"
//	@Success	200		{object}	models.SavedSegment
//	@Failure	404		{object}	echo.HTTPError
//	@Router		/api/segments/{id} [put]
func (a *App) UpdateSegment(c echo.Context) error {
	var req models.SavedSegment
	if err := c.Bind(&req); err != nil {
		return err
	}
	out, err := a.core.UpdateSegment(c.Request().Context(), tenantID(c), getID(c), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteSegment deletes a saved segment. Its people are untouched.
//
//	@ID			deleteSegment
//	@Summary	Delete a saved segment
//	@Tags		subscribers
//	@Produce	json
//	@Param		id	path		int	true	"Segment ID"
//	@Success	200	{object}	okResp
//	@Router		/api/segments/{id} [delete]
func (a *App) DeleteSegment(c echo.Context) error {
	if err := a.core.DeleteSegment(c.Request().Context(), tenantID(c), getID(c)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// CountSegments counts the people in each of several segments, among the
// lists the user can see, in the order given.
//
//	@ID			countSegments
//	@Summary	Count the people in segments
//	@Tags		subscribers
//	@Accept		json
//	@Produce	json
//	@Param		body	body	segmentCountsReq	true	"Segments to count"
//	@Success	200		{array}	int
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/segments/count [post]
func (a *App) CountSegments(c echo.Context) error {
	var req segmentCountsReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.Segments) > maxSegmentCounts {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "segments"))
	}

	listIDs, err := a.filterListQueryByPerm("list_id", url.Values{}, auth.GetUser(c))
	if err != nil {
		return err
	}
	out := make([]int, len(req.Segments))
	for i, s := range req.Segments {
		if out[i], err = a.core.CountSegment(c.Request().Context(), tenantID(c), s, listIDs); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// SegmentSQL returns the SQL condition a segment compiles to, for "Edit as
// SQL".
//
//	@ID			segmentSQL
//	@Summary	Get a segment's SQL condition
//	@Tags		subscribers
//	@Accept		json
//	@Produce	json
//	@Param		body	body		segments.Segment	true	"Segment"
//	@Success	200		{object}	segmentSQLResp
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/segments/sql [post]
func (a *App) SegmentSQL(c echo.Context) error {
	var req segments.Segment
	if err := c.Bind(&req); err != nil {
		return err
	}
	exp, err := a.core.CompileSegment(req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{segmentSQLResp{SQL: exp}})
}

// segmentQuery compiles QuerySubscribers' optional segment parameter, a
// JSON segment, to a SQL condition ("" when absent).
func (a *App) segmentQuery(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	var s segments.Segment
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("subscribers.segments.invalid", "error", err.Error()))
	}
	exp, err := a.core.CompileSegment(s)
	if err != nil || exp == "TRUE" {
		return "", err
	}
	return exp, nil
}

// withSegment ANDs a segment's compiled condition onto query (either may
// be empty).
func (a *App) withSegment(query string, s *segments.Segment) (string, error) {
	if s == nil {
		return query, nil
	}
	exp, err := a.core.CompileSegment(*s)
	if err != nil {
		return "", err
	}
	if exp == "TRUE" {
		return query, nil
	}
	return andSQL(query, exp), nil
}

// andSQL joins two SQL conditions with AND, either of which may be empty.
func andSQL(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return "(" + a + ") AND " + b
}
