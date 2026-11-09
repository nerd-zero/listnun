import {
  beforeEach, describe, expect, it, vi,
} from 'vitest';

// The module reads storage once on import, so each test loads it fresh.
async function fresh() {
  vi.resetModules();
  return (await import('./useNavOpen')).useNavOpen();
}

describe('useNavOpen', () => {
  beforeEach(() => localStorage.clear());

  it('follows the active page until the user toggles a group', async () => {
    const nav = await fresh();
    expect(nav.isOpen('lists', true)).toBe(true);
    expect(nav.isOpen('lists', false)).toBe(false);
  });

  it('keeps a closed group closed after a reload, even on its own page', async () => {
    (await fresh()).setOpen('subscribers', false);
    const nav = await fresh();
    expect(nav.isOpen('subscribers', true)).toBe(false);
  });

  it('ignores a corrupt saved value', async () => {
    localStorage.setItem('ln-nav-open', '[1,2');
    const nav = await fresh();
    expect(nav.isOpen('lists', true)).toBe(true);
  });
});
