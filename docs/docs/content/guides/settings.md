# Settings

**Workspace › Settings** holds your workspace's configuration, one section per item on the left. Changes aren't saved until you click **Save** on the bar that appears at the bottom; **Discard** undoes them. If you try to leave with unsaved changes, listnun asks first.

On hosted listnun, some settings, such as SMTP and Mail Validation, may be managed for you and shown read-only.

## General

- **Root URL**, **Site name**, **Logo URL** and **Favicon URL**: your workspace's public address and branding on its public pages.
- **Default `from` email** for new campaigns.
- **Admin notification e-mails**: who gets told when, for example, an import or a campaign finishes.
- **Language** of the app and the public pages.
- Whether to show the **public subscription page** and the **public mailing list archive** (and whether its RSS feed has the full content).
- Double opt-in behaviour: whether to send the confirmation email and show the confirmation page.

## Performance

How fast campaigns send: how many messages go out at once, a rate limit per second, and an optional sliding window (at most N messages per period) to stay within your mail provider's limits. Leave these alone unless your provider asks you to send more slowly.

## Privacy

| Setting | What it does |
| :------ | :----------- |
| **Individual subscriber tracking** | Records which subscriber opened or clicked. Off counts opens and clicks only in total. Inactive cleanup needs it on. |
| **Disable tracking** | Turns off open and click tracking completely. |
| **Include `List-Unsubscribe` header** | Lets email apps show their own unsubscribe button. Recommended. |
| **Allow blocklisting**, **preference changes**, **exporting**, **wiping** | What subscribers can do for themselves from the links in your emails. |
| **Record opt-in IP address** | Keeps the IP address people confirm their subscription from, as proof of consent. |
| **Domain blocklist** / **allowlist** | Refuse signups and imports from some email domains, or only accept certain ones. |

## Security

- **Single sign-on (OIDC)**: let your team sign in with Google, Microsoft or another provider. See [OIDC SSO](../oidc.md).
- **Captcha**: protect your signup forms, and optionally the sign-in page, with Cloudflare Turnstile.
- **Require two-step verification**: everyone who signs in with a password must set up an authenticator app. See [Your team](team.md#requiring-it-for-everyone).
- **Trusted redirect URLs**: the addresses signup forms may send people to afterwards.

## Media uploads

Where uploaded files are stored, which file types are allowed, and the **maximum upload size**.

- **Local filesystem** keeps files on the listnun server.
- **S3-compatible** uses any S3-style object storage: AWS S3, Hetzner Object Storage, Cloudflare R2, MinIO, RustFS and others. Set the **Endpoint** to the provider's S3 URL (for example `https://fsn1.your-objectstorage.com` for Hetzner), plus the region, bucket and access keys. A **public** bucket gives files plain links; a **private** one uses links that expire.

See [Media storage](../operators/media-storage.md) for setting a bucket up.

## SMTP

The mail servers that send your email. You can add several, and listnun spreads sending across the enabled ones. Use **Test connection** after changing one.

## Bounces

What happens when an email bounces (nothing, unsubscribe, blocklist or delete, separately for hard and soft bounces and complaints), and how bounces reach listnun: from your provider's webhooks (Amazon SES, SendGrid, Postmark and others) or by checking a bounce mailbox. See [Bounce processing](../bounces.md).

## Messengers

Optional extra ways to send, such as SMS, through HTTP webhooks ("postback" messengers). See [Messengers](../messengers.md).

## Appearance

Custom CSS and JavaScript for the admin app and for your public pages (signup, unsubscribe, archive).

## Mail Validation

Checks your subscribers' email addresses with Scrub, so campaigns don't go to addresses that would bounce. You can validate a list on demand, and imports are checked in the background. While a list is being validated, campaigns to it can't start.

## Inactive cleanup

Subscribers who never open or click hurt how often your emails reach inboxes. Inactive cleanup deals with them on a schedule. It's **off** until you turn on **Run inactive cleanup**.

Pick **Managed** to use listnun's recommended rules (shown read-only), or **Custom** to set your own:

| Rule | Options |
| :--- | :------ |
| **Inactive for** | No activity for 30 to 365 days. |
| **Counts as activity** | Opened, clicked, or either. |
| **Action** | Unsubscribe from all lists, blocklist, or delete. Delete can't be undone. |
| **Applies to** | All lists, or only some. |
| **Runs** | Daily, weekly or monthly, at 02:00. |

To be safe, someone only counts as inactive if they joined the list before the window began and a campaign actually went to that list during it. Blocklisted subscribers are left alone.

The card at the bottom shows when it runs next and how many subscribers match today. **Preview affected subscribers** counts matches for the rules on screen, before you save.

Cleanup needs **Individual subscriber tracking** (in Privacy) to be on. Without it, everyone would look inactive, so listnun won't run it and shows a warning.

## Maintenance

Tidy-up jobs you run by hand. Each one asks you to confirm, and none can be undone.

- **Subscribers**: delete orphans (subscribers on no list) or all blocklisted subscribers, with their activity.
- **Subscriptions**: remove double opt-in subscriptions still unconfirmed after a date.
- **Analytics**: delete view and click records before a date (campaign totals stay), or export them first.
- **Database**: vacuum the database on a schedule to keep it fast.

## Logs

Your workspace's recent activity: settings changes and who made them, campaigns starting, pausing and finishing, imports finishing, webhook retries and failures, and inactive cleanup runs. It shows the latest 500 lines and keeps 30 days. Turn on **Auto-refresh** to watch it live.
