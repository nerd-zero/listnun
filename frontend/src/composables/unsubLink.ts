// A tag that renders an unsubscribe or preferences link. Mirrors
// models.Campaign.HasUnsubscribeLink, which refuses to start or schedule a
// campaign without one: Gmail and Yahoo require a visible link from bulk
// senders.
const UNSUB_LINK_RE = /\{\{-?\s*(UnsubscribeURL|ManageURL)\b/;

export function hasUnsubLink(src: string | null | undefined): boolean {
  return UNSUB_LINK_RE.test(String(src || ''));
}

// Whether a campaign's email has a link: in its body, or in the template
// around it. A visual campaign renders without its template.
export function campaignHasUnsubLink(contentType: string, body: string, templateBody: string | null | undefined): boolean {
  if (hasUnsubLink(body)) return true;
  return contentType !== 'visual' && hasUnsubLink(templateBody);
}

const htmlFooter = '<p style="font-size: 12px; color: #888;"><a href="{{ UnsubscribeURL }}" style="color: #888;">Unsubscribe</a></p>';

const footers: Record<string, string> = {
  richtext: htmlFooter,
  html: htmlFooter,
  markdown: '[Unsubscribe]({{ UnsubscribeURL }})',
  plain: 'Unsubscribe: {{ UnsubscribeURL }}',
};

// withUnsubFooter appends a small unsubscribe footer in the body's own
// format. A visual body is blocks edited in the builder, so it's returned
// unchanged.
export function withUnsubFooter(contentType: string, body: string): string {
  const footer = footers[contentType];
  if (!footer) return body;
  const trimmed = String(body || '').trimEnd();
  return trimmed ? `${trimmed}\n\n${footer}` : footer;
}
