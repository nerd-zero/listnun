# Tracking opens and clicks

listnun can tell you who opened a campaign and which links they clicked. You set it up per campaign in the **Tracking** card on the **Content** step. Nothing needs adding to the email by hand.

<div class="ui">
<div class="ui-card" style="max-width: 560px; margin: 0 auto;">
<div class="ui-card-head">Tracking</div>
<div class="ui-card-pad ui-col">
<div class="ui-row"><span class="ui-toggle is-on"></span><span><strong>Track link clicks</strong><br><span class="ui-small">Counts who clicks each link, in every editor. No codes to add.</span></span></div>
<div>
<span class="ui-label">Tracking pixels</span>
<div class="ui-col" style="gap: 6px;">
<div class="ui-chip"><span class="ui-mono ui-grow">https://www.facebook.com/tr?id=123&amp;ev=PageView&amp;noscript=1</span><span class="ui-x">✕</span></div>
<div class="ui-row"><span class="ui-input is-area ui-grow ui-mono">Paste a pixel link, or the &lt;img&gt; code from Meta, Google Ads, LinkedIn…</span><span class="ui-btn">Add pixel</span></div>
<span class="ui-small">For ad and analytics platforms that count opens with an image. Many email apps block images until the reader allows them, so these counts run low.</span>
</div>
</div>
</div>
</div>
<p class="ui-caption">The Tracking card on a campaign's Content step, with one pixel added.</p>
</div>

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
