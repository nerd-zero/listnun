# Subscribers

Open **Subscribers › All subscribers** to see everyone in your workspace. Pick a segment on the left to narrow the table down, search by email or name, and click a subscriber to open their details on the right.

## A subscriber's details

The detail panel shows:

- their email, name and status,
- the lists they're on, and whether each subscription is confirmed, unconfirmed or unsubscribed,
- their attributes: any extra data you keep about them, such as a city or a plan (see [Concepts](../concepts.md#attributes)),
- their activity: campaigns sent to them, opens, clicks and bounces.

From there you can edit them, change their lists, blocklist them or delete them.

## Statuses

A subscriber has one overall status:

| Status | Meaning |
| :----- | :------ |
| **Enabled** | Normal. They get campaigns for the lists they're on. |
| **Blocklisted** | They get nothing, whatever lists they're on. Used for people who asked never to be emailed, or addresses that keep bouncing. |

Each list subscription has its own status too: **Unconfirmed** (double opt-in lists only: waiting for them to click the confirmation link), **Confirmed**, or **Unsubscribed**.

### Confirming someone yourself

If someone confirmed some other way, such as by replying or on the phone, you can confirm them yourself instead of waiting for them to click the link:

- **One person:** open them, go to the **Subscriptions** tab and click **Mark as confirmed** next to the list.
- **Many people:** select them, click **Add to list**, choose **Mark as confirmed** and pick the lists.

This only confirms subscriptions that are waiting. Unsubscribed people stay unsubscribed, so you can't re-subscribe someone by accident, and nobody gets added to a list they weren't on.

## Segments

A segment is a group of subscribers who match some conditions, such as "lives in Harare and joined in the last 90 days". The **Segments** column on the left of **Subscribers** has built-in ones:

| Segment | Who's in it |
| :------ | :---------- |
| **Everyone** | All subscribers. |
| **Engaged** | Opened a campaign in the last 90 days. |
| **New this month** | Joined in the last 30 days. |
| **Never opened** | Have never opened a campaign. |
| **Not confirmed yet** | Have at least one double opt-in subscription waiting for confirmation. |
| **Blocklisted** | Blocklisted subscribers. |

Below them, under **Saved by your team**, are the segments your team saved.

### Building a segment

Click **New segment** (or change the conditions of any segment) and choose whether people must match **all** or **any** of the conditions. Each condition is a field, a test and usually a value:

| Field | Tests |
| :---- | :---- |
| **Attribute** (enter its name, e.g. `city`) | is, is not, contains, is set |
| **Email**, **Name** | is, is not, contains, is set |
| **Joined** | in the last / not in the last N days, before / after a date |
| **On list** | is, is not (unsubscribed people don't count as on the list) |
| **Status** | is, is not: enabled or blocklisted |
| **Waiting to confirm** | is |
| **Last opened**, **Last clicked** | in the last / not in the last N days, before / after a date, never |

The table and the count above it update as you edit. A segment can have up to 20 conditions. Click **Save segment** to give it a name and keep it for your team; **Delete segment** removes a saved segment but not the people in it. Saving and deleting segments needs the `subscribers:manage` permission.

### Using a segment

Whatever segment you're viewing also limits:

- **select all**: ticking the header checkbox selects everyone in the segment, not everyone in the workspace,
- the bulk actions on that selection (add to or remove from lists, unsubscribe, blocklist, delete),
- **Export**, which downloads only the people in the segment.

### Edit as SQL

If you have the `subscribers:sql_query` permission, **Edit as SQL** turns the segment's conditions into an SQL expression in the advanced search box, for cases the builder can't express. See [Querying and segmenting subscribers](../querying-and-segmentation.md). Segments themselves don't need that permission.

## Working with many at once

Tick subscribers (or use the header checkbox to select everyone in the current segment and search) and the action bar appears above the table. You can add them to lists, remove them from lists, mark their subscriptions confirmed or unsubscribed, blocklist or delete them.

## Bounces

When an email can't be delivered, the mail provider reports a bounce. **Subscribers › Bounces** lists them, filtered by type:

| Type | Meaning |
| :--- | :------ |
| **Hard** | The address doesn't exist or permanently refuses mail. |
| **Soft** | A temporary problem, such as a full mailbox. |
| **Complaint** | The person marked your email as spam. |

Search by email or campaign, and filter by type, campaign or source. Click a bounce to see what the provider reported: its type, description and the mail server's reply.

Select bounces to **Delete** the records or **Blocklist** the subscribers behind them. **Delete all** and selecting everything act on exactly what the list shows, so with a filter or search on, only the matching bounces are affected. Deleting bounce records doesn't change the subscribers.

Bounces are only recorded for subscribers. A bounce for an address that isn't one, such as a [transactional email](../developers/transactional.md) to a non-subscriber or a subscriber you've since deleted, is skipped.

What happens after a bounce (nothing, unsubscribe, blocklist or delete) is set in **Settings › Bounces and complaints**. See [Bounce processing](../bounces.md) for how bounces reach listnun.

## Inactive subscribers

People who never open or click hurt how often your emails reach the inbox. **Settings › Inactive subscribers** can tidy them up on a schedule. See [Settings](settings.md#inactive-subscribers).
