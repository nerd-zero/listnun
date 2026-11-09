package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

// Form statuses. A draft form's public page and embed don't work yet.
const (
	FormStatusLive  = "live"
	FormStatusDraft = "draft"
)

// Form field keys other than attributes, which are FormAttribPrefix+name.
const (
	FormFieldEmail   = "email"
	FormFieldName    = "name"
	FormAttribPrefix = "attribs."
)

// Form event kinds, for a form's figures.
const (
	FormEventView    = "view"
	FormEventSignup  = "signup"
	FormEventBlocked = "blocked"
)

// MaxFormFields caps a form's fields; a signup form longer than this
// loses people.
const MaxFormFields = 10

var reAttribKey = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// FormField is one field on a signup form.
type FormField struct {
	// Key is email, name, or attribs.<name> for an attribute.
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

// Attrib is the attribute a field saves to, or "" for email and name.
func (f FormField) Attrib() string {
	return strings.TrimPrefix(f.Key, FormAttribPrefix)
}

// IsAttrib says whether the field saves to an attribute.
func (f FormField) IsAttrib() bool {
	return strings.HasPrefix(f.Key, FormAttribPrefix)
}

// FormFields is a form's fields in order, stored as JSON.
type FormFields []FormField

// Scan implements the sql.Scanner interface.
func (f *FormFields) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*f = FormFields{}
		return nil
	case []byte:
		return json.Unmarshal(v, f)
	case string:
		return json.Unmarshal([]byte(v), f)
	}
	return fmt.Errorf("could not decode type %T -> %T", src, f)
}

// Value implements the driver.Valuer interface.
func (f FormFields) Value() (driver.Value, error) {
	if f == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(f)
}

// Validate checks the fields: email first and required, each key once,
// attribute names lower case.
func (f FormFields) Validate() error {
	if len(f) == 0 || f[0].Key != FormFieldEmail || !f[0].Required {
		return fmt.Errorf("the first field must be a required email")
	}
	if len(f) > MaxFormFields {
		return fmt.Errorf("a form can have at most %d fields", MaxFormFields)
	}
	seen := map[string]bool{}
	for _, fl := range f {
		if seen[fl.Key] {
			return fmt.Errorf("field %q appears twice", fl.Key)
		}
		seen[fl.Key] = true
		switch {
		case fl.Key == FormFieldEmail, fl.Key == FormFieldName:
		case fl.IsAttrib() && reAttribKey.MatchString(fl.Attrib()):
		default:
			return fmt.Errorf("invalid field %q", fl.Key)
		}
		if strings.TrimSpace(fl.Label) == "" || len(fl.Label) > 200 {
			return fmt.Errorf("field %q needs a label of up to 200 characters", fl.Key)
		}
	}
	return nil
}

// Form is a saved signup form: the lists it subscribes people to, its
// fields and wording.
type Form struct {
	ID             int           `db:"id" json:"id"`
	UUID           string        `db:"uuid" json:"uuid"`
	Slug           string        `db:"slug" json:"slug"`
	Name           string        `db:"name" json:"name"`
	Status         string        `db:"status" json:"status"`
	Heading        string        `db:"heading" json:"heading"`
	Intro          string        `db:"intro" json:"intro"`
	ListIDs        pq.Int64Array `db:"list_ids" json:"list_ids" swaggertype:"array,integer"`
	Fields         FormFields    `db:"fields" json:"fields"`
	ButtonLabel    string        `db:"button_label" json:"button_label"`
	SuccessMessage string        `db:"success_message" json:"success_message"`
	Captcha        bool          `db:"captcha" json:"captcha"`
	// Signups is how many people signed up in the last 30 days (lists of
	// forms only).
	Signups   int       `db:"signups" json:"signups"`
	CreatedAt null.Time `db:"created_at" json:"created_at" swaggertype:"string"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at" swaggertype:"string"`
} // @name Form

// FormStats is a form's figures over a period.
type FormStats struct {
	Views     int `db:"views" json:"views"`
	Signups   int `db:"signups" json:"signups"`
	Confirmed int `db:"confirmed" json:"confirmed"`
	Blocked   int `db:"blocked" json:"blocked"`
} // @name FormStats
