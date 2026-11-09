<template>
  <div class="ln-page ana">
    <page-header :title="$t('analytics.title')" :subtitle="$t('analytics.subtitle', { n: MAX_CAMPAIGNS })">
      <PvButton
        :label="$t('analytics.exportCsv')"
        severity="secondary"
        outlined
        :disabled="!hasData"
        @click="exportCsv"
      />
    </page-header>

    <!-- The one banner: tracking is off, which someone can fix. -->
    <PvMessage
      v-if="privacy.disable_tracking && !trackingNoticeHidden"
      severity="warn"
      icon="pi pi-exclamation-triangle"
      closable
      @close="hideTrackingNotice"
    >
      {{ $t('analytics.trackingDisabled') }}
      <router-link v-if="$can('settings:get')" :to="{ name: 'settings', params: { section: 'privacy' } }" class="ana-fix">{{ $t('analytics.turnOnTracking') }}</router-link>
    </PvMessage>

    <section class="ln-card ana-filters">
      <PvAutoComplete
        v-model="form.campaigns"
        :suggestions="queriedCampaigns"
        multiple
        option-label="name"
        :placeholder="form.campaigns.length ? '' : $t('analytics.addCampaign')"
        :loading="isSearchLoading"
        class="ana-camps"
        :aria-label="$t('globals.terms.campaigns')"
        @complete="(e) => queryCampaigns(e.query)"
        @focus="queryCampaigns('')"
        @update:model-value="onCampaigns"
      >
        <template #chip="{ value, removeCallback }">
          <span class="ana-chip">
            <span class="ana-chip-dot" :style="{ background: colorFor(value.id) }" />
            {{ value.name }}
            <button type="button" class="ana-chip-x" :aria-label="$t('globals.buttons.delete')" @click="removeCallback">
              <i class="pi pi-times" />
            </button>
          </span>
        </template>
      </PvAutoComplete>
      <PvDatePicker
        v-model="range"
        selection-mode="range"
        :manual-input="false"
        show-icon
        icon-display="input"
        date-format="d M yy"
        :max-date="new Date()"
        class="ana-range"
        :aria-label="$t('analytics.range')"
        @update:model-value="onRange"
      />
    </section>

    <empty-state
      v-if="!form.campaigns.length"
      icon="pi-chart-line"
      :title="$t('analytics.emptyTitle')"
      :text="$t('analytics.emptyText')"
    />

    <template v-else>
      <!-- Metric cards double as the chart's metric picker -->
      <div class="ana-metrics" role="radiogroup" :aria-label="$t('analytics.metric')">
        <button
          v-for="m in metricCards"
          :key="m.key"
          type="button"
          role="radio"
          class="ln-tile ana-metric"
          :class="{ 'is-on': metric === m.key }"
          :aria-checked="metric === m.key"
          :disabled="m.key === 'links'"
          @click="metric = m.key as Metric"
        >
          <span class="ln-tile-label">{{ m.label }}</span>
          <span class="ln-tile-value">
            <PvSkeleton v-if="loadingAny" width="80px" height="28px" />
            <template v-else>{{ m.value.toLocaleString() }}</template>
          </span>
          <span class="ln-small ana-sub">
            <template v-if="m.rates">
              <span v-for="r in m.rates" :key="r.id" class="ana-rate">
                <span
                  v-if="form.campaigns.length > 1"
                  class="ana-chip-dot"
                  role="img"
                  :aria-label="nameFor(r.id)"
                  :style="{ background: colorFor(r.id) }"
                />{{ r.rate }}
              </span>
              <span v-if="m.rates.length" class="ana-of">{{ $t('analytics.ofSendsShort') }}</span>
            </template>
            <template v-else>{{ m.sub }}</template>
          </span>
        </button>
      </div>
      <p v-if="!privacy.disable_tracking && !privacy.individual_tracking" class="ln-small ana-note">{{ $t('analytics.nonUnique') }}</p>

      <section class="ln-card ana-chart">
        <div class="ana-chart-head">
          <h2 class="ln-card-title">{{ $t('analytics.perDay', { metric: metricLabel }) }}</h2>
          <ul class="ana-legend">
            <li v-for="c in form.campaigns" :key="c.id"><span :style="{ background: colorFor(c.id) }" />{{ c.name }}</li>
          </ul>
        </div>
        <PvSkeleton v-if="data[metric].loading" height="240px" />
        <ln-chart
          v-else-if="chartSeries.some((s) => s.data.some((v) => v > 0))"
          type="line"
          :labels="chartLabels"
          :series="chartSeries"
          height="240px"
          :label="metricLabel"
        />
        <div v-else class="ana-chart-empty ln-small">{{ $t('analytics.noData') }}</div>
      </section>

      <div class="ana-lower">
        <section class="ln-card ana-links">
          <div class="ln-card-head"><h2 class="ln-card-title">{{ $t('analytics.topLinks') }}</h2></div>
          <table-skeleton v-if="data.links.loading" :rows="5" :cols="2" :two-line="false" />
          <p v-else-if="!links.length" class="ln-small ana-pad">{{ $t('analytics.noClicks') }}</p>
          <ul v-else class="ana-link-rows ln-enter">
            <li v-for="l in links" :key="l.url">
              <a :href="l.url" target="_blank" rel="noopener noreferrer" class="ln-mono ana-link-url" :title="l.url">{{ shortUrl(l.url) }}</a>
              <span class="ana-link-bar"><span :style="{ width: `${(l.count / links[0].count) * 100}%` }" /></span>
              <span class="ana-link-n">{{ l.count.toLocaleString() }}</span>
            </li>
          </ul>
        </section>

        <section class="ln-card ln-card-pad ana-bounces">
          <h2 class="ln-card-title">{{ $t('globals.terms.bounces') }}</h2>
          <div v-if="bounceTypes.total" class="ana-btypes">
            <div class="ana-bounce-bar" aria-hidden="true">
              <span v-for="b in bounceTypes.rows" :key="b.type" :class="`is-${b.type}`" :style="{ width: `${b.share}%` }" />
            </div>
            <ul class="ana-bounce-rows">
              <li v-for="b in bounceTypes.rows" :key="b.type">
                <span><span class="ana-chip-dot" :class="`is-${b.type}`" />{{ $t(`bounces.${b.type}`) }}</span>
                <strong>{{ b.count.toLocaleString() }}</strong>
              </li>
            </ul>
          </div>
          <router-link v-if="$can('bounces:get')" :to="{ name: 'bounces', query: { campaign_id: form.campaigns[0]?.id } }" class="ana-review">
            {{ $t('analytics.reviewBounces') }} →
          </router-link>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, reactive, ref,
} from 'vue';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import LnChart, { type ChartSeries } from '../components/ui/LnChart.vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';

