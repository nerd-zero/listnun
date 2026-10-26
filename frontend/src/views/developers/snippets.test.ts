import { describe, expect, it } from 'vitest';
import { SNIPPET_LANGS, snippet } from './snippets';

const input = {
  rootUrl: 'https://acme.listnun.com/',
  keyName: 'checkout-svc',
  templateId: 4,
  templateName: 'Order receipt',
  recipient: 'ana@example.com',
  data: { order_id: '1042', note: "it's here" },
};

describe('snippet', () => {
  it('uses the workspace URL, key, template and data in every language', () => {
    SNIPPET_LANGS.forEach((lang) => {
      const s = snippet(lang, input);
      expect(s).toContain('https://acme.listnun.com/api/tx');
      expect(s).toContain('checkout-svc');
      expect(s).toContain('"template_id": 4');
      expect(s).toContain('LISTNUN_API_KEY');
    });
  });

  it('tells the assistant to keep the key server-side and explains unknown recipients', () => {
    const s = snippet('llm', input);
    expect(s).toContain('server-side');
    expect(s).toContain('"subscriber_mode": "external"');
    expect(s).toContain('order_id, note');
  });

  it('escapes single quotes in the curl body', () => {
    expect(snippet('curl', input)).toContain("it'\\''s here");
  });
});
