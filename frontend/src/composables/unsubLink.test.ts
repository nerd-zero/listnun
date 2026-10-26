import { describe, expect, it } from 'vitest';
import { campaignHasUnsubLink, hasUnsubLink, withUnsubFooter } from './unsubLink';

describe('hasUnsubLink', () => {
  it('finds the unsubscribe and preferences tags', () => {
    expect(hasUnsubLink('<a href="{{ UnsubscribeURL }}">x</a>')).toBe(true);
    expect(hasUnsubLink('[Manage]({{ManageURL}})')).toBe(true);
    expect(hasUnsubLink('Leave: {{- UnsubscribeURL -}}')).toBe(true);
  });

  it("doesn't count the word alone", () => {
    expect(hasUnsubLink('<p>Reply STOP to unsubscribe</p>')).toBe(false);
    expect(hasUnsubLink(null)).toBe(false);
  });
});

describe('campaignHasUnsubLink', () => {
  const tpl = '{{ template "content" . }}<a href="{{ UnsubscribeURL }}">Unsubscribe</a>';

  it("counts the template's link", () => {
    expect(campaignHasUnsubLink('richtext', '<p>Hi</p>', tpl)).toBe(true);
  });

  it('ignores the template for a visual campaign, which renders without it', () => {
    expect(campaignHasUnsubLink('visual', '<table>Hi</table>', tpl)).toBe(false);
  });
});

describe('withUnsubFooter', () => {
  it('adds a footer the check then finds, in each format', () => {
    for (const type of ['richtext', 'html', 'markdown', 'plain']) {
      const out = withUnsubFooter(type, 'Hello');
      expect(out.startsWith('Hello\n\n')).toBe(true);
      expect(hasUnsubLink(out)).toBe(true);
    }
  });

  it('leaves a visual body to the builder', () => {
    expect(withUnsubFooter('visual', '<table/>')).toBe('<table/>');
  });
});
