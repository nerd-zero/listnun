<template>
  <div class="ln-page camps-ov">
    <page-header :title="$t('globals.terms.campaigns')" :subtitle="summary">
      <template #search>
        <form class="ln-search camps-ov-search" role="search" @submit.prevent="onSearch">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="query" name="query" :placeholder="$t('campaigns.overview.search')" :aria-label="$t('campaigns.overview.search')" data-cy="query" />
          </PvIconField>
        </form>
      </template>
      <PvButton v-if="$can('campaigns:manage')" icon="pi pi-plus" :label="$t('campaigns.newCampaign')" data-cy="btn-new" @click="$router.push({ name: 'campaign', params: { id: 'new' } })" />
    </page-header>

    <table-skeleton v-if="!data" :cols="3" />

    <empty-state
      v-else-if="!data.campaigns.length"
      icon="pi-megaphone"
      :title="$t('campaigns.overview.none')"
      :action-label="$can('campaigns:manage') ? $t('campaigns.newCampaign') : ''"
      @action="$router.push({ name: 'campaign', params: { id: 'new' } })"
    />

    <template v-else>
      <!-- Paused campaigns -->
      <div v-for="c in paused" :key="c.id" role="status" class="camps-ov-paused" data-cy="paused-banner">
        <span class="camps-ov-paused-text">
          <i class="pi pi-pause" aria-hidden="true" />
          <span>
            <strong>{{ $t('campaigns.overview.pausedTitle', { name: c.name }) }}</strong>
            <span>{{ pausedText(c) }}</span>
            <span v-if="c.pauseReason" class="ln-small">{{ $t(`campaigns.pauseReason.${c.pauseReason}`) }}</span>
          </span>
        </span>
        <span class="camps-ov-paused-actions">
          <PvButton :label="$t('campaigns.overview.review')" severity="secondary" outlined size="small" @click="open(c)" />
          <PvButton v-if="$can('campaigns:send')" :label="$t('campaigns.overview.resume')" size="small" data-cy="btn-resume" @click="askStatus(c, 'running')" />
        </span>
      </div>

      <!-- Sending now -->
      <section v-if="running.length" :aria-label="$t('campaigns.overview.sendingNow')" class="camps-ov-section">
        <h2 class="camps-ov-h2">{{ $t('campaigns.overview.sendingNow') }}</h2>
        <article v-for="c in running" :key="c.id" class="ln-card camps-ov-live" data-cy="running">
          <div class="camps-ov-live-head">
            <div class="camps-ov-live-title">
              <h3>{{ c.name }}</h3>
              <span class="ln-small">{{ $t('campaigns.overview.startedTo', { lists: listNames(c), time: startedLabel(c) }) }}</span>
            </div>
            <span class="camps-ov-live-actions">
              <PvButton v-if="$can('campaigns:send')" icon="pi pi-pause" :label="$t('campaigns.confirm.paused.action')" severity="secondary" outlined @click="askStatus(c, 'paused')" />
              <PvButton :label="$t('campaigns.overview.liveResults')" severity="secondary" outlined @click="open(c)" />
            </span>
          </div>
          <div class="camps-ov-progress" role="progressbar" :aria-valuenow="sentOf(c)" aria-valuemin="0" :aria-valuemax="totalOf(c)">
            <span :style="{ width: `${pct(sentOf(c), totalOf(c))}%` }" />
          </div>
          <dl class="camps-ov-figures">
            <div>
              <dt>{{ $t('campaigns.overview.sent') }}</dt>
              <dd>{{ $t('campaigns.overview.sentOf', { sent: $utils.niceNumber(sentOf(c)), total: $utils.niceNumber(totalOf(c)) }) }}</dd>
            </div>
            <div>
              <dt>{{ $t('campaigns.overview.speed') }}</dt>
              <dd>{{ $t('campaigns.overview.perMinute', { n: $utils.niceNumber(rateOf(c)) }) }}</dd>
            </div>
            <div>
              <dt>{{ $t('campaigns.overview.timeLeft') }}</dt>
              <dd>{{ timeLeftLabel(c) }}</dd>
            </div>
            <div>
              <dt>{{ $t('campaigns.overview.openedSoFar') }}</dt>
              <dd>{{ rate(c.opened, sentOf(c)) }}</dd>
            </div>
          </dl>
        </article>
      </section>

      <div v-if="upNext.length || drafts.length" class="camps-ov-pair">
        <!-- Up next -->
        <section v-if="upNext.length" class="camps-ov-section" :aria-label="$t('campaigns.overview.upNext')">
          <h2 class="camps-ov-h2">{{ $t('campaigns.overview.upNext') }}</h2>
          <div class="ln-card camps-ov-next">
            <template v-for="d in upNext" :key="d.day">
              <h3 class="camps-ov-day">{{ d.label }}</h3>
              <div v-for="c in d.items" :key="c.id" class="camps-ov-next-row" data-cy="scheduled">
                <time class="ln-mono" :datetime="c.sendAt">{{ hhmm(c.sendAt) }}</time>
                <span class="camps-ov-next-name">
                  <strong>{{ c.name }}</strong>
                  <span class="ln-small">{{ $t('campaigns.overview.audience', { lists: listNames(c), n: $utils.niceNumber(c.audience) }) }}</span>
                </span>
                <router-link :to="{ name: 'campaign', params: { id: c.id } }">{{ $t('globals.buttons.edit') }}</router-link>
              </div>
            </template>
          </div>
        </section>

        <!-- Drafts -->
        <section v-if="drafts.length" class="camps-ov-section" :aria-label="$t('campaigns.overview.draftsTitle')">
          <div class="camps-ov-section-head">
            <h2 class="camps-ov-h2">{{ $t('campaigns.overview.draftsTitle') }}</h2>
            <router-link :to="{ name: 'allCampaigns', query: { status: 'draft' } }" class="ln-small">{{ $t('campaigns.overview.allDrafts', { n: count('draft') }) }}</router-link>
          </div>
          <router-link
            v-for="c in drafts"
            :key="c.id"
            :to="{ name: 'campaign', params: { id: c.id } }"
            class="ln-card camps-ov-draft"
            data-cy="draft"
          >
            <strong>{{ c.name }}</strong>
            <span class="ln-small">{{ draftMeta(c) }}</span>
            <span class="camps-ov-continue">{{ $t('campaigns.overview.continue') }}</span>
          </router-link>
        </section>
      </div>

      <!-- Sent -->
      <section v-if="sent.length" class="camps-ov-section" :aria-label="$t('campaigns.overview.sentTitle')">
        <div class="camps-ov-section-head">
          <h2 class="camps-ov-h2">{{ $t('campaigns.overview.sentTitle') }}</h2>
          <router-link v-if="$can('campaigns:get_analytics')" :to="{ name: 'campaignAnalytics' }" class="ln-small">{{ $t('campaigns.overview.compare') }}</router-link>
        </div>
        <div class="ln-card camps-ov-sent">
          <div v-if="!isMobile" class="camps-ov-sent-row is-head" aria-hidden="true">
            <span>{{ $t('campaigns.overview.col.campaign') }}</span>
            <span>{{ $t('campaigns.overview.col.sent') }}</span>
            <span class="is-num">{{ $t('campaigns.overview.col.delivered') }}</span>
            <span>{{ $t('campaigns.overview.col.opened') }}</span>
            <span class="is-num">{{ $t('campaigns.overview.col.clicked') }}</span>
          </div>
          <router-link
            v-for="c in sent"
            :key="c.id"
            :to="{ name: 'campaign', params: { id: c.id } }"
            class="camps-ov-sent-row"
            data-cy="sent"
          >
            <span class="camps-ov-sent-name">
              <strong>{{ c.name }}</strong>
              <span class="ln-small">{{ listNames(c) }}</span>
            </span>
            <template v-if="!isMobile">
              <span class="ln-small">{{ shortDate(c.startedAt || c.updatedAt) }}</span>
              <span class="is-num">{{ $utils.niceNumber(delivered(c)) }}</span>
              <span class="camps-ov-open">
                <span class="camps-ov-bar" aria-hidden="true"><span :style="{ width: `${pct(c.opened, delivered(c))}%` }" /></span>
                <span class="is-num">{{ rate(c.opened, delivered(c)) }}</span>
              </span>
              <span class="is-num">{{ rate(c.clicked, delivered(c)) }}</span>
            </template>
            <span v-else class="camps-ov-sent-m ln-small">
              <span>{{ shortDate(c.startedAt || c.updatedAt) }}</span>
              <span>{{ $t('campaigns.overview.opened', { n: rate(c.opened, delivered(c)) }) }}</span>
            </span>
          </router-link>
          <div class="camps-ov-sent-foot ln-small">
            <span>{{ $t('campaigns.overview.mostRecent', { n: sent.length, total: count('finished') }) }}</span>
            <router-link :to="{ name: 'allCampaigns', query: { status: 'finished' } }">{{ $t('campaigns.overview.showAllSent') }}</router-link>
          </div>
        </div>
      </section>
    </template>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :danger="false"
      :busy="confirm.busy"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, onUnmounted, reactive, ref, watch,
} from 'vue';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import { useMedia } from '../composables/useMedia';
import PageHeader from '../components/ui/PageHeader.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';

