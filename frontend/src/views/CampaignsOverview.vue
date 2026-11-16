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
            <span class="camps-ov-stats ln-small" data-cy="paused-stats">
              <span v-for="f in soFar(c)" :key="f.key">{{ f.label }} <strong>{{ f.value }}</strong></span>
            </span>
          </span>
        </span>
        <span class="camps-ov-paused-actions">
          <PvButton :label="$t('campaigns.overview.review')" severity="secondary" outlined size="small" @click="edit(c)" />
          <PvButton v-if="$can('campaigns:send')" :label="$t('campaigns.overview.resume')" size="small" data-cy="btn-resume" @click="askStatus(c, 'running')" />
          <row-icon-button icon="pi-ellipsis-h" :label="$t('globals.buttons.rowActions')" cy="btn-row-menu" menu @click="(e) => openMenu(e, c)" />
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
              <PvButton :label="$t('campaigns.overview.liveResults')" severity="secondary" outlined @click="openResults(c)" />
              <row-icon-button icon="pi-ellipsis-h" :label="$t('globals.buttons.rowActions')" cy="btn-row-menu" menu @click="(e) => openMenu(e, c)" />
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
            <div v-for="f in soFar(c)" :key="f.key">
              <dt>{{ f.label }}</dt>
              <dd>{{ f.value }}</dd>
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
                <span class="camps-ov-row-actions">
                  <router-link :to="{ name: 'campaign', params: { id: c.id } }">{{ $t('globals.buttons.edit') }}</router-link>
                  <row-icon-button icon="pi-ellipsis-h" :label="$t('globals.buttons.rowActions')" cy="btn-row-menu" menu @click="(e) => openMenu(e, c)" />
                </span>
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
          <div v-for="c in drafts" :key="c.id" class="ln-card camps-ov-draft" data-cy="draft">
            <router-link :to="{ name: 'campaign', params: { id: c.id } }" class="camps-ov-draft-name">{{ c.name }}</router-link>
            <span class="ln-small">{{ draftMeta(c) }}</span>
            <span class="camps-ov-draft-actions">
              <router-link :to="{ name: 'campaign', params: { id: c.id } }" class="camps-ov-continue">{{ $t('campaigns.overview.continue') }}</router-link>
              <span class="camps-ov-row-actions">
                <PvButton
                  v-if="sendAction(c) && c.lists.length"
                  :icon="`pi ${sendAction(c)!.icon}`"
                  :label="sendAction(c)!.label"
                  :data-cy="sendAction(c)!.cy"
                  size="small"
                  severity="secondary"
                  outlined
                  @click="sendAction(c)!.run()"
                />
                <row-icon-button icon="pi-eye" :label="$t('campaigns.preview')" cy="btn-preview" @click="preview(c)" />
                <row-icon-button v-if="$can('campaigns:manage')" icon="pi-copy" :label="$t('globals.buttons.clone')" cy="btn-clone" @click="askClone(c)" />
                <row-icon-button icon="pi-ellipsis-h" :label="$t('globals.buttons.rowActions')" cy="btn-row-menu" menu @click="(e) => openMenu(e, c)" />
              </span>
            </span>
          </div>
        </section>
      </div>

      <!-- Sent -->
      <section v-if="sent.length" class="camps-ov-section" :aria-label="$t('campaigns.overview.sentTitle')">
        <div class="camps-ov-section-head">
          <h2 class="camps-ov-h2">{{ $t('campaigns.overview.sentTitle') }}</h2>
          <router-link v-if="$can('campaigns:get_analytics')" :to="{ name: 'campaignAnalytics' }" class="ln-small">{{ $t('campaigns.overview.compare') }}</router-link>
        </div>
        <div class="ln-card camps-ov-sent">
          <div v-if="!isMobile" class="camps-ov-sent-row is-head" :style="sentGrid" aria-hidden="true">
            <span>{{ $t('campaigns.overview.col.campaign') }}</span>
            <span>{{ $t('campaigns.overview.col.sent') }}</span>
            <span class="is-num">{{ $t('campaigns.overview.col.delivered') }}</span>
            <span>{{ $t('campaigns.overview.col.opened') }}</span>
            <span class="is-num">{{ $t('campaigns.overview.col.clicked') }}</span>
            <span class="is-num">{{ $t('campaigns.overview.col.bounced') }}</span>
            <span v-if="showReplies" class="is-num">{{ $t('campaigns.overview.col.replied') }}</span>
            <span />
          </div>
          <div
            v-for="c in sent"
            :key="c.id"
            class="camps-ov-sent-row"
            :style="isMobile ? undefined : sentGrid"
            data-cy="sent"
            @click="edit(c)"
          >
            <span class="camps-ov-sent-name">
              <router-link :to="{ name: 'campaign', params: { id: c.id } }" @click.stop>{{ c.name }}</router-link>
              <span class="ln-small">{{ listNames(c) }}</span>
            </span>
            <template v-if="!isMobile">
              <span class="ln-small">{{ shortDate(c.startedAt || c.updatedAt) }}</span>
              <span class="is-num">{{ $utils.niceNumber(delivered(c)) }}</span>
              <span class="camps-ov-open">
                <span v-if="perPerson" class="camps-ov-bar" aria-hidden="true"><span :style="{ width: `${pct(c.opened, delivered(c))}%` }" /></span>
                <span class="is-num">{{ result(c.opened, delivered(c)) }}</span>
              </span>
              <span class="is-num">{{ result(c.clicked, delivered(c)) }}</span>
              <span class="is-num">
                <router-link
                  v-if="c.bounced && $can('bounces:get')"
                  :to="{ name: 'bounces', query: { campaign_id: c.id } }"
                  class="camps-ov-bounces"
                  @click.stop
                >{{ $utils.niceNumber(c.bounced) }}</router-link>
                <template v-else>{{ $utils.niceNumber(c.bounced) }}</template>
              </span>
              <span v-if="showReplies" class="is-num">{{ result(c.replied, delivered(c)) }}</span>
            </template>
            <span v-else class="camps-ov-sent-m ln-small">
              <span>{{ shortDate(c.startedAt || c.updatedAt) }}</span>
              <span>{{ $t('campaigns.overview.opened', { n: result(c.opened, delivered(c)) }) }}</span>
            </span>
            <span class="camps-ov-row-actions" @click.stop>
              <row-icon-button v-if="!isMobile && $can('campaigns:get_analytics')" icon="pi-chart-bar" :label="$t('globals.terms.analytics')" cy="btn-analytics" @click="analytics(c)" />
              <row-icon-button v-if="!isMobile && $can('campaigns:manage')" icon="pi-copy" :label="$t('globals.buttons.clone')" cy="btn-clone" @click="askClone(c)" />
              <row-icon-button icon="pi-ellipsis-h" :label="$t('globals.buttons.rowActions')" cy="btn-row-menu" menu @click="(e) => openMenu(e, c)" />
            </span>
          </div>
          <div class="camps-ov-sent-foot ln-small">
            <span>{{ $t('campaigns.overview.mostRecent', { n: sent.length, total: count('finished') }) }}</span>
            <router-link :to="{ name: 'allCampaigns', query: { status: 'finished' } }">{{ $t('campaigns.overview.showAllSent') }}</router-link>
          </div>
        </div>
      </section>
    </template>

    <PvMenu ref="menu" :model="menuItems" popup />

    <campaign-preview v-if="previewItem" type="campaign" :id="previewItem.id" :title="previewItem.name" @close="previewItem = null" />

    <dialog-form
      v-model:visible="clone.open"
      :title="$t('globals.buttons.clone')"
      :submit-label="$t('globals.buttons.clone')"
      :busy="clone.busy"
      :submit-disabled="!clone.name.trim()"
      @submit="doClone"
    >
      <div class="ln-field">
        <label for="clone-name">{{ $t('globals.fields.name') }}</label>
        <PvInputText id="clone-name" v-model="clone.name" autofocus />
      </div>
    </dialog-form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :confirm-text="confirm.typed"
      :danger="confirm.danger"
      :busy="confirm.busy"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, onUnmounted, ref, watch,
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
import DialogForm from '../components/ui/DialogForm.vue';
import RowIconButton from '../components/ui/RowIconButton.vue';
import CampaignPreview from '../components/CampaignPreview.vue';
import { useCampaignActions } from '../composables/useCampaignActions';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';

