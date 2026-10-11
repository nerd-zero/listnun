import {
  describe, it, expect, vi, beforeEach,
} from 'vitest';

// Mock showToast before importing the mutator so the error interceptor
// can call it without needing a real PrimeVue instance.
vi.mock('../toastService', () => ({
  showToast: vi.fn(),
}));

import { showToast } from '../toastService';
import { httpMutator } from './mutator';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockAxiosResponse(data: unknown, status = 200) {
  return {
    data,
    status,
    statusText: 'OK',
    headers: {},
    config: { headers: {} } as any,
  };
}

// ---------------------------------------------------------------------------
// Envelope unwrapping
// ---------------------------------------------------------------------------

describe('httpMutator — response envelope unwrapping', () => {
  it('unwraps the { data: T } envelope the listnun API uses', async () => {
    const inner = [{ id: 1, name: 'Test' }];
    const adapter = vi.fn().mockResolvedValue(mockAxiosResponse({ data: inner }));

    const result = await httpMutator({ url: '/api/lists', adapter });
    expect(result).toEqual([{ id: 1, name: 'Test' }]);
  });

  it('returns the raw payload when there is no data envelope', async () => {
    const payload = { version: '2.0' };
    const adapter = vi.fn().mockResolvedValue(mockAxiosResponse(payload));

    const result = await httpMutator({ url: '/api/config', adapter });
    expect(result).toEqual({ version: '2.0' });
  });
});

// ---------------------------------------------------------------------------
// camelCase key conversion
// ---------------------------------------------------------------------------

describe('httpMutator — snake_case → camelCase conversion', () => {
  it('converts top-level snake_case keys in a response object', async () => {
    const adapter = vi.fn().mockResolvedValue(
      mockAxiosResponse({ data: { first_name: 'Alice', last_name: 'Smith' } }),
    );

    const result = await httpMutator({ url: '/api/profile', adapter }) as any;
    expect(result.firstName).toBe('Alice');
    expect(result.lastName).toBe('Smith');
  });

  it('converts keys inside arrays', async () => {
    const adapter = vi.fn().mockResolvedValue(
      mockAxiosResponse({ data: [{ subscriber_count: 5 }, { subscriber_count: 10 }] }),
    );

    const result = await httpMutator({ url: '/api/lists', adapter }) as any[];
    expect(result[0].subscriberCount).toBe(5);
    expect(result[1].subscriberCount).toBe(10);
  });

  it('recursively converts nested keys', async () => {
    const adapter = vi.fn().mockResolvedValue(
      mockAxiosResponse({ data: { email_config: { smtp_host: 'mail.example.com' } } }),
    );

    const result = await httpMutator({ url: '/api/settings', adapter }) as any;
    expect(result.emailConfig.smtpHost).toBe('mail.example.com');
  });

  it('leaves non-object payloads (strings, numbers) unchanged', async () => {
    const adapter = vi.fn().mockResolvedValue(mockAxiosResponse({ data: 'ok' }));
    const result = await httpMutator({ url: '/api/health', adapter });
    expect(result).toBe('ok');
  });
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

describe('httpMutator — error handling', () => {
  beforeEach(() => {
    vi.mocked(showToast).mockClear();
  });

  it('shows a toast with the API error message and rejects', async () => {
    const apiError = {
      response: { data: { message: 'Record not found' }, status: 404 },
      toString: () => 'AxiosError',
    };
    const adapter = vi.fn().mockRejectedValue(apiError);

    await expect(httpMutator({ url: '/api/lists/999', adapter })).rejects.toBe(apiError);
    expect(showToast).toHaveBeenCalledWith('Record not found', 'is-danger', 4000);
  });

  it('falls back to err.toString() when response has no message', async () => {
    const apiError = {
      response: { data: {}, status: 500 },
      toString: () => 'Network Error',
    };
    const adapter = vi.fn().mockRejectedValue(apiError);

    await expect(httpMutator({ url: '/api/lists', adapter })).rejects.toBe(apiError);
    expect(showToast).toHaveBeenCalledWith('Network Error', 'is-danger', 4000);
  });
});
