<template>
  <div class="ln-page dash">
    <page-header
      :title="$t('menu.dashboard')"
      :subtitle="orgName ? $t('dashboard.subtitleOrg', { name: orgName }) : $t('dashboard.subtitle')"
    >
      <PvSelectButton
        v-model="period"
        :options="periodOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        :aria-label="$t('dashboard.period')"
      />
      <PvButton
        v-if="$can('campaigns:manage')"
        icon="pi pi-plus"
        :label="$t('menu.newCampaign')"
        @click="$router.push({ name: 'campaign', params: { id: 'new' } })"
      />
    </page-header>

    <!-- Key numbers -->
    <section class="ln-tiles" :aria-label="$t('dashboard.keyNumbers')">
      <div class="ln-tile" data-cy="subscribers">
        <span class="ln-tile-label">{{ $t('globals.terms.subscribers') }}</span>
        <span class="ln-tile-value">{{ ov ? ov.subscribers.toLocaleString() : '—' }}</span>
        <span class="ln-tile-sub" :class="ov && ov.new ? 'is-success' : ''">
          {{ ov ? $t('dashboard.newInPeriod', { n: ov.new.toLocaleString(), days: period }) : '' }}
        </span>
      </div>
      <div class="ln-tile" data-cy="messages">
        <span class="ln-tile-label">{{ $t('dashboard.emailsSent') }}</span>
        <span class="ln-tile-value">{{ ov ? ov.sent.toLocaleString() : '—' }}</span>
        <span class="ln-tile-sub">{{ ov ? $t('dashboard.campaignsStarted', { n: ov.campaignsStarted }) : '' }}</span>
      </div>
      <div class="ln-tile" data-cy="bounce-rate">
        <span class="ln-tile-label">{{ $t('dashboard.bounceRate') }}</span>
        <span class="ln-tile-value">{{ ov ? `${(ov.bounceRate * 100).toFixed(1)}%` : '—' }}</span>
        <span v-if="ov" class="ln-tile-sub" :class="bounceTone">{{ bounceText }}</span>
      </div>
      <div class="ln-tile" data-cy="queue">
        <span class="ln-tile-label">{{ $t('dashboard.queue') }}</span>
        <span class="ln-tile-value">{{ ov ? ov.queued.toLocaleString() : '—' }}</span>
        <span v-if="ov" class="ln-tile-sub" :class="ov.running ? 'is-success' : ''">
          {{ ov.running ? $t('dashboard.running', { n: ov.running }) : $t('dashboard.idle') }}
        </span>
      </div>
    </section>

    <!-- Mail validation (Scrub) -->
    <section v-if="scrubEnabled" class="ln-tiles" :aria-label="$t('dashboard.scrubTitle')">
      <div class="ln-tile" data-cy="scrub-risky">
        <span class="ln-tile-label">{{ $t('dashboard.riskySubscribers') }}</span>
        <span class="ln-tile-value">{{ risky.toLocaleString() }}</span>
        <span class="ln-tile-sub" :class="risky ? 'is-warn' : ''">{{ $t('dashboard.scrubTitle') }}</span>
      </div>
      <div class="ln-tile" data-cy="scrub-jobs">
        <span class="ln-tile-label">{{ $t('dashboard.scrubEmailsValidated') }}</span>
        <span class="ln-tile-value">{{ (scrubValidated + scrubInvalid).toLocaleString() }}</span>
        <span class="ln-tile-sub">
          {{ scrubValidated.toLocaleString() }} {{ $t('settings.scrub.historyStatus.valid') }}&emsp;
          {{ scrubInvalid.toLocaleString() }} {{ $t('settings.scrub.historyStatus.invalid') }}
          <template v-if="scrubActiveJobs">&emsp;{{ $t('dashboard.scrubJobsRunning', { count: scrubActiveJobs }) }}</template>
        </span>
      </div>
      <div class="ln-tile" data-cy="auto-paused">
        <span class="ln-tile-label">{{ $t('dashboard.autoPausedCampaigns') }}</span>
        <span class="ln-tile-value">{{ autoPaused.length }}</span>
        <span class="ln-tile-sub" :class="autoPaused.length ? 'is-danger' : 'is-success'">
          {{ autoPaused.length ? $t('dashboard.seeNeedsAttention') : $t('dashboard.noAutoPausedCampaigns') }}
        </span>
      </div>
    </section>

    <!-- Charts -->
    <div class="dash-charts">
      <section class="ln-card dash-chart">
        <div class="dash-chart-head">
          <h2 class="ln-card-title">{{ $t('dashboard.activity') }}</h2>
          <ul class="dash-legend">
            <li><span class="is-success" />{{ $t('dashboard.delivered') }}</li>
            <li><span class="is-warn" />{{ $t('dashboard.bounced') }}</li>
            <li><span class="is-danger" />{{ $t('dashboard.failed') }}</li>
          </ul>
        </div>
        <ln-chart
          v-if="ov && ov.sent + ov.bounced > 0"
          type="bar"
          stacked
          :labels="dayLabels"
          :series="activitySeries"
          height="200px"
          :label="$t('dashboard.activity')"
        />
        <div v-else-if="ov" class="dash-chart-empty">
          <span class="ln-small">{{ $t('dashboard.noActivity', { days: period }) }}</span>
        </div>
        <PvSkeleton v-else height="200px" />
      </section>

      <section class="ln-card dash-chart">
        <div class="dash-chart-head">
          <h2 class="ln-card-title">{{ $t('dashboard.growth') }}</h2>
          <strong v-if="ov" :class="growth >= 0 ? 'dash-up' : 'dash-down'">{{ growth >= 0 ? '+' : '' }}{{ growth.toLocaleString() }}</strong>
        </div>
        <template v-if="ov">
          <ln-chart type="line" area :labels="dayLabels" :series="growthSeries" height="150px" :label="$t('dashboard.growth')" />
          <div class="dash-growth-ends ln-small">
            <span>{{ growthStart.toLocaleString() }}</span>
            <span>{{ ov.subscribers.toLocaleString() }}</span>
          </div>
          <div class="dash-growth-foot">
            <span>{{ $t('dashboard.new') }} <strong>{{ ov.new.toLocaleString() }}</strong></span>
            <span>{{ $t('dashboard.unsubscribed') }} <strong>{{ ov.unsubscribed.toLocaleString() }}</strong></span>
          </div>
        </template>
        <PvSkeleton v-else height="190px" />
      </section>
    </div>

    <div class="dash-lower">
      <!-- Recent campaigns -->
      <section v-if="$can('campaigns:get_all', 'campaigns:get')" class="ln-card dash-recent" data-cy="recent-campaigns">
        <div class="ln-card-head">
          <h2 class="ln-card-title">{{ $t('dashboard.recentCampaigns') }}</h2>
          <router-link :to="{ name: 'campaigns' }" class="dash-link">{{ $t('menu.allCampaigns') }}</router-link>
        </div>
        <table-skeleton v-if="!campaigns" :rows="5" :cols="3" />
        <empty-state
          v-else-if="!campaigns.length"
          bare
          icon="pi-megaphone"
          :title="$t('dashboard.noCampaigns')"
          :text="$t('dashboard.noCampaignsText')"
          :action-label="$can('campaigns:manage') ? $t('menu.newCampaign') : ''"
          @action="$router.push({ name: 'campaign', params: { id: 'new' } })"
        />
        <ul v-else class="dash-rows">
          <li v-for="c in campaigns" :key="c.id">
            <router-link :to="{ name: 'campaign', params: { id: c.id } }" class="dash-row">
              <span class="dash-row-name">
                <span class="ln-row-title">{{ c.name }}</span>
                <span class="ln-row-sub">{{ campaignLine(c) }}</span>
              </span>
              <status-tag :status="c.status" :label="$t(`campaigns.status.${c.status}`)" />
              <span class="dash-rate">{{ $t('dashboard.views') }} <strong>{{ rate(c.views, c.sent) }}</strong></span>
              <span class="dash-rate">{{ $t('dashboard.clicks') }} <strong>{{ rate(c.clicks, c.sent) }}</strong></span>
            </router-link>
          </li>
        </ul>
      </section>

      <!-- Needs attention -->
      <section class="ln-card dash-attention">
        <div class="ln-card-head">
          <h2 class="ln-card-title">{{ $t('dashboard.attention') }}</h2>
        </div>
        <ul v-if="attention.length" class="dash-notes">
          <li v-for="a in attention" :key="a.key" :class="`is-${a.tone}`">
            <div>
              <component :is="a.to ? 'router-link' : 'strong'" :to="a.to" class="dash-note-title">{{ a.title }}</component>
              <p class="ln-small">{{ a.text }}</p>
            </div>
          </li>
        </ul>
        <p v-else class="dash-clear ln-small">{{ $t('dashboard.allClear') }}</p>
      </section>
    </div>

    <p v-if="settings['app.cache_slow_queries']" class="dash-cache-note ln-small">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferrer">{{ $t('globals.buttons.learnMore') }}</a>
    </p>
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, ref, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { storeToRefs } from 'pinia';
import type { RouteLocationRaw } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import { getDashboardOverview, type DashboardOverview, type DashboardPeriod } from '../api/dashboard';
import { getDashboard } from '../api/generated/endpoints/dashboard/dashboard';
import { getCampaigns } from '../api/generated/endpoints/campaigns/campaigns';
import { getSettings } from '../api/generated/endpoints/settings/settings';
import PageHeader from '../components/ui/PageHeader.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import LnChart, { type ChartSeries } from '../components/ui/LnChart.vue';

