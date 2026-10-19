<template>
  <div class="ln-page bnc">
    <page-header :title="$t('globals.terms.bounces')" :count="counts ? counts.total : null">
      <template #search>
        <form class="ln-search" role="search" @submit.prevent="onSearch">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="queryParams.query" name="query" :placeholder="$t('bounces.search')" :aria-label="$t('bounces.search')" data-cy="query" />
          </PvIconField>
        </form>
      </template>
      <router-link
        v-if="$can('settings:get')"
        :to="{ name: 'settings', params: { section: 'bounces' } }"
        class="p-button p-component p-button-secondary p-button-outlined bnc-settings"
      >
        {{ $t('bounces.settings') }}
      </router-link>
    </page-header>

    <div class="bnc-filters">
      <PvSelectButton
        v-model="queryParams.type"
        :options="typeOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        :aria-label="$t('globals.fields.type')"
        @change="onSearch"
      />
      <span v-if="campaignFilter || queryParams.source" class="ln-attrs">
        <span v-if="campaignFilter" class="ln-attr ln-attr--filled">
          {{ $t('globals.terms.campaign', 1) }}: {{ campaignFilter }}
        </span>
        <span v-if="queryParams.source" class="ln-attr ln-attr--filled">
          {{ $t('bounces.source') }}: {{ queryParams.source }}
        </span>
        <button type="button" class="ln-link-btn" @click="clearFilters">{{ $t('states.clearFilters') }}</button>
      </span>
    </div>

    <table-card
      :selected-count="numSelected"
      :selected-label="bulk.all ? $t('campaigns.allMatching', { num: bounces.total }) : undefined"
      :error="error"
      @clear="clearSelection"
      @retry="getBounces"
    >
      <template #bulk>
        <a v-if="!bulk.all && (bounces.total || 0) > (bounces.perPage || 0)" href="#" @click.prevent="bulk.all = true">
          {{ $t('globals.messages.selectAll', { num: bounces.total }) }}
        </a>
        <template v-if="$can('bounces:manage')">
          <button type="button" class="ln-bulk-btn" data-cy="btn-manage-blocklist" @click="askBlocklist">
            {{ $t('import.blocklist') }}
          </button>
          <button type="button" class="ln-bulk-btn" data-cy="btn-delete" @click="askDeleteSelected">
            {{ $t('bounces.deleteNum', { num: fmt(numSelected) }) }}
          </button>
          <button v-if="!bulk.all && (bounces.total || 0) > bulk.checked.length" type="button" class="ln-bulk-btn" @click="askDeleteAll">
            {{ $t('bounces.deleteAll', { num: fmt(bounces.total) }) }}
          </button>
        </template>
      </template>

      <PvDataTable
        v-model:selection="bulk.checked"
        v-model:expanded-rows="expandedRows"
        :value="rows"
        data-key="id"
        lazy
        paginator
        :rows="bounces.perPage"
        :total-records="bounces.total"
        :first="(queryParams.page - 1) * (bounces.perPage || 20)"
        :loading="loading.bounces && rows.length > 0"
        sort-field="created_at"
        :sort-order="-1"
        @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        @row-select="onTableCheck"
        @row-unselect="onTableCheck"
        @row-select-all="onTableCheck"
        @row-unselect-all="onTableCheck"
      >
        <template #empty>
          <table-skeleton v-if="loading.bounces" :cols="5" />
          <empty-state
            v-else-if="hasFilters"
            bare
            icon="pi-search"
            :title="$t('states.noMatches')"
            :text="queryParams.query ? $t('states.noMatchesText', { query: queryParams.query }) : $t('developers.log.noMatchesFilters')"
            :action-label="$t('states.clearFilters')"
            secondary
            @action="clearFilters"
          />
          <empty-state
            v-else
            bare
            icon="pi-inbox"
            :title="$t('bounces.emptyTitle')"
            :text="$t('bounces.emptyText')"
            :action-label="$can('settings:get') ? $t('bounces.openSettings') : ''"
            secondary
            @action="$router.push({ name: 'settings', params: { section: 'bounces' } })"
          />
        </template>

        <PvColumn v-if="$can('bounces:manage')" selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="email" :header="$t('globals.terms.subscriber', 1)" sortable>
          <template #body="{ data }">
            <span class="bnc-sub">
              <router-link
                class="ln-row-title"
                :class="{ 'is-blocked': data.subscriberStatus === 'blocklisted' }"
                :to="{ name: 'subscriber', params: { id: data.subscriberId } }"
              >
                {{ data.email }}
              </router-link>
              <status-tag
                v-if="data.subscriberStatus && data.subscriberStatus !== 'enabled'"
                :status="data.subscriberStatus"
                :label="$t(`subscribers.status.${data.subscriberStatus}`)"
                data-cy="blocklisted"
              />
            </span>
          </template>
        </PvColumn>

        <PvColumn field="campaign_name" :header="$t('globals.terms.campaign', 1)" sortable>
          <template #body="{ data }">
            <router-link
              v-if="data.campaign"
              class="bnc-link"
              :to="{ name: 'bounces', query: { campaign_id: data.campaign.id } }"
              @click="campaignFilter = data.campaign.name"
            >
              {{ data.campaign.name }}
            </router-link>
            <span v-else class="ln-small">—</span>
          </template>
        </PvColumn>

        <PvColumn field="type" :header="$t('globals.fields.type')" sortable header-style="width:130px">
          <template #body="{ data }">
            <status-tag :status="data.type" :label="$t(`bounces.${data.type}`)" />
          </template>
        </PvColumn>

        <PvColumn field="source" :header="$t('bounces.source')" sortable header-style="width:120px">
          <template #body="{ data }">
            <router-link class="ln-mono bnc-src" :to="{ name: 'bounces', query: { source: data.source } }">{{ data.source }}</router-link>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('bounces.when')" sortable header-style="width:140px">
          <template #body="{ data }">
            <span v-tooltip.bottom="$utils.niceDate(data.createdAt, true)" class="ln-date">{{ shortDate(data.createdAt, true) }}</span>
          </template>
        </PvColumn>

        <PvColumn header-style="width:88px" body-style="text-align:right">
          <template #body="{ data }">
            <div class="bnc-actions">
              <button
                v-tooltip.bottom="$t('bounces.details')"
                type="button"
                class="ln-row-menu"
                :aria-label="$t('bounces.details')"
                :aria-expanded="isExpanded(data)"
                @click="toggleRow(data)"
              >
                <i :class="isExpanded(data) ? 'pi pi-chevron-up' : 'pi pi-chevron-down'" />
              </button>
              <button
                v-if="$can('bounces:manage')"
                v-tooltip.bottom="$t('globals.buttons.delete')"
                type="button"
                class="ln-row-menu"
                :aria-label="$t('globals.buttons.delete')"
                data-cy="btn-delete"
                @click="askDeleteOne(data)"
              >
                <i class="pi pi-trash" />
              </button>
            </div>
          </template>
        </PvColumn>

        <template #expansion="{ data }">
          <pre class="ln-code bnc-meta">{{ prettyMeta(data.meta) }}</pre>
        </template>
      </PvDataTable>
    </table-card>

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
  ref, reactive, computed, watch, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import { makeQueryHandlers } from '../utils';