// Campaigns and analytics rows are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
type Metric = 'views' | 'clicks' | 'bounces';

interface Point { campaignId: number; count: number; timestamp: string }

// Comparing more than this many campaigns makes the chart unreadable.
const MAX_CAMPAIGNS = 5;
// Series colours, in campaign order (tokens board series order).
const SERIES: ChartSeries['tone'][] = ['success', 'warn', 'danger', 'neutral', 'ink'];

const { $utils } = useGlobal();
const { listCampaigns, getCampaign, getCampaignAnalytics } = campaignsApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { shortDate } = useTime();
const { serverConfig } = storeToRefs(useMainStore());

const privacy = computed(() => (serverConfig.value as Loose).privacy || {});
const isSearchLoading = ref(false);
const queriedCampaigns = ref<Loose[]>([]);
const form = reactive({ campaigns: [] as Loose[] });
const range = ref<Date[] | null>(null);
const metric = ref<Metric>('views');

// Bounce types in display order, worst first.
const BOUNCE_TYPES = ['hard', 'soft', 'complaint'] as const;

const data = reactive<Record<Metric | 'links' | 'bounce_types', { loading: boolean; rows: Loose[] }>>({
  views: { loading: false, rows: [] },
  clicks: { loading: false, rows: [] },
  bounces: { loading: false, rows: [] },
  links: { loading: false, rows: [] },
  bounce_types: { loading: false, rows: [] },
});

