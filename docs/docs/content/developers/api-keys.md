# API keys

An API key lets your code call the listnun API. Create one per service (your shop, your app's backend, a sync job) under **Developers › API keys**, so you can see where calls come from and revoke one without touching the others.

## Creating a key

Click **Create API key**, give it a name, and tick its scopes. Names are 2–63 characters of lowercase letters, numbers, dots, dashes and underscores, starting with a letter or number, such as `shop-backend`; the name is also what you authenticate with. listnun shows the secret **once**; store it in your secrets manager straight away. After that, only its last four characters are shown.

Secrets start with `lnk_live_`, so they're easy to spot if one leaks into a log or a repository.

<div class="ui">
<div class="ui-col">
<div class="ui-card ui-card-pad ui-col" style="gap: 8px;">
<strong>Copy your secret now</strong>
<span class="ui-small">This is the only time it's shown. Store it in your secrets manager.</span>
<div class="ui-row"><span class="ui-input ui-grow ui-mono">lnk_live_8f2c1d6b0e4a9f37c5b2e8d1a6f0c3b7</span><span class="ui-btn">Copy</span></div>
</div>
<div class="ui-card ui-divide">
<div class="ui-line"><span class="ui-grow"><strong>shop-backend</strong><br><span class="ui-small ui-mono">lnk_live_…c3b7</span></span><span class="ui-tag">tx:send</span><span class="ui-small">Last used 2 minutes ago</span><span class="ui-tag is-ok">Active</span></div>
<div class="ui-line"><span class="ui-grow"><strong>crm-sync</strong><br><span class="ui-small ui-mono">lnk_live_…91ae</span></span><span class="ui-tag">subscribers:read</span><span class="ui-small">Not used yet</span><span class="ui-tag is-ok">Active</span></div>
<div class="ui-line"><span class="ui-grow"><strong>old-mailer</strong><br><span class="ui-small ui-mono">lnk_live_…07d2</span></span><span class="ui-tag">tx:send</span><span class="ui-small">Last used 3 months ago</span><span class="ui-tag is-bad">Revoked</span></div>
</div>
</div>
<p class="ui-caption">Developers › API keys, just after creating a key.</p>
</div>

## Scopes

A key can only do what its scopes allow. It has no other permissions, whatever its creator can do.

| Scope | Allows |
| :---- | :----- |
| `tx:send` | Sending transactional email (`POST /api/tx`) and resending logged messages. |
| `tx:logs` | Reading the message log and sending stats. |
| `subscribers:read` | Reading subscribers. |
| `keys:manage` | Listing, creating, rotating and revoking API keys. |
| `webhooks:manage` | Managing webhook endpoints and deliveries. |

Give each key the fewest scopes it needs. A service that only sends receipts needs just `tx:send`.

## Authenticating

Send the key's name and secret in the `Authorization` header:

```shell
curl https://yourcompany.listnun.app/api/developers/messages \
  -H "Authorization: token my-service:lnk_live_…"
```

HTTP Basic auth with the same pair also works: `curl -u "my-service:lnk_live_…" …`.

A key belongs to one workspace and only works on that workspace's address.

## Rotating

**Rotate** generates a new secret and shows it once. The previous secret keeps working for **24 hours**, so you can deploy the new one without downtime. After that, only the new secret works.

## Revoking and deleting

**Revoke** stops a key working immediately, but keeps it in the list for your records. **Delete** removes it completely.

The list shows when each key was last used, which helps you find keys nothing uses any more.

## Keys migrated from API users

Older workspaces used "API users" (a username and token) instead of keys. Each one has been turned into a key with matching scopes, and its old `username:token` pair keeps working until you're ready to switch:

1. **Rotate** the migrated key to get a real `lnk_live_…` secret. Until you do, it shows "No key secret yet".
2. Update your code to use the key's name and new secret.
3. Choose **Disable old token** from the key's menu. The old pair stops working.
