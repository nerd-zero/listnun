package main

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/models"
)

const (
	// tplTag is the template tag that should be present in a template
	// as the placeholder for campaign bodies.
	tplTag = `{{ template "content" . }}`

	dummyTpl = `
		<p>Hi there</p>
		<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis et elit ac elit sollicitudin condimentum non a magna. Sed tempor mauris in facilisis vehicula. Aenean nisl urna, accumsan ac tincidunt vitae, interdum cursus massa. Interdum et malesuada fames ac ante ipsum primis in faucibus. Aliquam varius turpis et turpis lacinia placerat. Aenean id ligula a orci lacinia blandit at eu felis. Phasellus vel lobortis lacus. Suspendisse leo elit, luctus sed erat ut, venenatis fermentum ipsum. Donec bibendum neque quis.</p>

		<h3>Sub heading</h3>
		<p>Nam luctus dui non placerat mattis. Morbi non accumsan orci, vel interdum urna. Duis faucibus id nunc ut euismod. Curabitur et eros id erat feugiat fringilla in eget neque. Aliquam accumsan cursus eros sed faucibus.</p>

		<p>Here is a link to <a href="https://listmonk.app" target="_blank">listnun</a>.</p>`
)

var (
	regexpTplTag = regexp.MustCompile(`{{(\s+)?template\s+?"content"(\s+)?\.(\s+)?}}`)
)

