package models

import "testing"

func TestFormFieldsValidate(t *testing.T) {
	email := FormField{Key: FormFieldEmail, Label: "Email", Required: true}
	cases := []struct {
		name   string
		fields FormFields
		ok     bool
	}{
		{"email only", FormFields{email}, true},
		{"email, name, attribute", FormFields{email, {Key: "name", Label: "Name"}, {Key: "attribs.city", Label: "City"}}, true},
		{"empty", FormFields{}, false},
		{"email not first", FormFields{{Key: "name", Label: "Name"}, email}, false},
		{"email optional", FormFields{{Key: FormFieldEmail, Label: "Email"}}, false},
		{"twice", FormFields{email, {Key: "attribs.city", Label: "City"}, {Key: "attribs.city", Label: "Town"}}, false},
		{"bad attribute", FormFields{email, {Key: "attribs.City Name", Label: "City"}}, false},
		{"unknown key", FormFields{email, {Key: "password", Label: "Password"}}, false},
		{"no label", FormFields{email, {Key: "name", Label: " "}}, false},
	}
	for _, c := range cases {
		if err := c.fields.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.ok)
		}
	}

	long := FormFields{email}
	for i := 0; i < MaxFormFields; i++ {
		long = append(long, FormField{Key: "attribs.a" + string(rune('a'+i)), Label: "A"})
	}
	if err := long.Validate(); err == nil {
		t.Error("more than MaxFormFields fields passed")
	}
}
