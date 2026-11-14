import {
  afterEach, beforeEach, describe, expect, it, vi,
} from 'vitest';
import {
  readItem, writeItem, removeItem, readJSON, writeJSON,
} from './safeStorage';

beforeEach(() => localStorage.clear());
afterEach(() => vi.restoreAllMocks());

describe('safeStorage', () => {
  it('round-trips strings and JSON', () => {
    writeItem('a', '1');
    expect(readItem('a')).toBe('1');
    writeJSON('b', { x: [1, 2] });
    expect(readJSON('b')).toEqual({ x: [1, 2] });
    removeItem('a');
    expect(readItem('a')).toBeNull();
  });

  it('returns null for corrupt JSON instead of throwing', () => {
    localStorage.setItem('b', '{not json');
    expect(readJSON('b')).toBeNull();
  });

  it('swallows errors when storage is unavailable', () => {
    const fail = () => { throw new DOMException('denied', 'SecurityError'); };
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(fail);
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(fail);
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(fail);

    expect(readItem('a')).toBeNull();
    expect(readJSON('a')).toBeNull();
    expect(() => writeItem('a', '1')).not.toThrow();
    expect(() => writeJSON('a', { x: 1 })).not.toThrow();
    expect(() => removeItem('a')).not.toThrow();
  });
});
