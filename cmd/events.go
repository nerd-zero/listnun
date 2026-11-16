package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// EventStream serves an endpoint that never closes and pushes a
// live event stream (text/event-stream) such as a error messages.
func (a *App) EventStream(c echo.Context) error {
	hdr := c.Response().Header()
	hdr.Set(echo.HeaderContentType, "text/event-stream")
	hdr.Set(echo.HeaderCacheControl, "no-store")
	hdr.Set(echo.HeaderConnection, "keep-alive")

	// The stream carries the process's error log lines, which with
	// several tenants include other tenants' data. Until events are tagged
	// by tenant, a multi-tenant server sends none; the stream stays open so
	// the admin UI doesn't keep reconnecting.
	if a.cfg.MultiTenancyEnabled {
		<-c.Request().Context().Done()
		return nil
	}

	// Subscribe to the event stream with a random ID.
	id := fmt.Sprintf("api:%v", time.Now().UnixNano())
	sub, err := a.events.Subscribe(id)
	if err != nil {
		a.log.Printf("error subscribing to events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	ctx := c.Request().Context()
	for {
		select {
		case e := <-sub:
			b, err := json.Marshal(e)
			if err != nil {
				a.log.Printf("error marshalling event: %v", err)
				continue
			}

			c.Response().Write([]byte(fmt.Sprintf("retry: 3000\ndata: %s\n\n", b)))
			c.Response().Flush()

		case <-ctx.Done():
			// On HTTP connection close, unsubscribe.
			a.events.Unsubscribe(id)
			return nil
		}
	}

}