interface ListRef { id: number; name: string }
interface Item {
  id: number; name: string; status: string; pauseReason?: string | null;
  sendAt?: string; startedAt?: string; updatedAt?: string; updatedByName: string;
  toSend: number; sent: number; lists: ListRef[]; audience: number;
  // Results, counted as on the Analytics page (models.CampaignEngagement).
  bounced: number; delivered: number; opened: number; clicked: number; anonymous: number; replied: number;
}
interface Overview { campaigns: Item[]; counts: { status: string; count: number }[] }
// A running campaign's live figures from /api/campaigns/running/stats.
interface Live { id: number; sent: number; toSend: number; rate: number; netRate: number }

// How often running campaigns' figures refresh.
const LIVE_POLL_MS = 2000;

const { $utils, $can } = useGlobal();
const { getCampaignsOverview, getRunningCampaignStats } = campaignsApi();
const { t } = useI18n();
const router = useRouter();
const { ago, shortDate } = useTime();
const isMobile = useMedia('(max-width: 767px)');
const { refreshTick, serverConfig } = storeToRefs(useMainStore());

// Opens and clicks count people only with per-subscriber tracking on;
// without it they count every view and click, and a rate would overstate.
const perPerson = computed(() => !!(serverConfig.value as { privacy?: { individual_tracking?: boolean } }).privacy?.individual_tracking);

