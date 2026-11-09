package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/segments"
	"github.com/nerd-zero/listnun/models"
)

// GetSegments returns the workspace's saved segments by name.
func (c *Core) GetSegments(ctx context.Context, tenantID int) ([]models.SavedSegment, error) {
	out := []models.SavedSegment{}
	err := c.WithTenant(ctx, tenantID, &sql.TxOptions{ReadOnly: true}, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetSegments).Select(&out)
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{subscribers.segments.segments}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// CreateSegment saves a new segment, checking its conditions compile.
func (c *Core) CreateSegment(ctx context.Context, tenantID int, s models.SavedSegment, createdBy string) (models.SavedSegment, error) {
	conds, err := c.segmentJSON(s)
	if err != nil {
		return models.SavedSegment{}, err
	}
	var out models.SavedSegment
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateSegment).Get(&out, tenantID, s.Name, matchOrAll(s.Match), conds, createdBy)
	})
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{subscribers.segments.segment}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// UpdateSegment renames a segment and replaces its conditions.
func (c *Core) UpdateSegment(ctx context.Context, tenantID, id int, s models.SavedSegment) (models.SavedSegment, error) {
	conds, err := c.segmentJSON(s)
	if err != nil {
		return models.SavedSegment{}, err
	}
	var out models.SavedSegment
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.UpdateSegment).Get(&out, id, s.Name, matchOrAll(s.Match), conds)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return out, echo.NewHTTPError(http.StatusNotFound, c.i18n.Ts("globals.messages.notFound", "name", "{subscribers.segments.segment}"))
	}
	if err != nil {
		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{subscribers.segments.segment}", "error", pqErrMsg(err)))
	}
	return out, nil
}

// DeleteSegment deletes a saved segment. Its people are untouched.
func (c *Core) DeleteSegment(ctx context.Context, tenantID, id int) error {
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteSegment).Exec(id)
		return err
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{subscribers.segments.segment}", "error", pqErrMsg(err)))
	}
	return nil
}

// CountSegment counts the people in a segment, among those on listIDs
// when the user can only see some lists (nil means all).
func (c *Core) CountSegment(ctx context.Context, tenantID int, s segments.Segment, listIDs []int) (int, error) {
	exp, err := c.CompileSegment(s)
	if err != nil {
		return 0, err
	}
	if exp == "TRUE" {
		exp = ""
	}
	// The count query reads a NULL list filter as matching no one.
	if listIDs == nil {
		listIDs = []int{}
	}
	return c.getSubscriberCount(ctx, tenantID, "", exp, "", listIDs)
}

// CompileSegment returns a segment's SQL condition, or a 400 when it
// isn't valid.
func (c *Core) CompileSegment(s segments.Segment) (string, error) {
	exp, err := segments.Compile(s)
	if err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("subscribers.segments.invalid", "error", err.Error()))
	}
	return exp, nil
}

// segmentJSON validates a segment to save and returns its conditions as
// JSON.
func (c *Core) segmentJSON(s models.SavedSegment) ([]byte, error) {
	if s.Name == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}
	if _, err := c.CompileSegment(s.Segment()); err != nil {
		return nil, err
	}
	if s.Conditions == nil {
		s.Conditions = models.SegmentConditions{}
	}
	b, err := json.Marshal(s.Conditions)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, c.i18n.Ts("subscribers.segments.invalid", "error", err.Error()))
	}
	return b, nil
}

func matchOrAll(m segments.Match) segments.Match {
	if m == "" {
		return segments.MatchAll
	}
	return m
}