const loadingAny = computed(() => Object.values(data).some((d) => d.loading));
const hasData = computed(() => form.campaigns.length > 0 && !loadingAny.value);
const metricLabel = computed(() => t(metric.value === 'bounces' ? 'globals.terms.bounces' : `campaigns.${metric.value}`));

function colorFor(id: number): string {
  const i = form.campaigns.findIndex((c) => c.id === id);
  const tone = SERIES[(i < 0 ? 0 : i) % SERIES.length];
  return `var(--ln-${tone === 'ink' ? 'ink' : tone})`;
}

function nameFor(id: number): string {
  return form.campaigns.find((c) => c.id === id)?.name || '';
}

function total(rows: Loose[], id?: number): number {
  return rows.reduce((n, r) => (id === undefined || r.campaignId === id ? n + (r.count || 0) : n), 0);
}

// rates is each campaign's total as a share of its sends, leaving out
// campaigns at zero (or with nothing sent) so only what happened shows.
function rates(rows: Loose[]): { id: number; rate: string }[] {
  return form.campaigns
    .filter((c) => c.sent && total(rows, c.id) > 0)
    .map((c) => {
      const pct = (total(rows, c.id) / c.sent) * 100;
      return { id: c.id, rate: `${pct < 10 ? pct.toFixed(1) : Math.round(pct)}%` };
    });
}

// The tracking-off banner, once dismissed, stays hidden in this browser.
const TRACKING_NOTICE_KEY = 'ln-analytics-tracking-dismissed';
const trackingNoticeHidden = ref(localStorage.getItem(TRACKING_NOTICE_KEY) === '1');
function hideTrackingNotice() {
  localStorage.setItem(TRACKING_NOTICE_KEY, '1');
  trackingNoticeHidden.value = true;
}

const links = computed(() => data.links.rows.slice(0, 10) as { url: string; count: number }[]);

const metricCards = computed(() => [
  {
    key: 'views', label: t('campaigns.views'), value: total(data.views.rows), rates: rates(data.views.rows), sub: '',
  },
  {
    key: 'clicks', label: t('campaigns.clicks'), value: total(data.clicks.rows), rates: rates(data.clicks.rows), sub: '',
  },
  {
    key: 'bounces', label: t('globals.terms.bounces'), value: total(data.bounces.rows), rates: rates(data.bounces.rows), sub: '',
  },
  {
    key: 'links', label: t('analytics.links'), value: data.links.rows.length, rates: null, sub: t('analytics.clickedOnce'),
  },
]);

// Ranges under a week come back hourly, longer ones daily.
const hourly = computed(() => {
  const [a, b] = range.value || [];
  return !!a && !!b && dayjs(b).diff(dayjs(a), 'day') < 7;
});

const buckets = computed(() => {
  const [a, b] = range.value || [];
  if (!a || !b) return [] as dayjs.Dayjs[];
  const unit = hourly.value ? 'hour' : 'day';
  const out: dayjs.Dayjs[] = [];
  let cur = dayjs(a).startOf(unit);
  const end = dayjs(b).endOf(unit);
  while (cur.isBefore(end) && out.length < 2000) {
    out.push(cur);
    cur = cur.add(1, unit);
  }
  return out;
});

const chartLabels = computed(() => buckets.value.map((d) => (hourly.value ? d.format('D MMM HH:mm') : shortDate(d.toDate()))));

const chartSeries = computed<ChartSeries[]>(() => {
  const unit = hourly.value ? 'hour' : 'day';
  const rows = data[metric.value].rows as Point[];
  return form.campaigns.map((c, i) => {
    const byBucket = new Map<number, number>();
    rows.filter((r) => r.campaignId === c.id).forEach((r) => {
      const k = dayjs(r.timestamp).startOf(unit).valueOf();
      byBucket.set(k, (byBucket.get(k) || 0) + r.count);
    });
    return {
      label: c.name,
      tone: SERIES[i % SERIES.length],
      data: buckets.value.map((d) => byBucket.get(d.valueOf()) || 0),
    };
  });
});

