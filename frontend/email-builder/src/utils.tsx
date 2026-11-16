import { renderToStaticMarkup } from '@usewaypoint/email-builder';
import { DEFAULT_LAYOUT_WIDTH } from './documents/blocks/EmailLayout/EmailLayoutPropsSchema';
import { TEditorConfiguration } from './documents/editor/core';
import { postProcessForOutlook } from './outlook';

const VIEWPORT_META = '<meta name="viewport" content="width=device-width, initial-scale=1.0">';
const MSO_DOCUMENT_SETTINGS = '<!--[if mso]><noscript><xml xmlns:o="urn:schemas-microsoft-com:office:office"><o:OfficeDocumentSettings><o:AllowPNG/><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->';
const HTML_ATTRIBUTE_ESCAPES: Record<string, string> = {
  '&': '&amp;',
  '"': '&quot;',
  "'": '&#x27;',
  '<': '&lt;',
  '>': '&gt;',
};

function injectHeadContents(html: string, contents: string) {
  const headMatch = html.match(/<head\b([^>]*)>/i);
  if (headMatch) {
    return html.replace(/<head\b([^>]*)>/i, `<head$1>${contents}`);
  }

  const htmlMatch = html.match(/<html\b([^>]*)>/i);
  if (htmlMatch) {
    return html.replace(/<html\b([^>]*)>/i, `<html$1><head>${contents}</head>`);
  }

  return `<head>${contents}</head>${html}`;
}

function collectImageEmbedURLs(document: TEditorConfiguration): string[] {
  // The upstream renderer strips the custom `embed` prop before rendering, so
  // collect URLs from blocks marked for embedding and re-tag the matching <img>
  // with a `data-embed` flag after rendering. The backend resolves the src
  // filename to a media item at compile time.
  const embedURLs: string[] = [];

  for (const block of Object.values(document)) {
    if (!block || (block as { type?: string }).type !== 'Image') {
      continue;
    }

    const props = ((block as { data?: { props?: { url?: string; embed?: boolean } } }).data || {}).props || {};
    if (props.embed && props.url) {
      embedURLs.push(props.url);
    }
  }

  return embedURLs;
}

function applyImageEmbeds(html: string, embedURLs: string[]): string {
  let output = html;

  for (const url of embedURLs) {
    const re = new RegExp(`<img\\b[^>]*?\\ssrc="${escapeRegExp(escapeHtmlAttribute(url))}"[^>]*>`, 'g');
    output = output.replace(re, (tag) => (
      /\bdata-embed\b/.test(tag) ? tag : tag.replace(/(\ssrc="[^"]*")/, '$1 data-embed="true"')
    ));
  }

  return output;
}

// The upstream renderer always gives the email's outer table this width.
const RENDERER_LAYOUT_WIDTH = 600;

function layoutWidth(document: TEditorConfiguration, rootBlockId: string): number {
  const root = document[rootBlockId] as { type?: string; data?: { width?: number | null } } | undefined;
  return (root?.type === 'EmailLayout' && root.data?.width) || DEFAULT_LAYOUT_WIDTH;
}

// applyLayoutWidth sets the template's content width on the email's outer
// table, the first one in the output, which the upstream renderer fixes
// at 600px.
export function applyLayoutWidth(html: string, width: number): string {
  if (width === RENDERER_LAYOUT_WIDTH) {
    return html;
  }
  const outer = new RegExp(`(<table\\b[^>]*?style="[^"]*?)max-width:${RENDERER_LAYOUT_WIDTH}px`);
  return html.replace(outer, `$1max-width:${width}px`);
}

export function renderHtmlWithMeta(
  document: TEditorConfiguration,
  options: { rootBlockId: string; outlook?: boolean }
): string {
  const embedURLs = collectImageEmbedURLs(document);
  const width = layoutWidth(document, options.rootBlockId);
  const html = applyLayoutWidth(renderToStaticMarkup(document, options), width);
  const rendered = options.outlook ? postProcessForOutlook(html, width) : html;
  const output = applyImageEmbeds(rendered, embedURLs);
  const head = options.outlook ? `${VIEWPORT_META}${MSO_DOCUMENT_SETTINGS}` : VIEWPORT_META;

  return injectHeadContents(output, head);
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function escapeHtmlAttribute(s: string): string {
  return s.replace(/[&"'<>]/g, (ch) => HTML_ATTRIBUTE_ESCAPES[ch]);
}
