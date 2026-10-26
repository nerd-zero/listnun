<template>
  <div class="ln-page dev-log" :class="{ 'has-drawer': !!openId && !isMobile }">
    <page-header :title="$t('developers.log.title')" :count="countLabel">
      <template v-if="isMobile" #before-title>
        <span class="ln-small ln-page-eyebrow">{{ $t('globals.terms.developers') }}</span>
      </template>
    </page-header>

    <div class="dev-log-filters">
      <form class="dev-log-search" role="search" @submit.prevent="onFilter">
        <PvIconField>
          <PvInputIcon class="pi pi-search" />
          <PvInputText
            v-model="filters.query"
            :placeholder="$t('developers.log.search')"
            :aria-label="$t('developers.log.search')"
            data-cy="query"
          />
        </PvIconField>
      </form>
      <PvSelectButton
        v-model="filters.status"
        :options="statusOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        :aria-label="$t('globals.fields.status')"
        class="dev-log-status"
        @change="onFilter"
      >
        <template #option="{ option }">
          {{ option.label }}<span v-if="option.count !== null" class="dev-log-count">{{ option.count }}</span>
        </template>
      </PvSelectButton>
      <PvSelect
        v-if="!isMobile && templates.length"
        v-model="filters.templateId"
        :options="templateOptions"
        option-label="name"
        option-value="id"
        :placeholder="$t('developers.log.allTemplates')"
        show-clear
        class="dev-log-select"
        @change="onFilter"
      />
      <PvDatePicker
        v-if="!isMobile"
        v-model="filters.range"
        selection-mode="range"
        :manual-input="false"
        show-icon
        icon-display="input"
        :placeholder="$t('developers.log.anyTime')"
        show-button-bar
        class="dev-log-date"
        @update:model-value="onRange"
      />
    </div>

    <table-card :error="loadError" @retry="fetchPage">
      <!-- Mobile: one row per message, opening the detail drawer. -->
      <template v-if="isMobile">
        <table-skeleton v-if="loading && !page.results.length" :cols="1" />
        <ul v-else-if="page.results.length" class="dev-log-list ln-enter">
          <li v-for="m in page.results" :key="m.id">
            <button type="button" class="dev-log-item" @click="open(m.id)">
              <span class="dev-log-item-text">
                <span class="dev-log-item-to">{{ m.recipient }}</span>
                <span class="dev-log-item-sub">
                  <span :class="`dev-log-fg is-${tone(m.status)}`">{{ $t(`developers.log.status.${m.status}`) }}</span>
                  <span>{{ m.templateName || '—' }}</span>
                </span>
              </span>
              <time class="ln-mono dev-log-item-time">{{ hhmm(m.createdAt) }}</time>
              <i class="pi pi-angle-right" aria-hidden="true" />
            </button>
          </li>
        </ul>
        <empty-log v-else :filtered="isFiltered" :query="filters.query" @clear="clearFilters" />
      </template>

      <PvDataTable
        v-else
        :value="page.results"
        data-key="id"
        lazy
        paginator
        :rows="perPage"
        :total-records="page.total"
        :first="(currentPage - 1) * perPage"
        :loading="loading && page.results.length > 0"
        row-hover
        :row-class="(m) => (m.id === openId ? 'is-open' : '')"
        @page="(e) => { currentPage = e.page + 1; fetchPage(); }"
        @row-click="(e) => open(e.data.id)"
      >
        <template #empty>
          <table-skeleton v-if="loading" :cols="5" :two-line="false" />
          <empty-log v-else :filtered="isFiltered" :query="filters.query" @clear="clearFilters" />
        </template>
        <template #paginatorstart>
          <span class="ln-small">{{ rangeLabel }}</span>
        </template>

        <PvColumn :header="$t('globals.fields.status')" header-style="width:140px">
          <template #body="{ data }">
            <status-tag :status="data.status" :label="$t(`developers.log.status.${data.status}`)" />
          </template>
        </PvColumn>
        <PvColumn :header="$t('developers.log.recipient')">
          <template #body="{ data }">
            <button type="button" class="dev-log-recipient" @click.stop="open(data.id)">{{ data.recipient }}</button>
          </template>
        </PvColumn>
        <PvColumn :header="$t('developers.log.template')">
          <template #body="{ data }">{{ data.templateName || '—' }}</template>
        </PvColumn>
        <PvColumn v-if="!openId" :header="$t('developers.log.key')">
          <template #body="{ data }"><span class="ln-mono">{{ data.sender }}</span></template>
        </PvColumn>
        <PvColumn v-if="!openId" :header="$t('developers.log.sent')" header-style="width:160px">
          <template #body="{ data }"><span class="ln-date">{{ shortDate(data.createdAt) }} {{ hhmmss(data.createdAt) }}</span></template>
        </PvColumn>
      </PvDataTable>
    </table-card>

    <message-drawer :message-id="openId" @close="close" @resent="onResent" />
  </div>
