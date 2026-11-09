# API / Signup forms

Saved signup forms, as described in [Signup forms](../guides/lists-and-forms.md#signup-forms).

| Method | Endpoint                                        | Description                                  |
| :----- | :---------------------------------------------- | :------------------------------------------- |
| GET    | [/api/forms](#get-apiforms)                     | Retrieve all signup forms, with their sharing snippets. |
| GET    | [/api/forms/{id}/stats](#get-apiformsidstats)   | A form's figures for the last 30 days.       |
| POST   | [/api/forms](#post-apiforms)                    | Create a form.                               |
| PUT    | [/api/forms/{id}](#put-apiformsid)              | Update, publish or unpublish a form.         |
| DELETE | [/api/forms/{id}](#delete-apiformsid)           | Delete a form.                               |

Reading forms needs the `lists:get_all` permission; creating, changing and deleting them needs `lists:manage_all`. These are user permissions: no [API key](../developers/api-keys.md) scope covers forms yet.

______________________________________________________________________

## The form object

| Name            | Type     | Description |
| :-------------- | :------- | :---------- |
| name            | string   | The form's name, up to 200 characters. While the form is a draft, its `slug` follows the name. |
| status          | string   | `draft` or `live`. A draft's page, embed and popup don't work. |
| heading         | string   | Up to 200 characters. |
| intro           | string   | Up to 1000 characters. |
| list_ids        | []int    | The lists people can join. They must be public lists. A `live` form needs at least one. |
| fields          | []object | The fields, in order. See below. |
| button_label    | string   | Up to 100 characters. |
| success_message | string   | Shown after signing up, up to 1000 characters. Empty shows the standard message. |
| captcha         | bool     | Whether to require a Cloudflare Turnstile check. Turnstile must be set up under **Settings › Sign-in and security**. |

Each field is `{"key": "...", "label": "...", "required": true}`. The first field must be `email` and required. The other keys are `name` and `attribs.<name>`, which saves to the subscriber attribute `<name>` (lower-case letters, digits and `_`, up to 64 characters). A form can have up to 10 fields, each key once, each with a label of up to 200 characters.

Responses also include `id`, `uuid`, `slug`, `signups` (signups in the last 30 days, in lists of forms), `created_at`, `updated_at`, and the ways to share the form:

| Name         | Description |
| :----------- | :---------- |
| url          | The hosted page, `https://<workspace>/f/<slug>`. |
| embed_html   | HTML to paste into a website. |
| popup_script | A `<script>` tag that shows the form as a popup. |

______________________________________________________________________

#### GET /api/forms

Retrieve all signup forms.

##### Example Response

```json
{
  "data": [
    {
      "id": 1,
      "uuid": "0b3c3c8e-5a8d-4a43-9f7e-8a4b8d1e2c90",
      "slug": "newsletter",
      "name": "Newsletter",
      "status": "live",
      "heading": "Get the newsletter",
      "intro": "One email a month. No spam.",
      "list_ids": [3],
      "fields": [
        {"key": "email", "label": "Email", "required": true},
        {"key": "name", "label": "Name", "required": false},
        {"key": "attribs.city", "label": "City", "required": false}
      ],
      "button_label": "Subscribe",
      "success_message": "",
      "captcha": true,
      "signups": 128,
      "created_at": "2026-09-14T10:02:00Z",
      "updated_at": "2026-10-01T08:30:00Z",
      "url": "https://yourcompany.listnun.app/f/newsletter",
      "embed_html": "<form method=\"post\" action=\"https://yourcompany.listnun.app/f/newsletter\" …>…</form>",
      "popup_script": "<script src=\"https://yourcompany.listnun.app/f/newsletter/popup.js\" async></script>"
    }
  ]
}
```

______________________________________________________________________

#### GET /api/forms/{id}/stats

A form's figures for the last 30 days.

##### Example Response

```json
{"data": {"views": 2140, "signups": 128, "confirmed": 97, "blocked": 12}}
```

`confirmed` counts signups whose double opt-in subscription has been confirmed. `blocked` counts submissions stopped by Turnstile or the hidden bot-trap field.

______________________________________________________________________

#### POST /api/forms

Create a form. The body is a form object. Returns the form.

##### Example Request

```shell
curl -u "username:token" -X POST 'https://yourcompany.listnun.app/api/forms' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Newsletter",
    "status": "draft",
    "heading": "Get the newsletter",
    "intro": "One email a month. No spam.",
    "list_ids": [3],
    "fields": [
      {"key": "email", "label": "Email", "required": true},
      {"key": "name", "label": "Name", "required": false}
    ],
    "button_label": "Subscribe",
    "success_message": "",
    "captcha": false
  }'
```

______________________________________________________________________

#### PUT /api/forms/{id}

Update a form. Takes the full form object, as for `POST /api/forms`. Set `status` to `live` to publish it or `draft` to unpublish it. While a form is live its `slug` doesn't change when it's renamed, so links keep working. A draft's `slug` follows its name, so renaming a form after unpublishing it can change its address.

______________________________________________________________________

#### DELETE /api/forms/{id}

Delete a form. Its page, embeds and popup stop working and its figures are deleted. People who signed up through it stay subscribed.

______________________________________________________________________

## Public form endpoints

These are served on the workspace's address and need no auth. They only work for `live` forms.

| Method | Endpoint | Description |
| :----- | :------- | :---------- |
| GET    | `/f/{slug}` | The hosted signup page. |
| POST   | `/f/{slug}` | Submit a signup (`application/x-www-form-urlencoded`, one parameter per field key; `l` with a list UUID per chosen list when the form has several lists). The embed HTML posts here. |
| GET    | `/f/{slug}/popup.js` | The popup script. |
| GET    | `/subscription/form?l={list_uuid}` | A simple signup page for one public list, with no saved form needed. Without `l`, or with an unknown UUID, it shows every public list. |
