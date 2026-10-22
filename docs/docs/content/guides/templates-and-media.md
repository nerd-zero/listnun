# Templates and media

## Templates

A template is the layout around a campaign's content: the header, footer, styles and unsubscribe link. Open **Campaigns › Templates** to see them as cards with a preview, filtered by type:

| Type | Used for |
| :--- | :------- |
| **Campaign / HTML** | Rich text, Raw HTML, Markdown and Plain text campaigns. The campaign's content is placed into the template. |
| **Campaign / Visual** | A starting design for Visual campaigns. Importing one copies its blocks into the campaign, which you then edit freely. |
| **Transactional** | Transactional emails sent through the API. These are edited under **Developers › Email templates**. See [Transactional email](../developers/transactional.md). |

One campaign template is the **Default**: new campaigns use it unless you pick another. Set it from a template's **⋯** menu, where you can also clone and delete templates. The default template can't be deleted; make another one the default first.

### Writing an HTML template

An HTML campaign template is a full HTML email with one required line where the campaign's content goes:

```html
{{ template "content" . }}
```

Templates use the listmonk template language, so you can add the subscriber's name, unsubscribe link and so on. Include these two so your emails work properly:

- `{{ UnsubscribeURL }}`: the unsubscribe link. Every bulk email should have one.
- `{{ TrackView }}`: the invisible image that counts opens. Without it, opens aren't counted for campaigns using this template.

See the [Templating reference](../templating.md) for everything you can use. **Preview** shows the template with sample content and a sample subscriber.

## Media

**Campaigns › Media** is your workspace's library of images and files, used for images in campaigns, campaign attachments and your email logo.

- Drop files on the upload area, or click to choose them. Uploads over your workspace's size limit (set in **Settings › Media uploads**) are refused.
- Click a file to open it. Tick files to delete several at once.
- The storage panel shows how much space your files use.

Files are stored wherever your workspace is set up to keep them: on the server, or in S3-compatible storage. See [Settings](settings.md#media-uploads).