interface ListRef { id: number; name: string }
interface Item {
  id: number; name: string; status: string; pauseReason?: string | null;
  sendAt?: string; startedAt?: string; updatedAt?: string; updatedByName: string;
  toSend: number; sent: number; lists: ListRef[]; audience: number; bounced: number; opened: number; clicked: number;
}
interface Overview { campaigns: Item[]; counts: { status: string; count: number }[] }
// A running campaign's live figures from /api/campaigns/running/stats.
interface Live { id: number; sent: number; toSend: number; rate: number; netRate: number }

// How often running campaigns' figures refresh.
const LIVE_POLL_MS = 2000;

const { $utils } = useGlobal();
const { getCampaignsOverview, getRunningCampaignStats, updateCampaignStatus } = campaignsApi();
const { t } = useI18n();
const router = useRouter();
const { ago, shortDate } = useTime();
const isMobile = useMedia('(max-width: 767px)');
const { refreshTick } = storeToRefs(useMainStore());

const data = ref<Overview | null>(null);
const live = ref<Record<number, Live>>({});
const query = ref('');
let pollTimer: ReturnType<typeof setInterval> | null = null;
const confirm = reactive({
  open: false, title: '', text: '', label: '', busy: false, run: () => {},
});

const byStatus = (s: string) => (data.value?.campaigns || []).filter((c) => c.status === s);
const paused = computed(() => byStatus('paused'));
const running = computed(() => byStatus('running'));
const drafts = computed(() => byStatus('draft').sort((a, b) => dayjs(b.updatedAt).valueOf() - dayjs(a.updatedAt).valueOf()));
const sent = computed(() => byStatus('finished').sort((a, b) => dayjs(b.startedAt || b.updatedAt).valueOf() - dayjs(a.startedAt || a.updatedAt).valueOf()));
// Scheduled campaigns grouped by the day they go out.
const upNext = computed(() => {
  const days: { day: string; label: string; items: Item[] }[] = [];
  byStatus('scheduled')
    .sort((a, b) => dayjs(a.sendAt).valueOf() - dayjs(b.sendAt).valueOf())
    .forEach((c) => {
      const day = dayjs(c.sendAt).format('YYYY-MM-DD');
      let d = days.find((x) => x.day === day);
      if (!d) {
        d = { day, label: dayjs(c.sendAt).format('dddd D MMMM'), items: [] };
        days.push(d);
      }
      d.items.push(c);
    });
  return days;
});

