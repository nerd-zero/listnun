<template>
  <div class="ln-page camps">
    <page-header :title="$t('globals.terms.campaigns')" :count="campaigns.total ?? null">
      <template #search>
        <form class="ln-search" role="search" @submit.prevent="onSearch">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText
              v-model="queryParams.query"
              name="query"
              :placeholder="$t('campaigns.searchPlaceholder')"
              :aria-label="$t('campaigns.searchPlaceholder')"
            />
          </PvIconField>
        </form>
      </template>
      <PvButton
        v-if="$can('campaigns:manage')"
        icon="pi pi-plus"
        :label="$t('menu.newCampaign')"
        data-cy="btn-new"
        @click="$router.push({ name: 'campaign', params: { id: 'new' } })"
      />
    </page-header>

    <PvSelectButton
      v-model="statusFilter"
      :options="statusOptions"
      option-label="label"
      option-value="value"
      :allow-empty="false"
      :aria-label="$t('globals.fields.status')"
      class="camps-status"
      @change="onFilter"
    >
      <template #option="{ option }">
        {{ option.label }}<span v-if="option.count !== null" class="camps-count">{{ option.count }}</span>
      </template>
    </PvSelectButton>

    <table-card
      :selected-count="numSelected"
      :selected-label="bulk.all ? $t('campaigns.allMatching', { num: campaigns.total }) : undefined"
      @clear="clearSelection"
    >
      <template #bulk>
        <a v-if="!bulk.all && (campaigns.total || 0) > (campaigns.perPage || 0)" href="#" data-cy="select-all-campaigns" @click.prevent="bulk.all = true">
          {{ $t('globals.messages.selectAll', { num: campaigns.total }) }}
        </a>
        <button v-if="$can('campaigns:manage')" type="button" class="ln-bulk-btn" data-cy="btn-delete-campaigns" @click="askDeleteSelected">
          {{ $t('globals.buttons.delete') }}
        </button>
      </template>

      <!-- Mobile: one card per campaign -->
      <template v-if="isMobile">
        <table-skeleton v-if="loading.campaigns && !rows.length" :cols="1" />
        <ul v-else-if="rows.length" class="camps-list">
          <li v-for="c in rows" :key="c.id">
            <router-link :to="{ name: 'campaign', params: { id: c.id } }" class="camps-item">
              <span class="camps-item-top">
                <span class="ln-row-title">{{ c.name }}</span>
                <status-tag :status="c.status" :label="$t(`campaigns.status.${c.status}`)" />
              </span>
              <span class="ln-row-sub">{{ mobileLine(c) }}</span>
            </router-link>
          </li>
        </ul>
        <empty-campaigns v-else />
      </template>

      <PvDataTable
        v-else
        v-model:selection="bulk.checked"
        :value="rows"
        data-key="id"
        lazy
        paginator
        :rows="campaigns.perPage"
        :total-records="campaigns.total"
        :first="(queryParams.page - 1) * (campaigns.perPage || 20)"
        :loading="loading.campaigns && rows.length > 0"
        :row-class="(c) => (c.status === 'running' ? 'is-running' : '')"
        @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
      >
        <template #empty>
          <table-skeleton v-if="loading.campaigns" :cols="4" />
          <empty-campaigns v-else />
        </template>
        <template #paginatorstart>
          <span class="ln-small">{{ rangeLabel }}</span>
        </template>

        <PvColumn v-if="$can('campaigns:manage')" selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="name" :header="$t('globals.terms.campaign')" sortable header-class="cy-name">
          <template #body="{ data }">
            <div class="camps-name">
              <router-link :to="{ name: 'campaign', params: { id: data.id } }" class="ln-row-title" data-cy="campaign-name" :data-id="data.id">{{ data.name }}</router-link>
              <span v-if="data.type === 'optin'" class="ln-attr">{{ $t('lists.optin') }}</span>
            </div>
            <span class="ln-row-sub">{{ listNames(data) }}</span>
          </template>
        </PvColumn>

        <PvColumn field="status" :header="$t('globals.fields.status')" sortable header-class="cy-status" header-style="width:130px">
          <template #body="{ data }">
            <status-tag
              v-tooltip.bottom="data.pauseReason ? $t(`campaigns.pauseReason.${data.pauseReason}`) : null"
              :status="data.status"
              :label="$t(`campaigns.status.${data.status}`)"
            />
          </template>
        </PvColumn>

        <PvColumn :header="$t('campaigns.performance')">
          <template #body="{ data }">
            <template v-if="data.status === 'running'">
              <PvProgressBar :value="progress(data)" :show-value="false" class="camps-progress" />
              <span class="ln-small camps-perf">
                {{ $t('campaigns.sentOf', { sent: stats(data).sent.toLocaleString(), total: stats(data).toSend.toLocaleString() }) }}
                <template v-if="stats(data).rate"> · {{ Math.round(stats(data).rate) }}/{{ $t('campaigns.rateMinuteShort') }}</template>
              </span>
            </template>
            <span v-else-if="data.status === 'scheduled'" class="ln-small">{{ $t('campaigns.goesTo', { n: (data.toSend || 0).toLocaleString() }) }}</span>
            <span v-else-if="data.status === 'draft'" class="ln-small ln-muted">{{ data.sendAt ? $t('campaigns.willSchedule') : $t('campaigns.notScheduled') }}</span>
            <span v-else class="camps-metrics">
              <span>{{ $t('campaigns.sent') }} <strong>{{ (data.sent || 0).toLocaleString() }}</strong></span>
              <span>{{ $t('dashboard.views') }} <strong>{{ rate(data.views, data.sent) }}</strong></span>
              <span>{{ $t('dashboard.clicks') }} <strong>{{ rate(data.clicks, data.sent) }}</strong></span>
              <router-link v-if="data.bounces && $can('bounces:get')" :to="{ name: 'bounces', query: { campaign_id: data.id } }" class="camps-bounces">
                {{ $t('globals.terms.bounces') }} <strong>{{ data.bounces.toLocaleString() }}</strong>
              </router-link>
            </span>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('campaigns.date')" sortable header-class="cy-timestamp" header-style="width:170px">
          <template #body="{ data }">
            <span class="ln-date" :title="$utils.niceDate(data.createdAt, true)">{{ dateLine(data) }}</span>
          </template>
        </PvColumn>

        <PvColumn header-style="width:120px" body-style="text-align:right">
          <template #body="{ data }">
            <div class="camps-actions">
              <button
                v-if="primary(data)"
                v-tooltip.bottom="primary(data)!.label"
                type="button"
                class="ln-row-menu"
                :aria-label="primary(data)!.label"
                :data-cy="primary(data)!.cy"
                @click="primary(data)!.run()"
              >
                <i :class="['pi', primary(data)!.icon]" />
              </button>
              <button
                v-if="$can('campaigns:manage')"
                v-tooltip.bottom="$t('globals.buttons.clone')"
                type="button"
                class="ln-row-menu"
                :aria-label="$t('globals.buttons.clone')"
                data-cy="btn-clone"
                @click="askClone(data)"
              >
                <i class="pi pi-copy" />
              </button>
              <button
                type="button"
                class="ln-row-menu"
                :aria-label="$t('globals.buttons.rowActions')"
                aria-haspopup="true"
                data-cy="btn-row-menu"
                @click="(e) => openMenu(e, data)"
              >
                <i class="pi pi-ellipsis-h" />
              </button>
            </div>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
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
  computed, defineComponent, h, onMounted, onUnmounted, reactive, ref, watch,
} from 'vue';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import { useMedia } from '../composables/useMedia';
import { makeQueryHandlers } from '../utils';
import CampaignPreview from '../components/CampaignPreview.vue';
import PageHeader from '../components/ui/PageHeader.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import DialogForm from '../components/ui/DialogForm.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';
import { getDashboard } from '../api/generated/endpoints/dashboard/dashboard';