// GetTemplate handles the retrieval of a template
//
//	@ID			getTemplate
//	@Summary		Get a template
//	@Tags			templates
//	@Produce		json
//	@Param			id		path		int		true	"Template ID"
//	@Param			no_body	query		bool	false	"Omit template body from response"
//	@Success		200		{object}	models.Template
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/templates/{id} [get]
func (a *App) GetTemplate(c echo.Context) error {
	// If no_body is true, blank out the body of the template from the response.
	noBody, _ := strconv.ParseBool(c.QueryParam("no_body"))

	// Get the template from the DB.
	id := getID(c)
	out, err := a.core.GetTemplate(c.Request().Context(), tenantID(c), id, noBody)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetTemplates handles retrieval of templates.
//
//	@ID			listTemplates
//	@Summary		List templates
//	@Tags			templates
//	@Produce		json
//	@Param			no_body	query		bool	false	"Omit template bodies from response"
//	@Success		200		{object}	[]models.Template
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/templates [get]
func (a *App) GetTemplates(c echo.Context) error {
	// If no_body is true, blank out the body of the template from the response.
	noBody, _ := strconv.ParseBool(c.QueryParam("no_body"))

	// Fetch templates from the DB.
	out, err := a.core.GetTemplates(c.Request().Context(), tenantID(c), "", noBody)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// PreviewTemplate renders the HTML preview of a template in the DB.
//
//	@ID			previewTemplate
//	@Summary		Preview a template
//	@Tags			templates
//	@Produce		html
//	@Param			id	path		int	true	"Template ID"
//	@Success		200	{string}	string
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/templates/{id}/preview [get]
func (a *App) PreviewTemplate(c echo.Context) error {
	// Fetch one template from the DB.
	id := getID(c)
	tpl, err := a.core.GetTemplate(c.Request().Context(), tenantID(c), id, false)
	if err != nil {
		return err
	}

	// Render the template.
	out, err := a.previewTemplate(tenantID(c), tpl)
	if err != nil {
		return err
	}

	return c.HTML(http.StatusOK, string(out))
}

// PreviewTemplateBody renders the HTML preview of a template given its type and body.
//
//	@ID			previewTemplateBody
//	@Summary		Preview a template body
//	@Tags			templates
//	@Accept			mpfd
//	@Produce		html
//	@Param			template_type	formData	string	false	"Template type (campaign, tx)"
//	@Param			body			formData	string	true	"Template body HTML"
//	@Success		200				{string}	string
//	@Failure		400				{object}	echo.HTTPError
//	@Router			/api/templates/preview [post]
func (a *App) PreviewTemplateBody(c echo.Context) error {
	tpl := models.Template{
		Type: c.FormValue("template_type"),
		Body: c.FormValue("body"),
	}

	// Body is posted with the request.
	if tpl.Type == "" {
		tpl.Type = models.TemplateTypeCampaign
	}

	if tpl.Type == models.TemplateTypeCampaign && !regexpTplTag.MatchString(tpl.Body) {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
	}

	// Render the template.
	out, err := a.previewTemplate(tenantID(c), tpl)
	if err != nil {
		return err
	}

	return c.HTML(http.StatusOK, string(out))
}

// CreateTemplate handles template creation.
//
//	@ID			createTemplate
//	@Summary		Create a template
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			template	body		models.Template	true	"Template to create"
//	@Success		200			{object}	models.Template
//	@Failure		400			{object}	echo.HTTPError
//	@Router			/api/templates [post]
func (a *App) CreateTemplate(c echo.Context) error {
	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}
	if err := a.validateTemplate(o); err != nil {
		return err
	}

	// Subject is only relevant for fixed tx templates. For campaigns,
	// the subject changes per campaign and is on models.Campaign.
	var funcs template.FuncMap
	if o.Type == models.TemplateTypeCampaign || o.Type == models.TemplateTypeCampaignVisual {
		o.Subject = ""
		funcs = a.manager.TemplateFuncs(&models.Campaign{TenantID: tenantID(c)})
	} else {
		funcs = a.manager.GenericTemplateFuncs()
	}

	// Compile the template and validate.
	if err := o.Compile(funcs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Create the template the in the DB.
	out, err := a.core.CreateTemplate(c.Request().Context(), tenantID(c), o.Name, o.Type, o.Subject, []byte(o.Body), o.BodySource)
	if err != nil {
		return err
	}

	// If it's a transactional template, cache it in the manager
	// to be used for arbitrary incoming tx message pushes.
	if o.Type == models.TemplateTypeTx {
		if _, err := a.recacheTxTemplate(c.Request().Context(), tenantID(c), out.ID); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateTemplate handles template modification.
//
//	@ID			updateTemplate
//	@Summary		Update a template
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int				true	"Template ID"
//	@Param			template	body		models.Template	true	"Updated template"
//	@Success		200			{object}	models.Template
//	@Failure		400			{object}	echo.HTTPError
//	@Failure		404			{object}	echo.HTTPError
//	@Router			/api/templates/{id} [put]
func (a *App) UpdateTemplate(c echo.Context) error {
	var o models.Template
	if err := c.Bind(&o); err != nil {
		return err
	}
	if err := a.validateTemplate(o); err != nil {
		return err
	}

	// Subject is only relevant for fixed tx templates. For campaigns,
	// the subject changes per campaign and is on models.Campaign.
	var funcs template.FuncMap
	if o.Type == models.TemplateTypeCampaign || o.Type == models.TemplateTypeCampaignVisual {
		o.Subject = ""
		funcs = a.manager.TemplateFuncs(&models.Campaign{TenantID: tenantID(c)})
	} else {
		funcs = a.manager.GenericTemplateFuncs()
	}

	// Compile the template and validate.
	if err := o.Compile(funcs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Update the template in the DB.
	id := getID(c)
	out, err := a.core.UpdateTemplate(c.Request().Context(), tenantID(c), id, o.Name, o.Subject, []byte(o.Body), o.BodySource)
	if err != nil {
		return err
	}

	// If it's a transactional template, cache it.
	if out.Type == models.TemplateTypeTx {
		if _, err := a.recacheTxTemplate(c.Request().Context(), tenantID(c), out.ID); err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, okResp{out})

}

// TemplateSetDefault sets a template as the default.
//
//	@ID			setDefaultTemplate
//	@Summary		Set default template
//	@Tags			templates
//	@Produce		json
//	@Param			id	path		int	true	"Template ID"
//	@Success		200	{object}	[]models.Template
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/templates/{id}/default [put]
func (a *App) TemplateSetDefault(c echo.Context) error {
	// Update the template in the DB.
	id := getID(c)
	if err := a.core.SetDefaultTemplate(c.Request().Context(), tenantID(c), id); err != nil {
		return err
	}

	return a.GetTemplates(c)
}

// DeleteTemplate handles template deletion.
//
//	@ID			deleteTemplate
//	@Summary		Delete a template
//	@Tags			templates
//	@Produce		json
//	@Param			id	path		int	true	"Template ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/templates/{id} [delete]
func (a *App) DeleteTemplate(c echo.Context) error {
	// Delete the template from the DB.
	id := getID(c)
	if err := a.core.DeleteTemplate(c.Request().Context(), tenantID(c), id); err != nil {
		return err
	}

	// Delete cached in-memory template.
	a.manager.DeleteTpl(id)

	return c.JSON(http.StatusOK, okResp{true})
}

// compileTemplate validates template fields.
func (a *App) validateTemplate(o models.Template) error {
	if !strHasLen(o.Name, 1, stdInputMaxLen) {
		return errors.New(a.i18n.T("campaigns.fieldInvalidName"))
	}

	if o.Type == models.TemplateTypeCampaign && !regexpTplTag.MatchString(o.Body) {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
	}

	if o.Type == models.TemplateTypeTx && strings.TrimSpace(o.Subject) == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.missingFields", "name", "subject"))
	}

	return nil
}

// previewTemplate renders the HTML preview of a template for a tenant.
func (a *App) previewTemplate(tenantID int, tpl models.Template) ([]byte, error) {
	var out []byte
	if tpl.Type == models.TemplateTypeCampaign || tpl.Type == models.TemplateTypeCampaignVisual {
		camp := models.Campaign{
			// Tracked links in the template are registered under this
			// tenant (links.tenant_id); without it the insert fails its FK.
			TenantID:     tenantID,
			UUID:         dummyUUID,
			Name:         a.i18n.T("templates.dummyName"),
			Subject:      a.i18n.T("templates.dummySubject"),
			FromEmail:    "dummy-campaign@listmonk.app",
			TemplateBody: tpl.Body,
			Body:         dummyTpl,
		}

		if err := camp.CompileTemplate(a.manager.TemplateFuncs(&camp)); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("templates.errorCompiling", "error", err.Error()))
		}

		// Render the message body.
		msg, err := a.manager.NewCampaignMessage(&camp, dummySubscriber)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("templates.errorRendering", "error", err.Error()))
		}
		out = msg.Body()
	} else {
		// Compile transactional template.
		if err := tpl.Compile(a.manager.GenericTemplateFuncs()); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		m := models.TxMessage{
			Subject: tpl.Subject,
		}

		// Render the message.
		if err := m.Render(dummySubscriber, &tpl, a.manager.GenericTemplateFuncs()); err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		out = m.Body
	}

	return out, nil
}

