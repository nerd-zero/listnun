import {
  describe, it, expect, vi, beforeEach,
} from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import type { MenuItem } from 'primevue/menuitem';
import { createI18n } from 'vue-i18n';
import { useCampaignActions, type CampaignRef } from './useCampaignActions';

let perms: string[] = [];
const push = vi.fn();
const updateCampaignStatus = vi.fn(() => Promise.resolve({}));

vi.mock('./useGlobal', () => ({
  useGlobal: () => ({
    $can: (...p: string[]) => p.some((x) => perms.includes(x)),
    $utils: { toast: vi.fn(), niceNumber: (n: number) => String(n) },
  }),
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }));
vi.mock('../api/generated/endpoints/campaigns/campaigns', () => ({
  getCampaigns: () => ({
    getCampaign: vi.fn(), createCampaign: vi.fn(), deleteCampaign: vi.fn(), updateCampaignStatus,
  }),
}));

const ALL = ['campaigns:send', 'campaigns:manage', 'campaigns:get_analytics'];

// Keys come back as themselves, except the resume text, whose count the
// test checks.
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: { campaigns: { overview: { resumeText: '{n} still waiting' } } } },
  missing: (_locale, key) => key,
});
const CANCEL = 'globals.buttons.cancel';
const ANALYTICS = 'globals.terms.analytics';
const DELETE = 'globals.buttons.delete';

function setup(onChanged = () => {}) {
  let api!: ReturnType<typeof useCampaignActions>;
  mount(defineComponent({
    setup() {
      api = useCampaignActions(onChanged);
      return () => h('div');
    },
  }), { global: { plugins: [i18n] } });
  return api;
}

const camp = (status: string, extra: Partial<CampaignRef> = {}): CampaignRef => ({
  id: 7, name: 'Launch', status, toSend: 100, sent: 40, ...extra,
});

// The menu's labels, in order, with separators as '|'.
const labels = (items: MenuItem[]) => items.map((i) => (i.separator ? '|' : i.label));

beforeEach(() => {
  perms = [...ALL];
  push.mockClear();
  updateCampaignStatus.mockClear();
});

describe('useCampaignActions', () => {
  it('offers a draft start, or schedule once it has a send date', () => {
    const a = setup();
    expect(a.sendAction(camp('draft'))?.cy).toBe('btn-start');
    expect(a.sendAction(camp('draft', { sendAt: '2026-12-01T09:00:00Z' }))?.cy).toBe('btn-schedule');
    expect(a.sendAction(camp('finished'))).toBeNull();
  });

  it("doesn't offer sending without the send permission", () => {
    perms = ['campaigns:manage', 'campaigns:get_analytics'];
    const a = setup();
    expect(a.sendAction(camp('draft'))).toBeNull();
    expect(a.primary(camp('running'))?.cy).toBe('btn-preview');
    expect(labels(a.itemsFor(camp('running')))).not.toContain(CANCEL);
  });

  it('shows the action that fits each status first', () => {
    const a = setup();
    expect(a.primary(camp('running'))?.cy).toBe('btn-pause');
    expect(a.primary(camp('paused'))?.cy).toBe('btn-resume');
    expect(a.primary(camp('finished'))?.cy).toBe('btn-analytics');
    expect(a.primary(camp('draft'))?.cy).toBe('btn-preview');
  });

  it('lets a sending campaign be cancelled, and a draft not', () => {
    const a = setup();
    const paused = labels(a.itemsFor(camp('paused')));
    expect(paused).toContain(CANCEL);
    expect(paused).toContain(ANALYTICS);
    const draft = labels(a.itemsFor(camp('draft')));
    expect(draft).not.toContain(CANCEL);
    expect(draft).not.toContain(ANALYTICS);
    expect(draft[draft.length - 1]).toBe(DELETE);
  });

  it('says how many are still waiting when resuming, and refreshes after', async () => {
    const onChanged = vi.fn();
    const a = setup(onChanged);
    a.askStatus(camp('paused'), 'running');
    expect(a.confirm.open).toBe(true);
    expect(a.confirm.text).toBe('60 still waiting');
    expect(a.confirm.danger).toBe(false);

    a.confirm.run();
    await vi.waitFor(() => expect(onChanged).toHaveBeenCalled());
    expect(updateCampaignStatus).toHaveBeenCalledWith(7, { status: 'running' });
    expect(a.confirm.open).toBe(false);
  });

  it('confirms cancelling as a destructive action', () => {
    const a = setup();
    a.askStatus(camp('running'), 'cancelled');
    expect(a.confirm.danger).toBe(true);
  });
});
