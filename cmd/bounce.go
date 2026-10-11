package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/bounce"
	"github.com/nerd-zero/listnun/models"
)

// GetBounce handles retrieval of a specific bounce record by ID.
//
//	@ID			getBounce
//	@Summary		Get a bounce record
//	@Tags			bounces
//	@Produce		json
//	@Param			id	path		int	true	"Bounce ID"
//	@Success		200	{object}	models.Bounce
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/bounces/{id} [get]
func (a *App) GetBounce(c echo.Context) error {
	// Fetch one bounce from the DB.
	id := getID(c)
	out, err := a.core.GetBounce(c.Request().Context(), tenantID(c), id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetBounces handles retrieval of bounce records.
//
//	@ID			listBounces
//	@Summary		List bounce records
//	@Tags			bounces
//	@Produce		json
//	@Param			campaign_id	query		int		false	"Filter by campaign ID"
//	@Param			source		query		string	false	"Filter by bounce source"
//	@Param			order_by	query		string	false	"Field to order results by"
//	@Param			order		query		string	false	"Sort order (ASC or DESC)"
//	@Param			page		query		int		false	"Page number"
//	@Param			per_page	query		int		false	"Results per page"
//	@Success		200			{object}	models.PageResults
//	@Failure		500			{object}	echo.HTTPError
//	@Router			/api/bounces [get]
func (a *App) GetBounces(c echo.Context) error {
	var (
		campID, _ = strconv.Atoi(c.QueryParam("campaign_id"))
		source    = c.FormValue("source")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Query and fetch bounces from the DB.
	res, total, err := a.core.QueryBounces(c.Request().Context(), tenantID(c), campID, 0, source, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	// No results.
	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.Bounce{}}})
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberBounces retrieves a subscriber's bounce records.
//
//	@ID			getSubscriberBounces
//	@Summary		Get bounces for a subscriber
//	@Tags			bounces
//	@Produce		json
//	@Param			id	path		int	true	"Subscriber ID"
//	@Success		200	{object}	[]models.Bounce
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/subscribers/{id}/bounces [get]
func (a *App) GetSubscriberBounces(c echo.Context) error {
	// Query and fetch bounces from the DB.
	subID := getID(c)
	out, _, err := a.core.QueryBounces(c.Request().Context(), tenantID(c), 0, subID, "", "", "", 0, 1000)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteBounces handles bounce deletion of a list.
//
//	@ID			deleteBounces
//	@Summary		Delete bounce records
//	@Tags			bounces
//	@Produce		json
//	@Param			all	query		bool	false	"Delete all bounces"
//	@Param			id	query		[]int	false	"Bounce IDs to delete"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Router			/api/bounces [delete]
func (a *App) DeleteBounces(c echo.Context) error {
	all, _ := strconv.ParseBool(c.QueryParam("all"))

	var ids []int
	if !all {
		// There are multiple IDs in the query string.
		res, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}
		if len(res) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
		}

		ids = res
	}

	// Delete bounces from the DB.
	if err := a.core.DeleteBounces(c.Request().Context(), tenantID(c), ids, all); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteBounce handles bounce deletion of a single bounce record.
//
//	@ID			deleteBounce
//	@Summary		Delete a bounce record
//	@Tags			bounces
//	@Produce		json
//	@Param			id	path		int	true	"Bounce ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/bounces/{id} [delete]
func (a *App) DeleteBounce(c echo.Context) error {
	// Delete bounces from the DB.
	id := getID(c)
	if err := a.core.DeleteBounces(c.Request().Context(), tenantID(c), []int{id}, false); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistBouncedSubscribers handles blocklisting of all bounced subscribers.
//
//	@ID			blocklistBouncedSubscribers
//	@Summary		Blocklist all bounced subscribers
//	@Tags			bounces
//	@Produce		json
//	@Success		200
//	@Failure		500	{object}	echo.HTTPError
//	@Router			/api/bounces/blocklist [put]
func (a *App) BlocklistBouncedSubscribers(c echo.Context) error {
	if err := a.core.BlocklistBouncedSubscribers(c.Request().Context(), tenantID(c)); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BounceWebhook handles incoming bounce webhook notifications from various providers.
//
//	@ID			receiveBounceWebhook
//	@Summary		Receive a bounce webhook
//	@Tags			bounces
//	@Accept			json
//	@Produce		json
//	@Param			bounce	body		models.Bounce	false	"Bounce payload (native webhook only)"
//	@Success		200
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError	"Bounce webhooks are disabled for this tenant"
//	@Router			/webhooks/bounce [post]
func (a *App) BounceWebhook(c echo.Context) error {
	// Webhooks are configured per tenant: use the requesting tenant's own
	// settings and credentials, not the boot-time (tenant 1) bounce manager.
	tID := tenantID(c)
	hooks, enabled, err := a.tenantBounceWebhooks(c.Request().Context(), tID)
	if err != nil {
		return err
	}
	if !enabled {
		return echo.NewHTTPError(http.StatusNotFound, a.i18n.Ts("globals.messages.notFound", "name", "webhook"))
	}

	// Read the request body instead of using c.Bind() to read to save the entire raw request as meta.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading ses notification body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	var (
		service = c.Param("service")

		bounces []models.Bounce
	)
	switch true {
	// Native internal webhook.
	case service == "":
		var b models.Bounce
		if err := json.Unmarshal(rawReq, &b); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidData")+":"+err.Error())
		}

		if bv, err := a.validateBounceFields(c.Request().Context(), tenantID(c), b); err != nil {
			return err
		} else {
			b = bv
		}

		if len(b.Meta) == 0 {
			b.Meta = json.RawMessage("{}")
		}

		if b.CreatedAt.Year() == 0 {
			b.CreatedAt = time.Now()
		}

		bounces = append(bounces, b)

	// Amazon SES.
	case service == "ses" && hooks.SES != nil:
		switch c.Request().Header.Get("X-Amz-Sns-Message-Type") {
		// SNS webhook registration confirmation. Only after these are processed will the endpoint
		// start getting bounce notifications.
		case "SubscriptionConfirmation", "UnsubscribeConfirmation":
			if err := hooks.SES.ProcessSubscription(rawReq); err != nil {
				a.log.Printf("error processing SNS (SES) subscription: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}

		// Bounce notification.
		case "Notification":
			b, err := hooks.SES.ProcessBounce(rawReq)
			if err != nil {
				a.log.Printf("error processing SES notification: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}
			bounces = append(bounces, b)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

	// Azure ACS through Event Grid.
	case service == "azure" && hooks.Azure != nil:
		switch c.Request().Header.Get("aeg-event-type") {
		// Event Grid webhook registration validation.
		case "SubscriptionValidation", "SubscriptionValidationEvent":
			res, err := hooks.Azure.ProcessSubscription(rawReq)
			if err != nil {
				a.log.Printf("error processing Azure Event Grid subscription validation: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}
			return c.JSONBlob(http.StatusOK, res)

		// Regular event delivery.
		case "", "Notification":
			bs, err := hooks.Azure.ProcessBounce(c.Request(), rawReq)
			if err != nil {
				a.log.Printf("error processing Azure Event Grid notification: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}
			bounces = append(bounces, bs...)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

	// SendGrid.
	case service == "sendgrid" && hooks.Sendgrid != nil:
		var (
			sig = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Signature")
			ts  = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Timestamp")
		)

		// Sendgrid sends multiple bounces.
		bs, err := hooks.Sendgrid.ProcessBounce(sig, ts, rawReq)
		if err != nil {
			a.log.Printf("error processing sendgrid notification: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Postmark.
	case service == "postmark" && hooks.Postmark != nil:
		bs, err := hooks.Postmark.ProcessBounce(rawReq, c)
		if err != nil {
			a.log.Printf("error processing postmark notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// ForwardEmail.
	case service == "forwardemail" && hooks.Forwardemail != nil:
		var (
			sig = c.Request().Header.Get("X-Webhook-Signature")
		)

		bs, err := hooks.Forwardemail.ProcessBounce(sig, rawReq)
		if err != nil {
			a.log.Printf("error processing forwardemail notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Lettermint.
	case service == "lettermint" && hooks.Lettermint != nil:
		sig := c.Request().Header.Get("X-Lettermint-Signature")
		bs, err := hooks.Lettermint.ProcessBounce(sig, rawReq)
		if err != nil {
			a.log.Printf("error processing lettermint notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("bounces.unknownService"))
	}

	// Insert bounces into the DB, scoped to the tenant the webhook came in
	// for so its subscribers, campaigns and bounce actions are used.
	for _, b := range bounces {
		b.TenantID = tID
		if b.CreatedAt.IsZero() {
			b.CreatedAt = time.Now()
		}
		if err := a.core.RecordBounce(b); err != nil {
			a.log.Printf("error recording bounce: %v", err)
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// tenantBounceWebhooks builds the bounce webhook processors from a tenant's
// own settings. enabled is false when the tenant has bounce processing or
// webhooks switched off. Settings are read on every call (webhook traffic
// is low), so changes apply without a restart or cache invalidation.
func (a *App) tenantBounceWebhooks(ctx context.Context, tenantID int) (bounce.Webhooks, bool, error) {
	s, err := a.core.GetSettings(ctx, tenantID)
	if err != nil {
		return bounce.Webhooks{}, false, err
	}
	if !s.BounceEnabled || !s.BounceEnableWebhooks {
		return bounce.Webhooks{}, false, nil
	}

	opt := bounce.Opt{
		SESEnabled:              s.SESEnabled,
		AzureEnabled:            s.BounceAzure.Enabled,
		AzureSharedSecret:       s.BounceAzure.SharedSecret,
		AzureSharedSecretHeader: s.BounceAzure.SharedSecretHeader,
		SendgridEnabled:         s.SendgridEnabled,
		SendgridKey:             s.SendgridKey,
	}
	opt.Postmark.Enabled = s.BouncePostmark.Enabled
	opt.Postmark.Username = s.BouncePostmark.Username
	opt.Postmark.Password = s.BouncePostmark.Password
	opt.ForwardEmail.Enabled = s.BounceForwardEmail.Enabled
	opt.ForwardEmail.Key = s.BounceForwardEmail.Key
	opt.Lettermint.Enabled = s.BounceLettermint.Enabled
	opt.Lettermint.Key = s.BounceLettermint.Key

	return bounce.NewWebhooks(opt, a.log), true, nil
}

func (a *App) validateBounceFields(ctx context.Context, tenantID int, b models.Bounce) (models.Bounce, error) {
	if b.Email == "" && b.SubscriberUUID == "" {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email / subscriber_uuid"))
	}

	if b.SubscriberUUID != "" && !reUUID.MatchString(b.SubscriberUUID) {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_uuid"))
	}

	if b.Email != "" {
		imp, err := a.importers.Get(ctx, tenantID)
		if err != nil {
			return b, err
		}

		em, err := imp.SanitizeEmail(b.Email)
		if err != nil {
			return b, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		b.Email = em
	}

	if b.Type != models.BounceTypeHard && b.Type != models.BounceTypeSoft && b.Type != models.BounceTypeComplaint {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}

	return b, nil
}
