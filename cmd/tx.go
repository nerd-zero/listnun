package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/internal/manager"
	"github.com/nerd-zero/listnun/models"
)

// txSendAsHeader lets a signed-in user's test send be attributed to an API
// key in the message log (Developers > Send test). It's ignored on
// requests that authenticated with a key or API user.
const txSendAsHeader = "Listnun-Send-As"

// txOrigin is who sent a transactional message and from where, for the log.
type txOrigin struct {
	APIKeyID int
	Sender   string
	SourceIP string
	// Accepted is the first event's detail, e.g. "POST /api/tx from 41.79.x.x".
	Accepted string
}

// txPayload is the single-recipient request kept in the message log so a
// message can be inspected and resent. Attachment contents aren't kept.
type txPayload struct {
	SubscriberMode  string         `json:"subscriber_mode"`
	SubscriberEmail string         `json:"subscriber_email,omitempty"`
	SubscriberID    int            `json:"subscriber_id,omitempty"`
	TemplateID      int            `json:"template_id"`
	Data            map[string]any `json:"data"` // Arbitrary template data supplied by the caller.
	FromEmail       string         `json:"from_email,omitempty"`
	Headers         models.Headers `json:"headers,omitempty"`
	ContentType     string         `json:"content_type,omitempty"`
	Messenger       string         `json:"messenger,omitempty"`
	Subject         string         `json:"subject,omitempty"`
	AltBody         string         `json:"altbody,omitempty"`
	Attachments     []string       `json:"attachments,omitempty"`
}

// txLogRef identifies a recorded message: the public ID and its row.
type txLogRef struct {
	UUID string
	ID   int64
}

// SendTxMessage handles the sending of a transactional message.
//
// Every recipient is recorded in the message log. The response carries the
// log IDs in the Listnun-Message-Id header, comma-separated in recipient
// order; the body is unchanged.
func (a *App) SendTxMessage(c echo.Context) error {
	m, err := a.parseTxRequest(c)
	if err != nil {
		return err
	}

	ids, err := a.sendTx(c.Request().Context(), tenantID(c), m, a.txOriginFor(c))
	if len(ids) > 0 {
		c.Response().Header().Set(models.TxMessageIDHeader, strings.Join(ids, ","))
	}
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// parseTxRequest reads a JSON or multipart (with file attachments) tx request.
func (a *App) parseTxRequest(c echo.Context) (models.TxMessage, error) {
	var m models.TxMessage

	if !strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if err := c.Bind(&m); err != nil {
			return m, err
		}
		return m, nil
	}

	form, err := c.MultipartForm()
	if err != nil {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidFields", "name", err.Error()))
	}

	data, ok := form.Value["data"]
	if !ok || len(data) != 1 {
		return m, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "data"))
	}

	// Parse the JSON data.
	if err := json.Unmarshal([]byte(data[0]), &m); err != nil {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("data: %s", err.Error())))
	}

	// Attach files.
	for _, f := range form.File["file"] {
		file, err := f.Open()
		if err != nil {
			return m, echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("file: %s", err.Error())))
		}
		b, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return m, echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("globals.messages.invalidFields", "name", fmt.Sprintf("file: %s", err.Error())))
		}

		m.Attachments = append(m.Attachments, models.Attachment{
			Name:    f.Filename,
			Header:  manager.MakeAttachmentHeader(f.Filename, "base64", f.Header.Get("Content-Type")),
			Content: b,
		})
	}

	return m, nil
}

// txOriginFor describes the request for the message log: the API key or
// user that sent it and the caller's IP.
func (a *App) txOriginFor(c echo.Context) txOrigin {
	user := auth.GetUser(c)
	ip := c.RealIP()
	o := txOrigin{
		APIKeyID: user.APIKeyID,
		Sender:   user.Username,
		SourceIP: ip,
		Accepted: fmt.Sprintf("%s %s from %s", c.Request().Method, c.Request().URL.Path, ip),
	}

	// A signed-in user testing from the browser may send as one of the
	// workspace's keys, so the log shows the key the test was for.
	if user.Type == auth.UserTypeAPI {
		return o
	}
	name := strings.TrimSpace(c.Request().Header.Get(txSendAsHeader))
	if name == "" {
		return o
	}
	keys, err := a.core.GetAPIKeys(c.Request().Context(), tenantID(c), name)
	if err != nil {
		return o
	}
	for _, k := range keys {
		if k.Name == name && k.Status == models.APIKeyStatusActive {
			o.APIKeyID = k.ID
			o.Sender = k.Name
			break
		}
	}
	return o
}

