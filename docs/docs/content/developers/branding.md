# Email branding

**Developers › Email branding** sets the logo your transactional emails show.

- **Logo**: upload or pick an image from your media library.
- **Dark-mode logo** (optional): shown by email apps in dark mode. Without one, your logo sits on a white tile in dark mode so it stays readable.
- **Width**, **alignment** and an optional **link** for the logo.

The preview shows the result in light and dark mode.

## Using it in templates

Put this where the logo should go in a transactional template:

```html
{{ template "brand-logo" . }}
```

It renders nothing until a logo is set, so it's safe in every template. To lay the logo out your own way, define your own `brand-logo` block in the template and it replaces the built-in one.

Campaign templates don't use branding. Put your logo straight into the campaign template's HTML instead.
