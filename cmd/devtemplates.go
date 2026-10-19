package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
	null "gopkg.in/volatiletech/null.v6"
)

// txTemplateReq is the body for creating or saving a transactional
// template in the Developers template editor.
type txTemplateReq struct {
	Name       string          `json:"name"`
	Subject    string          `json:"subject"`
	Body       string          `json:"body"`
	AltBody    string          `json:"alt_body"`
	Headers    models.Headers  `json:"headers"`
	SampleData json.RawMessage `json:"sample_data" swaggertype:"object"`
} // @name TxTemplateRequest

// txTemplateOut is a transactional template with its recent use.
type txTemplateOut struct {
	models.Template
	UsedThisWeek int `json:"used_this_week"`
} // @name TxTemplate

// txPreviewReq is a template body to render with sample data.
type txPreviewReq struct {
	Subject string         `json:"subject"`
	Body    string         `json:"body"`
	AltBody string         `json:"alt_body"`
	Data    map[string]any `json:"data"` // Arbitrary sample data, as a caller would send in `data`.
} // @name TxPreviewRequest

// txPreviewOut is a rendered transactional email.
type txPreviewOut struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
	From    string `json:"from"`
} // @name TxPreview

// GetTxTemplates lists transactional templates with their bodies, sample
// data and messages sent in the last 7 days.
//
//	@ID			listTxTemplates
//	@Summary		List transactional templates
//	@Tags			developers
//	@Produce		json
//	@Success		200	{object}	[]txTemplateOut
//	@Router			/api/developers/templates [get]
func (a *App) GetTxTemplates(c echo.Context) error {
	ctx := c.Request().Context()
	tpls, err := a.core.GetTemplates(ctx, tenantID(c), models.TemplateTypeTx, false)
	if err != nil {
		return err
	}
	usage, err := a.core.GetTxTemplateUsage(ctx, tenantID(c))
	if err != nil {
		return err
	}

	out := make([]txTemplateOut, 0, len(tpls))
	for _, t := range tpls {
		out = append(out, txTemplateOut{Template: t, UsedThisWeek: usage[t.ID]})
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateTxTemplate creates a transactional template.
//
//	@ID			createTxTemplate
//	@Summary		Create a transactional template
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		txTemplateReq	true	"Template"
//	@Success		200		{object}	models.Template
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/developers/templates [post]
func (a *App) CreateTxTemplate(c echo.Context) error {
	req, err := a.bindTxTemplate(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	out, err := a.core.CreateTemplate(ctx, tenantID(c), req.Name, models.TemplateTypeTx, req.Subject, []byte(req.Body), null.String{})
	if err != nil {
		return err
	}
	if err := a.core.UpdateTxTemplateExtras(ctx, tenantID(c), out.ID, req.SampleData, req.AltBody, req.Headers); err != nil {
		return err
	}

	return a.respondTxTemplate(c, out.ID)
}

// UpdateTxTemplate saves a transactional template.
//
//	@ID			updateTxTemplate
//	@Summary		Save a transactional template
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Template ID"
//	@Param			body	body		txTemplateReq	true	"Template"
//	@Success		200		{object}	models.Template
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/templates/{id} [put]
func (a *App) UpdateTxTemplate(c echo.Context) error {
	req, err := a.bindTxTemplate(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	id := getID(c)
	cur, err := a.core.GetTemplate(ctx, tenantID(c), id, true)
	if err != nil {
		return err
	}
	if cur.Type != models.TemplateTypeTx {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.templates.notTx"))
	}

	if _, err := a.core.UpdateTemplate(ctx, tenantID(c), id, req.Name, req.Subject, []byte(req.Body), null.String{}); err != nil {
		return err
	}
	if err := a.core.UpdateTxTemplateExtras(ctx, tenantID(c), id, req.SampleData, req.AltBody, req.Headers); err != nil {
		return err
	}

	return a.respondTxTemplate(c, id)
}

// PreviewTxTemplate renders a template body with sample data on the
// server, exactly as POST /api/tx would.
//
//	@ID			previewTxTemplate
//	@Summary		Preview a transactional template
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		txPreviewReq	true	"Template and sample data"
//	@Success		200		{object}	txPreviewOut
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/developers/templates/preview [post]
func (a *App) PreviewTxTemplate(c echo.Context) error {
	var req txPreviewReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	tpl := models.Template{Type: models.TemplateTypeTx, Subject: req.Subject, Body: req.Body}
	if err := tpl.Compile(a.manager.GenericTemplateFuncs()); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	m := models.TxMessage{Data: req.Data, AltBody: req.AltBody}
	if err := m.Render(dummySubscriber, &tpl, a.manager.GenericTemplateFuncs()); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{txPreviewOut{
		Subject: m.Subject,
		HTML:    string(m.Body),
		Text:    m.AltBody,
		From:    a.cfg.FromEmail,
	}})
}

// bindTxTemplate reads and validates a template editor request.
func (a *App) bindTxTemplate(c echo.Context) (txTemplateReq, error) {
	var req txTemplateReq
	if err := c.Bind(&req); err != nil {
		return req, err
	}
	req.Name = strings.TrimSpace(req.Name)

	if err := a.validateTemplate(models.Template{Name: req.Name, Type: models.TemplateTypeTx, Subject: req.Subject, Body: req.Body}); err != nil {
		return req, err
	}

	if len(req.SampleData) > 0 {
		var probe map[string]any
		if err := json.Unmarshal(req.SampleData, &probe); err != nil {
			return req, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.templates.invalidSample"))
		}
	}

	// Compile it now so a broken template is rejected before it's saved.
	tpl := models.Template{Type: models.TemplateTypeTx, Subject: req.Subject, Body: req.Body}
	if err := tpl.Compile(a.manager.GenericTemplateFuncs()); err != nil {
		return req, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return req, nil
}

// respondTxTemplate recaches a saved template and returns it.
func (a *App) respondTxTemplate(c echo.Context, id int) error {
	out, err := a.recacheTxTemplate(c.Request().Context(), tenantID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// recacheTxTemplate loads a transactional template from the DB, with its
// plain-text body and headers, and caches it for POST /api/tx.
func (a *App) recacheTxTemplate(ctx context.Context, tenantID, id int) (models.Template, error) {
	out, err := a.core.GetTemplate(ctx, tenantID, id, false)
	if err != nil {
		return out, err
	}
	if out.Type != models.TemplateTypeTx {
		return out, nil
	}

	tpl := out
	if err := tpl.Compile(a.manager.GenericTemplateFuncs()); err != nil {
		return out, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	a.manager.CacheTpl(tenantID, id, &tpl)
	return out, nil
}
