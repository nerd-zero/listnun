package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/core"
	"github.com/nerd-zero/listnun/internal/webhooks"
	"github.com/nerd-zero/listnun/models"
)

const (
	// webhookPollInterval is how often the worker looks for due deliveries.
	webhookPollInterval = 5 * time.Second

	// webhookPruneInterval is how often finished deliveries are pruned.
	webhookPruneInterval = 6 * time.Hour
)

// webhookEndpointReq is the body for adding or changing an endpoint.
type webhookEndpointReq struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	// Status is "active" or "disabled"; empty keeps an endpoint active.
	Status string `json:"status"`
} // @name WebhookEndpointRequest

// webhookSecretOut is an endpoint's signing secret.
type webhookSecretOut struct {
	Secret string `json:"secret"`
} // @name WebhookSecret

// webhookTestOut is the delivery a test event was queued as.
type webhookTestOut struct {
	DeliveryID string `json:"delivery_id"`
} // @name WebhookTest

// GetWebhookEndpoints lists the workspace's webhook endpoints with their
// last-24h success rate, median response time and retrying count.
//
//	@ID			listWebhookEndpoints
//	@Summary		List webhook endpoints
//	@Tags			developers
//	@Produce		json
//	@Success		200	{object}	[]models.WebhookEndpoint
//	@Router			/api/developers/webhooks [get]
func (a *App) GetWebhookEndpoints(c echo.Context) error {
	out, err := a.core.GetWebhookEndpoints(c.Request().Context(), tenantID(c), 0)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// CreateWebhookEndpoint adds an endpoint. It must be HTTPS.
//
//	@ID			createWebhookEndpoint
//	@Summary		Add a webhook endpoint
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			body	body		webhookEndpointReq	true	"Endpoint"
//	@Success		200		{object}	models.WebhookEndpoint
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/developers/webhooks [post]
func (a *App) CreateWebhookEndpoint(c echo.Context) error {
	req, err := a.bindWebhookEndpoint(c)
	if err != nil {
		return err
	}
	out, err := a.core.CreateWebhookEndpoint(c.Request().Context(), tenantID(c), req.URL, req.Events)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateWebhookEndpoint changes an endpoint's URL, events or status.
//
//	@ID			updateWebhookEndpoint
//	@Summary		Update a webhook endpoint
//	@Tags			developers
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Endpoint ID"
//	@Param			body	body		webhookEndpointReq	true	"Endpoint"
//	@Success		200		{object}	models.WebhookEndpoint
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/{id} [put]
func (a *App) UpdateWebhookEndpoint(c echo.Context) error {
	req, err := a.bindWebhookEndpoint(c)
	if err != nil {
		return err
	}
	out, err := a.core.UpdateWebhookEndpoint(c.Request().Context(), tenantID(c), getID(c), req.URL, req.Events, req.Status)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteWebhookEndpoint removes an endpoint and its delivery history.
//
//	@ID			deleteWebhookEndpoint
//	@Summary		Delete a webhook endpoint
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Endpoint ID"
//	@Success		200	{object}	okResp
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/{id} [delete]
func (a *App) DeleteWebhookEndpoint(c echo.Context) error {
	if err := a.core.DeleteWebhookEndpoint(c.Request().Context(), tenantID(c), getID(c)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// GetWebhookEndpointSecret reveals an endpoint's signing secret.
//
//	@ID			getWebhookEndpointSecret
//	@Summary		Reveal a webhook signing secret
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Endpoint ID"
//	@Success		200	{object}	webhookSecretOut
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/{id}/secret [get]
func (a *App) GetWebhookEndpointSecret(c echo.Context) error {
	s, err := a.core.GetWebhookEndpointSecret(c.Request().Context(), tenantID(c), getID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{webhookSecretOut{Secret: s}})
}

// RotateWebhookEndpointSecret replaces an endpoint's signing secret.
//
//	@ID			rotateWebhookEndpointSecret
//	@Summary		Rotate a webhook signing secret
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Endpoint ID"
//	@Success		200	{object}	webhookSecretOut
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/{id}/secret [post]
func (a *App) RotateWebhookEndpointSecret(c echo.Context) error {
	s, err := a.core.RotateWebhookEndpointSecret(c.Request().Context(), tenantID(c), getID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{webhookSecretOut{Secret: s}})
}

// SendWebhookTest queues a webhook.test event to an endpoint.
//
//	@ID			sendWebhookTest
//	@Summary		Send a test event
//	@Tags			developers
//	@Produce		json
//	@Param			id	path		int	true	"Endpoint ID"
//	@Success		200	{object}	webhookTestOut
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/{id}/test [post]
func (a *App) SendWebhookTest(c echo.Context) error {
	id, err := a.core.SendWebhookTest(c.Request().Context(), tenantID(c), getID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{webhookTestOut{DeliveryID: id}})
}

// QueryWebhookDeliveries returns recent deliveries, newest first.
//
//	@ID			listWebhookDeliveries
//	@Summary		List webhook deliveries
//	@Tags			developers
//	@Produce		json
//	@Param			endpoint_id	query		int		false	"Endpoint ID"
//	@Param			status		query		string	false	"pending, success, retrying or failed"
//	@Param			page		query		int		false	"Page"
//	@Param			per_page	query		int		false	"Results per page"
//	@Success		200			{object}	models.PageResults
//	@Router			/api/developers/webhooks/deliveries [get]
func (a *App) QueryWebhookDeliveries(c echo.Context) error {
	pg := a.pg.NewFromURL(c.Request().URL.Query())
	endpointID, _ := strconv.Atoi(c.QueryParam("endpoint_id"))

	res, total, err := a.core.QueryWebhookDeliveries(c.Request().Context(), tenantID(c), endpointID, c.QueryParam("status"), pg.Offset, pg.Limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{models.PageResults{
		Results: res, Total: total, Page: pg.Page, PerPage: pg.PerPage,
	}})
}

// GetWebhookDelivery returns a delivery with its payload and attempts.
//
//	@ID			getWebhookDelivery
//	@Summary		Get a webhook delivery
//	@Tags			developers
//	@Produce		json
//	@Param			uuid	path		string	true	"Delivery ID (whd_…)"
//	@Success		200		{object}	models.WebhookDelivery
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/deliveries/{uuid} [get]
func (a *App) GetWebhookDelivery(c echo.Context) error {
	out, err := a.core.GetWebhookDelivery(c.Request().Context(), tenantID(c), c.Param("uuid"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// RetryWebhookDelivery sends a delivery again now.
//
//	@ID			retryWebhookDelivery
//	@Summary		Resend a webhook delivery
//	@Tags			developers
//	@Produce		json
//	@Param			uuid	path		string	true	"Delivery ID (whd_…)"
//	@Success		200		{object}	okResp
//	@Failure		404		{object}	echo.HTTPError
//	@Router			/api/developers/webhooks/deliveries/{uuid}/retry [post]
func (a *App) RetryWebhookDelivery(c echo.Context) error {
	if err := a.core.RetryWebhookDelivery(c.Request().Context(), tenantID(c), c.Param("uuid")); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// bindWebhookEndpoint reads and validates an endpoint request.
func (a *App) bindWebhookEndpoint(c echo.Context) (webhookEndpointReq, error) {
	var req webhookEndpointReq
	if err := c.Bind(&req); err != nil {
		return req, err
	}

	req.URL = strings.TrimSpace(req.URL)
	if !a.validWebhookURL(req.URL) {
		return req, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.webhooks.invalidURL"))
	}
	if !core.ValidWebhookEvents(req.Events) {
		return req, echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("developers.webhooks.invalidEvents"))
	}
	switch req.Status {
	case "":
		req.Status = models.WebhookEndpointActive
	case models.WebhookEndpointActive, models.WebhookEndpointDisabled:
	default:
		return req, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "status"))
	}
	return req, nil
}

// validWebhookURL accepts HTTPS URLs with a host. Plain HTTP is allowed
// only when private addresses are, for local development. The address
// itself is checked again at send time, after DNS resolution.
func (a *App) validWebhookURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User != nil || len(s) > stdInputMaxLen {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && a.cfg.WebhooksAllowPrivate
}

// runWebhooks starts the delivery worker and the pruning loop.
func (a *App) runWebhooks(ctx context.Context) {
	w := webhooks.New(a.core, a.cfg.WebhooksAllowPrivate, a.log)
	go w.Run(ctx, webhookPollInterval)

	go func() {
		for {
			if n, err := a.core.PruneWebhookDeliveries(); err != nil {
				a.log.Printf("error pruning webhook deliveries: %v", err)
			} else if n > 0 {
				a.log.Printf("pruned %d webhook deliveries", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(webhookPruneInterval):
			}
		}
	}()
}
