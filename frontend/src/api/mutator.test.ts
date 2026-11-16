import {
  describe, it, expect, vi, beforeEach,
} from 'vitest';

// Mock showToast before importing the mutator so the error interceptor
// can call it without needing a real PrimeVue instance.
vi.mock('../toastService', () => ({
  showToast: vi.fn(),
}));

import { showToast } from '../toastService';
import { httpMutator, transformResponse } from './mutator';

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

// ---------------------------------------------------------------------------
// Free-form JSON keeps its keys
// ---------------------------------------------------------------------------

describe('transformResponse — free-form JSON', () => {
  it('keeps subscriber attrib keys in a list response while camelCasing the rest', () => {
    const body = {
      data: {
        results: [{
          id: 1,
          created_at: '2026-01-01',
          attribs: { first_name: 'Alice', 'home-city': 'Harare', nested_obj: { zip_code: '00263' } },
          lists: [{ subscription_status: 'confirmed', meta: { opt_in_ip: '1.2.3.4' } }],
        }],
        total: 1,
        per_page: 20,
      },
    };

    const out = transformResponse(body, '/api/subscribers') as any;
    expect(out.perPage).toBe(20);
    expect(out.results[0].createdAt).toBe('2026-01-01');
    expect(out.results[0].attribs).toEqual({
      first_name: 'Alice', 'home-city': 'Harare', nested_obj: { zip_code: '00263' },
    });
    expect(out.results[0].lists[0].subscriptionStatus).toBe('confirmed');
    expect(out.results[0].lists[0].meta).toEqual({ opt_in_ip: '1.2.3.4' });
  });

  it('keeps attrib keys on a single subscriber', () => {
    const out = transformResponse(
      { data: { last_opened_at: null, attribs: { first_name: 'Alice' } } },
      '/api/subscribers/7',
    );
    expect(out).toEqual({ lastOpenedAt: null, attribs: { first_name: 'Alice' } });
  });

  it('keeps campaign header names and archive meta keys', () => {
    const out = transformResponse({
      data: {
        results: [{
          to_send: 3,
          headers: [{ 'X-Custom_Header': 'v' }],
          archive_meta: { first_name: 'Reader' },
        }],
      },
    }, '/api/campaigns') as any;
    expect(out.results[0].toSend).toBe(3);
    expect(out.results[0].headers).toEqual([{ 'X-Custom_Header': 'v' }]);
    expect(out.results[0].archiveMeta).toEqual({ first_name: 'Reader' });
  });

  it('keeps bounce meta and template sample data keys', () => {
    const bounces = transformResponse(
      { data: { results: [{ subscriber_id: 1, meta: { bounce_type: 'hard' } }] } },
      '/api/bounces',
    ) as any;
    expect(bounces.results[0].subscriberId).toBe(1);
    expect(bounces.results[0].meta).toEqual({ bounce_type: 'hard' });

    const tpls = transformResponse(
      { data: [{ is_default: true, sample_data: { order_id: 9 } }] },
      '/api/templates',
    );
    expect(tpls).toEqual([{ isDefault: true, sampleData: { order_id: 9 } }]);
  });

  it('keeps inbox check params only on the inbox check endpoint', () => {
    const body = { data: [{ params: { list_name: 'x' } }] };
    expect(transformResponse(body, '/api/campaigns/3/inbox-check')).toEqual([{ params: { list_name: 'x' } }]);
    expect(transformResponse(body, '/api/campaigns/3')).toEqual([{ params: { listName: 'x' } }]);
  });

  it('leaves settings and server config keys as sent', () => {
    const settings = { 'maintenance.db': { vacuum: true, vacuum_cron_interval: '0 2 * * *' } };
    expect(transformResponse({ data: settings }, '/api/settings')).toEqual(settings);
    expect(transformResponse({ data: { root_url: 'x' } }, '/api/config')).toEqual({ root_url: 'x' });
    // Other settings sub-resources are still camelCased.
    expect(transformResponse({ data: { per_page: 1 } }, '/api/settings/logs')).toEqual({ perPage: 1 });
  });

  it('applies the skip list through the axios interceptor', async () => {
    const adapter = vi.fn().mockResolvedValue(mockAxiosResponse({
      data: { results: [{ created_at: 'x', attribs: { first_name: 'Alice' } }] },
    }));
    const result = await httpMutator({ url: '/api/subscribers', adapter }) as any;
    expect(result.results[0]).toEqual({ createdAt: 'x', attribs: { first_name: 'Alice' } });
  });
});
