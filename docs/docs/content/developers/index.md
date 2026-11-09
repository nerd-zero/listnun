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

**Developers › Overview** walks you through four steps and ticks each one off as you go:

1. **Create an API key** with the `tx:send` scope under **Developers › API keys**, one per service. Copy the secret: it's shown once, so keep it in your secrets manager.
2. **Pick a transactional template** under **Developers › Email templates**, or create one. Note its ID.
3. **Send a test** from your code. The Overview shows the call in **curl**, **Node**, **Python** and **Go**, already filled in with your workspace's address, your key's name, your template's ID and its sample data. The examples read the secret from a `LISTNUN_API_KEY` environment variable and use HTTP Basic auth (key name as the username, secret as the password):

    === "curl"

        ```shell
        curl -X POST https://yourcompany.listnun.app/api/tx \
          -u "my-service:$LISTNUN_API_KEY" \
          -H "Content-Type: application/json" \
          -d '{
            "subscriber_email": "ada@example.com",
            "template_id": 3,
            "data": {"order_id": "1042", "total": "$25.00"}
          }'
        ```

    === "Node"

        ```js
        const auth = Buffer.from(
          `my-service:${process.env.LISTNUN_API_KEY}`,
        ).toString("base64");

        await fetch("https://yourcompany.listnun.app/api/tx", {
          method: "POST",
          headers: {
            "Authorization": `Basic ${auth}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            subscriber_email: "ada@example.com",
            template_id: 3,
            data: { order_id: "1042", total: "$25.00" },
          }),
        });
        ```

    === "Python"

        ```python
        import os, requests

        requests.post(
            "https://yourcompany.listnun.app/api/tx",
            auth=("my-service", os.environ["LISTNUN_API_KEY"]),
            json={
                "subscriber_email": "ada@example.com",
                "template_id": 3,
                "data": {"order_id": "1042", "total": "$25.00"},
            },
            timeout=10,
        ).raise_for_status()
        ```

    === "Go"

        ```go
        body := strings.NewReader(`{
          "subscriber_email": "ada@example.com",
          "template_id": 3,
          "data": {"order_id": "1042", "total": "$25.00"}
        }`)
        req, _ := http.NewRequest("POST", "https://yourcompany.listnun.app/api/tx", body)
        req.SetBasicAuth("my-service", os.Getenv("LISTNUN_API_KEY"))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        ```

    The `Authorization: token my-service:<secret>` header works too. See [Auth](../apis/apis.md#auth).

4. **Check the log** under **Developers › Message log**. The message appears within seconds, with a timeline from accepted to delivered or bounced.

By default the recipient must already be a subscriber in your workspace, or the call fails with HTTP 400. To email anyone, for example receipts to guest customers, add `"subscriber_mode": "external"`. See the [Transactional API reference](../apis/transactional.md).

### Let an AI assistant write it

The **AI prompt** tab, next to the code tabs, gives you a prompt instead of code. Paste it into Claude, ChatGPT, Cursor or Copilot inside your project and it writes the integration in your own language and framework.

The prompt is filled in with your workspace's address, key name, template ID and the template's variables. It also tells the assistant to:

- keep the key on the server, read from `LISTNUN_API_KEY`, never in browser code,
- use `subscriber_mode` when the recipients aren't subscribers,
- retry only on HTTP 429 and 5xx, with backoff, and log 400s instead of retrying,
- use a 10-second timeout and wrap the call in one reusable function.

Review what the assistant writes like any other code before you ship it.

## The rest of the API

Everything you can do in the app can also be done through the REST API: lists, subscribers, campaigns, templates, media, bounces and more. See the [API introduction](../apis/apis.md) and the reference pages under **API**.

Template syntax for both campaigns and transactional email is in the [Templating reference](../templating.md).
