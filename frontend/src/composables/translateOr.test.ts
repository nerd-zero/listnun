import { describe, expect, it } from 'vitest';
import { translateOr } from './translateOr';

const msgs: Record<string, string> = { 'globals.terms.campaign': 'Campaign', 'globals.terms.lists': 'Lists' };
const t = (k: string) => msgs[k] ?? k;

describe('translateOr', () => {
  it('translates flat dotted keys', () => {
    expect(translateOr(t, 'globals.terms.campaign', 'x')).toBe('Campaign');
  });

  it('falls back when the message is missing or the key is empty', () => {
    expect(translateOr(t, 'globals.terms.nope', 'nope')).toBe('nope');
    expect(translateOr(t, '', 'fallback')).toBe('fallback');
  });

  it('passes the plural count through', () => {
    const tp = (k: string, n?: number) => (k === 'globals.terms.lists' && n === 2 ? 'Lists' : k);
    expect(translateOr(tp, 'globals.terms.lists', 'lists', 2)).toBe('Lists');
  });
});