interface CampaignRow {
  id: number;
  name: string;
  status: string;
  toSend: number;
  sent: number;
  views: number;
  clicks: number;
  sendAt: string | null;
  startedAt: string | null;
  updatedAt: string;
}

interface AutoPaused { id: number; name: string; pauseReason: string }

interface Attention {
  key: string;
  tone: 'warn' | 'danger' | 'neutral';
  title: string;
  text: string;
  to?: RouteLocationRaw;
}

// Bounce rates above this are flagged; mailbox providers start filtering
// around 2%.
const BOUNCE_WARN = 0.02;
const PERIOD_KEY = 'ln-dash-period';

const { t } = useI18n();
const { $can } = useGlobal();
const { ago, shortDate } = useTime();
const {
  refreshTick, settings, serverConfig,
} = storeToRefs(useMainStore());
const { getDashboardCounts } = getDashboard();
const { listCampaigns, getAutoPausedCampaigns } = getCampaigns();
const { getScrubListStatus } = getSettings();

const savedPeriod = (() => {
  try { return Number(localStorage.getItem(PERIOD_KEY)); } catch { return 0; }
})();
const period = ref<DashboardPeriod>(([7, 30, 90].includes(savedPeriod) ? savedPeriod : 30) as DashboardPeriod);
const ov = ref<DashboardOverview | null>(null);
const campaigns = ref<CampaignRow[] | null>(null);
const autoPaused = ref<AutoPaused[]>([]);
const risky = ref(0);
const scrubValidated = ref(0);
const scrubInvalid = ref(0);
const scrubActiveJobs = ref(0);

