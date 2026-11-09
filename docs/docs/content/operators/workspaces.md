# Workspaces and the operator API

One listnun instance can host many **workspaces** (tenants). Each has its own subscribers, campaigns, users and settings, isolated in the database by Postgres row-level security. Workspaces are created and managed through the **operator API**, not the admin app.

A plain self-hosted install doesn't need any of this. With multi-tenancy off (the default), everything belongs to a single workspace.

## Turning on multi-tenancy

```toml
[app]
multi_tenancy_enabled = true
root_domain = "listnun.example.com"

# Where suspended workspaces' admin UI redirects to (optional).
console_url = "https://console.example.com"
```

Every request must then arrive on a workspace's own host:

- `<slug>.listnun.example.com`, from the workspace's slug, or
- the workspace's custom domain, if one is set.

Requests for unknown hosts are rejected. Suspended or disabled workspaces show a "workspace unavailable" page: their public pages and API return `503`, and their admin app redirects to `console_url` when it's set.

You'll need a wildcard DNS record (`*.listnun.example.com`) and a wildcard TLS certificate, for example from Let's Encrypt using the DNS-01 challenge.

Sessions and API keys belong to one workspace. A session cookie or key from one workspace is refused on another's host.

Links listnun generates use the host the request came in on, so each workspace's links point at its own address: password-reset emails, signup forms' hosted pages, embed code and popup scripts, and the archive feed. The `root_url` setting is only used for single-workspace installs. If a workspace has a custom domain, open its admin app on that domain to get links on it.

!!! note "Testing locally"
    Browsers resolve `*.localhost` to your machine, so `acme.localhost:9000` works for reading pages. Signing in through the browser needs a real domain, though; an `/etc/hosts` entry isn't enough.

## The operator API

The operator API is a separate, cross-workspace API under `/api/operator`, for provisioning and billing systems. It's off until you set a token:

```toml
[operator]
token = "a-long-random-secret"

# A separate Postgres role with BYPASSRLS, for reading across workspaces.
db_user = "listnun_operator"
db_password = "…"

# "dev" makes generated workspace URLs <slug>-dev.<root_domain>, so a dev
# and a prod instance can share a root domain. Empty for prod.
env = ""
```

Authenticate with the token as a bearer token:

```shell
curl -H "Authorization: Bearer a-long-random-secret" https://listnun.example.com/api/operator/tenants
```

!!! warning
    The operator token can create and delete any workspace. Keep it in your secrets manager, never in a browser. Use a dedicated Postgres role with `BYPASSRLS` for `db_user`; the app's own role must never have it, or row-level security stops protecting workspaces from each other.

### Endpoints

