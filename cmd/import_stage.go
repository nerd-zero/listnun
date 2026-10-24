package main

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/importstage"
	"github.com/nerd-zero/listnun/internal/subimporter"
)

const (
	// maxImportFileSize is the largest file the Import page accepts.
	maxImportFileSize = 50 << 20

	// numRecentImports is how many past imports the Import page lists.
	numRecentImports = 5
)

// importFileResp describes an uploaded file: its rows, and each column with
// a few values and a guess at what it holds.
type importFileResp struct {
	FileID     string    `json:"file_id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	subimporter.Inspection
} // @name ImportFile

// checkImportReq asks what importing a staged file with these columns
// would do.
type checkImportReq struct {
	FileID  string               `json:"file_id"`
	Columns []subimporter.Column `json:"columns"`
} // @name CheckImportReq

// UploadImportFile keeps an uploaded CSV or ZIP for importing and describes
// it, so its columns can be matched before the import starts.
//
//	@ID			uploadImportFile
//	@Summary		Upload a file to import
//	@Tags			import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"CSV, or a ZIP with a CSV in it"
//	@Success		200		{object}	importFileResp
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/import/subscribers/file [post]
func (a *App) UploadImportFile(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("import.invalidFile", "error", err.Error()))
	}
	if file.Size > maxImportFileSize {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.fileTooLarge"))
	}

	f := importstage.File{TenantID: tenantID(c), Name: file.Filename, Size: file.Size, At: time.Now()}
	if err := a.stageUpload(file, &f); err != nil {
		return err
	}

	ins, err := subimporter.Inspect(f.Path)
	if err != nil {
		os.RemoveAll(f.Dir)
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("import.invalidFile", "error", err.Error()))
	}
	f.Delim = rune(ins.Delim[0])

	id, err := a.importStage.Put(f, f.At)
	if err != nil {
		os.RemoveAll(f.Dir)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{importFileResp{
		FileID: id, Name: f.Name, Size: f.Size, UploadedAt: f.At, Inspection: ins,
	}})
}

// stageUpload copies an upload into a new temporary directory, taking the
// CSV out of a ZIP, and sets f's path and dir.
func (a *App) stageUpload(file *multipart.FileHeader, f *importstage.File) error {
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("import.invalidFile", "error", err.Error()))
	}
	defer src.Close()

	dir, err := os.MkdirTemp("", "listnun-import")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}
	path := filepath.Join(dir, "upload")
	out, err := os.Create(path)
	if err == nil {
		_, err = io.Copy(out, src)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		os.RemoveAll(dir)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("import.errorCopyingFile", "error", err.Error()))
	}
	f.Path, f.Dir = path, dir

	if strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		csvPath, csvDir, err := subimporter.ExtractFirstCSV(path)
		os.RemoveAll(dir)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("import.errorProcessingZIP", "error", err.Error()))
		}
		f.Path, f.Dir = csvPath, csvDir
	}
	return nil
}

// CheckImport says what importing an uploaded file with the given columns
// would do (new people, people already here, rows that won't import)
// without importing anything.
//
//	@ID			checkImport
//	@Summary		Check an import before starting it
//	@Tags			import
//	@Accept			json
//	@Produce		json
//	@Param			data	body		checkImportReq	true	"File and columns"
//	@Success		200		{object}	subimporter.CheckResult
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/import/subscribers/check [post]
func (a *App) CheckImport(c echo.Context) error {
	var req checkImportReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	tID := tenantID(c)
	f, ok := a.importStage.Get(tID, req.FileID, time.Now())
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.fileExpired"))
	}

	imp, err := a.importers.Get(c.Request().Context(), tID)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	out, err := imp.Check(f.Path, f.Delim, req.Columns, func(emails []string) (int, error) {
		return a.core.CountExistingEmails(ctx, tID, emails)
	})
	if errors.Is(err, subimporter.ErrNoEmailColumn) {
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("import.noEmailColumn"))
	}
	if err != nil {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			return he
		}
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("import.invalidFile", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetImportProblems returns the rows the current or last import left out.
//
//	@ID			getImportProblems
//	@Summary		Rows an import didn't import
//	@Tags			import
//	@Produce		json
//	@Success		200	{array}	subimporter.Problem
//	@Router			/api/import/subscribers/problems [get]
func (a *App) GetImportProblems(c echo.Context) error {
	imp, err := a.importers.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{imp.GetProblems()})
}

// GetRecentImports returns the latest finished imports.
//
//	@ID			getRecentImports
//	@Summary		Recent imports
//	@Tags			import
//	@Produce		json
//	@Success		200	{array}	models.ImportRecord
//	@Router			/api/import/subscribers/history [get]
func (a *App) GetRecentImports(c echo.Context) error {
	out, err := a.core.GetRecentImports(c.Request().Context(), tenantID(c), numRecentImports)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, okResp{out})
}
