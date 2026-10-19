package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

// brandingKey is the settings row that holds the workspace's email branding.
const brandingKey = "branding"

// GetBranding returns the workspace's email branding.
//
//	@ID			getBranding
//	@Summary		Get email branding
//	@Tags			developers
//	@Produce		json
//	@Success		200	{object}	models.Branding
//	@Router			/api/developers/branding [get]
func (a *App) GetBranding(c echo.Context) error {
	s, err := a.core.GetSettings(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	b := s.Branding
	if b.Width == 0 {
		b.Width = models.BrandWidthDefault
	}
	if b.Align == "" {
		b.Align = models.BrandAlignLeft
	}
	return c.JSON(http.StatusOK, okResp{b})
}

// UpdateBranding saves the workspace's email branding. It applies to the
// next message sent, without a restart.
//
//	@ID			updateBranding
//	@Summary		Save email branding
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.Branding	true	"Branding"
//	@Success		200		{object}	models.Branding
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/developers/branding [put]
func (a *App) UpdateBranding(c echo.Context) error {
	var b models.Branding
	if err := c.Bind(&b); err != nil {
		return err
	}

	b.LinkURL = strings.TrimSpace(b.LinkURL)
	switch {
	case b.Width < models.BrandWidthMin || b.Width > models.BrandWidthMax:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("developers.branding.invalidWidth",
			"min", strconv.Itoa(models.BrandWidthMin), "max", strconv.Itoa(models.BrandWidthMax)))
	case b.Align != models.BrandAlignLeft && b.Align != models.BrandAlignCenter:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "align"))
	case !models.ValidBrandLink(b.LinkURL):
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.branding.invalidLink"))
	case !validLogoURL(b.Logo) || !validLogoURL(b.LogoDark):
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.branding.invalidLogo"))
	}

	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	if err := a.core.UpdateSettingsByKey(c.Request().Context(), tenantID(c), brandingKey, raw); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{b})
}

// validLogoURL accepts no logo, or one with an http(s) URL.
func validLogoURL(l *models.BrandLogo) bool {
	return l == nil || (l.URL != "" && models.ValidBrandLink(l.URL))
}

// brandBlock is the tenant's logo for the brand-logo template block, or nil
// when there's no logo or it can't be read (the message still sends).
func (a *App) brandBlock(ctx context.Context, tenantID int) *models.BrandBlock {
	s, err := a.core.GetSettings(ctx, tenantID)
	if err != nil {
		a.log.Printf("error reading branding for tenant %d: %v", tenantID, err)
		return nil
	}

	alt := s.AppSiteName
	if a.cfg.MultiTenancyEnabled {
		if name, err := a.core.GetTenantOrganizationName(tenantID); err == nil && name != "" {
			alt = name
		}
	}
	return s.Branding.Block(alt)
}