import PageHeader from '../components/ui/PageHeader.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getBounces as bouncesApi } from '../api/generated/endpoints/bounces/bounces';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

// Bounce rows are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface Counts { total: number; hard: number; soft: number; complaint: number }

// Deleting every bounce asks the operator to type this first.
const DELETE_ALL_WORD = 'delete';

const { $utils } = useGlobal();
const {
  listBounces, getBounceCounts, deleteBounce, deleteBounces, blocklistBouncedSubscribers,
} = bouncesApi();
const { blocklistSubscribers } = subscribersApi();
const { t, n } = useI18n();
const route = useRoute();
const { shortDate } = useTime();
const { refreshTick, loading } = storeToRefs(useMainStore());

const bounces = ref<Loose>({});
const counts = ref<Counts | null>(null);
const error = ref<Loose | null>(null);
const expandedRows = ref<Loose[]>([]);
const campaignFilter = ref('');
const bulk = reactive({ checked: [] as Loose[], all: false });
const queryParams = reactive({
  page: 1,
  orderBy: 'created_at',
  order: 'desc',
  campaignId: 0,
  source: '',
  type: '',
  query: '',
});
const confirm = reactive({
  open: false, title: '', text: '', label: '', typed: '', danger: true, busy: false, run: () => {},
});

const rows = computed<Loose[]>(() => bounces.value.results || []);
const numSelected = computed(() => (bulk.all ? (bounces.value.total || 0) : bulk.checked.length));
const hasFilters = computed(() => !!(queryParams.query || queryParams.type || queryParams.source || queryParams.campaignId));

const typeOptions = computed(() => {
  const c = counts.value;
  const label = (key: string, num?: number) => (c ? `${t(key)} ${fmt(num)}` : t(key));
  return [
    { value: '', label: label('developers.log.all', c?.total) },
    { value: 'hard', label: label('bounces.hard', c?.hard) },
    { value: 'soft', label: label('bounces.soft', c?.soft) },
    { value: 'complaint', label: label('bounces.complaint', c?.complaint) },
  ];
});

function fmt(v: unknown): string {
  return n(Number(v) || 0);
}

function prettyMeta(meta: unknown): string {
  try {
    return JSON.stringify(typeof meta === 'string' ? JSON.parse(meta) : meta, null, 2);
  } catch {
    return String(meta);
  }
}

function filterParams() {
  return {
    campaign_id: queryParams.campaignId || undefined,
    source: queryParams.source || undefined,
    query: queryParams.query.trim() || undefined,
  };
}

function getCounts() {
  getBounceCounts(filterParams()).then((d) => { counts.value = d as unknown as Counts; });
}

