# Lists and forms

## Lists

A list is a group of subscribers you send campaigns to. Open **Lists › All lists** to see them, with how many subscribers each has and how many are still waiting to confirm.

Each list has two settings that matter:

| Setting | Options |
| :------ | :------ |
| **Type** | **Public** lists can be added to signup forms and appear on subscribers' preferences page. **Private** lists are for your own segments; people can't join them themselves. |
| **Opt-in** | **Single**: people are subscribed straight away. **Double**: each new subscriber gets an email with a confirmation link, and campaigns only go to subscribers who confirmed. |

Use double opt-in for lists people sign up to on your website. It keeps typos and fake addresses off your list and helps your emails reach inboxes.

Click a list to open its panel:

- **Growth**: how many people joined and left each day over the last 30 days.
- **Campaigns**: its latest campaigns, with how many people opened and clicked.
- **Signup form** (public lists only): a link to a ready-made signup page for the list. See [A quick link for one list](#a-quick-link-for-one-list).

From a list's **⋯** menu you can view its subscribers, import into it, or draft an opt-in campaign that asks its unconfirmed subscribers to confirm.

### People waiting to confirm

When people haven't confirmed a double opt-in subscription yet, a banner at the top of **Lists** says how many. **Resend confirmation** emails them the confirmation again. Anyone who was sent one, or joined a list, in the last day is skipped, so nobody gets two in a row. Only one resend runs at a time. You can also [confirm people yourself](subscribers.md#confirming-someone-yourself).

Deleting a list doesn't delete its subscribers. They just stop being on that list.

## Signup forms

**Lists › Forms** is where you keep your signup forms. Each form has its own lists, fields and wording, and you can share it three ways: as a hosted page, as HTML embedded in your website, or as a popup.

### Making a form

Click **New form**, then set:

| Setting | What it does |
| :------ | :----------- |
| **Form name** | For your team. While the form is a draft, it also sets the hosted page's address. |
| **Lists people can join** | Only public lists can be added. With one list, people join it; with several, they tick the ones they want. The form tells you if any of them use double opt-in, in which case new people get a confirmation email before they're added. |
| **Fields** | Email always comes first and is required. Add **Name**, or up to 10 fields in all with **Add a field**: each extra field (e.g. *City*) is saved to the subscriber's attributes. Mark fields required or optional and drag them into order. |
| **Heading**, **Intro**, **Button label** | The text people see. |
| **After someone signs up** | The thank-you message. Left empty, people see a standard message, telling them to check their inbox when a list uses double opt-in. |
| **Bot protection** | Adds a Cloudflare Turnstile check. Set up Turnstile under **Settings › Sign-in and security** first. Every form also has a hidden trap field that catches simple bots. |

The preview shows the form in light and dark themes. **Save form** keeps your changes.

### Publishing and sharing

A new form is a **Draft**: its page, embed and popup don't work until you click **Publish form**. A live form needs at least one list. While it's live, its address stays fixed even if you rename it, so links keep working. **Unpublish** turns it back into a draft: its page, embed and popup stop working, and renaming it then can change its address.

A live form's **Share** section has:

| Way to share | How |
| :----------- | :-- |
| **Hosted page** | A page on your workspace's address, at `/f/<form>`. Link to it from social posts, email signatures or a QR code. |
| **Embed** | HTML to paste into your website where the form should go. It's unstyled, so it picks up your site's CSS, and it posts to the hosted page, which shows the result. |
| **Popup** | A `<script>` tag to add to your website's pages. The form pops up 8 seconds after a page opens, at most once a month per visitor. |

### How a form is doing

Each form shows its signups in the last 30 days in the list of forms. Open a form to see, for the last 30 days, its **Views**, **Signups**, how many of those **Confirmed**, and how many submissions were **Blocked by Turnstile**.

Deleting a form stops its page, embeds and popup, and deletes its figures. People who signed up through it stay subscribed.

### A quick link for one list

Every public list also has a ready-made signup link. Open the list and go to its **Signup form** tab to copy it. The link shows a simple form for just that list, with no setup needed. **Customise the form** takes you to **Forms** when you want more control.

## What subscribers see

Every campaign includes an unsubscribe link. Depending on your **Settings › Privacy and consent**, subscribers can also:

- manage which of your public lists they're on,
- download the data you hold about them,
- delete their data.

People who unsubscribe land on a page where they can resubscribe if it was a mistake.
