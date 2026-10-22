package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/models"
)

const (
	thumbPrefix   = "thumb_"
	thumbnailSize = 250
)

var (
	vectorExts = []string{"svg"}
	imageExts  = []string{"gif", "png", "jpg", "jpeg"}
)

// UploadMedia handles media file uploads.
//
//	@ID			uploadMedia
//	@Summary		Upload a media file
//	@Tags			media
//	@Accept			mpfd
//	@Produce		json
//	@Param			file	formData	file	true	"Media file to upload"
//	@Success		200		{object}	object
//	@Failure		400		{object}	echo.HTTPError
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/media [post]
func (a *App) UploadMedia(c echo.Context) error {
	ctx := c.Request().Context()
	tID := tenantID(c)

	ms, settings, err := a.media.Get(ctx, tID)
	if err != nil {
		a.log.Printf("error resolving media store: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("media.invalidFile", "error", err.Error()))
	}

	// Enforce the workspace's size limit (Settings › Media, in KB).
	if max := settings.UploadMaxFileSize; max > 0 && file.Size > int64(max)*1024 {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge,
			a.i18n.Ts("media.fileTooLarge", "max", strconv.FormatFloat(float64(max)/1024, 'f', 1, 64)))
	}

	// Read the file from the HTTP form.
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("media.errorReadingFile", "error", err.Error()))
	}
	defer src.Close()

	var (
		// Naive check for content type and extension.
		ext         = strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), ".")
		contentType = file.Header.Get("Content-Type")
	)

	// Validate file extension.
	if !inArray("*", settings.UploadExtensions) {
		if ok := inArray(ext, settings.UploadExtensions); !ok {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("media.unsupportedFileType", "type", ext))
		}
	}

	// Sanitize the filename.
	fName := makeFilename(file.Filename)

	// If the filename already exists in the DB, make it unique by adding a random suffix.
	if _, err := a.core.GetMedia(ctx, tID, 0, "", fName, ms); err == nil {
		suffix, err := generateRandomString(6)
		if err != nil {
			a.log.Printf("error generating random string: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}

		fName = appendSuffixToFilename(fName, suffix)
	}

	// Upload the file to the media store.
	fName, err = ms.Put(fName, contentType, src)
	if err != nil {
		a.log.Printf("error uploading file: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("media.errorUploading", "error", err.Error()))
	}

	// This keeps track of whether the file has to be deleted from the DB and the store
	// if any of the subsequent steps fail.
	var (
		cleanUp    = false
		thumbfName = ""
	)
	defer func() {
		if cleanUp {
			ms.Delete(fName)

			if thumbfName != "" {
				ms.Delete(thumbfName)
			}
		}
	}()

	// Thumbnail width and height.
	var width, height int

	// Create thumbnail from file for non-vector formats.
	isImage := inArray(ext, imageExts)
	if isImage {
		thumbFile, wi, he, err := processImage(file)
		if err != nil {
			cleanUp = true
			a.log.Printf("error resizing image: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("media.errorResizing", "error", err.Error()))
		}
		width = wi
		height = he

		// Upload thumbnail.
		tf, err := ms.Put(thumbPrefix+fName, contentType, thumbFile)
		if err != nil {
			cleanUp = true
			a.log.Printf("error saving thumbnail: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				a.i18n.Ts("media.errorSavingThumbnail", "error", err.Error()))
		}
		thumbfName = tf
	}
	if inArray(ext, vectorExts) {
		thumbfName = fName
	}

	// Images have metadata.
	meta := models.JSON{}
	if isImage {
		meta = models.JSON{
			"width":  width,
			"height": height,
		}
	}

	// Insert the media into the DB.
	m, err := a.core.InsertMedia(ctx, tID, fName, thumbfName, contentType, meta, settings.UploadProvider, file.Size, ms)
	if err != nil {
		cleanUp = true
		return err
	}

	return c.JSON(http.StatusOK, okResp{m})
}

// GetAllMedia handles retrieval of uploaded media.
//
//	@ID			listMedia
//	@Summary		List all media
//	@Tags			media
//	@Produce		json
//	@Param			query	query		string	false	"Search query"
//	@Param			page	query		int		false	"Page number"
//	@Param			per_page	query	int		false	"Results per page"
//	@Success		200		{object}	models.PageResults
//	@Failure		500		{object}	echo.HTTPError
//	@Router			/api/media [get]
func (a *App) GetAllMedia(c echo.Context) error {
	var (
		query = c.FormValue("query")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)

	ctx := c.Request().Context()
	tID := tenantID(c)

	ms, settings, err := a.media.Get(ctx, tID)
	if err != nil {
		a.log.Printf("error resolving media store: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Fetch the media items from the DB.
	res, total, err := a.core.QueryMedia(ctx, tID, settings.UploadProvider, ms, query, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetMedia handles retrieval of a media item by ID.
//
//	@ID			getMedia
//	@Summary		Get a media item
//	@Tags			media
//	@Produce		json
//	@Param			id	path		int	true	"Media ID"
//	@Success		200	{object}	object
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/media/{id} [get]
func (a *App) GetMedia(c echo.Context) error {
	ctx := c.Request().Context()
	tID := tenantID(c)

	ms, _, err := a.media.Get(ctx, tID)
	if err != nil {
		a.log.Printf("error resolving media store: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Fetch the media item from the DB.
	id := getID(c)
	out, err := a.core.GetMedia(ctx, tID, id, "", "", ms)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteMedia handles deletion of uploaded media.
//
//	@ID			deleteMedia
//	@Summary		Delete a media item
//	@Tags			media
//	@Produce		json
//	@Param			id	path		int	true	"Media ID"
//	@Success		200
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError
//	@Router			/api/media/{id} [delete]
func (a *App) DeleteMedia(c echo.Context) error {
	ctx := c.Request().Context()
	tID := tenantID(c)

	ms, _, err := a.media.Get(ctx, tID)
	if err != nil {
		a.log.Printf("error resolving media store: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Delete the media from the DB. The query returns the filename.
	id := getID(c)
	fname, err := a.core.DeleteMedia(ctx, tID, id)
	if err != nil {
		return err
	}

	// Delete the files from the media store.
	ms.Delete(fname)
	ms.Delete(thumbPrefix + fname)

	return c.JSON(http.StatusOK, okResp{true})
}

// ServeS3Media serves media files stored in S3 when the public URL is a relative path.
func (a *App) ServeS3Media(c echo.Context) error {
	key := c.Param("filepath")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing media file path")
	}

	ms, _, err := a.media.Get(c.Request().Context(), tenantID(c))
	if err != nil {
		a.log.Printf("error resolving media store: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching media")
	}

	b, err := ms.GetBlob(key)
	if err != nil {
		a.log.Printf("error fetching media from s3 %s: %v", key, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching media")
	}

	return c.Stream(http.StatusOK, http.DetectContentType(b), bytes.NewReader(b))
}

// processImage reads the image file and returns thumbnail bytes and
// the original image's width, and height.
func processImage(file *multipart.FileHeader) (*bytes.Reader, int, int, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, 0, err
	}
	defer src.Close()

	img, err := imaging.Decode(src)
	if err != nil {
		return nil, 0, 0, err
	}

	// Encode the image into a byte slice as PNG.
	var (
		thumb = imaging.Resize(img, thumbnailSize, 0, imaging.Lanczos)
		out   bytes.Buffer
	)
	if err := imaging.Encode(&out, thumb, imaging.PNG); err != nil {
		return nil, 0, 0, err
	}

	b := img.Bounds().Max
	return bytes.NewReader(out.Bytes()), b.X, b.Y, nil
}
