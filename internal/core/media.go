package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/media"
	"github.com/nerd-zero/listnun/models"
	"gopkg.in/volatiletech/null.v6"
)

// QueryMedia returns media entries optionally filtered by a query string.
func (c *Core) QueryMedia(ctx context.Context, tenantID int, provider string, s media.Store, query string, offset, limit int) ([]media.Media, int, error) {
	out := []media.Media{}

	if query != "" {
		query = strings.ToLower(query)
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.QueryMedia).Select(&out, fmt.Sprintf("%%%s%%", query), provider, offset, limit)
	})
	if err != nil {
		return out, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total

		for i := 0; i < len(out); i++ {
			out[i].URL = s.GetURL(out[i].Filename)

			if out[i].Thumb != "" {
				out[i].ThumbURL = null.String{Valid: true, String: s.GetURL(out[i].Thumb)}
			}
		}
	}

	return out, total, nil
}

// GetMedia returns a media item.
func (c *Core) GetMedia(ctx context.Context, tenantID int, id int, uuid, fileName string, s media.Store) (media.Media, error) {
	var uu any
	if uuid != "" {
		uu = uuid
	}

	var out media.Media
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetMedia).Get(&out, id, uu, fileName)
	})
	if err != nil {
		// If it's ` sql: no rows in result set`, return a 404.
		if err == sql.ErrNoRows {
			return out, ErrNotFound
		}

		return out, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	out.URL = s.GetURL(out.Filename)
	if out.Thumb != "" {
		out.ThumbURL = null.String{Valid: true, String: s.GetURL(out.Thumb)}
	}

	return out, nil
}

// InsertMedia inserts a new media file into the DB.
func (c *Core) InsertMedia(ctx context.Context, tenantID int, fileName, thumbName, contentType string, meta models.JSON, provider string, s media.Store) (media.Media, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Write to the DB.
	var newID int
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.InsertMedia).Get(&newID, uu, fileName, thumbName, contentType, provider, meta, tenantID)
	})
	if err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return media.Media{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return c.GetMedia(ctx, tenantID, newID, "", "", s)
}

// DeleteMedia deletes a given media item and returns the filename of the deleted item.
func (c *Core) DeleteMedia(ctx context.Context, tenantID int, id int) (string, error) {
	var fname string
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.DeleteMedia).Get(&fname, id)
	})
	if err != nil {
		c.log.Printf("error inserting uploaded file to db: %v", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.media}", "error", pqErrMsg(err)))
	}

	return fname, nil
}
