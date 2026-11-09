package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/media"
)

const (
	// mediaSizeBatch is how many unmeasured files the backfill sizes per
	// pass; mediaSizePause spaces passes out so S3 HEAD requests stay gentle.
	mediaSizeBatch = 200
	mediaSizePause = 2 * time.Second
)

// backfillMediaSizes measures files uploaded before sizes were recorded,
// in the background, until none are left. New uploads record their size,
// so this only has work to do once after the v6.28 upgrade.
func (a *App) backfillMediaSizes(ctx context.Context) {
	storeFor := func(tenantID int) (media.Store, string, error) {
		st, settings, err := a.media.Get(ctx, tenantID)
		if err != nil {
			return nil, "", err
		}
		return st, settings.UploadProvider, nil
	}

	total := 0
	for {
		rows, err := a.core.GetUnsizedMedia(mediaSizeBatch)
		if err != nil {
			a.log.Printf("error listing unmeasured media: %v", err)
			return
		}
		if len(rows) == 0 {
			if total > 0 {
				a.log.Printf("measured %d existing media files", total)
			}
			return
		}

		n, err := media.MeasureSizes(rows, storeFor, a.core.SetMediaSize)
		total += n
		if err != nil {
			a.log.Printf("error recording media sizes: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(mediaSizePause):
		}
	}
}

// GetMediaUsage returns the storage the workspace's media takes up.
//
//	@ID			getMediaUsage
//	@Summary	Get media storage usage
//	@Tags		media
//	@Produce	json
//	@Success	200	{object}	models.MediaUsage
//	@Failure	500	{object}	echo.HTTPError
//	@Router		/api/media/usage [get]
func (a *App) GetMediaUsage(c echo.Context) error {
	out, err := a.core.GetMediaUsage(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetOperatorTenantStorage returns a tenant's media storage usage, so the
// operator can show it and enforce plan limits.
//
//	@ID				getOperatorTenantStorage
//	@Summary		Get a tenant's media storage usage (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token.
//	@Tags			operator
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Tenant ID"
//	@Success		200	{object}	models.MediaUsage
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Router			/api/operator/tenants/{id}/storage [get]
func (a *App) GetOperatorTenantStorage(c echo.Context) error {
	id := getID(c)
	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	out, err := a.core.GetMediaUsage(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}