function count(status: string): number {
  return data.value?.counts.find((c) => c.status === status)?.count || 0;
}

const summary = computed(() => {
  if (!data.value) return '';
  const parts = [];
  if (count('running')) parts.push(t('campaigns.overview.sending', { n: count('running') }));
  if (count('scheduled')) parts.push(t('campaigns.overview.scheduled', { n: count('scheduled') }));
  if (count('draft')) parts.push(t('campaigns.overview.drafts', { n: count('draft') }, count('draft')));
  return parts.join(', ');
});

function listNames(c: Item): string {
  return c.lists.map((l) => l.name).join(', ');
}

const sentOf = (c: Item) => live.value[c.id]?.sent ?? c.sent;
const totalOf = (c: Item) => live.value[c.id]?.toSend ?? c.toSend;
const rateOf = (c: Item) => live.value[c.id]?.rate || live.value[c.id]?.netRate || 0;
const delivered = (c: Item) => Math.max(0, c.sent - c.bounced);

function pct(n: number, of: number): number {
  return of ? Math.min(100, (n / of) * 100) : 0;
}

function rate(n: number, of: number): string {
  return of ? `${((n / of) * 100).toFixed(1)}%` : '–';
}

function hhmm(stamp?: string): string {
  return stamp ? dayjs(stamp).format('HH:mm') : '';
}