// Campaign rows are the API's camelCased campaign objects.
type Campaign = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface PopupMenu { toggle: (e: Event) => void }
interface RowAction { label: string; icon: string; cy: string; run: () => void }
interface CampaignPage { results?: Campaign[]; total?: number; perPage?: number }

const STATUSES = ['running', 'scheduled', 'draft', 'paused', 'finished', 'cancelled'];

const { $utils, $can } = useGlobal();
const {
  listCampaigns, getCampaign, createCampaign, deleteCampaign, deleteCampaigns,
  updateCampaignStatus, getRunningCampaignStats,
} = campaignsApi();
const { getDashboardCounts } = getDashboard();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { shortDate, ago } = useTime();
const isMobile = useMedia('(max-width: 767px)');
const store = useMainStore();
const { refreshTick, loading } = storeToRefs(store);
const campaigns = computed<CampaignPage>({
  get: () => (store.campaigns as CampaignPage) || {},
  set: (v) => { store.campaigns = v; },
});

const previewItem = ref<Campaign | null>(null);
const pollID = ref<number | undefined>(undefined);
const runningStats = ref<Record<number, Campaign>>({});
const statusCounts = ref<Record<string, number>>({});
const statusFilter = ref<string>((route.query.status as string) || '');
const bulk = reactive({ checked: [] as Campaign[], all: false });
const queryParams = reactive({
  page: 1, query: '', orderBy: 'created_at', order: 'desc',
});