| Method | Endpoint | Does |
| :----- | :------- | :--- |
| `GET` | `/api/operator/tenants` | List workspaces, with status and user and subscriber counts. |
| `GET` | `/api/operator/tenants/:id` | One workspace. |
| `POST` | `/api/operator/tenants` | Create a workspace (slug, name, first admin, optional organization). Returns a one-time setup link for the admin to set their own password. |
| `POST` | `/api/operator/tenants/:id/setup-link` | Issue a fresh setup link. Links expire after 7 days, work once, and only last as long as the server process, so reissue after a restart. |
| `GET` | `/api/operator/tenants/:id/setup-link?token=&email=` | Check a setup link without using it up: `{"valid": …, "password_set": …}`. `valid` is whether it still works for that admin; `password_set` is whether the admin has a password, meaning the link was used. Doesn't count towards the link's attempt limit. |
| `PUT` | `/api/operator/tenants/:id/status` | Suspend, reactivate or disable: `{"status": "suspended"}`. Suspended workspaces don't send campaigns. |
| `DELETE` | `/api/operator/tenants/:id` | Permanently delete a workspace and all its data. Workspace 1 can't be deleted. |
| `PUT` | `/api/operator/tenants/:id/smtp` | Set the workspace's SMTP server. It's used from the next send. |
| `PUT` | `/api/operator/tenants/:id/from-email` | Set the workspace's default From address, which new campaigns start with: `{"from_email": "Acme <news@acme.com>"}` or a bare address. New workspaces start with a placeholder that can't send, so set this once a real sending address is verified. The workspace can still change it in its settings. |
| `PUT` | `/api/operator/tenants/:id/scrub` | Configure email validation (Scrub) for the workspace. |
| `GET` | `/api/operator/tenants/:id/scrub/usage` | The workspace's validation usage. |
| `PUT` | `/api/operator/tenants/:id/bounce-webhooks` | Configure bounce webhooks for the workspace's mail provider. |
| `PUT` | `/api/operator/tenants/:id/replies` | Set up [reply tracking](#reply-tracking) for the workspace. |
| `PUT` | `/api/operator/tenants/:id/custom-domain` | Set or clear the workspace's custom domain. |
| `GET` | `/api/operator/tenants/:id/storage` | Media storage the workspace uses. |
| `GET` | `/api/operator/tenants/:id/usage` | Emails sent and validations over `?from=&to=` (RFC 3339), optionally `&group=day`. |
| `GET` | `/api/operator/usage` | The same for every workspace at once, plus a daily series. |
| `GET` | `/api/operator/tenants/:id/activity` | The workspace's latest activity log lines (its **Settings › Activity log**). `?source=sending` returns only send failures; `?limit=` up to 500. |
| `GET` | `/api/operator/tenants/:id/sending-health` | Campaign email sent, hard bounces and spam complaints over the last 30 days, rated against Gmail's limits the same way as the campaign [inbox check](../guides/campaigns.md#inbox-check), plus the workspace's unsent campaigns that have no unsubscribe link and so can't start. |
| `GET` `POST` `DELETE` | `/api/operator/organizations…` | Group workspaces by customer. Deleting an organization detaches its workspaces; it doesn't delete them. |

The full request and response shapes are in the OpenAPI spec at `/api/swagger/index.html`, under **operator**.

### Reply tracking

Campaign [reply tracking](../guides/campaigns.md#replies) uses each workspace's own Postmark server and its inbound stream. To turn it on for a workspace, send its setup:

```shell
curl -X PUT -H "Authorization: Bearer a-long-random-secret" \
  -H "Content-Type: application/json" \
  https://listnun.example.com/api/operator/tenants/42/replies \
  -d '{
    "inbound_address": "abc123@inbound.postmarkapp.com",
    "secret": "at-least-16-characters",
    "webhook_username": "replies",
    "webhook_password": "…"
  }'
```

| Field | Notes |
| :---- | :---- |
| `inbound_address` | The Postmark server's inbound address, or the same local part on a [reply domain](#reply-domains). Each email's Reply-To adds a token to it (`<local>+<token>@<domain>`). Empty turns reply tracking off for the workspace. |
| `secret` | Signs the reply tokens, at least 16 characters. Keep it stable: changing it orphans replies to emails already sent. |
| `webhook_username`, `webhook_password` | The basic-auth credentials Postmark sends with each inbound webhook. |

Then point the Postmark server's inbound webhook at `https://<workspace host>/webhooks/replies/postmark` with the same credentials. A reply is only accepted if it arrives at an address with that workspace's inbound local part and a token signed by its secret. The domain isn't checked, so the address can move to a reply domain without breaking replies to emails already sent. The setup is stored apart from the workspace's settings, so the workspace's admins can't see or change it, and the **Track replies** switch only appears in its campaigns once it's set.

#### Reply domains

Reply addresses can use the workspace's own domain instead of `inbound.postmarkapp.com`:

1. The workspace publishes an MX record for a subdomain it uses only for replies: `reply.acme.com MX 10 inbound.postmarkapp.com`. Not its main domain: that MX would send all of its own mail to Postmark.
2. Set that domain as the Postmark server's inbound domain (`InboundDomain` on `PUT /servers/{id}`). Postmark lets an inbound domain belong to only one server.
3. Send the replies setup again with `inbound_address` set to the server's inbound local part on the new domain, e.g. `abc123@reply.acme.com`, and the **same** secret.

Keeping the local part means replies to emails sent before the switch, addressed to `abc123+<token>@inbound.postmarkapp.com`, still match. To undo it, send the setup with the original address first, then clear the server's inbound domain, so no new email carries an address Postmark no longer accepts.

On hosted listnun, the console does all of this from the workspace page's **Reply domain** section.

### Usage for billing

listnun counts each workspace's billable usage when it happens: every email sent through an SMTP messenger (one per recipient), and every address submitted for validation. Counts are kept per hour. A `from`/`to` window counts each hour once, so back-to-back billing periods never double-count. A deleted workspace's usage stays available, so it can still be invoiced.
