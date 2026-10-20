<template>
  <div class="ln-page subs" :class="{ 'has-drawer': !!openSub && !isMobile }">
    <page-header :title="$t('globals.terms.subscribers')" :count="subscribers.total ?? null">
      <template #after-title>
        <span v-if="currentList" class="ln-small">
          / {{ currentList.name }}<template v-if="queryParams.subStatus"> ({{ $t(`subscribers.status.${queryParams.subStatus}`) }})</template>
        </span>
      </template>
      <PvButton
        severity="secondary"
        outlined
        icon="pi pi-download"
        :label="$t('subscribers.export')"
        data-cy="btn-export-subscribers"
        @click="askExport"
      />
      <PvButton
        v-if="$can('subscribers:manage')"
        icon="pi pi-plus"
        :label="$t('subscribers.newSubscriber')"
        data-cy="btn-new"
        @click="showNewForm"
      />
    </page-header>

    <div class="subs-filters">
      <form class="subs-search" role="search" @submit.prevent="onSubmit">
        <PvIconField>
          <PvInputIcon class="pi pi-search" />
          <PvInputText
            v-model="queryInput"
            :placeholder="$t('subscribers.searchPlaceholder')"
            :aria-label="$t('subscribers.searchPlaceholder')"
            :disabled="isSearchAdvanced"
            data-cy="search"
            @input="onSimpleQueryInput"
          />
        </PvIconField>
      </form>
      <PvSelect
        :model-value="queryParams.listID"
        :options="listOptions"
        option-label="name"
        option-value="id"
        :placeholder="$t('subscribers.allLists')"
        show-clear
        filter
        class="subs-list"
        @update:model-value="onListPick"
      />
      <PvButton
        :label="isSearchAdvanced ? $t('subscribers.reset') : $t('subscribers.advancedQuery')"
        :icon="isSearchAdvanced ? 'pi pi-times' : 'pi pi-filter'"
        severity="secondary"
        :outlined="!isSearchAdvanced"
        data-cy="btn-advanced-search"
        @click="toggleAdvancedSearch"
      />
    </div>

    <form v-if="isSearchAdvanced" class="ln-card ln-card-pad subs-advanced" @submit.prevent="onSubmit">
      <PvTextarea
        v-model="queryParams.queryExp"
        rows="2"
        class="ln-mono"
        placeholder="subscribers.attribs->>'city' = 'Harare' AND subscribers.created_at > NOW() - INTERVAL '90 days'"
        data-cy="query"
        :aria-label="$t('subscribers.advancedQuery')"
        @keydown="onAdvancedQueryEnter"
      />
      <div class="subs-advanced-foot">
        <span class="ln-small">
          {{ advancedHint }}.
          <a href="https://listmonk.app/docs/querying-and-segmentation" target="_blank" rel="noopener noreferrer">{{ $t('globals.buttons.learnMore') }}</a>
        </span>
        <PvButton type="submit" size="small" icon="pi pi-search" :label="$t('subscribers.query')" data-cy="btn-query" />
      </div>
    </form>

    <table-card
      :selected-count="numSelected"
      :selected-label="bulk.all ? $t('campaigns.allMatching', { num: (subscribers.total || 0).toLocaleString() }) : undefined"
      @clear="clearSelection"
    >
      <template #bulk>
        <a v-if="!bulk.all && (subscribers.total || 0) > (subscribers.perPage || 0)" href="#" @click.prevent="bulk.all = true">
          {{ $t('subscribers.selectAllMatching', { num: (subscribers.total || 0).toLocaleString() }) }}
        </a>
        <template v-if="$can('subscribers:manage')">
          <button type="button" class="ln-bulk-btn" data-cy="btn-manage-lists" @click="isBulkListFormVisible = true">{{ $t('subscribers.manageLists') }}</button>
          <button type="button" class="ln-bulk-btn" data-cy="btn-manage-blocklist" @click="askBlocklistSelected">{{ $t('subscribers.blocklist') }}</button>
          <button type="button" class="ln-bulk-btn" data-cy="btn-delete-subscribers" @click="askDeleteSelected">{{ $t('globals.buttons.delete') }}</button>
        </template>
      </template>

      <PvDataTable
        v-model:selection="bulk.checked"
        :value="rows"
        data-key="id"
        lazy
        paginator
        :rows="subscribers.perPage"
        :total-records="subscribers.total"
        :first="(queryParams.page - 1) * (subscribers.perPage || 20)"
        :loading="loading.subscribers && rows.length > 0"
        :row-class="(s) => (openSub && s.id === openSub.id ? 'is-open' : '')"
        @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        @row-click="(e) => openDrawer(e.data)"
        @row-select="onTableCheck"
        @row-unselect="onTableCheck"
        @row-select-all="onTableCheck"
        @row-unselect-all="onTableCheck"
      >
        <template #empty>
          <table-skeleton v-if="loading.subscribers" :cols="4" />
          <empty-state
            v-else-if="isFiltered"
            bare
            icon="pi-search"
            :title="$t('states.noMatches')"
            :text="queryInput ? $t('states.noMatchesText', { query: queryInput }) : $t('developers.log.noMatchesFilters')"
            :action-label="$t('states.clearFilters')"
            secondary
            @action="clearFilters"
          />
          <empty-state
            v-else
            bare
            icon="pi-users"
            :title="$t('subscribers.emptyTitle')"
            :text="$t('subscribers.emptyText')"
            :action-label="$can('subscribers:import') ? $t('subscribers.importAction') : ''"
            @action="$router.push({ name: 'import' })"
          />
        </template>
        <template #paginatorstart>
          <span class="ln-small">{{ rangeLabel }}</span>
        </template>

        <PvColumn selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="email" :header="$t('globals.terms.subscriber')" sortable header-class="cy-email" header-style="width:42%">
          <template #body="{ data }">
            <a
              :href="$router.resolve(`/subscribers/${data.id}`).href"
              class="ln-row-title"
              :class="{ 'subs-blocked': data.status === 'blocklisted' }"
              data-cy="email"
              :data-id="data.id"
              @click.prevent.stop="openDrawer(data)"
            >{{ data.email }}</a>
            <span class="ln-row-sub" data-cy="name">{{ data.name }}</span>
            <status-tag
              v-if="data.scrubStatus === 'risky' || data.scrubStatus === 'unchecked_error'"
              class="subs-scrub"
              status="pending"
              :label="$t(`subscribers.scrubStatus.${data.scrubStatus}`)"
              data-cy="scrub-status"
            />
          </template>
        </PvColumn>

        <PvColumn field="status" :header="$t('globals.fields.status')" header-style="width:134px">
          <template #body="{ data }">
            <span class="subs-status" :class="`is-${data.status}`" :data-cy="data.status === 'blocklisted' ? 'blocklisted' : undefined">
              <span class="ln-dot" />{{ $t(`subscribers.status.${data.status}`) }}
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('globals.terms.lists')" header-class="cy-lists">
          <template #body="{ data }">
            <span class="ln-attrs" data-cy="lists">
              <span
                v-for="l in activeLists(data).slice(0, 2)"
                :key="l.id"
                class="ln-attr"
                :title="awaitingOptin(l) ? $t('subscribers.status.unconfirmed') : ''"
              >
                <span v-if="awaitingOptin(l)" class="subs-unconf" aria-hidden="true" />
                {{ l.name }}
                <span v-if="awaitingOptin(l)" class="subs-sr">({{ $t('subscribers.status.unconfirmed') }})</span>
              </span>
              <span v-if="activeLists(data).length > 2" class="ln-attr ln-attr--filled">+{{ activeLists(data).length - 2 }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn v-if="!openSub" field="created_at" :header="$t('globals.fields.createdAt')" sortable header-class="cy-created_at" header-style="width:134px">
          <template #body="{ data }"><span class="ln-date">{{ shortDate(data.createdAt) }}</span></template>
        </PvColumn>

        <PvColumn v-if="!openSub" header-style="width:56px" body-style="text-align:right">
          <template #body="{ data }">
            <button type="button" class="ln-row-menu" data-cy="row-menu" :aria-label="$t('globals.buttons.rowActions')" aria-haspopup="true" @click.stop="(e) => openMenu(e, data)">
              <i class="pi pi-ellipsis-h" />
            </button>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
    <PvMenu ref="menu" :model="menuItems" popup />

    <subscriber-drawer
      :subscriber="openSub"
      @close="closeDrawer"
      @edit="(s) => showEditForm(s)"
      @blocklist="(s) => askBlocklistOne(s)"
    />

    <!-- Manage lists -->
    <PvDialog v-model:visible="isBulkListFormVisible" :style="{ width: '500px' }" :show-header="false" :closable="false" modal>
      <subscriber-bulk-list :num-subscribers="numSelected" @finished="bulkChangeLists" @close="isBulkListFormVisible = false" />
    </PvDialog>

    <!-- Add / edit -->
    <PvDialog v-model:visible="isFormVisible" :style="{ width: '760px', maxWidth: 'calc(100vw - 32px)' }" class="subscriber-modal" :show-header="false" :closable="false" modal @hide="onFormClose">
      <subscriber-form :data="curItem" :is-editing="isEditing" @finished="onFormFinished" @close="closeForm" />
    </PvDialog>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :confirm-text="confirm.typed"
      :danger="confirm.danger"
      :busy="confirm.busy"
      @confirm="confirm.run"
    >
      <template v-if="confirm.secondary" #footer-start>
        <PvButton :label="confirm.secondary.label" text severity="secondary" @click="confirm.secondary.run" />
      </template>
    </confirm-action>
  </div>
</template>

<script setup lang="ts">
import {
  computed, nextTick, onMounted, reactive, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import { useTime } from '../composables/useTime';
import { useMedia } from '../composables/useMedia';
import { makeQueryHandlers } from '../utils';
import { uris } from '../constants';
import PageHeader from '../components/ui/PageHeader.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import SubscriberBulkList from './SubscriberBulkList.vue';
import SubscriberForm from './SubscriberForm.vue';
import SubscriberDrawer from './SubscriberDrawer.vue';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

// Subscribers and lists are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

const { $utils, $can } = useGlobal();
const {
  listSubscribers, getSubscriber, deleteSubscriber, deleteSubscribers,
  blocklistSubscribers, blocklistSubscribersByQuery, deleteSubscribersByQuery,
  manageSubscriberLists, manageSubscriberListsByQuery,
} = subscribersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { refreshTick, loading } = storeToRefs(store);
const subscribers = computed(() => (store.subscribers as Loose) || {});
const lists = computed(() => (store.lists as Loose) || {});
const { shortDate } = useTime();
const isMobile = useMedia('(max-width: 991px)');

const {
  curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
} = useFormDialog<Loose>();
const isSearchAdvanced = ref(false);
const isBulkListFormVisible = ref(false);
const openSub = ref<Loose | null>(null);
const bulk = reactive({ checked: [] as Loose[], all: false });
const queryInput = ref('');
const queryParams = reactive({
  queryExp: '',
  search: '',
  listID: null as number | null,
  page: 1,
  orderBy: 'id',
  order: 'desc',
  subStatus: null as string | null,
});

const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Loose | null>(null);
const confirm = reactive({
  open: false,
  title: '',
  text: '',
  label: '',
  typed: '',
  danger: true,
  busy: false,
  run: () => {},
  secondary: null as { label: string; run: () => void } | null,
});

const rows = computed<Loose[]>(() => subscribers.value.results || []);
const numSelected = computed(() => (bulk.all ? (subscribers.value.total || 0) : bulk.checked.length));
const listOptions = computed(() => lists.value.results || []);
const currentList = computed(() => (queryParams.listID ? listOptions.value.find((l: Loose) => l.id === queryParams.listID) : null));
const isFiltered = computed(() => !!(queryParams.search || queryParams.queryExp || queryParams.listID || queryParams.subStatus));

const advancedHint = computed(() => {
  const help = t('subscribers.advancedQueryHelp');
  if (!queryParams.queryExp || subscribers.value.total === undefined) return help;
  return `${help} · ${t('subscribers.matchCount', { n: (subscribers.value.total || 0).toLocaleString() })}`;
});

const rangeLabel = computed(() => {
  const total = subscribers.value.total || 0;
  const per = subscribers.value.perPage || 20;
  if (!total) return '';
  const from = (queryParams.page - 1) * per + 1;
  return t('developers.log.range', { from, to: Math.min(queryParams.page * per, total), total: total.toLocaleString() });
});

function activeLists(s: Loose): Loose[] {
  return (s.lists || []).filter((l: Loose) => l.subscriptionStatus !== 'unsubscribed');
}

// Single opt-in subscriptions are stored as unconfirmed too, so only a
// double opt-in list's unconfirmed subscription is still waiting.
function awaitingOptin(l: Loose): boolean {
  return l.optin === 'double' && l.subscriptionStatus === 'unconfirmed';
}

function clearSelection() {
  bulk.checked = [];
  bulk.all = false;
}

function onTableCheck() {
  if (bulk.checked.length !== (subscribers.value.total || 0)) bulk.all = false;
}

function toggleAdvancedSearch() {
  isSearchAdvanced.value = !isSearchAdvanced.value;
  queryParams.search = '';
  if (!isSearchAdvanced.value) {
    queryInput.value = '';
    queryParams.queryExp = '';
    queryParams.page = 1;
    querySubscribers();
    return;
  }
  const q = queryInput.value.replace(/'/, "''").trim();
  if (q) {
    queryParams.queryExp = $utils.validateEmail(q)
      ? `email = '${q.toLowerCase()}'`
      : `(name ~* '${q}' OR email ~* '${q.toLowerCase()}')`;
  }
}

function onSimpleQueryInput(v: Event | string) {
  const raw = typeof v === 'string' ? v : (v.target as HTMLInputElement).value;
  queryParams.queryExp = '';
  queryParams.page = 1;
  queryParams.search = raw.replace(/'/, "''").trim().toLowerCase();
}

function onAdvancedQueryEnter(e: KeyboardEvent) {
  if (e.ctrlKey && e.key === 'Enter') onSubmit();
}

function onSubmit() { querySubscribers({ page: 1 }); }

function onListPick(id: number | null) {
  if (id) router.push({ name: 'subscribers_list', params: { listID: id } });
  else router.push({ name: 'subscribers' });
}

function clearFilters() {
  queryInput.value = '';
  queryParams.search = '';
  queryParams.queryExp = '';
  isSearchAdvanced.value = false;
  if (queryParams.listID || queryParams.subStatus) {
    router.push({ name: 'subscribers' });
    return;
  }
  querySubscribers({ page: 1 });
}

function querySubscribers(params?: Partial<typeof queryParams>) {
  if (params) Object.assign(queryParams, params);
  const qp: Record<string, unknown> = {
    list_id: queryParams.listID,
    search: queryParams.search,
    query: queryParams.queryExp,
    page: queryParams.page,
    subscription_status: queryParams.subStatus,
    order_by: queryParams.orderBy,
    order: queryParams.order,
  };
  if (queryParams.queryExp) delete qp.search;
  else delete qp.query;

  nextTick(() => {
    store.setLoading({ model: 'subscribers', status: true });
    listSubscribers(qp as never).then((data) => {
      store.setModelResponse({ model: 'subscribers', data });
      clearSelection();
      if (openSub.value) {
        const fresh = ((data as Loose).results || []).find((s: Loose) => s.id === openSub.value!.id);
        if (fresh) openSub.value = fresh;
      }
    }).finally(() => {
      store.setLoading({ model: 'subscribers', status: false });
    });
  });
}

const { onPageChange, onSort } = makeQueryHandlers(queryParams, () => querySubscribers());

function openDrawer(s: Loose) {
  openSub.value = s;
  window.history.replaceState(window.history.state, '', `/subscribers/${s.id}`);
}

function closeDrawer() {
  openSub.value = null;
  if (route.params.id) router.replace({ name: 'subscribers' });
  else window.history.replaceState(window.history.state, '', route.fullPath);
}

function onFormClose() {
  // Deep links to /subscribers/:id open the drawer, not the form.
}

function onFormFinished() {
  querySubscribers();
}

// query-wide bulk actions use the current filters.
function byQuery(): Loose {
  return {
    search: queryParams.search,
    query: queryParams.queryExp,
    list_ids: queryParams.listID ? [queryParams.listID] : null,
    subscription_status: queryParams.subStatus,
  };
}

function askDeleteOne(s: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.confirmDeleteOneTitle', { name: s.email }),
    text: t('subscribers.confirmDeleteOneText'),
    label: t('globals.buttons.delete'),
    typed: '',
    danger: true,
    secondary: null,
    run: () => {
      confirm.busy = true;
      deleteSubscriber(s.id).then(() => {
        confirm.open = false;
        if (openSub.value?.id === s.id) closeDrawer();
        querySubscribers();
        $utils.toast(t('globals.messages.deleted', { name: s.email }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askBlocklistOne(s: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.confirmBlocklistOneTitle', { name: s.email }),
    text: t('subscribers.confirmBlocklistOneText'),
    label: t('subscribers.blocklist'),
    typed: '',
    danger: true,
    secondary: null,
    run: () => {
      confirm.busy = true;
      blocklistSubscribers({ ids: [s.id] } as never).then(() => {
        confirm.open = false;
        querySubscribers();
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function blocklistSelected() {
  confirm.busy = true;
  const req = !bulk.all && bulk.checked.length > 0
    ? blocklistSubscribers({ ids: bulk.checked.map((s) => s.id) } as never)
    : blocklistSubscribersByQuery(byQuery() as never);
  return req.then(() => {
    confirm.open = false;
    querySubscribers();
  }).finally(() => { confirm.busy = false; });
}

function askBlocklistSelected() {
  const n = numSelected.value;
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.confirmBlocklistManyTitle', { n: n.toLocaleString() }),
    text: t('subscribers.confirmBlocklistManyText'),
    label: t('subscribers.blocklist'),
    typed: bulk.all ? String(n) : '',
    danger: true,
    secondary: null,
    run: blocklistSelected,
  });
}

function askDeleteSelected() {
  const n = numSelected.value;
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.confirmDeleteManyTitle', { n: n.toLocaleString() }),
    text: t('subscribers.confirmDeleteManyText'),
    label: t('subscribers.deleteN', { n: n.toLocaleString() }),
    typed: bulk.all ? String(n) : '',
    danger: true,
    secondary: { label: t('subscribers.blocklistInstead'), run: blocklistSelected },
    run: () => {
      confirm.busy = true;
      const req = !bulk.all && bulk.checked.length > 0
        ? deleteSubscribers({ id: bulk.checked.map((s) => s.id) } as never)
        : deleteSubscribersByQuery({ all: !queryParams.queryExp.trim() && !queryParams.search.trim(), ...byQuery() } as never);
      req.then(() => {
        confirm.open = false;
        querySubscribers();
        $utils.toast(t('subscribers.subscribersDeleted', { num: n }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askExport() {
  const n = !bulk.all && bulk.checked.length > 0 ? bulk.checked.length : (subscribers.value.total || 0);
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.exportTitle', { n: n.toLocaleString() }),
    text: t('subscribers.exportText'),
    label: t('subscribers.export'),
    typed: '',
    danger: false,
    secondary: null,
    run: () => {
      confirm.open = false;
      const q = new URLSearchParams();
      if (queryParams.search) q.append('search', queryParams.search);
      else if (queryParams.queryExp) q.append('query', queryParams.queryExp);
      if (queryParams.listID) q.append('list_id', String(queryParams.listID));
      if (queryParams.subStatus) q.append('subscription_status', queryParams.subStatus);
      if (!bulk.all && bulk.checked.length > 0) bulk.checked.forEach((s) => q.append('id', String(s.id)));
      document.location.href = `${uris.exportSubscribers}?${q.toString()}`;
    },
  });
}

function bulkChangeLists(action: string, preconfirm: boolean, listItems: Loose[]) {
  const data: Loose = {
    action,
    search: queryParams.search,
    list_ids: queryParams.listID ? [queryParams.listID] : null,
    target_list_ids: listItems.map((l) => l.id),
  };
  if (preconfirm) data.status = 'confirmed';
  let req;
  if (!bulk.all && bulk.checked.length > 0) {
    data.ids = bulk.checked.map((s) => s.id);
    req = manageSubscriberLists(data as never);
  } else {
    data.query = queryParams.queryExp;
    data.subscription_status = queryParams.subStatus;
    req = manageSubscriberListsByQuery(data as never);
  }
  req.then(() => {
    querySubscribers();
    $utils.toast(t('subscribers.listChangeApplied'));
  });
}

const menuItems = computed<MenuItem[]>(() => {
  const s = menuFor.value;
  if (!s) return [];
  const items: MenuItem[] = [
    { label: t('subscribers.view'), icon: 'pi pi-eye', command: () => openDrawer(s) },
    { label: t('subscribers.downloadData'), icon: 'pi pi-download', command: () => { document.location.href = `/api/subscribers/${s.id}/export`; } },
  ];
  if ($can('subscribers:manage')) {
    items.splice(1, 0, { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => showEditForm(s) });
    if (s.status !== 'blocklisted') {
      items.push({ separator: true }, {
        label: t('subscribers.blocklist'), icon: 'pi pi-ban', class: 'is-danger', command: () => askBlocklistOne(s),
      });
    }
    items.push({
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDeleteOne(s),
    });
  }
  return items;
});

function openMenu(e: Event, s: Loose) {
  menuFor.value = s;
  menu.value?.toggle(e);
}

watch(refreshTick, () => querySubscribers());

onMounted(() => {
  if (route.params.listID) queryParams.listID = parseInt(route.params.listID as string, 10);
  if (route.query.subscription_status) queryParams.subStatus = route.query.subscription_status as string;
  querySubscribers();
  if (route.params.id) {
    getSubscriber(parseInt(route.params.id as string, 10)).then((data) => { openSub.value = data as unknown as Loose; });
  }
});
</script>

<style scoped lang="scss">
// An amber dot marks a subscription that is still waiting for opt-in.
.subs-unconf {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-right: 4px;
  border-radius: 50%;
  background: var(--ln-warn);
  vertical-align: 1px;
}
.subs-sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}
.subs { transition: padding-right .15s; }
.subs.has-drawer { padding-right: 440px; }

.subs-filters { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.subs-search {
  flex: 1 1 280px;
  max-width: 360px;

  .p-inputtext { width: 100%; }
  @media (max-width: 767px) { max-width: none; }
}
.subs-list { width: 200px; }
.subs-advanced { display: flex; flex-direction: column; gap: 10px; }
.subs-advanced-foot { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }

.subs-blocked { color: var(--ln-text-3); text-decoration: line-through; }
.subs-scrub { margin-top: 4px; }
.subs-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--ln-success-fg);

  .ln-dot { background: var(--ln-success); }
  &.is-blocklisted { color: var(--ln-danger-fg); .ln-dot { background: var(--ln-danger); } }
  &.is-disabled { color: var(--ln-text-3); .ln-dot { background: var(--ln-neutral); } }
}
:deep(tr.is-open) { background: var(--ln-hover) !important; box-shadow: inset 3px 0 0 var(--ln-ink); }
:deep(.p-datatable-tbody > tr) { cursor: pointer; }
</style>