// sendTx validates, renders, logs and queues a transactional message for
// each of its recipients. It returns the log ID of every recipient it
// recorded, including ones that failed.
func (a *App) sendTx(ctx context.Context, tenantID int, m models.TxMessage, origin txOrigin) ([]string, error) {
	// Validate fields.
	m, err := a.validateTxMessage(ctx, tenantID, m)
	if err != nil {
		return nil, err
	}

	// Get the cached tx template. A template that doesn't exist still leaves
	// a failed entry per recipient in the message log, so the mistake shows
	// up there and not only in the caller's error handling.
	tpl, err := a.manager.GetTpl(tenantID, m.TemplateID)
	if err != nil {
		msg := a.i18n.Ts("globals.messages.notFound", "name", fmt.Sprintf("template %d", m.TemplateID))
		var ids []string
		for _, r := range m.Recipients() {
			if ref := a.logTx(ctx, tenantID, m, nil, origin, models.Subscriber{Email: r.Label}, r.ID, msg); ref.UUID != "" {
				ids = append(ids, ref.UUID)
			}
		}
		return ids, echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	// The workspace logo, for templates that include the brand-logo block.
	m.Brand = a.brandBlock(ctx, tenantID)

	// The template's plain-text body applies unless the request brings its own.
	if m.AltBody == "" {
		m.AltBody = tpl.AltBody
	}

	var (
		num      = len(m.SubscriberEmails)
		isEmails = true
		ids      []string
	)
	if len(m.SubscriberIDs) > 0 {
		num = len(m.SubscriberIDs)
		isEmails = false
	}
	record := func(sub models.Subscriber, subID int, failure string) txLogRef {
		ref := a.logTx(ctx, tenantID, m, tpl, origin, sub, subID, failure)
		if ref.UUID != "" {
			ids = append(ids, ref.UUID)
		}
		return ref
	}

	notFound := []string{}
	for n := range num {
		var (
			sub      models.Subscriber
			subID    int
			subEmail string
		)
		if isEmails {
			subEmail = m.SubscriberEmails[n]
		} else {
			subID = m.SubscriberIDs[n]
		}

		if m.SubscriberMode == models.TxSubModeExternal {
			// `external`: Always create an ephemeral "subscriber" and don't
			// lookup in the DB.
			sub = models.Subscriber{Email: subEmail}
		} else {
			// Default/fallback mode: lookup subscriber in DB.
			sub, err = a.core.GetSubscriber(ctx, tenantID, subID, "", subEmail)
			if err != nil {
				er, ok := err.(*echo.HTTPError)
				if !ok || er.Code != http.StatusBadRequest {
					return ids, err
				}

				if m.SubscriberMode != models.TxSubModeFallback {
					// `default`: log error and continue.
					msg := fmt.Sprintf("%v", er.Message)
					notFound = append(notFound, msg)
					recipient := subEmail
					if recipient == "" {
						recipient = fmt.Sprintf("#%d", subID)
					}
					record(models.Subscriber{Email: recipient}, subID, msg)
					continue
				}

				// `fallback`: Create an ephemeral "subscriber" if the subscriber wasn't found.
				sub = models.Subscriber{Email: subEmail}
			}
		}

		// Render the message.
		if err := m.Render(sub, tpl, a.manager.GenericTemplateFuncs()); err != nil {
			msg := a.i18n.Ts("templates.errorRendering", "error", err.Error())
			record(sub, subID, msg)
			return ids, echo.NewHTTPError(http.StatusBadRequest, msg)
		}

		ref := record(sub, subID, "")

		// Prepare the final message.
		msg := models.Message{}
		msg.TenantID = tenantID
		msg.Subscriber = sub
		msg.To = []string{sub.Email}
		msg.From = m.FromEmail
		msg.Subject = m.Subject
		msg.ContentType = m.ContentType
		msg.Messenger = m.Messenger
		msg.Body = m.Body
		msg.AltBody = []byte(m.AltBody)
		msg.TxMessageID = ref.ID
		for _, a := range m.Attachments {
			msg.Attachments = append(msg.Attachments, models.Attachment{
				Name:     a.Name,
				Header:   a.Header,
				Content:  a.Content,
				IsInline: a.IsInline,
			})
		}
		msg.Attachments = append(msg.Attachments, tpl.Attachments...)

		msg.Headers = models.MergeHeaders(tpl.Headers, m.Headers)
		if ref.UUID != "" {
			msg.Headers.Set(models.TxMessageIDHeader, ref.UUID)
		}

		if err := a.manager.PushMessage(msg); err != nil {
			a.log.Printf("error sending message (%s): %v", msg.Subject, err)
			a.core.RecordTxSendResult(tenantID, ref.ID, m.Messenger, err)
			return ids, err
		}
	}

	if len(notFound) > 0 {
		return ids, echo.NewHTTPError(http.StatusBadRequest, strings.Join(notFound, "; "))
	}

	return ids, nil
}

// logTx records one recipient's message in the log. failure is set when
// the message won't be sent. A message that can't be recorded is still
// sent, so the returned ref is empty in that case.
func (a *App) logTx(ctx context.Context, tenantID int, m models.TxMessage, tpl *models.Template, origin txOrigin,
	sub models.Subscriber, subID int, failure string) txLogRef {
	p := txPayload{
		SubscriberMode: m.SubscriberMode,
		TemplateID:     m.TemplateID,
		Data:           m.Data,
		FromEmail:      m.FromEmail,
		Headers:        m.Headers,
		ContentType:    m.ContentType,
		Messenger:      m.Messenger,
		Subject:        m.Subject,
		AltBody:        m.AltBody,
	}
	if subID > 0 {
		p.SubscriberID = subID
	} else {
		p.SubscriberEmail = sub.Email
	}
	for _, att := range m.Attachments {
		p.Attachments = append(p.Attachments, att.Name)
	}
	payload, err := json.Marshal(p)
	if err != nil {
		payload = []byte("{}")
	}

	// tpl is nil when the requested template doesn't exist: the entry then
	// links to no template, and the payload keeps the ID that was asked for.
	var (
		tplID            int
		tplName, tplSubj string
	)
	if tpl != nil {
		tplID, tplName, tplSubj = m.TemplateID, tpl.Name, tpl.Subject
	}

	rec := models.NewTxLogMessage{
		UUID:         core.NewTxMessageUUID(),
		APIKeyID:     origin.APIKeyID,
		Sender:       origin.Sender,
		TemplateID:   tplID,
		TemplateName: tplName,
		SubscriberID: sub.ID,
		Recipient:    sub.Email,
		Subject:      core.TxLogSubject(m.Subject, tplSubj, failure != ""),
		Status:       models.TxStatusQueued,
		Messenger:    m.Messenger,
		SourceIP:     origin.SourceIP,
		Payload:      payload,
		Body:         string(m.Body),
		Events:       []models.NewTxLogEvent{{Type: models.TxEventAccepted, Detail: origin.Accepted}},
	}
	if failure != "" {
		rec.Status = models.TxStatusFailed
		rec.Error = failure
		rec.Body = ""
		rec.Events = append(rec.Events, models.NewTxLogEvent{Type: models.TxEventFailed, Detail: failure})
	} else {
		rec.Events = append(rec.Events, models.NewTxLogEvent{
			Type:   models.TxEventRendered,
			Detail: fmt.Sprintf("Template #%d with %d variables", m.TemplateID, len(m.Data)),
		})
	}

	id, err := a.core.CreateTxMessage(ctx, tenantID, rec)
	if err != nil {
		return txLogRef{}
	}
	return txLogRef{UUID: rec.UUID, ID: id}
}

// validateTxMessage validates the tx message fields.
func (a *App) validateTxMessage(ctx context.Context, tenantID int, m models.TxMessage) (models.TxMessage, error) {
	if len(m.SubscriberEmails) > 0 && m.SubscriberEmail != "" {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidFields", "name", "do not send `subscriber_email`"))
	}
	if len(m.SubscriberIDs) > 0 && m.SubscriberID != 0 {
		return m, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidFields", "name", "do not send `subscriber_id`"))
	}

	if m.SubscriberEmail != "" {
		m.SubscriberEmails = append(m.SubscriberEmails, m.SubscriberEmail)
	}

	if m.SubscriberID != 0 {
		m.SubscriberIDs = append(m.SubscriberIDs, m.SubscriberID)
	}

	// Validate subscriber_mode.
	if m.SubscriberMode == "" {
		m.SubscriberMode = models.TxSubModeDefault
	}

	switch m.SubscriberMode {
	case models.TxSubModeDefault:
		// Need subscriber_emails OR subscriber_ids, but not both.
		if (len(m.SubscriberEmails) == 0 && len(m.SubscriberIDs) == 0) || (len(m.SubscriberEmails) > 0 && len(m.SubscriberIDs) > 0) {
			return m, echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidFields", "name", "send subscriber_emails OR subscriber_ids"))
		}
	case models.TxSubModeFallback, models.TxSubModeExternal:
		// `fallback` and `external` can only use subscriber_emails.
		if len(m.SubscriberIDs) > 0 {
			return m, echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_ids not allowed in fallback or external mode"))
		}
		if len(m.SubscriberEmails) == 0 {
			return m, echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_emails"))
		}
	default:
		return m, echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_mode"))
	}

	imp, err := a.importers.Get(ctx, tenantID)
	if err != nil {
		return m, err
	}

	for n, email := range m.SubscriberEmails {
		if email != "" {
			em, err := imp.SanitizeEmail(email)
			if err != nil {
				return m, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			m.SubscriberEmails[n] = em
		}
	}

	if m.FromEmail == "" {
		m.FromEmail = a.tenantFromEmail(ctx, tenantID)
	}

	if m.Messenger == "" {
		m.Messenger = emailMsgr
	} else if !a.manager.HasMessenger(m.Messenger) {
		return m, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("campaigns.fieldInvalidMessenger", "name", m.Messenger))
	}

	return m, nil
}