const cfg = computed(() => serverConfig.value as { organization_name?: string; scrub_enabled?: boolean; needs_restart?: boolean });
const orgName = computed(() => cfg.value.organization_name || '');
const scrubEnabled = computed(() => !!cfg.value.scrub_enabled);

const periodOptions = [7, 30, 90].map((d) => ({ value: d, label: `${d} d` }));

const dayLabels = computed(() => (ov.value?.days || []).map((d) => shortDate(d.date)));

const activitySeries = computed<ChartSeries[]>(() => {
  const days = ov.value?.days || [];
  return [
    { label: t('dashboard.delivered'), tone: 'success', data: days.map((d) => d.delivered) },
    { label: t('dashboard.bounced'), tone: 'warn', data: days.map((d) => d.bounced) },
    { label: t('dashboard.failed'), tone: 'danger', data: days.map((d) => d.failed) },
  ];
});

// Growth is the running total, working back from today's count.
const growthStart = computed(() => (ov.value ? ov.value.subscribers - ov.value.new : 0));
const growth = computed(() => (ov.value ? ov.value.new - ov.value.unsubscribed : 0));
const growthSeries = computed<ChartSeries[]>(() => {
  let total = growthStart.value;
  const data = (ov.value?.days || []).map((d) => {
    total += d.newSubscribers;
    return total;
  });
  return [{ label: t('globals.terms.subscribers'), tone: 'ink', data }];
});

