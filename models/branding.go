package models

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"
)

// BrandLogoTplName is the template block transactional templates include
// to show the workspace logo: {{ template "brand-logo" . }}.
const BrandLogoTplName = "brand-logo"

// brandLogoDefine is parsed before every transactional template body so the
// block always exists; it renders nothing until a logo is set. A template
// may define its own brand-logo block to override it.
const brandLogoDefine = `{{ define "` + BrandLogoTplName + `" }}{{ if .Brand }}{{ .Brand.HTML }}{{ end }}{{ end }}`

const (
	// BrandWidthDefault is the logo width when none is set.
	BrandWidthDefault = 140
	// BrandWidthMin and BrandWidthMax bound the width an admin can pick.
	BrandWidthMin = 40
	BrandWidthMax = 600
)

// BrandBlock is what transactional templates see as .Brand.
type BrandBlock struct {
	HTML template.HTML
}

// brandHTML is the logo markup. Email clients that support dark mode swap
// to the dark logo, or put the light one on a white tile when there's no
// dark logo, through the prefers-color-scheme media query.
var brandHTML = template.Must(template.New("brand").Parse(`<style>
@media (prefers-color-scheme: dark) {
  .ln-brand-light { display: none !important; }
  .ln-brand-dark { display: inline-block !important; }
  .ln-brand-tile { background: #ffffff !important; padding: 8px 12px !important; border-radius: 6px !important; }
}
</style>
<div class="ln-brand" style="text-align: {{ .Align }}; padding: 0 0 16px;">
{{- if .Link }}<a href="{{ .Link }}" style="text-decoration: none; border: 0;">{{ end -}}
{{- if .Dark -}}
<img class="ln-brand-light" src="{{ .Light }}" width="{{ .Width }}" alt="{{ .Alt }}" style="display: inline-block; width: {{ .Width }}px; max-width: 100%; height: auto; border: 0;">
<img class="ln-brand-dark" src="{{ .Dark }}" width="{{ .Width }}" alt="{{ .Alt }}" style="display: none; width: {{ .Width }}px; max-width: 100%; height: auto; border: 0;">
{{- else -}}
<span class="ln-brand-tile" style="display: inline-block;"><img src="{{ .Light }}" width="{{ .Width }}" alt="{{ .Alt }}" style="display: block; width: {{ .Width }}px; max-width: 100%; height: auto; border: 0;"></span>
{{- end -}}
{{- if .Link }}</a>{{ end -}}
</div>`))

// Block renders the logo for the brand-logo template block, or nil when no
// logo is set. alt is the logo's alt text, usually the workspace name.
func (b Branding) Block(alt string) *BrandBlock {
	if b.Logo == nil || b.Logo.URL == "" {
		return nil
	}

	width := b.Width
	if width < BrandWidthMin || width > BrandWidthMax {
		width = BrandWidthDefault
	}
	align := BrandAlignLeft
	if b.Align == BrandAlignCenter {
		align = BrandAlignCenter
	}

	data := struct {
		Light, Dark, Alt, Align string
		Link                    string
		Width                   int
	}{Light: b.Logo.URL, Alt: alt, Align: align, Width: width}
	if b.LogoDark != nil {
		data.Dark = b.LogoDark.URL
	}
	if ValidBrandLink(b.LinkURL) {
		data.Link = b.LinkURL
	}

	var out bytes.Buffer
	if err := brandHTML.Execute(&out, data); err != nil {
		return nil
	}
	// The markup above is built by html/template, so it's already escaped.
	return &BrandBlock{HTML: template.HTML(out.String())}
}

// ValidBrandLink reports whether s is an http(s) URL, or empty.
func ValidBrandLink(s string) bool {
	if s == "" {
		return true
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// UsesBrandLogo reports whether a template body includes the brand-logo block.
func UsesBrandLogo(body string) bool {
	return strings.Contains(body, `"`+BrandLogoTplName+`"`)
}