// templateTestReq sends a campaign template, as it is in the editor, to a
// few people.
type templateTestReq struct {
	// Type is campaign or campaign_visual.
	Type string `json:"type"`
	Body string `json:"body"`
	// Emails must be existing subscribers the user can see.
	Emails []string `json:"emails"`
} // @name TemplateTestReq

// maxTemplateTestEmails caps one test send; it's a check, not a campaign.
const maxTemplateTestEmails = 10

// TestTemplate sends the editor's campaign template, wrapped around the
// same sample campaign the preview shows, to the given subscribers.
//
//	@ID			testTemplate
//	@Summary		Send a test of a campaign template
//	@Tags			templates
//	@Accept			json
//	@Produce		json
//	@Param			test	body		templateTestReq	true	"Template body and who gets the test"
//	@Success		200		{object}	bool
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/templates/test [post]
func (a *App) TestTemplate(c echo.Context) error {
	var req templateTestReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Type == "" {
		req.Type = models.TemplateTypeCampaign
	}
	if req.Type != models.TemplateTypeCampaign && req.Type != models.TemplateTypeCampaignVisual {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}
	if !regexpTplTag.MatchString(req.Body) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.placeholderHelp", "placeholder", tplTag))
	}

	emails := make([]string, 0, len(req.Emails))
	for _, e := range req.Emails {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" && !slices.Contains(emails, e) {
			emails = append(emails, e)
		}
	}
	if len(emails) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("campaigns.noSubsToTest"))
	}
	if len(emails) > maxTemplateTestEmails {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("templates.testTooMany", "num", strconv.Itoa(maxTemplateTestEmails)))
	}

	// Only to existing subscribers this user may see, like a campaign test.
	ctx, tID := c.Request().Context(), tenantID(c)
	subs, err := a.core.GetSubscribersByEmail(ctx, tID, emails)
	if err != nil {
		return err
	}
	ids := make([]int, len(subs))
	for i, s := range subs {
		ids[i] = s.ID
	}
	if err := a.hasSubPerm(ctx, tID, auth.GetUser(c), ids); err != nil {
		return err
	}

	from := a.tenantFromEmail(ctx, tID)
	camp := models.Campaign{
		TenantID:     tID,
		UUID:         dummyUUID,
		Name:         a.i18n.T("templates.dummyName"),
		Subject:      a.i18n.T("templates.testSubject"),
		FromEmail:    from,
		Messenger:    emailMsgr,
		ContentType:  models.CampaignContentTypeRichtext,
		TemplateBody: req.Body,
		Body:         dummyTpl,
	}
	for _, s := range subs {
		if err := a.sendTestMessage(s, &camp); err != nil {
			a.log.Printf("error sending template test: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("campaigns.errorSendTest", "error", err.Error()))
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}
