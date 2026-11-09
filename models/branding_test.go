package models

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestBrandingBlock(t *testing.T) {
	if (Branding{}).Block("Acme") != nil {
		t.Error("no logo should render no block")
	}

	b := Branding{
		Logo:    &BrandLogo{URL: "https://cdn.example/acme.png"},
		Width:   5000,
		Align:   "center",
		LinkURL: "javascript:alert(1)",
	}
	got := string(b.Block(`Acme "Inc"`).HTML)

	if !strings.Contains(got, `width="140"`) {
		t.Error("an out-of-range width should fall back to the default")
	}
	if !strings.Contains(got, "text-align: center") {
		t.Error("alignment lost")
	}
	if strings.Contains(got, "javascript:") {
		t.Error("a non-http link must not be rendered")
	}
	if strings.Contains(got, `"Inc"`) {
		t.Error("alt text must be escaped")
	}
	if !strings.Contains(got, "ln-brand-tile") {
		t.Error("without a dark logo the light one should sit on a tile")
	}

	b.LogoDark = &BrandLogo{URL: "https://cdn.example/acme-dark.png"}
	b.LinkURL = "https://acme.example"
	got = string(b.Block("Acme").HTML)
	if !strings.Contains(got, "acme-dark.png") || !strings.Contains(got, `href="https://acme.example"`) {
		t.Error("dark logo or link missing")
	}
}

func TestTemplateBrandLogoBlock(t *testing.T) {
	tpl := Template{Body: `<p>{{ template "brand-logo" . }}hello</p>`}
	if err := tpl.Compile(template.FuncMap{}); err != nil {
		t.Fatal(err)
	}

	render := func(brand *BrandBlock) string {
		var b bytes.Buffer
		data := struct{ Brand *BrandBlock }{brand}
		if err := tpl.Tpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}

	if got := render(nil); got != "<p>hello</p>" {
		t.Errorf("without a logo the block should be empty, got %q", got)
	}
	logo := Branding{Logo: &BrandLogo{URL: "https://cdn.example/a.png"}}.Block("Acme")
	if got := render(logo); !strings.Contains(got, "https://cdn.example/a.png") {
		t.Errorf("logo missing: %q", got)
	}
}

func TestUsesBrandLogo(t *testing.T) {
	if !UsesBrandLogo(`{{ template "brand-logo" . }}`) || UsesBrandLogo(`<p>no logo</p>`) {
		t.Error("UsesBrandLogo misread the body")
	}
}