const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Campaign | null>(null);
const clone = reactive({
  open: false, busy: false, name: '', source: null as Campaign | null,
});
const confirm = reactive({
  open: false, title: '', text: '', label: '', typed: '', danger: true, busy: false, run: () => {},
});

const rows = computed<Campaign[]>(() => campaigns.value.results || []);
const numSelected = computed(() => (bulk.all ? (campaigns.value.total || 0) : bulk.checked.length));

const statusOptions = computed(() => {
  const total = Object.values(statusCounts.value).reduce((a, b) => a + b, 0);
  return [
    { value: '', label: t('developers.log.all'), count: total ? total.toLocaleString() : null },
    ...STATUSES
      .filter((s) => s !== 'cancelled' || statusCounts.value.cancelled)
      .map((s) => ({
        value: s,
        label: t(`campaigns.status.${s}`),
        count: statusCounts.value[s] ? statusCounts.value[s].toLocaleString() : null,
      })),
  ];
});

const rangeLabel = computed(() => {
  const total = campaigns.value.total || 0;
  const per = campaigns.value.perPage || 20;
  if (!total) return '';
  const from = (queryParams.page - 1) * per + 1;
  return t('developers.log.range', { from, to: Math.min(queryParams.page * per, total), total: total.toLocaleString() });
});

const stats = (c: Campaign) => (c.id in runningStats.value ? runningStats.value[c.id] : c);
const progress = (c: Campaign) => {
  const s = stats(c);
  return s.toSend ? Math.min(100, (s.sent / s.toSend) * 100) : 0;
};

function rate(n: number, of: number): string {
  if (!of || !n) return '—';
  return `${((n / of) * 100).toFixed(1)}%`;
}

function listNames(c: Campaign): string {
  return (c.lists || []).map((l: { name: string }) => l.name).join(', ');
}

function dateLine(c: Campaign): string {
  const s = stats(c);
  switch (c.status) {
    case 'running': {
      const started = dayjs(s.startedAt);
      return started.isSame(dayjs(), 'day')
        ? t('campaigns.startedToday', { time: started.format('HH:mm') })
        : t('campaigns.startedOn', { date: shortDate(s.startedAt) });
    }
    case 'scheduled': return c.sendAt ? shortDate(c.sendAt, true) : '—';
    case 'draft': return t('dashboard.lineDraft', { when: ago(c.updatedAt) });
    case 'paused': return t('campaigns.pausedOn', { date: shortDate(c.updatedAt) });
    default: return shortDate(s.updatedAt || c.updatedAt);
  }
}

