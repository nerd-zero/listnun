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
| `POST` | `/api/operator/tenants/:id/setup-link` | Issue a fresh setup link. Links only last as long as the server process, so reissue after a restart. |
| `PUT` | `/api/operator/tenants/:id/status` | Suspend, reactivate or disable: `{"status": "suspended"}`. Suspended workspaces don't send campaigns. |
| `DELETE` | `/api/operator/tenants/:id` | Permanently delete a workspace and all its data. Workspace 1 can't be deleted. |
| `PUT` | `/api/operator/tenants/:id/smtp` | Set the workspace's SMTP server. |
| `PUT` | `/api/operator/tenants/:id/scrub` | Configure Mail Validation (Scrub) for the workspace. |
| `GET` | `/api/operator/tenants/:id/scrub/usage` | The workspace's validation usage. |
| `PUT` | `/api/operator/tenants/:id/bounce-webhooks` | Configure bounce webhooks for the workspace's mail provider. |
| `PUT` | `/api/operator/tenants/:id/custom-domain` | Set or clear the workspace's custom domain. |
| `GET` | `/api/operator/tenants/:id/storage` | Media storage the workspace uses. |
| `GET` | `/api/operator/tenants/:id/usage` | Emails sent and validations over `?from=&to=` (RFC 3339), optionally `&group=day`. |
| `GET` | `/api/operator/usage` | The same for every workspace at once, plus a daily series. |
| `GET` `POST` `DELETE` | `/api/operator/organizations…` | Group workspaces by customer. Deleting an organization detaches its workspaces; it doesn't delete them. |

The full request and response shapes are in the OpenAPI spec at `/api/swagger/index.html`, under **operator**.

### Usage for billing

listnun counts each workspace's billable usage when it happens: every email sent through an SMTP messenger (one per recipient), and every address submitted for validation. Counts are kept per hour. A `from`/`to` window counts each hour once, so back-to-back billing periods never double-count. A deleted workspace's usage stays available, so it can still be invoiced.
