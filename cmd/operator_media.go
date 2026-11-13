package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Media storage providers a tenant can be given.
const (
	mediaProviderFilesystem = "filesystem"
	mediaProviderS3         = "s3"
)

// operatorMediaReq is a tenant's media storage, set by the platform.
type operatorMediaReq struct {
	// Provider is "s3" or "filesystem".
	Provider string `json:"provider"`

	// S3 is required when Provider is "s3".
	S3 *operatorS3 `json:"s3,omitempty"`
}

// operatorS3 is an S3-compatible bucket for a tenant's media.
type operatorS3 struct {
	// URL is the S3 endpoint, e.g. https://s3.eu-west-1.amazonaws.com or
	// a MinIO/R2 endpoint.
	URL string `json:"url"`
	// PublicURL is the base URL media is served from, when it differs
	// from the bucket's own (a CDN, say). Optional.
	PublicURL       string `json:"public_url"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	// BucketPath is the prefix inside the bucket, e.g. "/tenants/42".
	// Defaults to "/".
	BucketPath string `json:"bucket_path"`
	// BucketType is "public" or "private" (served through signed URLs).
	// Defaults to "public".
	BucketType string `json:"bucket_type"`
	// Expiry is how long a private bucket's signed URLs last, as a Go
	// duration ("167h"). Defaults to "167h".
	Expiry string `json:"expiry"`
}

// tenantMediaSettings are the upload.* settings keys SetOperatorTenantMedia
// writes, in the settings table's dotted-key shape.
type tenantMediaSettings struct {
	Provider       string `json:"upload.provider"`
	S3URL          string `json:"upload.s3.url"`
	S3PublicURL    string `json:"upload.s3.public_url"`
	S3AccessKeyID  string `json:"upload.s3.aws_access_key_id"`
	S3SecretKey    string `json:"upload.s3.aws_secret_access_key"`
	S3Region       string `json:"upload.s3.aws_default_region"`
	S3Bucket       string `json:"upload.s3.bucket"`
	S3BucketPath   string `json:"upload.s3.bucket_path"`
	S3BucketType   string `json:"upload.s3.bucket_type"`
	S3Expiry       string `json:"upload.s3.expiry"`
	S3BucketDomain string `json:"upload.s3.bucket_domain"`
}

// Defaults for optional S3 fields, as in schema.sql.
const (
	defaultS3BucketPath = "/"
	defaultS3BucketType = "public"
	defaultS3Expiry     = "167h"
)

// toSettings validates r and returns the settings it sets. A filesystem
// provider clears the S3 credentials, so none are left behind.
func (r operatorMediaReq) toSettings() (tenantMediaSettings, string) {
	switch r.Provider {
	case mediaProviderFilesystem:
		return tenantMediaSettings{
			Provider:     mediaProviderFilesystem,
			S3BucketPath: defaultS3BucketPath,
			S3BucketType: defaultS3BucketType,
			S3Expiry:     defaultS3Expiry,
		}, ""

	case mediaProviderS3:
		s := r.S3
		if s == nil {
			return tenantMediaSettings{}, "s3 is required for the s3 provider"
		}
		if strings.TrimSpace(s.URL) == "" || strings.TrimSpace(s.Bucket) == "" || s.AccessKeyID == "" || s.SecretAccessKey == "" {
			return tenantMediaSettings{}, "s3.url, s3.bucket, s3.access_key_id and s3.secret_access_key are required"
		}

		out := tenantMediaSettings{
			Provider:      mediaProviderS3,
			S3URL:         strings.TrimRight(strings.TrimSpace(s.URL), "/"),
			S3PublicURL:   strings.TrimRight(strings.TrimSpace(s.PublicURL), "/"),
			S3AccessKeyID: s.AccessKeyID,
			S3SecretKey:   s.SecretAccessKey,
			S3Region:      strings.TrimSpace(s.Region),
			S3Bucket:      strings.TrimSpace(s.Bucket),
			S3BucketPath:  s.BucketPath,
			S3BucketType:  s.BucketType,
			S3Expiry:      s.Expiry,
		}
		if out.S3BucketPath == "" {
			out.S3BucketPath = defaultS3BucketPath
		}
		if !strings.HasPrefix(out.S3BucketPath, "/") {
			out.S3BucketPath = "/" + out.S3BucketPath
		}
		switch out.S3BucketType {
		case "":
			out.S3BucketType = defaultS3BucketType
		case "public", "private":
		default:
			return tenantMediaSettings{}, "s3.bucket_type must be public or private"
		}
		if out.S3Expiry == "" {
			out.S3Expiry = defaultS3Expiry
		} else if _, err := time.ParseDuration(out.S3Expiry); err != nil {
			return tenantMediaSettings{}, "s3.expiry must be a duration, e.g. 167h"
		}
		return out, ""
	}

	return tenantMediaSettings{}, "provider must be s3 or filesystem"
}

// SetOperatorTenantMedia sets where a tenant's media (uploaded images and
// attachments) is stored.
//
//	@ID			setOperatorTenantMedia
//	@Summary		Set a tenant's media storage (Operator API)
//	@Description	Fork-only, off by default (see [operator] config). Requires the Operator API bearer token. New tenants copy tenant 1's media storage until this is called; set each tenant its own bucket (or bucket path) so tenants' media are kept apart. The tenant's cached media store is rebuilt on its next upload.
//	@Tags			operator
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Tenant ID"
//	@Param			media	body		operatorMediaReq	true	"Media storage"
//	@Success		200	{object}	okResp
//	@Failure		400	{object}	echo.HTTPError
//	@Failure		401	{object}	echo.HTTPError
//	@Failure		404	{object}	echo.HTTPError	"Tenant not found"
//	@Router			/api/operator/tenants/{id}/media [put]
func (a *App) SetOperatorTenantMedia(c echo.Context) error {
	id := getID(c)

	if _, err := a.operator.GetTenant(id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	var req operatorMediaReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	set, msg := req.toSettings()
	if msg != "" {
		return echo.NewHTTPError(http.StatusBadRequest, msg)
	}

	b, err := json.Marshal(set)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error encoding media settings")
	}
	if _, err := a.operator.q.SetTenantSettings.Exec(id, b); err != nil {
		a.log.Printf("error setting tenant %d media storage: %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error setting tenant media storage")
	}

	// The tenant's media store is cached from its first use.
	a.media.Invalidate(id)

	return c.JSON(http.StatusOK, okResp{true})
}