const bounceTypes = computed(() => {
  const all = total(data.bounce_types.rows);
  return {
    total: all,
    rows: BOUNCE_TYPES.map((type) => {
      const count = data.bounce_types.rows.filter((r) => r.type === type).reduce((n, r) => n + (r.count || 0), 0);
      return { type, count, share: all ? (count / all) * 100 : 0 };
    }).filter((b) => b.count > 0),
  };
});

function shortUrl(u: string): string {
  try {
    const url = new URL(u);
    return `${url.hostname}${url.pathname}`.replace(/\/$/, '');
  } catch {
    return u;
  }
}

function fetchAll() {
  const [from, to] = range.value || [];
  if (!form.campaigns.length || !from || !to) return;
  (['views', 'clicks', 'bounces', 'links', 'bounce_types'] as const).forEach((typ) => {
    data[typ].loading = true;
    getCampaignAnalytics(typ, {
      id: form.campaigns.map((c) => c.id),
      from: dayjs(from).format('YYYY-MM-DD'),
      to: dayjs(to).format('YYYY-MM-DD'),
    } as never).then((rows) => {
      data[typ].rows = (rows as unknown as Loose[]) || [];
    }).finally(() => { data[typ].loading = false; });
  });
}

function syncQuery() {
  const [from, to] = range.value || [];
  router.replace({
    query: {
      id: form.campaigns.map((c) => String(c.id)),
      ...(from ? { from: String(dayjs(from).unix()) } : {}),
      ...(to ? { to: String(dayjs(to).unix()) } : {}),
    },
  });
}

function onCampaigns(v: Loose[]) {
  if (v.length > MAX_CAMPAIGNS) {
    form.campaigns = v.slice(0, MAX_CAMPAIGNS);
    $utils.toast(t('analytics.maxCampaigns', { n: MAX_CAMPAIGNS }), 'is-danger');
  }
  syncQuery();
  fetchAll();
}

function onRange(v: Date[] | null) {
  if (!v || !v[0] || !v[1]) return;
  syncQuery();
  fetchAll();
}

function queryCampaigns(q: string) {
  isSearchLoading.value = true;
  listCampaigns({
    query: q, order_by: 'created_at', order: 'DESC', no_body: true,
  } as never).then((page) => {
    const chosen = new Set(form.campaigns.map((c) => c.id));
    queriedCampaigns.value = ((page as unknown as Loose).results || []).filter((c: Loose) => !chosen.has(c.id));
  }).finally(() => { isSearchLoading.value = false; });
}

