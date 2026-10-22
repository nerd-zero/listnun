import { describe, expect, it } from 'vitest';
import { parsePixelInput } from './trackingPixel';

describe('parsePixelInput', () => {
  it('takes a bare URL', () => {
    expect(parsePixelInput('  https://px.example.com/a.gif?id=1 ')).toEqual({ url: 'https://px.example.com/a.gif?id=1' });
  });

  it('pulls the src out of an <img> snippet, including inside <noscript>', () => {
    const meta = `<script>!function(f){/* fbq */}();</script>
<noscript><img height="1" width="1" style="display:none"
src="https://www.facebook.com/tr?id=123&amp;ev=PageView&amp;noscript=1"/></noscript>`;
    expect(parsePixelInput(meta)).toEqual({ url: 'https://www.facebook.com/tr?id=123&ev=PageView&noscript=1' });
    expect(parsePixelInput("<img src='https://px.example.com/b' />")).toEqual({ url: 'https://px.example.com/b' });
  });

  it('explains when only a script was pasted', () => {
    expect(parsePixelInput('<script src="https://example.com/tag.js"></script>')).toEqual({ error: 'script' });
  });

  it('rejects anything that is not a plain web address', () => {
    ['example.com/a.gif', `${'javascript'}:alert(1)`, 'https://x.com/a b', 'https://x.com/"onerror', ''].forEach((s) => {
      expect(parsePixelInput(s)).toEqual({ error: 'invalid' });
    });
  });
});