function startedLabel(c: Item): string {
  if (!c.startedAt) return '';
  return dayjs(c.startedAt).isSame(dayjs(), 'day') ? t('campaigns.overview.today', { time: hhmm(c.startedAt) }) : shortDate(c.startedAt, true);
}

function timeLeftLabel(c: Item): string {
  const r = rateOf(c);
  const left = totalOf(c) - sentOf(c);
  if (!r || left <= 0) return '–';
  const mins = Math.ceil(left / r);
  return mins < 90 ? t('campaigns.overview.aboutMin', { n: mins }) : t('campaigns.overview.aboutHours', { n: Math.round(mins / 60) });
}

function pausedText(c: Item): string {
  return t('campaigns.overview.pausedText', {
    date: ago(c.updatedAt),
    sent: $utils.niceNumber(c.sent),
    total: $utils.niceNumber(c.toSend),
    left: $utils.niceNumber(Math.max(0, c.toSend - c.sent)),
    lists: listNames(c),
  });
}

function draftMeta(c: Item): string {
  const when = ago(c.updatedAt);
  const edited = c.updatedByName ? t('campaigns.overview.editedBy', { when, name: c.updatedByName }) : t('campaigns.overview.edited', { when });
  return `${edited}, ${c.lists.length ? listNames(c) : t('campaigns.overview.noLists')}`;
}

function open(c: Item) {
  router.push({ name: 'campaign', params: { id: c.id } });
}

function onSearch() {
  router.push({ name: 'allCampaigns', query: query.value ? { query: query.value } : {} });
}

function pollLive() {
  if (pollTimer) clearInterval(pollTimer);
  pollTimer = null;
  if (!running.value.length) return;
  pollTimer = setInterval(() => {
    getRunningCampaignStats().then((d) => {
      const list = (d as unknown as Live[]) || [];
      // A campaign that stopped running moves section: reload.
      if (list.length < running.value.length) {
        fetchOverview();
        return;
      }
      live.value = Object.fromEntries(list.map((l) => [l.id, l]));
    }).catch(() => {});
  }, LIVE_POLL_MS);
}

function fetchOverview() {
  getCampaignsOverview().then((d) => {
    data.value = d as unknown as Overview;
    pollLive();
  });
}

function askStatus(c: Item, status: 'running' | 'paused') {
  const resume = status === 'running';
  Object.assign(confirm, {
    open: true,
    title: resume ? t('campaigns.overview.resumeTitle', { name: c.name }) : t('campaigns.confirm.paused.title', { name: c.name }),
    text: resume
      ? t('campaigns.overview.resumeText', { n: $utils.niceNumber(Math.max(0, c.toSend - c.sent)) })
      : t('campaigns.confirm.paused.text'),
    label: resume ? t('campaigns.overview.resume') : t('campaigns.confirm.paused.action'),
    run: () => {
      confirm.busy = true;
      updateCampaignStatus(c.id, { status } as never).then(() => {
        confirm.open = false;
        $utils.toast(t('campaigns.statusChanged', { name: c.name, status: t(`campaigns.status.${status}`) }));
        fetchOverview();
      }).finally(() => { confirm.busy = false; });
    },
  });
}

watch(refreshTick, fetchOverview);
onMounted(fetchOverview);
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer); });
</script>

<style scoped lang="scss">
.camps-ov-search {
  width: 280px;

  // Phones: search fills the row beside New campaign, as in the artboard.
  @media (max-width: 767px) { flex: 1; width: auto; min-width: 0; }
}
@media (max-width: 767px) {
  .camps-ov :deep(.ln-page-header-actions) { flex-wrap: nowrap; width: 100%; }
}

.camps-ov-section { display: flex; flex-direction: column; gap: 10px; min-width: 0; }
.camps-ov-section-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;

  a { color: var(--ln-text-2); text-decoration: underline; }
}
.camps-ov-h2 { margin: 0; font-size: 15px; font-weight: 600; }

