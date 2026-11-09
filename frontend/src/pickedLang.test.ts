import { describe, expect, it } from 'vitest';
import { pickedLang } from './pickedLang';

describe('pickedLang', () => {
  it('reads the ln_lang cookie among others', () => {
    expect(pickedLang('a=1; ln_lang=pt-BR; b=2')).toBe('pt-BR');
    expect(pickedLang('ln_lang=de')).toBe('de');
  });

  it('ignores missing, malformed and look-alike cookies', () => {
    expect(pickedLang('')).toBe('');
    expect(pickedLang('xln_lang=de')).toBe('');
    expect(pickedLang('ln_lang=..%2F..%2Fetc')).toBe('');
    expect(pickedLang('ln_lang=%E0%A4%A')).toBe('');
  });
});
