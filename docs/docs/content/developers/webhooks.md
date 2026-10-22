# Webhooks

A webhook tells your system what happened to a transactional message: listnun sends an HTTP `POST` to your endpoint each time a message is sent, delivered, bounces or fails. Set them up under **Developers › Webhooks**.

## Adding an endpoint

Click **Add endpoint**, enter its URL, and choose the events it should get. listnun shows the endpoint's **signing secret** (starting `whsec_`). Store it; you'll use it to check requests really come from listnun. You can **Reveal** it again later. **Rotate secret** replaces it, and deliveries are signed with the new one straight away, so update your endpoint at the same time.

Endpoints must be public `https://` addresses. listnun refuses to call private, internal or loopback addresses.

**Send test event** sends a `webhook.test` event so you can check your endpoint before real traffic arrives.

<div class="ui">
<div class="ui-card ui-divide">
<div class="ui-card-head"><span class="ui-mono">https://api.acme.com/hooks/listnun</span><span class="ui-row"><span class="ui-btn is-sm">Send test event</span><span class="ui-tag is-ok">Active</span></span></div>
<div class="ui-line ui-small"><span>message.sent</span><span>message.delivered</span><span>message.bounced</span><span>message.failed</span><span class="ui-grow"></span><span>Last 24 hours: 1,204 delivered, 99.8% OK, median 140 ms</span></div>
<div class="ui-line"><span class="ui-tag is-ok">200</span><span class="ui-mono ui-grow">message.delivered</span><span class="ui-small">ada@example.com</span><span class="ui-small">1 attempt, 120 ms</span></div>
<div class="ui-line"><span class="ui-tag is-warn">503</span><span class="ui-mono ui-grow">message.bounced</span><span class="ui-small">alan@example.com</span><span class="ui-small">Retrying: attempt 3 in 1 h</span><span class="ui-btn is-sm">Retry</span></div>
<div class="ui-line"><span class="ui-tag is-ok">204</span><span class="ui-mono ui-grow">message.sent</span><span class="ui-small">grace@example.com</span><span class="ui-small">1 attempt, 95 ms</span></div>
</div>
<p class="ui-caption">A webhook endpoint with its recent deliveries.</p>
</div>

## Events

| Event | When |
| :---- | :--- |
| `message.sent` | The message was handed to the mail server. |
| `message.delivered` | The receiving server accepted it. |
| `message.bounced` | It bounced. |
| `message.failed` | listnun couldn't send it. |

## The request

```http
POST /your/endpoint HTTP/1.1
Content-Type: application/json
User-Agent: listnun-webhooks/1
Listnun-Event: message.bounced
Listnun-Delivery-Id: 7b1f…
Listnun-Signature: t=1761141600,v1=5f2c…

{
  "type": "message.bounced",
  "created_at": "2026-10-22T14:00:00Z",
  "data": {
    "message_id": "0c9e…",
    "template_id": 3,
    "recipient": "ada@example.com",
    "status": "bounced",
    "bounce": {"type": "hard", "detail": "550 5.1.1 user unknown"}
  }
}
```

`data.bounce` only appears on `message.bounced`, and `data.error` only on `message.failed`. `Listnun-Delivery-Id` is the same on every retry of a delivery, so you can use it to ignore duplicates.

## Checking the signature

`Listnun-Signature` is `t=<unix seconds>,v1=<signature>`, where the signature is the hex HMAC-SHA256 of `<t>.<raw request body>` using the endpoint's secret. To check a request:

1. Split the header into `t` and `v1`.
2. Compute HMAC-SHA256 of `t + "." + body` with your secret. Use the raw body bytes, before any JSON parsing.
3. Compare it with `v1` using a constant-time comparison.
4. Reject requests whose `t` is more than a few minutes old, so a captured request can't be replayed.

```python
import hashlib, hmac, time

def verify(secret: str, header: str, body: bytes, tolerance: int = 300) -> bool:
    parts = dict(p.split("=", 1) for p in header.split(","))
    t, sig = parts.get("t", ""), parts.get("v1", "")
    if not t.isdigit() or abs(time.time() - int(t)) > tolerance:
        return False
    expected = hmac.new(secret.encode(), f"{t}.".encode() + body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig)
```

```js
import crypto from "node:crypto";

export function verify(secret, header, rawBody, tolerance = 300) {
  const parts = Object.fromEntries(header.split(",").map((p) => p.split("=")));
  const t = Number(parts.t);
  if (!t || Math.abs(Date.now() / 1000 - t) > tolerance) return false;
  const expected = crypto.createHmac("sha256", secret).update(`${t}.${rawBody}`).digest("hex");
  return expected.length === parts.v1?.length &&
    crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(parts.v1));
}
```

## Responding and retries

Reply with any `2xx` status within **10 seconds** to accept a delivery. Do slow work after replying, not before.

Anything else, including a timeout or an error, counts as a failure, and listnun tries again after 4 minutes, 15 minutes, 1 hour, 4 hours, then every 6 hours. A delivery gets **8 attempts**, about 23 hours in all, before it's marked failed and recorded in **Settings › Logs**.

The endpoint's page shows recent deliveries with their status, response code and timing. Open one to see the payload and your endpoint's reply, or **Retry** it by hand.
