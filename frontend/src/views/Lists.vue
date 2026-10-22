<template>
  <div class="ln-page lists">
    <page-header :title="$t('globals.terms.lists')" :count="lists.total ?? null">
      <template #after-title>
        <span v-if="queryParams.status === 'archived'" class="ln-small">/ {{ $t('lists.archived') }}</span>
      </template>
      <template #search>
        <form class="ln-search" role="search" @submit.prevent="onSearch">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="queryParams.query" name="query" :placeholder="$t('lists.search')" :aria-label="$t('lists.search')" data-cy="query" />
          </PvIconField>
        </form>
      </template>
      <PvButton v-if="$can('lists:manage_all')" icon="pi pi-plus" :label="$t('lists.newList')" data-cy="btn-new" @click="showNewForm" />
    </page-header>

    <div class="lists-filters">
      <PvSelectButton
        v-model="typeFilter"
        :options="typeOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        :aria-label="$t('lists.type')"
        @change="onSearch"
      />
      <router-link
        :to="queryParams.status === 'archived' ? { name: 'lists' } : { name: 'lists', query: { status: 'archived' } }"
        class="ln-small lists-archived-link"
      >
        {{ queryParams.status === 'archived' ? $t('lists.viewActive') : $t('lists.viewArchived') }}
      </router-link>
    </div>

    <table-card
      :selected-count="numSelected"
      :selected-label="bulk.all ? $t('campaigns.allMatching', { num: lists.total }) : undefined"
      @clear="clearSelection"
    >
      <template #bulk>
        <a v-if="!bulk.all && (lists.total || 0) > (lists.perPage || 0)" href="#" data-cy="select-all-lists" @click.prevent="bulk.all = true">
          {{ $t('globals.messages.selectAll', { num: lists.total }) }}
        </a>
        <button v-if="$can('lists:manage_all')" type="button" class="ln-bulk-btn" data-cy="btn-delete-lists" @click="askDeleteSelected">
          {{ $t('globals.buttons.delete') }}
        </button>
      </template>

      <PvDataTable
        v-model:selection="bulk.checked"
        :value="rows"
        data-key="id"
        lazy
        paginator
        :rows="lists.perPage"
        :total-records="lists.total"
        :first="(queryParams.page - 1) * (lists.perPage || 20)"
        :loading="loading.listsFull && rows.length > 0"
        @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        @row-select="onTableCheck"
        @row-unselect="onTableCheck"
        @row-select-all="onTableCheck"
        @row-unselect-all="onTableCheck"
      >
        <template #empty>
          <table-skeleton v-if="loading.listsFull" :cols="4" />
          <empty-state
            v-else-if="queryParams.query || typeFilter"
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
            icon="pi-list"
            :title="$t('lists.emptyTitle')"
            :text="$t('lists.emptyText')"
            :action-label="$can('lists:manage_all') ? $t('lists.newList') : ''"
            @action="showNewForm"
          />
        </template>

        <PvColumn v-if="$can('lists:manage_all')" selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="name" :header="$t('globals.terms.list')" sortable header-class="cy-name">
          <template #body="{ data }">
            <a class="ln-row-title" :href="$router.resolve(`/lists/${data.id}`).href" :data-id="data.id" data-cy="list-name" @click.prevent="showEditForm(data)">{{ data.name }}</a>
            <span v-if="data.description" class="ln-row-sub lists-desc">{{ data.description }}</span>
            <span class="ln-attrs lists-tags">
              <status-tag
                v-if="scrubStatus[data.id]?.activeJobRequestId"
                status="pending"
                :label="$t('settings.scrub.validatingProgress', { count: $utils.niceNumber(scrubProgress[data.id] || 0) })"
              />
              <span v-else-if="scrubStatus[data.id]?.lastResult" class="ln-attr ln-attr--filled">
                {{ $t('settings.scrub.lastValidatedCounts', {
                  date: shortDate(scrubStatus[data.id].lastResult.completedAt),
                  valid: $utils.niceNumber(scrubStatus[data.id].lastResult.validCount || 0),
                  invalid: $utils.niceNumber(scrubStatus[data.id].lastResult.invalidCount || 0),
                }) }}
              </span>
              <span v-for="tag in data.tags" :key="tag" class="ln-attr ln-attr--filled">{{ tag }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn field="type" :header="$t('lists.type')" sortable header-class="cy-type" header-style="width:200px">
          <template #body="{ data }">
            <span class="ln-attrs">
              <span class="ln-attr" :data-cy="`type-${data.type}`">{{ $t(`lists.types.${data.type}`) }}</span>
              <span class="ln-attr" :data-cy="`optin-${data.optin}`">{{ $t(`lists.optins.${data.optin}`) }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn field="subscriber_count" :header="$t('globals.terms.subscribers')" sortable header-class="cy-subscribers" header-style="width:260px">
          <template #body="{ data }">
            <component
              :is="$can('subscribers:get_all', 'subscribers:get') ? 'router-link' : 'div'"
              :to="`/subscribers/lists/${data.id}`"
              class="lists-subs"
              data-cy="subscribers"
            >
              <span class="lists-subs-top">
                <strong>{{ (data.subscriberCount || 0).toLocaleString() }}</strong>
                <span v-if="awaiting(data)" class="ln-small">{{ $t('lists.awaiting', { n: awaiting(data).toLocaleString() }) }}</span>
              </span>
              <span class="lists-meter" aria-hidden="true">
                <span class="is-ok" :style="{ width: `${share(data, 'confirmed')}%` }" />
                <span class="is-wait" :style="{ width: `${share(data, 'unconfirmed')}%` }" />
              </span>
            </component>
          </template>
        </PvColumn>

        <PvColumn field="updated_at" :header="$t('globals.fields.updatedAt')" sortable header-class="cy-updated_at" header-style="width:130px">
          <template #body="{ data }"><span class="ln-date">{{ ago(data.updatedAt) }}</span></template>
        </PvColumn>

        <PvColumn header-style="width:120px" body-style="text-align:right">
          <template #body="{ data }">
            <div class="lists-actions">
              <router-link
                v-if="$can('campaigns:manage')"
                v-tooltip.bottom="$t('lists.sendCampaign')"
                :to="`/campaigns/new?list_id=${data.id}`"
                class="ln-row-menu"
                :aria-label="$t('lists.sendCampaign')"
                data-cy="btn-campaign"
              >
                <i class="pi pi-send" />
              </router-link>
              <button
                v-if="$can('lists:manage') || $canList(data.id, 'list:manage')"
                v-tooltip.bottom="$t('globals.buttons.edit')"
                type="button"
                class="ln-row-menu"
                :aria-label="$t('globals.buttons.edit')"
                data-cy="btn-edit"
                @click="showEditForm(data)"
              >
                <i class="pi pi-pencil" />
              </button>
              <button type="button" class="ln-row-menu" :aria-label="$t('globals.buttons.rowActions')" aria-haspopup="true" data-cy="btn-row-menu" @click="(e) => openMenu(e, data)">
                <i class="pi pi-ellipsis-h" />
              </button>
            </div>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
    <PvMenu ref="menu" :model="menuItems" popup />

    <div class="lists-legend ln-small" aria-hidden="true">
      <span><span class="is-ok" />{{ $t('lists.legendConfirmed') }}</span>
      <span><span class="is-wait" />{{ $t('lists.legendAwaiting') }}</span>
    </div>

    <p v-if="settings['app.cache_slow_queries']" class="ln-small">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferrer">{{ $t('globals.buttons.learnMore') }}</a>
    </p>

    <list-form v-if="isFormVisible" :data="curItem || {}" :is-editing="isEditing" @finished="fetchLists" @close="onFormClose" />

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
  computed, onMounted, onUnmounted, reactive, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import { useTime } from '../composables/useTime';
import { makeQueryHandlers } from '../utils';
import PageHeader from '../components/ui/PageHeader.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import ListForm from './ListForm.vue';
import { getLists as listsApi } from '../api/generated/endpoints/lists/lists';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';
import { getSettings as settingsApi } from '../api/generated/endpoints/settings/settings';

// Lists and Scrub statuses are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

// How often to refresh validation progress while a job runs.
const SCRUB_POLL_MS = 4000;

const {
  $utils, $api, $can, $canList,
} = useGlobal();
const {
  listLists, getList, deleteList, deleteLists,
} = listsApi();
const { createCampaign } = campaignsApi();
const { getScrubListStatus, getScrubListProgress, scrubList } = settingsApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { ago, shortDate } = useTime();
const {
  refreshTick, loading, settings, serverConfig,
} = storeToRefs(useMainStore());

const {
  curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
} = useFormDialog<Loose>();
const lists = ref<Loose>({});
const scrubStatus = ref<Record<number, Loose>>({});
const scrubProgress = ref<Record<number, number>>({});
let scrubPollTimer: ReturnType<typeof setTimeout> | null = null;
const bulk = reactive({ checked: [] as Loose[], all: false });
const typeFilter = ref<string>((route.query.type as string) || '');

const queryParams = reactive({
  page: 1,
  query: '',
  orderBy: 'id',
  order: 'asc',
  status: (route.query.status as string) || 'active',
});

const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Loose | null>(null);
const confirm = reactive({
  open: false, title: '', text: '', label: '', typed: '', danger: true, busy: false, run: () => {},
});

const rows = computed<Loose[]>(() => lists.value.results || []);
const numSelected = computed(() => (bulk.all ? (lists.value.total || 0) : bulk.checked.length));
const typeOptions = computed(() => [
  { value: '', label: t('developers.log.all') },
  { value: 'public', label: t('lists.types.public') },
  { value: 'private', label: t('lists.types.private') },
]);

const { onPageChange, onSort } = makeQueryHandlers(queryParams, fetchLists);

// awaiting is the number of double opt-in subscribers yet to confirm.
function awaiting(l: Loose): number {
  if (l.optin !== 'double') return 0;
  return l.subscriberStatuses?.unconfirmed || 0;
}

// share is the percentage of a list's subscribers in a status, for the bar.
function share(l: Loose, status: 'confirmed' | 'unconfirmed'): number {
  const s = l.subscriberStatuses || {};
  const total = (s.confirmed || 0) + (s.unconfirmed || 0) + (s.unsubscribed || 0);
  if (!total) return 0;
  if (l.optin !== 'double') {
    return status === 'confirmed' ? (((s.confirmed || 0) + (s.unconfirmed || 0)) / total) * 100 : 0;
  }
  return ((s[status] || 0) / total) * 100;
}

function clearSelection() {
  bulk.checked = [];
  bulk.all = false;
}

function onTableCheck() {
  if (bulk.checked.length !== (lists.value.total || 0)) bulk.all = false;
}

function refreshMinimalLists() {
  ($api.getLists as (p: unknown) => void)({ minimal: true, per_page: 'all', status: 'active' });
}

// Polls list-level Scrub status and, for lists with a job running, its
// live progress; it reschedules itself and stops once nothing is active.
function fetchScrubStatus() {
  getScrubListStatus().then((data) => {
    const m: Record<number, Loose> = {};
    (Array.isArray(data) ? data : []).forEach((l: Loose) => {
      m[l.id] = { activeJobRequestId: l.activeJobRequestId, lastResult: l.lastResult };
    });
    scrubStatus.value = m;

    const active = Object.entries(m).filter(([, v]) => v.activeJobRequestId);
    active.forEach(([id, v]) => {
      getScrubListProgress(Number(id), v.activeJobRequestId)
        .then((p) => { scrubProgress.value[Number(id)] = (p as unknown as Loose)?.validated || 0; })
        .catch(() => {});
    });

    if (scrubPollTimer) clearTimeout(scrubPollTimer);
    scrubPollTimer = active.length > 0 ? setTimeout(fetchScrubStatus, SCRUB_POLL_MS) : null;
  }).catch(() => {});
}

function fetchLists() {
  listLists({
    page: queryParams.page,
    query: queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
    order_by: queryParams.orderBy,
    order: queryParams.order,
    status: queryParams.status,
    ...(typeFilter.value ? { type: typeFilter.value } : {}),
  } as never).then((resp) => { lists.value = resp as unknown as Loose; });
  if ((serverConfig.value as Loose).scrub_enabled) fetchScrubStatus();
}

function onSearch() {
  queryParams.page = 1;
  fetchLists();
}

function clearFilters() {
  queryParams.query = '';
  typeFilter.value = '';
  onSearch();
}

function onFormClose() {
  closeForm();
  if (route.params.id) router.push({ name: 'lists' });
}

function askScrub(l: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('settings.scrub.scrubList'),
    text: t('settings.scrub.scrubListConfirm', { name: l.name }),
    label: t('settings.scrub.scrubList'),
    typed: '',
    danger: false,
    run: () => {
      confirm.open = false;
      scrubList(l.id).then(() => {
        $utils.toast(t('settings.scrub.scrubJobStarted'));
        fetchScrubStatus();
      });
    },
  });
}