.camps-ov-paused {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  padding: 14px 16px;
  border-radius: var(--ln-radius-card);
  border: 1px solid var(--ln-warn);
  background: var(--ln-warn-bg);

  @media (max-width: 767px) { flex-direction: column; align-items: flex-start; }
}
.camps-ov-paused-text {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  min-width: 0;

  i { margin-top: 3px; color: var(--ln-warn-fg); }
  > span { display: flex; flex-direction: column; gap: 2px; }
  strong { font-weight: 600; }
}
.camps-ov-paused-actions {
  display: flex;
  gap: 8px;
  flex: none;

  :deep(.p-button-outlined) { background: var(--ln-card); color: var(--ln-text); }
}

.camps-ov-live { padding: 20px 22px; display: flex; flex-direction: column; gap: 16px; }
.camps-ov-live-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}
.camps-ov-live-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;

  h3 { margin: 0; font-size: 17px; font-weight: 600; }
}
.camps-ov-live-actions { display: flex; gap: 8px; }
.camps-ov-progress {
  height: 8px;
  border-radius: 99px;
  background: var(--ln-muted);
  overflow: hidden;

  span { display: block; height: 100%; background: var(--ln-success); transition: width var(--ln-dur-slow) var(--ln-ease); }
}
.camps-ov-figures {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin: 0;

  dt { font-size: 13px; color: var(--ln-text-2); }
  dd { margin: 2px 0 0; font-size: 18px; line-height: 24px; font-weight: 600; font-variant-numeric: tabular-nums; }
  @media (max-width: 767px) { grid-template-columns: repeat(2, minmax(0, 1fr)); row-gap: 16px; }
}

.camps-ov-pair {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 24px;
  align-items: start;

  > :only-child { grid-column: 1 / -1; }
  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
}
.camps-ov-next { overflow: hidden; }
.camps-ov-day {
  margin: 0;
  padding: 10px 18px;
  font-size: 13px;
  font-weight: 600;
  background: var(--ln-muted);
  border-bottom: 1px solid var(--ln-border);

  &:not(:first-child) { border-top: 1px solid var(--ln-border); }
}
.camps-ov-next-row {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 18px;

  & + & { border-top: 1px solid var(--ln-border); }
  time { color: var(--ln-text-2); font-size: 13px; }
  a { color: var(--ln-text); font-size: 13px; font-weight: 500; text-decoration: underline; }
}
.camps-ov-next-name { display: flex; flex-direction: column; min-width: 0; strong { font-weight: 500; } }

.camps-ov-draft {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px 18px;
  color: var(--ln-text);
  text-decoration: none;

  &:hover { border-color: var(--ln-border-strong); text-decoration: none; }
  strong { font-weight: 500; }
}
.camps-ov-continue { margin-top: 6px; font-size: 13px; font-weight: 600; }

.camps-ov-sent { overflow: hidden; }
.camps-ov-sent-row {
  display: grid;
  grid-template-columns: minmax(0, 2.4fr) 90px 90px minmax(0, 1.6fr) 80px;
  gap: 16px;
  align-items: center;
  padding: 14px 18px;
  border-top: 1px solid var(--ln-border);
  color: var(--ln-text);
  text-decoration: none;

  &:hover:not(.is-head) { background: var(--ln-hover); text-decoration: none; }
  &.is-head {
    border-top: 0;
    padding-block: 10px;
    background: var(--ln-muted);
    font-size: 12px;
    font-weight: 600;
    color: var(--ln-text-3);
  }
  .is-num { text-align: right; font-variant-numeric: tabular-nums; }
  @media (max-width: 767px) { grid-template-columns: minmax(0, 1fr) auto; }
}
.camps-ov-sent-name { display: flex; flex-direction: column; min-width: 0; strong { font-weight: 500; } }
.camps-ov-sent-m { display: flex; flex-direction: column; align-items: flex-end; }
.camps-ov-open { display: grid; grid-template-columns: minmax(0, 1fr) 56px; gap: 10px; align-items: center; }
.camps-ov-bar {
  height: 6px;
  border-radius: 99px;
  background: var(--ln-muted);
  overflow: hidden;

  span { display: block; height: 100%; background: var(--ln-success); }
}
.camps-ov-sent-foot {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px;
  border-top: 1px solid var(--ln-border);

  a { color: var(--ln-text); font-weight: 500; text-decoration: underline; }
}
</style>