const bounceTone = computed(() => {
  if (!ov.value) return '';
  if (!ov.value.sent) return '';
  return ov.value.bounceRate > BOUNCE_WARN ? 'is-warn' : 'is-success';
});
const bounceText = computed(() => {
  if (!ov.value?.sent) return t('dashboard.nothingSent');
  return ov.value.bounceRate > BOUNCE_WARN ? t('dashboard.bounceHigh') : t('dashboard.bounceHealthy');
});

function rate(n: number, of: number): string {
  if (!of || !n) return '—';
  return `${((n / of) * 100).toFixed(1)}%`;
}

function campaignLine(c: CampaignRow): string {
  const pct = c.toSend ? Math.round((c.sent / c.toSend) * 100) : 0;
  switch (c.status) {
    case 'running': return t('dashboard.lineRunning', { pct, n: c.toSend.toLocaleString() });
    case 'paused': return t('dashboard.linePaused', { pct });
    case 'scheduled': return c.sendAt ? t('dashboard.lineScheduled', { date: shortDate(c.sendAt, true) }) : t('campaigns.status.scheduled');
    case 'finished': return t('dashboard.lineFinished', { date: shortDate(c.startedAt || c.updatedAt), n: c.sent.toLocaleString() });
    case 'cancelled': return t('dashboard.lineCancelled', { n: c.sent.toLocaleString() });
    default: return t('dashboard.lineDraft', { when: ago(c.updatedAt) });
  }
}

const attention = computed<Attention[]>(() => {
  const out: Attention[] = [];
  autoPaused.value.forEach((c) => out.push({
    key: `paused-${c.id}`,
    tone: 'danger',
    title: t('dashboard.attnPaused', { name: c.name }),
    text: t(`campaigns.pauseReason.${c.pauseReason}`),
    to: { name: 'campaign', params: { id: c.id } },
  }));
  const hot = ov.value?.bounceHotspot;
  if (hot && hot.hard > 0) {
    out.push({
      key: 'bounces',
      tone: 'warn',
      title: t('dashboard.attnBounces', { name: hot.listName }),
      text: t('dashboard.attnBouncesText', { n: hot.hard }),
      to: { name: 'bounces' },
    });
  }
  if (cfg.value.needs_restart) {
    out.push({
      key: 'restart', tone: 'warn', title: t('dashboard.attnRestart'), text: t('dashboard.attnRestartText'),
    });
  }
  if (risky.value > 0) {
    out.push({
      key: 'risky',
      tone: 'warn',
      title: t('dashboard.attnRisky', { n: risky.value.toLocaleString() }),
      text: t('dashboard.attnRiskyText'),
      to: { name: 'subscribers' },
    });
  }
  if (ov.value && ov.value.staleDrafts > 0) {
    out.push({
      key: 'drafts',
      tone: 'neutral',
      title: t('dashboard.attnDrafts', { n: ov.value.staleDrafts }),
      text: t('dashboard.attnDraftsText'),
      to: { name: 'campaigns', query: { status: 'draft' } },
    });
  }
  return out;
});

function fetchOverview() {
  getDashboardOverview(period.value).then((o) => { ov.value = o; });
}

