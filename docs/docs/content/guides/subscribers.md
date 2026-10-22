# Subscribers

Open **Subscribers › All subscribers** to see everyone in your workspace. Search by email or name, filter by list or status, and click a subscriber to open their details on the right.

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

## Working with many at once

Tick subscribers (or use the header checkbox to select everyone matching your search) and the action bar appears above the table. You can add them to lists, remove them from lists, mark subscriptions unsubscribed, blocklist or delete them.

## Advanced search

For precise segments, such as everyone in a given city who joined this year, use the advanced query, which takes an SQL expression. See [Querying and segmenting subscribers](../querying-and-segmentation.md).

## Bounces

When an email can't be delivered, the mail provider reports a bounce. **Subscribers › Bounces** lists them, filtered by type:

| Type | Meaning |
| :--- | :------ |
| **Hard** | The address doesn't exist or permanently refuses mail. |
| **Soft** | A temporary problem, such as a full mailbox. |
| **Complaint** | The person marked your email as spam. |

What happens after a bounce (nothing, unsubscribe, blocklist or delete) is set in **Settings › Bounces**. See [Bounce processing](../bounces.md) for how bounces reach listnun.

## Inactive subscribers

People who never open or click hurt how often your emails reach the inbox. **Settings › Inactive cleanup** can tidy them up on a schedule. See [Settings](settings.md#inactive-cleanup).
