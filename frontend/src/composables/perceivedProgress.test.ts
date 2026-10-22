import { describe, expect, it } from 'vitest';
import { perceivedProgress } from './perceivedProgress';

describe('perceivedProgress', () => {
  it('starts empty and ends full', () => {
    expect(perceivedProgress(0)).toBe(0);
    expect(perceivedProgress(100)).toBe(100);
  });

  it('draws behind the real value in between, so the end speeds up', () => {
    [10, 50, 90].forEach((p) => expect(perceivedProgress(p)).toBeLessThan(p));
    const early = perceivedProgress(20) - perceivedProgress(10);
    const late = perceivedProgress(100) - perceivedProgress(90);
    expect(late).toBeGreaterThan(early);
  });

  it('only moves forward as the real value does', () => {
    let last = -1;
    for (let p = 0; p <= 100; p += 1) {
      const v = perceivedProgress(p);
      expect(v).toBeGreaterThan(last);
      last = v;
    }
  });

  it('clamps values outside 0-100', () => {
    expect(perceivedProgress(-5)).toBe(0);
    expect(perceivedProgress(140)).toBe(100);
  });
});