const data = ref<Overview | null>(null);
const live = ref<Record<number, Live>>({});
const query = ref('');
let pollTimer: ReturnType<typeof setInterval> | null = null;
const {
  previewItem, confirm, clone, menu, menuItems, openMenu,
  edit, analytics, preview, askStatus, askClone, doClone, sendAction,
} = useCampaignActions(() => fetchOverview());

// Replies show where the workspace has reply tracking, or a campaign here
// got replies while it had (as on the Analytics page).
const showReplies = computed(() => !!(serverConfig.value as { reply_tracking?: boolean }).reply_tracking
  || (data.value?.campaigns || []).some((c) => c.replied > 0));

// The sent table's columns: campaign, date, delivered, opened, clicked,
// bounced, replied (when shown) and actions.
const sentGrid = computed(() => ({
  gridTemplateColumns: `minmax(0, 2.4fr) 80px 80px minmax(0, 1.4fr) 70px 70px${showReplies.value ? ' 70px' : ''} 108px`,
}));

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
const delivered = (c: Item) => c.delivered;

function pct(n: number, of: number): number {
  return of ? Math.min(100, (n / of) * 100) : 0;
}

function rate(n: number, of: number): string {
  return of ? `${((n / of) * 100).toFixed(1)}%` : '–';
}

// result is n as a share of `of` when counts are of people, else the count.
function result(n: number, of: number): string {
  return perPerson.value ? rate(n, of) : String($utils.niceNumber(n));
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

// soFar is a sending or paused campaign's results so far, counted as on
// the Analytics page.
function soFar(c: Item): { key: string; label: string; value: string }[] {
  const of = delivered(c);
  const out = [
    { key: 'opened', label: t('campaigns.overview.openedSoFar'), value: result(c.opened, of) },
    { key: 'clicked', label: t('campaigns.overview.col.clicked'), value: result(c.clicked, of) },
    { key: 'bounced', label: t('campaigns.overview.col.bounced'), value: String($utils.niceNumber(c.bounced)) },
  ];
  if (showReplies.value) out.push({ key: 'replied', label: t('campaigns.overview.col.replied'), value: result(c.replied, of) });
  return out;
}

function draftMeta(c: Item): string {
  const when = ago(c.updatedAt);
  const edited = c.updatedByName ? t('campaigns.overview.editedBy', { when, name: c.updatedByName }) : t('campaigns.overview.edited', { when });
  const who = c.lists.length
    ? t('campaigns.overview.audience', { lists: listNames(c), n: $utils.niceNumber(c.audience) })
    : t('campaigns.overview.noLists');
  return `${edited}, ${who}`;
}

// openResults shows a sending campaign's results on the Analytics page,
// from the day it started; without access to analytics, the campaign.
function openResults(c: Item) {
  if (!$can('campaigns:get_analytics')) {
    edit(c);
    return;
  }
  analytics(c);
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
.camps-ov-live-actions { display: flex; gap: 8px; align-items: center; }
.camps-ov-row-actions { display: inline-flex; gap: 2px; align-items: center; justify-content: flex-end; }
.camps-ov-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 16px;
  margin-top: 4px;

  strong { color: var(--ln-text); font-weight: 600; font-variant-numeric: tabular-nums; }
}
.camps-ov-progress {
  height: 8px;
  border-radius: 99px;
  background: var(--ln-muted);
  overflow: hidden;

  span { display: block; height: 100%; background: var(--ln-success); transition: width var(--ln-dur-slow) var(--ln-ease); }
}
.camps-ov-figures {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
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
  .camps-ov-row-actions > a { color: var(--ln-text); font-size: 13px; font-weight: 500; text-decoration: underline; margin-right: 6px; }
}
.camps-ov-next-name { display: flex; flex-direction: column; min-width: 0; strong { font-weight: 500; } }

.camps-ov-draft {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px 18px;
  color: var(--ln-text);

  &:hover { border-color: var(--ln-border-strong); }
}
.camps-ov-draft-name { color: var(--ln-text); font-weight: 500; text-decoration: none; &:hover { text-decoration: underline; } }
.camps-ov-draft-actions { display: flex; justify-content: space-between; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 6px; }
.camps-ov-continue { font-size: 13px; font-weight: 600; color: var(--ln-text); text-decoration: underline; }

.camps-ov-sent { overflow: hidden; }
.camps-ov-sent-row {
  display: grid;
  gap: 16px;
  align-items: center;
  padding: 14px 18px;
  border-top: 1px solid var(--ln-border);
  color: var(--ln-text);
  text-decoration: none;

  &:not(.is-head) { cursor: pointer; }
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
  @media (max-width: 767px) { grid-template-columns: minmax(0, 1fr) auto auto; }
}
.camps-ov-sent-name {
  display: flex;
  flex-direction: column;
  min-width: 0;

  a { color: var(--ln-text); font-weight: 500; text-decoration: none; &:hover { text-decoration: underline; } }
}
.camps-ov-bounces { color: var(--ln-text); text-decoration: underline; }
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
