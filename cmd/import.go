package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/auth"
	"github.com/nerd-zero/listnun/internal/subimporter"
	"github.com/nerd-zero/listnun/models"
)

// ImportSubscribers handles the uploading and bulk importing of
// a ZIP file of one or more CSV files.
//
//	@ID			importSubscribers
//	@Summary		Import subscribers from a CSV or ZIP file
//	@Tags			import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"CSV or ZIP file"
//	@Param			params	formData	string	true	"JSON-encoded import options (subimporter.SessionOpt)"
//	@Success		200		{object}	subimporter.Status
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/import/subscribers [post]
func (a *App) ImportSubscribers(c echo.Context) error {
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}

	// Is an import already running?
	if imp.GetStats().Status == subimporter.StatusImporting {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.alreadyRunning"))
	}

	// Unmarshal the JSON params.
	var opt subimporter.SessionOpt
	if err := json.Unmarshal([]byte(c.FormValue("params")), &opt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("import.invalidParams", "error", err.Error()))
	}

	// Filter list IDs against the current user's permitted lists.
	// Blocklist mode doesn't require list subscriptions.
	user := auth.GetUser(c)
	opt.ListIDs = user.FilterListsByPerm(auth.PermTypeManage, opt.ListIDs)
	if len(opt.ListIDs) == 0 && opt.Mode != subimporter.ModeBlocklist {
		return echo.NewHTTPError(http.StatusForbidden,
			a.i18n.Ts("globals.messages.permissionDenied", "name", "lists"))
	}

	// Validate mode.
	if opt.Mode != subimporter.ModeSubscribe && opt.Mode != subimporter.ModeBlocklist {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.invalidMode"))
	}

	// If no status is specified, pick a default one.
	if opt.SubStatus == "" {
		switch opt.Mode {
		case subimporter.ModeSubscribe:
			opt.SubStatus = models.SubscriptionStatusUnconfirmed
		case subimporter.ModeBlocklist:
			opt.SubStatus = models.SubscriptionStatusUnsubscribed
		}
	}

	if opt.SubStatus != models.SubscriptionStatusUnconfirmed &&
		opt.SubStatus != models.SubscriptionStatusConfirmed &&
		opt.SubStatus != models.SubscriptionStatusUnsubscribed {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.invalidSubStatus"))
	}

	if len(opt.Delim) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.invalidDelim"))
	}

	// Open the HTTP file.
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("import.invalidFile", "error", err.Error()))
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Copy it to a temp location.
	out, err := os.CreateTemp("", "listnun")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}
	defer out.Close()

	if _, err = io.Copy(out, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}

	// Start the importer session.
	opt.Filename = file.Filename
	opt.TenantID = tenantID(c)
	sess, err := imp.NewSession(opt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("import.errorStarting", "error", err.Error()))
	}
	go sess.Start()

	if strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		go sess.LoadCSV(out.Name(), rune(opt.Delim[0]))
	} else {
		// Only 1 CSV from the ZIP is considered. If multiple files have
		// to be processed, counting the net number of lines (to track progress),
		// keeping the global import state (failed / successful) etc. across
		// multiple files becomes complex. Instead, it's just easier for the
		// end user to concat multiple CSVs (if there are multiple in the first)
		// place and upload as one in the first place.
		dir, files, err := sess.ExtractZIP(out.Name(), 1)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("import.errorProcessingZIP", "error", err.Error()))
		}

		go sess.LoadCSV(dir+"/"+files[0], rune(opt.Delim[0]))
	}

	return c.JSON(http.StatusOK, okResp{imp.GetStats()})
}

// GetImportSubscribers returns import statistics.
//
//	@ID			getImportStatus
//	@Summary		Get current import status
//	@Tags			import
//	@Produce		json
//	@Success		200	{object}	subimporter.Status
//	@Router			/api/import/subscribers [get]
func (a *App) GetImportSubscribers(c echo.Context) error {
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{imp.GetStats()})
}

// GetImportSubscriberStats returns import log output.
//
//	@ID			getImportLogs
//	@Summary		Get import log
//	@Tags			import
//	@Produce		json
//	@Success		200	{object}	string
//	@Router			/api/import/subscribers/logs [get]
func (a *App) GetImportSubscriberStats(c echo.Context) error {
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{string(imp.GetLogs())})
}

// StopImportSubscribers sends a stop signal to the importer.
// If there's an ongoing import, it'll be stopped, and if an import
// is finished, it's state is cleared.
//
//	@ID			stopImport
//	@Summary		Stop or clear a subscriber import
//	@Tags			import
//	@Produce		json
//	@Success		200	{object}	subimporter.Status
//	@Router			/api/import/subscribers [delete]
func (a *App) StopImportSubscribers(c echo.Context) error {
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	imp.Stop()
	return c.JSON(http.StatusOK, okResp{imp.GetStats()})
}
