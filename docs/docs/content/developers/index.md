# Developers

The **Developers** area of the sidebar is for connecting your own systems to listnun: sending transactional email (receipts, password resets, alerts) from your code, and hearing back what happened to it.

| Page | What it's for |
| :--- | :------------ |
| **Overview** | A four-step quickstart, plus today's sending numbers and recent messages. |
| **API keys** | Credentials for your code, each limited to the scopes it needs. See [API keys](api-keys.md). |
| **Email templates** | The transactional templates your code sends. See [Transactional email](transactional.md). |
| **Send test** | Send a transactional template from the browser, with sample data, and copy the matching `curl` command. |
| **Message log** | Every transactional message: who it went to, its status, and the exact payload. |
| **Email branding** | Your logo for transactional emails, including a dark-mode version. See [Email branding](branding.md). |
| **Webhooks** | Endpoints listnun calls when a message is sent, delivered, bounces or fails. See [Webhooks](webhooks.md). |

## Quickstart

1. **Create an API key** with the `tx:send` scope under **Developers › API keys**. Copy the secret. It's shown once.
2. **Pick a transactional template** under **Developers › Email templates**, or create one. Note its ID.
3. **Send a test** from your code:

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

4. **Check the log** under **Developers › Message log**. The message appears within seconds, with a timeline from accepted to delivered or bounced.

## The rest of the API

Everything you can do in the app can also be done through the REST API: lists, subscribers, campaigns, templates, media, bounces and more. See the [API introduction](../apis/apis.md) and the reference pages under **API**.

Template syntax for both campaigns and transactional email is in the [Templating reference](../templating.md).