function askOptinCampaign(l: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('lists.sendOptinCampaign'),
    text: t('lists.optinCampaignText', { name: l.name, n: awaiting(l).toLocaleString() }),
    label: t('lists.createOptinCampaign'),
    typed: '',
    danger: false,
    run: () => {
      confirm.busy = true;
      createCampaign({
        name: t('lists.optinTo', { name: l.name }),
        subject: t('lists.confirmSub', { name: l.name }),
        lists: [l.id],
        from_email: (settings.value as Loose)['app.from_email'],
        content_type: 'richtext',
        messenger: 'email',
        type: 'optin',
      } as never).then((d) => {
        confirm.open = false;
        router.push({ name: 'campaign', hash: '#content', params: { id: (d as unknown as Loose).id } });
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askDelete(l: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('lists.deleteTitle', { name: l.name }),
    text: t('lists.deleteText', { n: (l.subscriberCount || 0).toLocaleString() }),
    label: t('lists.deleteList'),
    typed: '',
    danger: true,
    run: () => {
      confirm.busy = true;
      deleteList(l.id).then(() => {
        confirm.open = false;
        fetchLists();
        refreshMinimalLists();
        $utils.toast(t('globals.messages.deleted', { name: l.name }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askDeleteSelected() {
  const n = numSelected.value;
  const name = t('globals.terms.list', n);
  Object.assign(confirm, {
    open: true,
    title: t('lists.deleteManyTitle', { n: n.toLocaleString() }),
    text: t('lists.deleteManyText'),
    label: t('subscribers.deleteN', { n: n.toLocaleString() }),
    typed: bulk.all ? String(n) : '',
    danger: true,
    run: () => {
      const params: Loose = {};
      if (!bulk.all && bulk.checked.length > 0) {
        params.id = bulk.checked.map((l) => l.id);
      } else {
        params.query = queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' ');
        params.all = bulk.all;
      }
      confirm.busy = true;
      deleteLists(params as never).then(() => {
        confirm.open = false;
        clearSelection();
        fetchLists();
        refreshMinimalLists();
        $utils.toast(t('globals.messages.deletedCount', { num: n, name }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

const menuItems = computed<MenuItem[]>(() => {
  const l = menuFor.value;
  if (!l) return [];
  const canManageList = $can('lists:manage') || $canList(l.id, 'list:manage');
  const items: MenuItem[] = [];
  if ($can('subscribers:get_all', 'subscribers:get')) {
    items.push({ label: t('lists.viewSubscribers'), icon: 'pi pi-users', command: () => router.push(`/subscribers/lists/${l.id}`) });
  }
  if ($can('subscribers:import')) {
    items.push({ label: t('import.title'), icon: 'pi pi-upload', command: () => router.push({ name: 'import', query: { list_id: l.id } }) });
  }
  if (l.optin === 'double' && $can('campaigns:manage')) {
    items.push({ label: t('lists.sendOptinCampaign'), icon: 'pi pi-envelope', command: () => askOptinCampaign(l) });
  }
  if ((serverConfig.value as Loose).scrub_enabled && $can('settings:manage')) {
    items.push({
      label: t('settings.scrub.scrubList'),
      icon: 'pi pi-shield',
      disabled: !!scrubStatus.value[l.id]?.activeJobRequestId,
      command: () => askScrub(l),
    });
  }
  items.push({ label: t('lists.signupForm'), icon: 'pi pi-globe', command: () => router.push({ name: 'forms' }) });
  if (canManageList) {
    items.push({ separator: true }, {
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(l),
    });
  }
  return items;
});

function openMenu(e: Event, l: Loose) {
  menuFor.value = l;
  menu.value?.toggle(e);
}

watch(refreshTick, fetchLists);

onMounted(() => {
  fetchLists();
  if (route.params.id) {
    getList(parseInt(route.params.id as string, 10)).then((data) => { showEditForm(data as unknown as Loose); });
  }
});

onUnmounted(() => {
  if (scrubPollTimer) clearTimeout(scrubPollTimer);
});
</script>

<style scoped lang="scss">
.lists-filters { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.lists-archived-link { color: var(--ln-text-2); }
.lists-desc { max-width: 420px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.lists-tags { margin-top: 4px; &:empty { display: none; } }

.lists-subs {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--ln-text);
  text-decoration: none;

  &:hover { text-decoration: none; strong { text-decoration: underline; } }
}
.lists-subs-top { display: flex; justify-content: space-between; align-items: baseline; gap: 8px; strong { font-variant-numeric: tabular-nums; } }
.lists-meter {
  display: flex;
  height: 4px;
  border-radius: 99px;
  overflow: hidden;
  background: var(--ln-muted);

  .is-ok { background: var(--ln-success); }
  .is-wait { background: var(--ln-warn); }
}
.lists-actions { display: inline-flex; gap: 2px; justify-content: flex-end; }
.lists-legend {
  display: flex;
  gap: 16px;

  > span { display: inline-flex; align-items: center; gap: 6px; }
  .is-ok, .is-wait { width: 10px; height: 10px; border-radius: 2px; }
  .is-ok { background: var(--ln-success); }
  .is-wait { background: var(--ln-warn); }
}
</style>
