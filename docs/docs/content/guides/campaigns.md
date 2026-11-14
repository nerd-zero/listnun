# Writing and sending campaigns

A campaign is an email you send to one or more lists. **Campaigns › All campaigns** opens an overview of what's sending, what's next and what's done; **New campaign** starts one.

## The campaigns overview

| Section | What it shows |
| :------ | :------------ |
| **Paused** banner | One per paused campaign: how far it got and how many people are still waiting, with **Review** and **Resume sending**. |
| **Sending now** | Each running campaign's progress (sent of total), speed, time left and opens so far, refreshed every couple of seconds, with **Pause** and **Live results**, which opens the campaign in Analytics, refreshing every 30 seconds while it sends. |
| **Up next** | Scheduled campaigns by day, with their lists and audience. |
| **Drafts** | The latest three, with who edited them and when. **All drafts** shows the rest. |
| **Sent** | The latest four, with delivered, open rate and click rate. **Show all sent** and **Compare in analytics** go further. |

To find any campaign, use **Search campaigns**. Search, **All drafts** and **Show all sent** open the full table of campaigns, with filters and bulk actions.

## The campaign editor

The editor has four steps along the top: Settings, Content, Schedule and Review. Once the campaign is created you can move between them freely. **Continue** saves the step you're on and moves to the next.

### Settings

| Field | Notes |
| :---- | :---- |
| **Campaign name** | For you and your team; subscribers don't see it. |
| **Subject** | The subject line. It can use template expressions, such as `{{ .Subscriber.FirstName }}`. |
| **From address** | Who it's from, written as `Your Name <you@yourdomain.com>`. Defaults to your workspace's address. |
| **Lists to send to** | One or more lists. Someone on several of them gets the email once, and blocklisted subscribers are never sent to. The **Audience** card shows how many subscribers that is. |
| **Tags** | Optional labels to organise your campaigns. |

**Advanced** holds the messenger (normally email), custom email headers, and campaign attributes: a JSON object you can use in the content as `{{ .Campaign.Attribs.* }}`.

### Content

At the top, pick the **Format** and, for every format except Visual, the **Template** the content sits in (see [Templates and media](templates-and-media.md)).

| Format | Best for |
| :----- | :------- |
| **Visual** | Building an email from blocks (headings, text, images, buttons, columns) without code. Start from scratch or **Import visual template**. |
| **Rich text** | Writing like in a word processor: headings, lists, links and images. |
| **Raw HTML** | Pasting in HTML from another tool, or hand-writing it. |
| **Markdown** | Plain writing that becomes simple HTML. |
| **Plain text** | No formatting at all. |

Switching format converts what you've written, and some formatting may be lost, so listnun asks first.

Under the editor:

- **Attachments**: click **Add attachments** to pick files from your [media library](templates-and-media.md#media). Each attached file is sent with every email, so keep them small.
- **Tracking**: track link clicks, add tracking pixels and, if your workspace has it, track replies. See [Tracking opens and clicks](tracking.md).
- **Plain-text version**: for HTML formats, add a plain-text alternative for email apps that can't show HTML. Without one, those apps show nothing.

**Preview** (or <kbd>F9</kbd>) shows the email as a subscriber will see it, with sample data.

To personalise content, such as a subscriber's name, use template expressions like `{{ .Subscriber.Name }}`. See the [Templating reference](../templating.md).

### Schedule

Choose **Send now** to start as soon as you confirm on the Review step, or **Schedule for later** and pick a date and time. Times are in your browser's time zone.

**Archive** publishes the campaign on your public archive page, a web page listing past campaigns people can browse. Set the URL slug and the sample subscriber data shown in place of real names. See [Archives](../archives.md).

### Review

<div class="ui">
<div class="ui-col">
<div class="ui-row"><span class="ui-title ui-grow">Spring sale</span><span class="ui-tag">Draft</span><span class="ui-btn is-sm">Send test</span><span class="ui-btn is-sm is-primary">Start campaign</span></div>
<div class="ui-steps"><div class="ui-step is-done"><span class="ui-step-mark">✓</span>Settings</div><div class="ui-step is-done"><span class="ui-step-mark">✓</span>Content</div><div class="ui-step is-done"><span class="ui-step-mark">✓</span>Schedule</div><div class="ui-step is-current"><span class="ui-step-mark">4</span>Review</div></div>
<div class="ui-card ui-divide">
<div class="ui-check is-ok"><span class="ui-check-mark">✓</span><span class="ui-grow"><strong>Subject and sender</strong><span class="ui-small">Spring sale: 20% off everything&emsp;Acme &lt;news@acme.com&gt;</span></span></div>
<div class="ui-check is-ok"><span class="ui-check-mark">✓</span><span class="ui-grow"><strong>Audience</strong><span class="ui-small">Up to 10,000 subscribers in Newsletter</span></span></div>
<div class="ui-check is-warn"><span class="ui-check-mark">!</span><span class="ui-grow"><strong>Test email</strong><span class="ui-small">Send yourself a test before you start it.</span></span><span class="ui-btn is-sm">Send test</span></div>
<div class="ui-check is-info"><span class="ui-check-mark">i</span><span class="ui-grow"><strong>Tracking</strong><span class="ui-small">Link clicks are tracked.</span></span></div>
<div class="ui-check is-info"><span class="ui-check-mark">i</span><span class="ui-grow"><strong>Schedule</strong><span class="ui-small">Sends as soon as you start it.</span></span></div>
</div>
</div>
<p class="ui-caption">The Review step, before a test has been sent.</p>
</div>

A checklist of everything worth checking before sending:

- **Subject and sender** are set.
- **Audience**: how many people the campaign will go to, on which lists. Each person counts once, however many of the lists they're on, and only people it will actually be sent to count: not unsubscribed or blocklisted people, nor, for a regular campaign, people who haven't confirmed a double opt-in list.
- **Content** isn't empty.
- **Unsubscribe link**: whether the email has one. See [The unsubscribe link](#the-unsubscribe-link).
- **Test email**: amber until you've sent a test of exactly what's in the editor now. If you change the subject, content, attachments or tracking after testing, it turns amber again.
- **Tracking**: whether link clicks are tracked, how many pixels, and whether replies are tracked.
- **Schedule**.

Anything that needs attention has a button to fix it.

#### The unsubscribe link

Gmail and Yahoo filter bulk email that has no visible unsubscribe link, so listnun warns you when a campaign doesn't have one. The link counts if the campaign's content or its template has `{{ UnsubscribeURL }}` or `{{ ManageURL }}`. A **Visual** campaign doesn't use a template, so for those only the content counts.

If it's missing, the Review step shows an amber **Unsubscribe link** check and the confirmation before sending says so. You can still send it. **Add unsubscribe link** adds a small footer with the link to the end of the content, in its own format. For a Visual campaign, it opens the email builder so you can add one to a footer block.

The template editor warns you when a campaign template has no unsubscribe link.

#### Inbox check

The **Inbox check** on the Review step looks at the campaign as last saved for what commonly sends email to spam. It lists only what fails, each with what to do about it. Passing checks are folded away under **Passing checks**. None of them stop you sending, but the send confirmation tells you how many warnings there are.

| Group | Checks |
| :---- | :----- |
| **Content** | There's a plain-text version. The email is under Gmail's 102 KB limit, past which Gmail cuts it off and hides the rest, unsubscribe link included. It isn't mostly images. No links are empty, `#` or missing `https://`. No link shorteners. |
| **Subject** | It isn't empty or long enough to be cut off on phones (about 60 characters). It isn't mostly capitals. No repeated `!!` or `$$`. No phrases spam filters score. |
| **Sender** | The From address isn't on a free-mail domain such as gmail.com, which other servers can't send for. It's on the domain the workspace is set up to send from. |
| **Audience** | At most half the audience has gone 90 days without opening or clicking, the same rule as [inactive cleanup](settings.md#inactive-subscribers). Over the last 30 days, the workspace's spam complaints stayed under 0.1% and hard bounces under 2% of its campaign email. |

The complaint and bounce rates are only rated once the workspace has sent enough campaign email to measure them.

#### Send a test

Click **Send test** (on the Review step or at the top of the editor) and enter one or more email addresses. They must be subscribers in your workspace. The test uses what's in the editor, even if you haven't saved it, so you see exactly what will go out.

#### Start

**Start campaign** (or **Schedule**) asks you to confirm, with the number of subscribers it goes to. If you haven't sent a test of this version, the confirmation says so.

## While it sends

A running campaign shows on the overview under **Sending now**, where you can **Pause** it. A paused campaign gets a banner at the top; **Resume sending** picks up where it stopped, with the people still waiting.

In the full table of campaigns, each campaign's **⋯** menu can also **Start**, **Schedule** or **Unschedule** it, **Cancel** it, or open its **Analytics**. The copy button on any campaign **clones** it, a quick way to start a new campaign from an old one.

If your workspace uses [email validation](settings.md#email-validation), starting is held back while a list is still being validated.

## Results

**Campaigns › Analytics** compares campaigns: how many people opened, clicked, bounced and, with [Replies](#replies) set up, replied, bounces split into hard and soft, and which links were clicked most. Pick the campaigns at the top. The totals and rates are each campaign's results all time, the same figures as on the Campaigns overview; the chart, links and bounce types follow the date range, which starts from the day the earliest campaign went out until you pick one.

Opens and clicks count people, once each per campaign, and rates are a share of delivered emails (sent, less those that bounced). Anyone who clicked counts as having opened, since many email apps block the image that records an open. Rates need **Settings › Privacy › Individual subscriber tracking**: without it each view and click is counted on its own, so only the counts are shown.

Opens are counted with a tracking image, which many email apps block until the reader allows images, so open counts are always a little low. Clicks are reliable. See [Tracking opens and clicks](tracking.md).

## Replies

If your workspace has reply tracking, the **Tracking** card on the Content step has a **Track replies** switch. It's off by default. If you don't see it, reply tracking isn't set up for your workspace.

With it on, each email's Reply-To is a listnun reply address unique to that subscriber, so subscribers see that address when they reply. On listnun.app, you can put these addresses on your own domain, such as `reply.yourcompany.com`, under **Reply domain** on the workspace's page in the console. listnun records each reply and forwards it to the campaign's Reply-To address, or its From address if it has none, with the subscriber as the Reply-To. You can answer them straight from your own inbox.

Once the campaign starts sending, a **Replies** card on the campaign shows how many replies came in, who replied, and each reply. Out-of-office and other automatic replies are kept and marked **Auto-reply**, but aren't counted as replies.