function fetchData() {
  fetchOverview();

  if ($can('campaigns:get_all', 'campaigns:get')) {
    listCampaigns({
      per_page: 5, order_by: 'updated_at', order: 'desc', no_body: true,
    } as never).then((p) => {
      campaigns.value = ((p as unknown as { results: CampaignRow[] }).results) || [];
    });
  }

  if (scrubEnabled.value) {
    getDashboardCounts().then((data) => {
      risky.value = (data as unknown as { scrub?: { riskySubscribers?: number } }).scrub?.riskySubscribers || 0;
    });
    getAutoPausedCampaigns().then((data) => {
      autoPaused.value = (data as unknown as AutoPaused[]) || [];
    }).catch(() => { autoPaused.value = []; });

    // Aggregates GetScrubListStatus's per-list last_result across every
    // list in this tenant: there's no tenant-wide Scrub stats endpoint,
    // and Scrub's account-scoped job endpoints would leak other tenants'
    // jobs since one Scrub account is shared across tenants.
    getScrubListStatus().then((data) => {
      const lists = (Array.isArray(data) ? data : []) as { lastResult?: { validCount?: number; invalidCount?: number }; activeJobRequestId?: string }[];
      scrubValidated.value = lists.reduce((n, l) => n + (l.lastResult?.validCount || 0), 0);
      scrubInvalid.value = lists.reduce((n, l) => n + (l.lastResult?.invalidCount || 0), 0);
      scrubActiveJobs.value = lists.filter((l) => l.activeJobRequestId).length;
    }).catch(() => {});
  }
}

watch(period, (p) => {
  try { localStorage.setItem(PERIOD_KEY, String(p)); } catch { /* storage unavailable */ }
  ov.value = null;
  fetchOverview();
});
watch(refreshTick, fetchData);
onMounted(fetchData);
</script>

<style scoped lang="scss">
.dash-charts {
  display: grid;
  grid-template-columns: minmax(0, 1.75fr) minmax(0, 1fr);
  gap: 20px;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dash-chart { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.dash-chart-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.dash-legend {
  display: flex;
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 12px;
  color: var(--ln-text-2);

  li { display: inline-flex; align-items: center; gap: 6px; }
  span { width: 10px; height: 10px; border-radius: 2px; }
  .is-success { background: var(--ln-success); }
  .is-warn { background: var(--ln-warn); }
  .is-danger { background: var(--ln-danger); }
}
.dash-chart-empty { height: 200px; display: flex; align-items: center; justify-content: center; }
.dash-up { color: var(--ln-success-fg); font-size: 13px; }
.dash-down { color: var(--ln-danger-fg); font-size: 13px; }
.dash-growth-ends { display: flex; justify-content: space-between; color: var(--ln-text-3); }
.dash-growth-foot {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--ln-border);
  font-size: 13px;

  strong { font-weight: 600; }
}

.dash-lower {
  display: grid;
  grid-template-columns: minmax(0, 1.75fr) minmax(0, 1fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dash-recent, .dash-attention { overflow: hidden; }
.dash-link { font-size: 13px; color: var(--ln-text); text-decoration: underline; }
.dash-rows { list-style: none; margin: 0; padding: 0; }
.dash-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto 96px 96px;
  gap: 16px;
  align-items: center;
  padding: 12px 18px;
  border-top: 1px solid var(--ln-border);
  color: var(--ln-text);
  text-decoration: none;

  li:first-child & { border-top: 0; }
  &:hover { background: var(--ln-hover); text-decoration: none; .ln-row-title { text-decoration: underline; } }
  @media (max-width: 600px) {
    grid-template-columns: minmax(0, 1fr) auto;
    .dash-rate { display: none; }
  }
}
.dash-row-name { min-width: 0; }
.dash-rate { font-size: 13px; color: var(--ln-text-3); text-align: right; white-space: nowrap; strong { color: var(--ln-text); font-weight: 500; } }

.dash-notes { list-style: none; margin: 0; padding: 0; }
.dash-notes li {
  display: flex;
  gap: 12px;
  padding: 14px 18px;
  border-top: 1px solid var(--ln-border);

  &:first-child { border-top: 0; }
  .ln-small { margin: 4px 0 0; }
}
.dash-note-title { font-weight: 500; color: var(--ln-text); text-decoration: none; }
a.dash-note-title:hover { text-decoration: underline; }
.dash-clear { padding: 16px 18px; margin: 0; }
.dash-cache-note { margin: 0; }
</style>