</template>

<script setup lang="ts">
import {
  computed, defineComponent, h, onMounted, reactive, ref, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import dayjs from 'dayjs';
import { useMainStore } from '../../store';
import { useTime } from '../../composables/useTime';
import { useMedia } from '../../composables/useMedia';
import { getTemplates as templatesApi } from '../../api/generated/endpoints/templates/templates';
import {
  listTxMessages, TX_STATUSES, type TxLogPage, type TxStatus,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import TableCard from '../../components/ui/TableCard.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import StatusTag from '../../components/ui/StatusTag.vue';
import EmptyState from '../../components/ui/EmptyState.vue';
import { statusTone } from '../../components/ui/status';
import type { RequestError } from '../../components/ui/ErrorState.vue';
import MessageDrawer from './MessageDrawer.vue';

interface TemplateOption { id: number; name: string }

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { shortDate } = useTime();
const { refreshTick } = storeToRefs(useMainStore());
const isMobile = useMedia('(max-width: 767px)');

const perPage = 20;
const page = ref<TxLogPage>({
  results: [], counts: {}, total: 0, page: 1, perPage,
});
const currentPage = ref(1);
const loading = ref(true);
const loadError = ref<RequestError | null>(null);
const templates = ref<TemplateOption[]>([]);
const openId = ref<string | null>((route.query.id as string) || null);

const filters = reactive({
  query: (route.query.query as string) || '',
  status: ((route.query.status as string) || '') as TxStatus | '',
  templateId: Number(route.query.template_id) || null as number | null,
  range: null as Date[] | null,
});

const totalAll = computed(() => Object.values(page.value.counts).reduce((a, b) => a + (b || 0), 0));
const countLabel = computed(() => (totalAll.value ? totalAll.value.toLocaleString() : null));
const isFiltered = computed(() => !!(filters.query || filters.status || filters.templateId || filters.range));

const statusOptions = computed(() => [
  { value: '', label: t('developers.log.all'), count: totalAll.value.toLocaleString() },
  ...TX_STATUSES
    .filter((s) => s !== 'queued' || page.value.counts.queued)
    .map((s) => ({
      value: s,
      label: t(`developers.log.status.${s}`),
      count: page.value.counts[s] ? page.value.counts[s]!.toLocaleString() : null,
    })),
]);

const templateOptions = computed(() => templates.value);

const rangeLabel = computed(() => {
  if (!page.value.total) return '';
  const from = (currentPage.value - 1) * perPage + 1;
  const to = Math.min(currentPage.value * perPage, page.value.total);
  return t('developers.log.range', { from, to, total: page.value.total.toLocaleString() });
});

function tone(s: TxStatus) {
  return statusTone(s);
}

function hhmm(stamp: string) {
  return dayjs(stamp).format('HH:mm');
}

function hhmmss(stamp: string) {
  return dayjs(stamp).format('HH:mm:ss');
}

function toRequestError(err: unknown): RequestError {
  const e = err as { response?: { status?: number; data?: { message?: string } }; config?: { method?: string; url?: string } };
  return {
    status: e.response?.status, method: e.config?.method, url: e.config?.url, message: e.response?.data?.message,
  };
}

function fetchPage() {
  loading.value = true;
  const [from, to] = filters.range || [];
  listTxMessages({
    query: filters.query.trim(),
    status: filters.status,
    templateId: filters.templateId || undefined,
    from: from ? dayjs(from).startOf('day').toISOString() : undefined,
    to: to ? dayjs(to).endOf('day').toISOString() : undefined,
    page: currentPage.value,
    perPage,
  })
    .then((p) => { page.value = p; loadError.value = null; })
    .catch((err) => { loadError.value = toRequestError(err); })
    .finally(() => { loading.value = false; });
}

function syncQuery() {
  router.replace({
    query: {
      ...(filters.query ? { query: filters.query } : {}),
      ...(filters.status ? { status: filters.status } : {}),
      ...(filters.templateId ? { template_id: String(filters.templateId) } : {}),
      ...(openId.value ? { id: openId.value } : {}),
    },
  });
}

function onFilter() {
  currentPage.value = 1;
  syncQuery();
  fetchPage();
}

function onRange(v: Date[] | null) {
  // Wait for both ends of the range before filtering.
  if (v && v.length === 2 && (!v[0] || !v[1])) return;
  onFilter();
}

function clearFilters() {
  filters.query = '';
  filters.status = '';
  filters.templateId = null;
  filters.range = null;
  onFilter();
}

function open(id: string) {
  openId.value = id;
  syncQuery();
}

function close() {
  openId.value = null;
  syncQuery();
}

function onResent(id: string) {
  fetchPage();
  if (id) open(id);
}

// EmptyLog: no messages yet, or none matching the filters.
const EmptyLog = defineComponent({
  props: { filtered: Boolean, query: { type: String, default: '' } },
  emits: ['clear'],
  setup(p, { emit }) {
    return () => (p.filtered
      ? h(EmptyState, {
        bare: true,
        icon: 'pi-search',
        title: t('states.noMatches'),
        text: p.query ? t('states.noMatchesText', { query: p.query }) : t('developers.log.noMatchesFilters'),
        actionLabel: t('states.clearFilters'),
        secondary: true,
        onAction: () => emit('clear'),
      })
      : h(EmptyState, {
        bare: true,
        icon: 'pi-inbox',
        title: t('developers.log.emptyTitle'),
        text: t('developers.log.emptyText'),
        actionLabel: router.hasRoute('devSendTest') ? t('developers.log.sendTest') : '',
        onAction: () => router.push({ name: 'devSendTest' }),
      }));
  },
});

onMounted(() => {
  // Completes the "Check the log" quickstart step on the Overview.
  try { localStorage.setItem('ln-dev-log-seen', '1'); } catch { /* storage unavailable */ }
  fetchPage();
  templatesApi().listTemplates({ no_body: true }).then((list) => {
    templates.value = (list as unknown as { id: number; name: string; type: string }[])
      .filter((tpl) => tpl.type === 'tx')
      .map((tpl) => ({ id: tpl.id, name: tpl.name }));
  });
});

watch(refreshTick, fetchPage);
</script>

<style scoped lang="scss">
.dev-log { transition: padding-right var(--ln-dur-base) var(--ln-ease); }
.dev-log.has-drawer { padding-right: 460px; }

.dev-log-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.dev-log-search {
  flex: 1 1 280px;
  max-width: 360px;

  .p-inputtext { width: 100%; }
  @media (max-width: 767px) { max-width: none; }
}
.dev-log-status {
  overflow-x: auto;
  max-width: 100%;
  flex-wrap: nowrap;
}
.dev-log-count {
  margin-left: 6px;
  font-variant-numeric: tabular-nums;
  color: var(--ln-text-3);
}
.dev-log-select { width: 200px; }
.dev-log-date { width: 240px; }

:deep(tr.is-open) {
  background: var(--ln-hover) !important;
  box-shadow: inset 3px 0 0 var(--ln-ink);
}
:deep(.p-datatable-tbody > tr) { cursor: pointer; }

.dev-log-recipient {
  border: 0;
  background: none;
  padding: 0;
  font: inherit;
  color: var(--ln-text);
  cursor: pointer;
  text-align: left;

  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}

.dev-log-list { list-style: none; margin: 0; padding: 0; }
.dev-log-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 0;
  border-top: 1px solid var(--ln-border);
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  text-align: left;
  cursor: pointer;

  li:first-child & { border-top: 0; }
  i { color: var(--ln-text-3); font-size: 12px; }
}
.dev-log-item-text { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.dev-log-item-to { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dev-log-item-sub { display: flex; gap: 8px; font-size: 13px; color: var(--ln-text-2); }
.dev-log-fg {
  &.is-success { color: var(--ln-success-fg); }
  &.is-warn { color: var(--ln-warn-fg); }
  &.is-danger { color: var(--ln-danger-fg); }
}
.dev-log-item-time { font-size: 12px; color: var(--ln-text-3); align-self: flex-start; }
</style>
