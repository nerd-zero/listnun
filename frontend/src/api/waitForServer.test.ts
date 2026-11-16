import {
  describe, it, expect, vi, beforeEach, afterEach,
} from 'vitest';

vi.mock('./index', () => ({ getHealth: vi.fn() }));

import { waitForServer } from './waitForServer';

beforeEach(() => { vi.useFakeTimers(); });
afterEach(() => { vi.useRealTimers(); });

describe('waitForServer', () => {
  it('resolves once the health check answers', async () => {
    const health = vi.fn()
      .mockRejectedValueOnce(new Error('down'))
      .mockResolvedValue({});
    const done = waitForServer(10000, health);

    await vi.advanceTimersByTimeAsync(3000);
    await expect(done).resolves.toBeUndefined();
    expect(health).toHaveBeenCalledTimes(2);
  });

  it('rejects and stops polling when the server never comes back', async () => {
    const health = vi.fn().mockRejectedValue(new Error('down'));
    const done = waitForServer(5000, health);
    const settled = expect(done).rejects.toThrow('timed out');

    await vi.advanceTimersByTimeAsync(7000);
    await settled;

    const calls = health.mock.calls.length;
    await vi.advanceTimersByTimeAsync(5000);
    expect(health).toHaveBeenCalledTimes(calls);
  });
});