// exportCsv downloads the selected campaigns' daily (or hourly) numbers.
function exportCsv() {
  const header = ['campaign_id', 'campaign', 'metric', 'timestamp', 'count'];
  const lines = [header.join(',')];
  const name = (id: number) => (form.campaigns.find((c) => c.id === id)?.name || '').replace(/"/g, '""');
  (['views', 'clicks', 'bounces'] as const).forEach((m) => {
    (data[m].rows as Point[]).forEach((r) => {
      lines.push([r.campaignId, `"${name(r.campaignId)}"`, m, r.timestamp, r.count].join(','));
    });
  });
  const blob = new Blob([lines.join('\n')], { type: 'text/csv' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `campaign-analytics-${dayjs().format('YYYY-MM-DD')}.csv`;
  a.click();
  URL.revokeObjectURL(a.href);
}

onMounted(() => {
  const now = dayjs().endOf('day');
  const from = route.query.from ? dayjs.unix(Number(route.query.from)) : now.subtract(10, 'day').startOf('day');
  const to = route.query.to ? dayjs.unix(Number(route.query.to)) : now;
  range.value = [from.toDate(), to.toDate()];

  const ids = $utils.parseQueryIDs(route.query.id) as number[];
  if (!ids.length) return;
  isSearchLoading.value = true;
  Promise.allSettled(ids.slice(0, MAX_CAMPAIGNS).map((id) => getCampaign(id))).then((res) => {
    res.forEach((r) => { if (r.status === 'fulfilled') form.campaigns.push(r.value as unknown as Loose); });
    isSearchLoading.value = false;
    fetchAll();
  });
});
</script>

<style scoped lang="scss">
.ana-filters {
  display: flex;
  gap: 8px;
  padding: 10px;
  flex-wrap: wrap;

  .ana-camps { flex: 1 1 400px; :deep(.p-autocomplete-input-multiple) { width: 100%; border-color: var(--ln-border); } }
  .ana-range { width: 260px; }
}
.ana-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 6px 2px 8px;
  border-radius: 4px;
  background: var(--ln-muted);
  font-size: 13px;
  .app-dark & { background: var(--ln-hover); }
}
.ana-chip-dot { width: 8px; height: 8px; border-radius: 2px; display: inline-block; flex: none; }
.ana-chip-x { border: 0; background: none; padding: 0 2px; color: var(--ln-text-2); cursor: pointer; i { font-size: 10px; } }

.ana-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;

  @media (max-width: 900px) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
.ana-metric {
  text-align: left;
  font: inherit;
  color: var(--ln-text);
  cursor: pointer;

  &:disabled { cursor: default; opacity: 1; }
  &.is-on { border-color: var(--ln-ink); box-shadow: 0 0 0 1px var(--ln-ink); }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}
// Each campaign's rate next to its colour, then "of sends" once.
.ana-sub { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 10px; min-height: 1.4em; }
.ana-rate { display: inline-flex; align-items: center; gap: 5px; }
.ana-of { margin-left: -6px; }
.ana-note { margin: -4px 0 0; }
.ana-fix { color: inherit; font-weight: 600; text-decoration: underline; text-underline-offset: 3px; margin-left: 4px; }
.ana-chart { padding: 18px; display: flex; flex-direction: column; gap: 12px; }
.ana-chart-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.ana-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 12px;
  color: var(--ln-text-2);

  li { display: inline-flex; align-items: center; gap: 6px; }
  span { width: 12px; height: 2px; border-radius: 2px; }
}
.ana-chart-empty { height: 240px; display: flex; align-items: center; justify-content: center; }

.ana-lower {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(0, 1fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.ana-links { overflow: hidden; }
.ana-pad { padding: 16px 18px; margin: 0; }
.ana-link-rows { list-style: none; margin: 0; padding: 0; }
.ana-link-rows li {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px 64px;
  gap: 16px;
  align-items: center;
  padding: 10px 18px;
  border-top: 1px solid var(--ln-border);

  &:first-child { border-top: 0; }
}
.ana-link-url { font-size: 12px; color: var(--ln-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-decoration: none; &:hover { text-decoration: underline; } }
.ana-link-bar {
  height: 4px;
  border-radius: 99px;
  background: var(--ln-muted);
  overflow: hidden;

  span { display: block; height: 100%; background: var(--ln-success); }
}
.ana-link-n { text-align: right; font-variant-numeric: tabular-nums; font-size: 13px; }
.ana-bounces { display: flex; flex-direction: column; gap: 12px; }
.ana-bounce-bar {
  display: flex;
  height: 8px;
  border-radius: 99px;
  overflow: hidden;
  background: var(--ln-muted);
}
.ana-bounce-rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;

  li { display: flex; justify-content: space-between; gap: 8px; }
  span { display: inline-flex; align-items: center; gap: 8px; }
}
.ana-review { font-size: 13px; color: var(--ln-text); text-decoration: underline; }
.ana-btypes { display: flex; flex-direction: column; gap: 12px; }
// Hard bounces and complaints need action; soft ones usually clear up.
.ana-bounce-bar span, .ana-chip-dot {
  &.is-hard { background: var(--ln-danger); }
  &.is-soft { background: var(--ln-warn); }
  &.is-complaint { background: var(--ln-series-5); }
}
.ana-bounces h3 { margin: 4px 0 -4px; }
</style>