function mobileLine(c: Campaign): string {
  const s = stats(c);
  switch (c.status) {
    case 'running': return t('campaigns.mobileRunning', { pct: Math.round(progress(c)), sent: s.sent.toLocaleString(), total: s.toSend.toLocaleString() });
    case 'scheduled': return c.sendAt ? shortDate(c.sendAt, true) : '';
    case 'draft': return t('dashboard.lineDraft', { when: ago(c.updatedAt) });
    default: return `${dateLine(c)} · ${t('dashboard.views').toLowerCase()} ${rate(c.views, c.sent)} · ${t('dashboard.clicks').toLowerCase()} ${rate(c.clicks, c.sent)}`;
  }
}

function fetchCampaigns() {
  listCampaigns({
    page: queryParams.page,
    query: queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
    order_by: queryParams.orderBy,
    order: queryParams.order,
    no_body: true,
    ...(statusFilter.value ? { status: [statusFilter.value] } : {}),
  } as never).then((resp) => { campaigns.value = resp as unknown as CampaignPage; });
}

function fetchCounts() {
  getDashboardCounts().then((data) => {
    statusCounts.value = (data as unknown as { campaigns?: { byStatus?: Record<string, number> } }).campaigns?.byStatus || {};
  });
}

const { onPageChange, onSort } = makeQueryHandlers(queryParams, fetchCampaigns);

function onSearch() {
  queryParams.page = 1;
  fetchCampaigns();
}

function onFilter() {
  queryParams.page = 1;
  router.replace({ query: statusFilter.value ? { status: statusFilter.value } : {} });
  fetchCampaigns();
}

function clearSelection() {
  bulk.checked = [];
  bulk.all = false;
}

function pollStats() {
  window.clearInterval(pollID.value);
  pollID.value = window.setInterval(() => {
    getRunningCampaignStats().then((data) => {
      const list = (data as unknown as Campaign[]) || [];
      if (list.length === 0) {
        window.clearInterval(pollID.value);
        if (Object.keys(runningStats.value).length > 0) {
          fetchCampaigns();
          fetchCounts();
          runningStats.value = {};
        }
      } else {
        runningStats.value = list.reduce((obj: Record<number, Campaign>, cur: Campaign) => ({ ...obj, [cur.id]: cur }), {});
      }
    }, () => { window.clearInterval(pollID.value); });
  }, 1000);
}

function setStatus(c: Campaign, status: string) {
  confirm.busy = true;
  updateCampaignStatus(c.id, { status } as never)
    .then(() => {
      confirm.open = false;
      $utils.toast(t('campaigns.statusChanged', { name: c.name, status: t(`campaigns.status.${status}`) }));
      fetchCampaigns();
      fetchCounts();
      pollStats();
    })
    .finally(() => { confirm.busy = false; });
}

// askStatus confirms a status change, naming the campaign and saying what
// happens.
function askStatus(c: Campaign, status: string) {
  Object.assign(confirm, {
    open: true,
    title: t(`campaigns.confirm.${status}.title`, { name: c.name }),
    text: t(`campaigns.confirm.${status}.text`, { n: (c.toSend || 0).toLocaleString() }),
    label: t(`campaigns.confirm.${status}.action`),
    typed: '',
    danger: status === 'cancelled',
    run: () => setStatus(c, status),
  });
}

function askDelete(c: Campaign) {
  Object.assign(confirm, {
    open: true,
    title: t('campaigns.confirm.delete.title', { name: c.name }),
    text: t('campaigns.confirm.delete.text'),
    label: t('globals.buttons.delete'),
    typed: '',
    danger: true,
    run: () => {
      confirm.busy = true;
      deleteCampaign(c.id)
        .then(() => {
          confirm.open = false;
          fetchCampaigns();
          fetchCounts();
          $utils.toast(t('globals.messages.deleted', { name: c.name }));
        })
        .finally(() => { confirm.busy = false; });
    },
  });
}

