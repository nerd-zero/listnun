# Tracking opens and clicks

listnun can tell you who opened a campaign and which links they clicked. You set it up per campaign in the **Tracking** card on the **Content** step. Nothing needs adding to the email by hand.

## Track link clicks

With **Track link clicks** on, every web link in the email is tracked: when someone clicks, listnun counts it and sends them on to the real address. It works the same in every format, including Visual, Rich text, Raw HTML and Markdown.

It's on by default for new campaigns. Campaigns created before this switch existed start with it off, so they keep sending exactly as before; turn it on if you want clicks counted.

Some links are never rewritten:

- unsubscribe, preferences and view-in-browser links (listnun handles these itself),
- `mailto:` and other non-web links,
- links that are already tracked,
- links containing a [template expression](../templating.md), such as `https://example.com/{{ .Subscriber.UUID }}`.

Tracking applies to HTML emails. Plain-text campaigns, and the plain-text version of an HTML campaign, aren't tracked.

!!! tip "Tracking a single link by hand"
    In Raw HTML or Markdown you can still track one link without the switch by adding `@TrackLink` to its address: `https://example.com@TrackLink`. In Rich text, tick **Track link** in the link dialog. See the [Templating reference](../templating.md).

## Tracking pixels

Ad and analytics platforms, such as Meta, Google Ads and LinkedIn, can count email opens with a tiny invisible image called a pixel. To add one:

1. In the platform, find the pixel's image address, or the `<img …>` code it gives you.
2. Paste either into **Tracking pixels** and click **Add pixel**.

If you paste a whole snippet, listnun pulls out the image address, including from inside `<noscript>`. If the snippet is only a `<script>`, listnun explains that scripts don't run in email. Look for the image (noscript) version instead.

Pixel addresses must start with `https://` (or `http://`). You can add up to 10 per campaign, and remove one with its **×**. Pixels go at the end of the email and are added to every subscriber's copy.

## Open tracking

listnun counts opens with its own tracking image, which the default templates include as `{{ TrackView }}`. If you write your own template, include it to count opens. See [Templates](templates-and-media.md#templates).

## Why the numbers aren't exact

- Many email apps block images until the reader allows them, and some load every image automatically to protect privacy. Open counts (yours and any pixel's) are therefore approximate. Clicks are reliable.
- If **Settings › Privacy › Individual subscriber tracking** is off, listnun counts opens and clicks in total but not per subscriber.
- If **Disable tracking** is on in **Settings › Privacy**, nothing is tracked at all, whatever the campaign says.
