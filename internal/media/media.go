package media

import (
	"io"

	"github.com/nerd-zero/listnun/models"
	"gopkg.in/volatiletech/null.v6"
)

// Media represents an uploaded object.
type Media struct {
	ID          int         `db:"id" json:"id"`
	UUID        string      `db:"uuid" json:"uuid"`
	Filename    string      `db:"filename" json:"filename"`
	ContentType string      `db:"content_type" json:"content_type"`
	Thumb       string      `db:"thumb" json:"-"`
	CreatedAt   null.Time   `db:"created_at" json:"created_at"`
	ThumbURL    null.String `json:"thumb_url"`
	Provider    string      `json:"provider"`
	Meta        models.JSON `db:"meta" json:"meta"`
	URL         string      `json:"url"`
	// Size is the stored file's size in bytes: null until measured, and
	// SizeUnknown when it couldn't be read (see Sizer).
	Size null.Int `db:"size" json:"size"`

	Total int `db:"total" json:"-"`
}

// SizeUnknown marks a file whose size couldn't be read, so the backfill
// doesn't retry it forever. Usage totals leave it out.
const SizeUnknown = -1

// Sizer is implemented by stores that can report a stored file's size.
// It is used to measure files uploaded before sizes were recorded.
type Sizer interface {
	Size(name string) (int64, error)
}

// Store represents functions to store and retrieve media (files).
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	GetURL(string) string
	GetBlob(string) ([]byte, error)
}