function askDeleteSelected() {
  const n = numSelected.value;
  const name = t('globals.terms.campaign', n);
  Object.assign(confirm, {
    open: true,
    title: t('campaigns.confirm.deleteMany.title', { n: n.toLocaleString() }),
    text: t('campaigns.confirm.deleteMany.text'),
    label: t('campaigns.confirm.deleteMany.action', { n: n.toLocaleString() }),
    typed: bulk.all ? String(n) : '',
    danger: true,
    run: () => {
      const params: Record<string, unknown> = {};
      if (!bulk.all && bulk.checked.length > 0) {
        params.id = bulk.checked.map((c: Campaign) => c.id);
      } else {
        params.query = queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' ');
        params.all = bulk.all;
      }
      confirm.busy = true;
      deleteCampaigns(params as never)
        .then(() => {
          confirm.open = false;
          clearSelection();
          fetchCampaigns();
          fetchCounts();
          $utils.toast(t('globals.messages.deletedCount', { num: n, name }));
        })
        .finally(() => { confirm.busy = false; });
    },
  });
}

// primary is the one action a row shows directly, by status.
function primary(c: Campaign): RowAction | null {
  const send = $can('campaigns:send');
  if (c.status === 'running' && send) {
    return {
      label: t('campaigns.pause'), icon: 'pi-pause', cy: 'btn-pause', run: () => askStatus(c, 'paused'),
    };
  }
  if (c.status === 'paused' && send) {
    return {
      label: t('campaigns.send'), icon: 'pi-play', cy: 'btn-resume', run: () => askStatus(c, 'running'),
    };
  }
  if ((c.status === 'finished' || c.status === 'cancelled') && $can('campaigns:get_analytics')) {
    return {
      label: t('globals.terms.analytics'),
      icon: 'pi-chart-bar',
      cy: 'btn-analytics',
      run: () => router.push({ name: 'campaignAnalytics', query: { id: c.id } }),
    };
  }
  return {
    label: t('campaigns.preview'), icon: 'pi-eye', cy: 'btn-preview', run: () => { previewItem.value = c; },
  };
}

const menuItems = computed<MenuItem[]>(() => {
  const c = menuFor.value;
  if (!c) return [];
  const send = $can('campaigns:send');
  const items: MenuItem[] = [
    { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => router.push({ name: 'campaign', params: { id: c.id } }) },
    { label: t('campaigns.preview'), icon: 'pi pi-eye', command: () => { previewItem.value = c; } },
  ];
  if (send && c.status === 'draft' && !c.sendAt) {
    items.push({ label: t('campaigns.start'), icon: 'pi pi-send', command: () => askStatus(c, 'running') });
  }
  if (send && c.status === 'draft' && c.sendAt) {
    items.push({ label: t('campaigns.schedule'), icon: 'pi pi-clock', command: () => askStatus(c, 'scheduled') });
  }
  if (send && c.status === 'scheduled') {
    items.push({ label: t('campaigns.unschedule'), icon: 'pi pi-clock', command: () => askStatus(c, 'draft') });
  }
  if ($can('campaigns:get_analytics') && c.status !== 'draft') {
    items.push({
      label: t('globals.terms.analytics'),
      icon: 'pi pi-chart-bar',
      command: () => router.push({ name: 'campaignAnalytics', query: { id: c.id } }),
    });
  }
  if (send && (c.status === 'running' || c.status === 'paused')) {
    items.push({ separator: true }, {
      label: t('globals.buttons.cancel'), icon: 'pi pi-times-circle', class: 'is-danger', command: () => askStatus(c, 'cancelled'),
    });
  }
  if ($can('campaigns:manage')) {
    items.push({ separator: true }, {
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(c),
    });
  }
  return items;
});

function openMenu(e: Event, c: Campaign) {
  menuFor.value = c;
  menu.value?.toggle(e);
}

