# Writing and sending campaigns

A campaign is an email you send to one or more lists. **Campaigns › All campaigns** lists them with their status and results; **Create new** starts one.

The editor has four steps along the top. Once the campaign is created you can move between them freely. **Continue** saves the step you're on and moves to the next.

## Settings

| Field | Notes |
| :---- | :---- |
| **Campaign name** | For you and your team; subscribers don't see it. |
| **Subject** | The subject line. It can use template expressions, such as `{{ .Subscriber.FirstName }}`. |
| **From address** | Who it's from, written as `Your Name <you@yourdomain.com>`. Defaults to your workspace's address. |
| **Lists to send to** | One or more lists. Someone on several of them gets the email once, and blocklisted subscribers are never sent to. The **Audience** card shows how many subscribers that is. |
| **Tags** | Optional labels to organise your campaigns. |

**Advanced** holds the messenger (normally email), custom email headers, and campaign attributes: a JSON object you can use in the content as `{{ .Campaign.Attribs.* }}`.

## Content

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
- **Tracking**: track link clicks and add tracking pixels. See [Tracking opens and clicks](tracking.md).
- **Plain-text version**: for HTML formats, add a plain-text alternative for email apps that can't show HTML. Without one, those apps show nothing.

**Preview** (or <kbd>F9</kbd>) shows the email as a subscriber will see it, with sample data.

To personalise content, such as a subscriber's name, use template expressions like `{{ .Subscriber.Name }}`. See the [Templating reference](../templating.md).

## Schedule

Choose **Send now** to start as soon as you confirm on the Review step, or **Schedule for later** and pick a date and time. Times are in your browser's time zone.

**Archive** publishes the campaign on your public archive page, a web page listing past campaigns people can browse. Set the URL slug and the sample subscriber data shown in place of real names. See [Archives](../archives.md).

## Review

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
- **Audience**: how many subscribers, on which lists.
- **Content** isn't empty.
- **Test email**: amber until you've sent a test of exactly what's in the editor now. If you change the subject, content, attachments or tracking after testing, it turns amber again.
- **Tracking**: whether link clicks are tracked, and how many pixels.
- **Plain-text version** and **Schedule**.

Anything that needs attention has a button to fix it.

### Send a test

Click **Send test** (on the Review step or at the top of the editor) and enter one or more email addresses. They must be subscribers in your workspace. The test uses what's in the editor, even if you haven't saved it, so you see exactly what will go out.

### Start

**Start campaign** (or **Schedule**) asks you to confirm, with the number of subscribers it goes to. If you haven't sent a test of this version, the confirmation says so.

## While it sends

In **All campaigns**, a running campaign shows its progress. From its **⋯** menu you can **Pause** it and resume it later, **Cancel** it, **Unschedule** a scheduled one, or open its **Analytics**. The copy button on any campaign **clones** it, a quick way to start a new campaign from an old one.

If your workspace uses [Mail Validation](settings.md#mail-validation), starting is held back while a list is still being validated.

## Results

**Campaigns › Analytics** compares campaigns over time: views (opens), clicks, bounces split into hard and soft, and which links were clicked most. Pick the campaigns and the date range at the top.

Opens are counted with a tracking image, which many email apps block until the reader allows images, so open counts are always a little low. Clicks are reliable. See [Tracking opens and clicks](tracking.md).