function getBounces() {
  bulk.checked = [];
  bulk.all = false;
  error.value = null;
  listBounces({
    ...filterParams(),
    page: queryParams.page,
    order_by: queryParams.orderBy,
    order: queryParams.order,
    type: queryParams.type || undefined,
  }).then((data) => {
    bounces.value = data as unknown as Loose;
    // A campaign link from elsewhere carries only the ID; name it from a row.
    if (queryParams.campaignId && !campaignFilter.value) {
      campaignFilter.value = rows.value[0]?.campaign?.name || `#${queryParams.campaignId}`;
    }
  }).catch((e) => { error.value = e; });
}

const { onPageChange, onSort } = makeQueryHandlers(queryParams, getBounces);

function refresh() {
  getBounces();
  getCounts();
}

function onSearch() {
  queryParams.page = 1;
  refresh();
}

function clearFilters() {
  Object.assign(queryParams, {
    page: 1, query: '', type: '', source: '', campaignId: 0,
  });
  campaignFilter.value = '';
  refresh();
}

function clearSelection() {
  bulk.checked = [];
  bulk.all = false;
}

function onTableCheck() {
  if (bulk.checked.length !== bounces.value.total) bulk.all = false;
}

function isExpanded(b: Loose): boolean {
  return expandedRows.value.some((r) => r.id === b.id);
}

function toggleRow(b: Loose) {
  expandedRows.value = isExpanded(b) ? expandedRows.value.filter((r) => r.id !== b.id) : [...expandedRows.value, b];
}

function ask(opts: Partial<typeof confirm>) {
  Object.assign(confirm, {
    typed: '', danger: true, busy: false, ...opts, open: true,
  });
}

function done(msg: string) {
  confirm.open = false;
  refresh();
  $utils.toast(msg);
}

function askDeleteOne(b: Loose) {
  ask({
    title: t('bounces.deleteOneTitle', { email: b.email }),
    text: t('bounces.deleteText'),
    label: t('globals.buttons.delete'),
    run: () => {
      confirm.busy = true;
      deleteBounce(b.id).then(() => done(t('globals.messages.deleted', { name: b.email })))
        .finally(() => { confirm.busy = false; });
    },
  });
}

function runDelete(all: boolean) {
  const num = all ? (bounces.value.total || 0) : numSelected.value;
  confirm.busy = true;
  const params = all ? { all: true } : { id: bulk.checked.map((b) => b.id) };
  deleteBounces(params as never)
    .then(() => done(t('globals.messages.deletedCount', { name: t('globals.terms.bounces', 2), num })))
    .finally(() => { confirm.busy = false; });
}

function askDeleteSelected() {
  if (bulk.all) {
    askDeleteAll();
    return;
  }
  ask({
    title: t('bounces.deleteManyTitle', { num: fmt(numSelected.value) }),
    text: t('bounces.deleteText'),
    label: t('bounces.deleteNum', { num: fmt(numSelected.value) }),
    run: () => runDelete(false),
  });
}

function askDeleteAll() {
  ask({
    title: t('bounces.deleteAllTitle', { num: fmt(bounces.value.total) }),
    text: t('bounces.deleteAllText'),
    label: t('bounces.deleteAll', { num: fmt(bounces.value.total) }),
    typed: DELETE_ALL_WORD,
    run: () => runDelete(true),
  });
}

function askBlocklist() {
  const { all } = bulk;
  ask({
    title: all ? t('bounces.blocklistAllTitle') : t('bounces.blocklistTitle', { num: fmt(numSelected.value) }),
    text: t('bounces.blocklistText'),
    label: t('import.blocklist'),
    run: () => {
      confirm.busy = true;
      const req = all
        ? blocklistBouncedSubscribers()
        : blocklistSubscribers({ ids: bulk.checked.map((b) => b.subscriberId) } as never);
      req.then(() => done(t('globals.messages.done'))).finally(() => { confirm.busy = false; });
    },
  });
}

watch(refreshTick, refresh);

// Campaign and source links reload the view with new query params.
watch(() => [route.query.campaign_id, route.query.source], ([camp, src]) => {
  queryParams.campaignId = parseInt(String(camp || 0), 10) || 0;
  queryParams.source = String(src || '');
  if (!queryParams.campaignId) campaignFilter.value = '';
  queryParams.page = 1;
  refresh();
});

onMounted(() => {
  if (route.query.campaign_id) queryParams.campaignId = parseInt(route.query.campaign_id as string, 10) || 0;
  if (route.query.source) queryParams.source = route.query.source as string;
  refresh();
});
</script>

<style scoped lang="scss">
.bnc-settings { text-decoration: none; &:hover { text-decoration: none; } }
.bnc-filters {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  :deep(.p-selectbutton) { overflow-x: auto; max-width: 100%; }
}
.bnc-sub { display: inline-flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.ln-row-title.is-blocked { color: var(--ln-text-3); text-decoration: line-through; }
.bnc-link { color: var(--ln-text); text-decoration: none; &:hover { text-decoration: underline; } }
.bnc-src { font-size: 12px; color: var(--ln-text-2); text-decoration: none; &:hover { text-decoration: underline; } }
.bnc-actions { display: inline-flex; gap: 2px; }
.bnc-meta { margin: 4px 0 8px; max-height: 320px; overflow: auto; font-size: 12px; }
</style>