function askClone(c: Campaign) {
  clone.source = c;
  clone.name = t('campaigns.copyOf', { name: c.name });
  clone.open = true;
}

async function doClone() {
  const c = clone.source;
  if (!c) return;
  clone.busy = true;
  try {
    const full = await getCampaign(c.id) as unknown as Campaign;
    const now = dayjs();
    const sendLater = !!c.sendAt;
    let sendAt = null;
    if (sendLater) sendAt = dayjs(c.sendAt).isAfter(now) ? c.sendAt : now.add(7, 'day');
    const name = clone.name.trim();
    const data: Record<string, unknown> = {
      name,
      subject: c.subject,
      lists: (c.lists || []).map((l: { id: number }) => l.id),
      type: c.type,
      from_email: c.fromEmail,
      content_type: c.contentType,
      messenger: c.messenger,
      tags: c.tags,
      template_id: c.templateId,
      body: full.body,
      body_source: full.bodySource,
      altbody: c.altbody,
      headers: c.headers,
      send_later: sendLater,
      send_at: sendAt,
      archive: c.archive,
      archive_template_id: c.archiveTemplateId,
      archive_meta: c.archiveMeta,
      media: (c.media || []).map((m: { id: number }) => m.id),
    };
    if (c.archive) data.archive_slug = `${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
    const created = await createCampaign(data as never) as unknown as Campaign;
    clone.open = false;
    router.push({ name: 'campaign', params: { id: created.id } });
  } finally {
    clone.busy = false;
  }
}

// EmptyCampaigns: none yet, or none in the current filter.
const EmptyCampaigns = defineComponent({
  setup() {
    return () => {
      if (statusFilter.value || queryParams.query) {
        return h(EmptyState, {
          bare: true,
          icon: 'pi-search',
          title: t('states.noMatches'),
          text: queryParams.query ? t('states.noMatchesText', { query: queryParams.query }) : t('developers.log.noMatchesFilters'),
          actionLabel: t('states.clearFilters'),
          secondary: true,
          onAction: () => {
            queryParams.query = '';
            statusFilter.value = '';
            onFilter();
          },
        });
      }
      return h(EmptyState, {
        bare: true,
        icon: 'pi-megaphone',
        title: t('dashboard.noCampaigns'),
        text: t('dashboard.noCampaignsText'),
        actionLabel: $can('campaigns:manage') ? t('menu.newCampaign') : '',
        onAction: () => router.push({ name: 'campaign', params: { id: 'new' } }),
      });
    };
  },
});

watch(refreshTick, () => { fetchCampaigns(); fetchCounts(); });
onMounted(() => {
  fetchCampaigns();
  fetchCounts();
  pollStats();
});
onUnmounted(() => window.clearInterval(pollID.value));
</script>

<style scoped lang="scss">
.camps-status { overflow-x: auto; max-width: 100%; align-self: flex-start; }
.camps-count { margin-left: 6px; color: var(--ln-text-3); font-variant-numeric: tabular-nums; }

.camps-name { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.camps-progress { width: 100%; max-width: 280px; margin-bottom: 6px; }
.camps-perf { font-variant-numeric: tabular-nums; }
.camps-metrics {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  font-size: 13px;
  color: var(--ln-text-3);

  strong { color: var(--ln-text); font-weight: 500; font-variant-numeric: tabular-nums; }
}
.camps-bounces { color: var(--ln-text-3); text-decoration: none; &:hover { text-decoration: underline; } }
.camps-actions { display: inline-flex; gap: 2px; justify-content: flex-end; }

.camps-list { list-style: none; margin: 0; padding: 0; }
.camps-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 16px;
  border-top: 1px solid var(--ln-border);
  color: var(--ln-text);
  text-decoration: none;

  li:first-child & { border-top: 0; }
  &:hover { background: var(--ln-hover); text-decoration: none; }
}
.camps-item-top { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
</style>
