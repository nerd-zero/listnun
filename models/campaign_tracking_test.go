package models

import (
	"errors"
	"html/template"
	"strings"
	"testing"
)

func TestTrackingApplyLinks(t *testing.T) {
	tr := CampaignTracking{Links: true}
	cases := []struct {
		name, in, want string
	}{
		{"tracks a plain link", `<a href="https://example.com/a?b=1">x</a>`, `<a href="https://example.com/a?b=1@TrackLink">x</a>`},
		{"single quotes", `<a href='http://example.com'>x</a>`, `<a href='http://example.com@TrackLink'>x</a>`},
		{"already tracked", `<a href="https://example.com@TrackLink">x</a>`, `<a href="https://example.com@TrackLink">x</a>`},
		{"template tag", `<a href="{{ UnsubscribeURL }}">x</a>`, `<a href="{{ UnsubscribeURL }}">x</a>`},
		{"url with a tag", `<a href="https://example.com/{{ .Subscriber.UUID }}">x</a>`, `<a href="https://example.com/{{ .Subscriber.UUID }}">x</a>`},
		{"mailto", `<a href="mailto:a@example.com">x</a>`, `<a href="mailto:a@example.com">x</a>`},
		{"image src untouched", `<img src="https://example.com/i.png">`, `<img src="https://example.com/i.png">`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := tr.Apply(c.in); got != c.want {
				t.Errorf("Apply(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	if got := (CampaignTracking{}).Apply(cases[0].in); got != cases[0].in {
		t.Errorf("links off still changed the body: %q", got)
	}
}

func TestTrackingApplyPixels(t *testing.T) {
	tr := CampaignTracking{Pixels: []string{"https://px.example.com/a.gif?id=1", "https://px2.example.com/b"}}
	got := tr.Apply("<p>hi</p>")
	for _, p := range tr.Pixels {
		if !strings.Contains(got, `<img src="`+p+`" width="1" height="1"`) {
			t.Errorf("pixel %q missing from %q", p, got)
		}
	}
	if !strings.HasPrefix(got, "<p>hi</p>") {
		t.Errorf("pixels should go after the content: %q", got)
	}
}

func TestTrackingValidate(t *testing.T) {
	ok := CampaignTracking{Pixels: []string{"https://px.example.com/a.gif?id=1&x=2"}}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid pixel rejected: %v", err)
	}

	for _, bad := range []string{
		"javascript:alert(1)",
		`https://x.com/a" onerror="alert(1)`,
		"https://x.com/<script>",
		"ftp://x.com/a.gif",
		"x.com/a.gif",
	} {
		var e InvalidPixelError
		if err := (CampaignTracking{Pixels: []string{bad}}).Validate(); !errors.As(err, &e) || e.URL != bad {
			t.Errorf("Validate(%q) = %v, want InvalidPixelError", bad, err)
		}
	}

	many := CampaignTracking{}
	for i := 0; i <= MaxTrackingPixels; i++ {
		many.Pixels = append(many.Pixels, "https://px.example.com/a.gif")
	}
	if err := many.Validate(); !errors.Is(err, ErrTooManyPixels) {
		t.Errorf("Validate(%d pixels) = %v, want ErrTooManyPixels", len(many.Pixels), err)
	}
}

func TestTrackingNormalized(t *testing.T) {
	got := CampaignTracking{Links: true, Pixels: []string{" https://a.example.com ", "", "https://a.example.com", "https://b.example.com"}}.Normalized()
	want := []string{"https://a.example.com", "https://b.example.com"}
	if !got.Links || strings.Join(got.Pixels, ",") != strings.Join(want, ",") {
		t.Errorf("Normalized() = %+v, want links and %v", got, want)
	}
}

// The rewrite has to survive compilation: a tracked link must end up as a
// TrackLink call, and plain-text emails must be left alone.
func TestCompileTemplateUsesTracking(t *testing.T) {
	var tracked []string
	funcs := template.FuncMap{
		"TrackLink":      func(u string, _ any) string { tracked = append(tracked, u); return u },
		"TrackView":      func(any) template.HTML { return "" },
		"UnsubscribeURL": func(any) string { return "" },
		"ManageURL":      func(any) string { return "" },
		"OptinURL":       func(any) string { return "" },
		"MessageURL":     func(any) string { return "" },
	}

	c := Campaign{ContentType: CampaignContentTypeRichtext, Body: `<a href="https://example.com/x">go</a>`, Tracking: CampaignTracking{Links: true}}
	if err := c.CompileTemplate(funcs); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := c.Tpl.ExecuteTemplate(&b, BaseTpl, nil); err != nil {
		t.Fatal(err)
	}
	if len(tracked) != 1 || tracked[0] != "https://example.com/x" {
		t.Errorf("TrackLink calls = %v, want the one link", tracked)
	}

	p := Campaign{ContentType: CampaignContentTypePlain, Body: "hi", Tracking: CampaignTracking{Pixels: []string{"https://px.example.com/a.gif"}}}
	if err := p.CompileTemplate(funcs); err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if err := p.Tpl.ExecuteTemplate(&b, BaseTpl, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<img") {
		t.Errorf("plain-text email got a pixel: %q", b.String())
	}
}
