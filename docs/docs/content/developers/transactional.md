# Transactional email

A transactional email goes to one person because something happened: an order, a sign-up, a password reset. You send it from your code with `POST /api/tx` and a transactional template. Unlike a campaign, it isn't sent to a list. By default the recipient must exist as a subscriber in your workspace (on any list or none); set `subscriber_mode` to `fallback` or `external` to send to any address. See the [API reference](../apis/transactional.md#subscriber-modes).

## Templates

Create and edit transactional templates under **Developers › Email templates**. The editor shows a live preview next to the code, using sample data you can edit.

A transactional template has a **subject** and a **body**. Both can use:

- `{{ .Tx.Data.* }}`: the data your code sends, e.g. `{{ .Tx.Data.order_id }}`.
- `{{ .Subscriber.* }}`: the recipient, e.g. `{{ .Subscriber.Name }}`, when they're a subscriber.
- `{{ template "brand-logo" . }}`: your workspace logo from [Email branding](branding.md).

```html
{{ template "brand-logo" . }}
<p>Hi {{ .Subscriber.FirstName }},</p>
<p>Thanks for order #{{ .Tx.Data.order_id }}. Your total was {{ .Tx.Data.total }}.</p>
```

See the [Templating reference](../templating.md) for everything else you can use.

## Sending

```shell
curl -X POST https://yourcompany.listnun.app/api/tx \
  -H "Authorization: token my-service:lnk_live_…" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriber_email": "ada@example.com",
    "template_id": 3,
    "data": {"order_id": "1042", "total": "$25.00"}
  }'
```

Your key needs the `tx:send` scope. The full list of fields (several recipients, sending to people who aren't subscribers, attachments, custom headers) is in the [Transactional API reference](../apis/transactional.md).

To try a template without writing code, use **Developers › Send test**: pick the template, enter an address and sample data, and send. It also shows the `curl` command for the same request.

## The message log

**Developers › Message log** lists every transactional message, newest first. Filter by status, template, API key or date, or search by recipient or message ID. Open one to see its timeline and the exact data it was sent with.

| Status | Meaning |
| :----- | :------ |
| **queued** | Accepted, waiting to be sent. |
| **sent** | Handed to the mail server. |
| **delivered** | The receiving mail server accepted it (when your mail provider reports deliveries). |
| **bounced** | It couldn't be delivered. The bounce type is shown. |
| **failed** | listnun couldn't send it, for example because of a template error. |

**Resend** sends a logged message again with the same data. Reading the log needs the `tx:logs` scope; resending needs `tx:send`.

To be told about these changes as they happen, set up [webhooks](webhooks.md).
