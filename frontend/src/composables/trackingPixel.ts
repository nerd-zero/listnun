// Turns what someone pastes into a campaign's "Tracking pixels" field into
// a pixel URL. Ad and analytics platforms hand out either a bare URL or an
// HTML snippet; in email only the <img> part works (scripts never run), so
// that's what this pulls out. The server applies the same URL rule
// (models.CampaignTracking.Validate).

export type PixelResult =
  | { url: string }
  | { error: 'script' | 'invalid' };

// A plain http(s) URL with nothing that could break out of an attribute.
const PIXEL_URL = /^https?:\/\/[^\s"'<>]+$/;
const IMG_SRC = /<img\b[^>]*?\bsrc\s*=\s*(["'])(.*?)\1/i;

function decodeEntities(s: string): string {
  return s.replace(/&amp;/g, '&').replace(/&#38;/g, '&');
}

export function parsePixelInput(raw: string): PixelResult {
  const input = raw.trim();
  const img = IMG_SRC.exec(input);
  let url = input;
  if (img) {
    url = decodeEntities(img[2].trim());
  } else if (/<script\b/i.test(input)) {
    return { error: 'script' };
  }
  return PIXEL_URL.test(url) ? { url } : { error: 'invalid' };
}

export default parsePixelInput;
