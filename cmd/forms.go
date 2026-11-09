package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/nerd-zero/listnun/models"
)

const (
	// formStatsWindow is the period a form's figures cover.
	formStatsWindow = 30 * 24 * time.Hour

	maxFormName    = 200
	maxFormHeading = 200
	maxFormIntro   = 1000
	maxFormButton  = 100
	maxFormMessage = 1000
	maxFormValue   = 1000
)

// formReq creates or updates a form.
type formReq struct {
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	Heading        string            `json:"heading"`
	Intro          string            `json:"intro"`
	ListIDs        []int             `json:"list_ids"`
	Fields         models.FormFields `json:"fields"`
	ButtonLabel    string            `json:"button_label"`
	SuccessMessage string            `json:"success_message"`
	Captcha        bool              `json:"captcha"`
} // @name FormReq

// formResp is a form and the ways to share it.
type formResp struct {
	models.Form
	// URL is the form's hosted page.
	URL string `json:"url"`
	// EmbedHTML is a form to paste into a website.
	EmbedHTML string `json:"embed_html"`
	// PopupScript is a script tag that shows the form as a popup.
	PopupScript string `json:"popup_script"`
} // @name FormResp

// GetForms returns every signup form.
//
//	@ID			getForms
//	@Summary	Get signup forms
//	@Tags		forms
//	@Produce	json
//	@Success	200	{array}	formResp
//	@Router		/api/forms [get]
func (a *App) GetForms(c echo.Context) error {
	ctx, tID := c.Request().Context(), tenantID(c)
	forms, err := a.core.GetForms(ctx, tID)
	if err != nil {
		return err
	}
	lists, err := a.publicLists(ctx, tID)
	if err != nil {
		return err
	}

	out := make([]formResp, len(forms))
	for i, f := range forms {
		out[i] = a.formResp(c, f, lists)
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// GetFormStats returns a form's views, signups, confirmations and blocked
// submissions over the last 30 days.
//
//	@ID			getFormStats
//	@Summary	Get a form's figures
//	@Tags		forms
//	@Produce	json
//	@Param		id	path		int	true	"Form ID"
//	@Success	200	{object}	models.FormStats
//	@Router		/api/forms/{id}/stats [get]
func (a *App) GetFormStats(c echo.Context) error {
	out, err := a.core.GetFormStats(c.Request().Context(), tenantID(c), getID(c), time.Now().Add(-formStatsWindow))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateForm saves a new signup form.
//
//	@ID			createForm
//	@Summary	Create a signup form
//	@Tags		forms
//	@Accept		json
//	@Produce	json
//	@Param		data	body		formReq	true	"Form"
//	@Success	200		{object}	formResp
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/forms [post]
func (a *App) CreateForm(c echo.Context) error {
	ctx, tID := c.Request().Context(), tenantID(c)
	f, lists, err := a.validateForm(c)
	if err != nil {
		return err
	}
	out, err := a.core.CreateForm(ctx, tID, f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{a.formResp(c, out, lists)})
}

// UpdateForm saves changes to a signup form.
//
//	@ID			updateForm
//	@Summary	Update a signup form
//	@Tags		forms
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int		true	"Form ID"
//	@Param		data	body		formReq	true	"Form"
//	@Success	200		{object}	formResp
//	@Failure	400		{object}	echo.HTTPError
//	@Router		/api/forms/{id} [put]
func (a *App) UpdateForm(c echo.Context) error {
	ctx, tID := c.Request().Context(), tenantID(c)
	f, lists, err := a.validateForm(c)
	if err != nil {
		return err
	}
	out, err := a.core.UpdateForm(ctx, tID, getID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{a.formResp(c, out, lists)})
}

// DeleteForm deletes a signup form. Its page and embeds stop working.
//
//	@ID			deleteForm
//	@Summary	Delete a signup form
//	@Tags		forms
//	@Produce	json
//	@Param		id	path		int	true	"Form ID"
//	@Success	200	{object}	bool
//	@Router		/api/forms/{id} [delete]
func (a *App) DeleteForm(c echo.Context) error {
	if err := a.core.DeleteForm(c.Request().Context(), tenantID(c), getID(c)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// validateForm reads and checks a formReq. Lists must be the workspace's
// public lists; a live form needs at least one.
func (a *App) validateForm(c echo.Context) (models.Form, []models.List, error) {
	var req formReq
	if err := c.Bind(&req); err != nil {
		return models.Form{}, nil, err
	}

	bad := func(msg string) (models.Form, []models.List, error) {
		return models.Form{}, nil, echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	f := models.Form{
		Name:           strings.TrimSpace(req.Name),
		Status:         req.Status,
		Heading:        strings.TrimSpace(req.Heading),
		Intro:          strings.TrimSpace(req.Intro),
		Fields:         req.Fields,
		ButtonLabel:    strings.TrimSpace(req.ButtonLabel),
		SuccessMessage: strings.TrimSpace(req.SuccessMessage),
		Captcha:        req.Captcha,
	}
	switch {
	case f.Name == "" || len(f.Name) > maxFormName:
		return bad(a.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	case f.Status != models.FormStatusLive && f.Status != models.FormStatusDraft:
		return bad(a.i18n.Ts("globals.messages.invalidFields", "name", "status"))
	case len(f.Heading) > maxFormHeading, len(f.Intro) > maxFormIntro,
		len(f.ButtonLabel) > maxFormButton, len(f.SuccessMessage) > maxFormMessage:
		return bad(a.i18n.T("forms.textTooLong"))
	}
	if err := f.Fields.Validate(); err != nil {
		return bad(a.i18n.Ts("forms.invalidFields", "error", err.Error()))
	}

	lists, err := a.publicLists(c.Request().Context(), tenantID(c))
	if err != nil {
		return models.Form{}, nil, err
	}
	f.ListIDs = pq.Int64Array{}
	for _, id := range req.ListIDs {
		if !slices.ContainsFunc(lists, func(l models.List) bool { return l.ID == id }) {
			return bad(a.i18n.T("forms.onlyPublicLists"))
		}
		if !slices.Contains(f.ListIDs, int64(id)) {
			f.ListIDs = append(f.ListIDs, int64(id))
		}
	}
	if f.Status == models.FormStatusLive && len(f.ListIDs) == 0 {
		return bad(a.i18n.T("forms.needList"))
	}

	return f, lists, nil
}

// publicLists returns the workspace's active public lists, the only ones a
// form can subscribe people to.
func (a *App) publicLists(ctx context.Context, tID int) ([]models.List, error) {
	return a.core.GetLists(ctx, tID, models.ListTypePublic, models.ListStatusActive, true, nil)
}

// formLists returns f's lists that are still public, in f's order.
func formLists(f models.Form, public []models.List) []models.List {
	out := []models.List{}
	for _, id := range f.ListIDs {
		if i := slices.IndexFunc(public, func(l models.List) bool { return int64(l.ID) == id }); i >= 0 {
			out = append(out, public[i])
		}
	}
	return out
}

func (a *App) formURL(c echo.Context, f models.Form) string {
	return fmt.Sprintf("%s/f/%s", a.requestRootURL(c), f.Slug)
}

// formResp adds a form's sharing snippets.
func (a *App) formResp(c echo.Context, f models.Form, public []models.List) formResp {
	out := formResp{Form: f, URL: a.formURL(c, f)}

	var b bytes.Buffer
	data := formPageData{Form: f, Lists: formLists(f, public), Action: out.URL, Root: a.requestRootURL(c)}
	if f.Captcha {
		data.Captcha = a.captchaWidget()
	}
	if err := embedTpl.Execute(&b, data); err != nil {
		a.log.Printf("error rendering form embed: %v", err)
	}
	out.EmbedHTML = b.String()
	out.PopupScript = fmt.Sprintf(`<script src="%s/popup.js" async></script>`, out.URL)
	return out
}

// formPageData is what the hosted page and the embed snippet render.
type formPageData struct {
	publicTpl
	Form    models.Form
	Lists   []models.List
	Captcha captchaWidget
	Action  string
	// Root is the workspace's address, for the embed's captcha scripts.
	Root string
	// Embed renders the page without the workspace's header and footer,
	// for the popup's frame.
	Embed bool
	// Values and Error refill the form after a failed submission.
	Values map[string]string
	Error  string
}

// embedTpl is the HTML a website pastes in. It posts to the hosted page,
// which shows the result.
var embedTpl = template.Must(template.New("embed").Parse(`<form method="post" action="{{ .Action }}" class="listnun-form">
{{- if .Form.Heading }}
  <h3>{{ .Form.Heading }}</h3>
{{- end }}
{{- if .Form.Intro }}
  <p>{{ .Form.Intro }}</p>
{{- end }}
{{- range .Form.Fields }}
  <p>
    <label for="ln-{{ .Key }}">{{ .Label }}</label><br />
    <input id="ln-{{ .Key }}" name="{{ .Key }}" type="{{ if eq .Key "email" }}email{{ else }}text{{ end }}"{{ if .Required }} required{{ end }} />
  </p>
{{- end }}
  <input name="nonce" value="" tabindex="-1" autocomplete="off" aria-hidden="true" style="display:none" />
{{- if gt (len .Lists) 1 }}
{{- range .Lists }}
  <p><label><input type="checkbox" name="l" value="{{ .UUID }}" checked /> {{ .Name }}</label></p>
{{- end }}
{{- end }}
{{- if .Captcha.Enabled }}
{{- if eq .Captcha.Provider "turnstile" }}
  <div class="cf-turnstile" data-sitekey="{{ .Captcha.Key }}"></div>
  <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
{{- else if eq .Captcha.Provider "hcaptcha" }}
  <div class="h-captcha" data-sitekey="{{ .Captcha.Key }}"></div>
  <script src="https://js.hcaptcha.com/1/api.js" async defer></script>
{{- else if eq .Captcha.Provider "altcha" }}
  <altcha-widget challengeurl="{{ .Root }}/api/public/captcha/altcha"></altcha-widget>
  <script type="module" src="{{ .Root }}/public/static/altcha.umd.js" async defer></script>
{{- end }}
{{- end }}
  <button type="submit">{{ if .Form.ButtonLabel }}{{ .Form.ButtonLabel }}{{ else }}Subscribe{{ end }}</button>
</form>
`))

// liveForm returns the live form at :slug, or renders a not-found page.
func (a *App) liveForm(c echo.Context) (models.Form, bool) {
	f, err := a.core.GetFormBySlug(c.Request().Context(), tenantID(c), c.Param("slug"))
	if err != nil || f.Status != models.FormStatusLive {
		_ = c.Render(http.StatusNotFound, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.T("forms.notFound")))
		return f, false
	}
	return f, true
}

// formPage builds a form's page data.
func (a *App) formPage(c echo.Context, f models.Form) (formPageData, error) {
	public, err := a.publicLists(c.Request().Context(), tenantID(c))
	if err != nil {
		return formPageData{}, err
	}
	out := formPageData{
		Form:   f,
		Lists:  formLists(f, public),
		Action: a.formURL(c, f),
		Embed:  c.QueryParam("embed") == "1",
		Values: map[string]string{},
	}
	out.Title = f.Heading
	if out.Title == "" {
		out.Title = f.Name
	}
	out.Description = f.Intro
	if f.Captcha && a.captcha.IsEnabled() {
		out.Captcha = a.captchaWidget()
	}
	if out.Embed {
		out.Action += "?embed=1"
	}
	return out, nil
}

// FormPage shows a live form's hosted page.
func (a *App) FormPage(c echo.Context) error {
	f, ok := a.liveForm(c)
	if !ok {
		return nil
	}
	data, err := a.formPage(c, f)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingLists")))
	}

	if err := a.core.AddFormEvent(c.Request().Context(), tenantID(c), f.ID, models.FormEventView, 0); err != nil {
		a.log.Printf("error recording form view: %v", err)
	}
	return c.Render(http.StatusOK, "form-page", data)
}

// FormSubmit signs someone up through a live form.
func (a *App) FormSubmit(c echo.Context) error {
	f, ok := a.liveForm(c)
	if !ok {
		return nil
	}
	ctx, tID := c.Request().Context(), tenantID(c)

	data, err := a.formPage(c, f)
	if err != nil {
		return c.Render(http.StatusInternalServerError, tplMessage,
			makeMsgTpl(a.i18n.T("public.errorTitle"), "", a.i18n.Ts("public.errorFetchingLists")))
	}
	for _, fl := range f.Fields {
		data.Values[fl.Key] = c.FormValue(fl.Key)
	}

	// A filled honeypot, or a failed captcha, is a bot.
	blocked := c.FormValue("nonce") != "" || (data.Captcha.Enabled && !a.verifyCaptcha(c))
	if blocked {
		if err := a.core.AddFormEvent(ctx, tID, f.ID, models.FormEventBlocked, 0); err != nil {
			a.log.Printf("error recording blocked form submission: %v", err)
		}
		data.Error = a.i18n.T("public.invalidCaptcha")
		return c.Render(http.StatusBadRequest, "form-page", data)
	}

	sub, listUUIDs, msg := a.readFormSignup(c, f, data.Lists)
	if msg != "" {
		data.Error = msg
		return c.Render(http.StatusBadRequest, "form-page", data)
	}

	subID, hasOptin, err := a.subscribe(ctx, tID, sub, listUUIDs)
	if err != nil {
		data.Error = errMessage(err)
		return c.Render(http.StatusBadRequest, "form-page", data)
	}
	if err := a.core.AddFormEvent(ctx, tID, f.ID, models.FormEventSignup, subID); err != nil {
		a.log.Printf("error recording form signup: %v", err)
	}

	text := f.SuccessMessage
	if text == "" {
		text = a.i18n.T("public.subConfirmed")
		if hasOptin {
			text = a.i18n.T("public.subOptinPending")
		}
	}
	return c.Render(http.StatusOK, tplMessage, makeMsgTpl(a.i18n.T("public.subTitle"), "", text))
}

// readFormSignup reads a submitted form into a subscriber and the lists to
// join. The string is a message for the visitor when something's missing.
func (a *App) readFormSignup(c echo.Context, f models.Form, lists []models.List) (models.Subscriber, []string, string) {
	sub := models.Subscriber{Status: models.SubscriberStatusEnabled, Attribs: models.JSON{}}
	for _, fl := range f.Fields {
		v := strings.TrimSpace(c.FormValue(fl.Key))
		if len(v) > maxFormValue {
			return sub, nil, a.i18n.Ts("forms.fieldTooLong", "name", fl.Label)
		}
		if v == "" {
			if fl.Required {
				return sub, nil, a.i18n.Ts("forms.fieldRequired", "name", fl.Label)
			}
			continue
		}
		switch {
		case fl.Key == models.FormFieldEmail:
			sub.Email = v
		case fl.Key == models.FormFieldName:
			sub.Name = v
		case fl.IsAttrib():
			sub.Attribs[fl.Attrib()] = v
		}
	}

	// One list joins without asking; with several, the visitor picks.
	uuids := make([]string, 0, len(lists))
	picked := c.Request().Form["l"]
	for _, l := range lists {
		if len(lists) == 1 || slices.Contains(picked, l.UUID) {
			uuids = append(uuids, l.UUID)
		}
	}
	if len(uuids) == 0 {
		return sub, nil, a.i18n.T("public.noListsSelected")
	}
	return sub, uuids, ""
}

// errMessage is an error's message for a visitor.
func errMessage(err error) string {
	if he, ok := err.(*echo.HTTPError); ok {
		return fmt.Sprintf("%v", he.Message)
	}
	return err.Error()
}

// popupJS shows a form in a popup a little while after the page loads, at
// most once per visitor every popupEveryDays. Its verbs are the form's
// page, a storage key for when the visitor saw it, and the two constants.
const popupJS = `(function () {
  var url = %q, key = %q, every = %d * 864e5, delay = %d;
  try { var seen = +localStorage.getItem(key); if (seen && Date.now() - seen < every) return; } catch (e) {}
  setTimeout(function () {
    try { localStorage.setItem(key, String(Date.now())); } catch (e) {}
    var bg = document.createElement('div');
    bg.setAttribute('role', 'dialog');
    bg.setAttribute('aria-modal', 'true');
    bg.style.cssText = 'position:fixed;inset:0;z-index:2147483647;background:rgba(20,23,28,.55);display:flex;align-items:center;justify-content:center;padding:16px';
    var box = document.createElement('div');
    box.style.cssText = 'position:relative;width:100%%;max-width:440px;height:min(640px,90vh);background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 20px 60px rgba(0,0,0,.3)';
    var frame = document.createElement('iframe');
    frame.src = url + '?embed=1';
    frame.title = 'Sign up';
    frame.style.cssText = 'border:0;width:100%%;height:100%%';
    var close = document.createElement('button');
    close.type = 'button';
    close.setAttribute('aria-label', 'Close');
    close.innerHTML = '&times;';
    close.style.cssText = 'position:absolute;top:8px;right:8px;width:32px;height:32px;border:0;border-radius:16px;background:rgba(0,0,0,.06);font:20px/1 sans-serif;cursor:pointer';
    function shut() { bg.remove(); document.removeEventListener('keydown', esc); }
    function esc(e) { if (e.key === 'Escape') shut(); }
    close.onclick = shut;
    bg.onclick = function (e) { if (e.target === bg) shut(); };
    document.addEventListener('keydown', esc);
    box.appendChild(frame); box.appendChild(close); bg.appendChild(box);
    document.body.appendChild(bg);
    close.focus();
  }, delay);
})();
`

const (
	// popupEveryDays is how long a visitor who saw the popup isn't shown it
	// again.
	popupEveryDays = 30
	// popupDelayMS is how long after the page loads the popup appears.
	popupDelayMS = 8000
)

// FormPopup serves the script that shows a live form as a popup.
func (a *App) FormPopup(c echo.Context) error {
	f, err := a.core.GetFormBySlug(c.Request().Context(), tenantID(c), c.Param("slug"))
	c.Response().Header().Set(echo.HeaderContentType, "application/javascript; charset=utf-8")
	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=300")
	if err != nil || f.Status != models.FormStatusLive {
		return c.String(http.StatusOK, "/* this form isn't published */\n")
	}
	key := "listnun-popup-" + url.PathEscape(f.UUID)
	return c.String(http.StatusOK, fmt.Sprintf(popupJS, a.formURL(c, f), key, popupEveryDays, popupDelayMS))
}
