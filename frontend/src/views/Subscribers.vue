<template>
  <div class="subs-page">
    <!-- Page header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">
          {{ $t('globals.terms.subscribers') }}
          <span v-if="!isNaN(subscribers.total)" class="page-title-count">{{ subscribers.total }}</span>
          <template v-if="currentList">
            <span class="page-title-sub">/ {{ currentList.name }}</span>
            <span v-if="queryParams.subStatus" class="page-title-sub is-capitalized">({{ queryParams.subStatus }})</span>
          </template>
        </h1>
      </div>
      <PvButton
        v-if="$can('subscribers:manage')"
        severity="primary"
        icon="pi pi-plus"
        :label="$t('globals.buttons.new')"
        data-cy="btn-new"
        @click="showNewForm"
      />
    </div>

    <!-- Table card -->
    <div class="table-card">
      <!-- Search toolbar (outside table header for cleaner layout) -->
      <div class="search-toolbar">
        <form class="search-form" @submit.prevent="onSubmit">
          <PvIconField class="search-field">
            <PvInputIcon class="pi pi-search" />
            <PvInputText
              @input="onSimpleQueryInput"
              v-model="queryInput"
              :placeholder="$t('subscribers.queryPlaceholder')"
              ref="query"
              :disabled="isSearchAdvanced"
              data-cy="search"
              class="search-input"
            />
          </PvIconField>
        </form>

        <a href="#" class="advanced-toggle" @click.prevent="toggleAdvancedSearch" data-cy="btn-advanced-search">
          <i :class="['pi', isSearchAdvanced ? 'pi-times' : 'pi-sliders-h']" />
          {{ isSearchAdvanced ? $t('subscribers.reset') : $t('subscribers.advancedQuery') }}
        </a>

        <button type="button" class="toolbar-btn" @click.prevent="exportSubscribers" data-cy="btn-export-subscribers">
          <i class="pi pi-download" /> {{ $t('subscribers.export') }}
        </button>
      </div>

      <!-- Advanced query panel -->
      <div v-if="isSearchAdvanced" class="advanced-panel">
        <form @submit.prevent="onSubmit">
          <PvTextarea
            v-model="queryParams.queryExp"
            @keydown="onAdvancedQueryEnter"
            rows="3"
            placeholder="subscribers.name LIKE '%user%' or subscribers.status='blocklisted'"
            class="w-full"
            data-cy="query"
          />
          <div class="advanced-footer">
            <span class="advanced-help">
              {{ $t('subscribers.advancedQueryHelp') }}.
              <a href="https://listmonk.app/docs/querying-and-segmentation" target="_blank" rel="noopener noreferrer">
                {{ $t('globals.buttons.learnMore') }}
              </a>
            </span>
            <PvButton type="submit" severity="primary" size="small" icon="pi pi-search" :label="$t('subscribers.query')" data-cy="btn-query" />
          </div>
        </form>
      </div>

      <PvDataTable
        :value="subscribers.results ?? []"
        :loading="loading.subscribers"
        data-key="id"
        :rows="subscribers.perPage"
        :paginator="true"
        paginator-position="bottom"
        :total-records="subscribers.total"
        :lazy="true"
        @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        selection-mode="checkbox"
        v-model:selection="bulk.checked"
        @row-select="onTableCheck"
        @row-unselect="onTableCheck"
        @row-select-all="onTableCheck"
        @row-unselect-all="onTableCheck"
      >
        <template #header>
          <div v-if="bulk.checked.length > 0" class="bulk-bar">
            <span class="bulk-count">
              {{ $t('globals.messages.numSelected', { num: numSelectedSubscribers }) }}
              <template v-if="!bulk.all && subscribers.total > subscribers.perPage">
                &mdash;
                <a href="#" @click.prevent="selectAllSubscribers">
                  {{ $t('globals.messages.selectAll', { num: subscribers.total }) }}
                </a>
              </template>
            </span>
            <button type="button" class="bulk-btn" @click.prevent="showBulkListForm" data-cy="btn-manage-lists">
              <i class="pi pi-list" /> Manage lists
            </button>
            <button type="button" class="bulk-btn bulk-btn--danger" @click.prevent="onDeleteSubscribers" data-cy="btn-delete-subscribers">
              <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
            </button>
            <button type="button" class="bulk-btn bulk-btn--warn" @click.prevent="onBlocklistSubscribers" data-cy="btn-manage-blocklist">
              <i class="pi pi-user-minus" /> Blocklist
            </button>
          </div>
        </template>

        <PvColumn selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="email" :header="$t('subscribers.email')" header-class="cy-email" sortable>
          <template #body="{ data }">
            <div class="email-cell">
              <a class="row-name" :class="{ 'row-name--blocked': data.status === 'blocklisted' }"
                :href="`/subscribers/${data.id}`" @click.prevent="showEditForm(data)">
                {{ data.email }}
                <copy-text :text="`${data.email}`" hide-text />
              </a>
              <PvTag v-if="data.status !== 'enabled'" severity="danger" size="small" data-cy="blocklisted"
                :value="$t(`subscribers.status.${data.status}`)" />
              <PvTag v-if="data.scrubStatus === 'risky' || data.scrubStatus === 'unchecked_error'"
                severity="warn" size="small" data-cy="scrub-status"
                :value="$t(`subscribers.scrubStatus.${data.scrubStatus}`)" />
            </div>
            <div v-if="data.lists?.length" class="list-tags">
              <router-link v-for="l in data.lists" :key="l.id" :to="`/subscribers/lists/${l.id}`">
                <PvTag severity="secondary" size="small">
                  {{ l.name }}<sup v-if="l.optin === 'double' || l.subscriptionStatus === 'unsubscribed'"> {{ $t(`subscribers.status.${l.subscriptionStatus}`) }}</sup>
                </PvTag>
              </router-link>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="name" :header="$t('globals.fields.name')" header-class="cy-name" sortable style="width:18%">
          <template #body="{ data }">
            <a class="row-name" :class="{ 'row-name--blocked': data.status === 'blocklisted' }"
              :href="`/subscribers/${data.id}`" @click.prevent="showEditForm(data)">
              {{ data.name }}
            </a>
          </template>
        </PvColumn>

        <PvColumn field="lists" :header="$t('globals.terms.lists')" header-class="cy-lists" style="width:6rem; text-align:center">
          <template #body="{ data }">
            <span class="list-count">{{ listCount(data.lists) }}</span>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('globals.fields.createdAt')" header-class="cy-created_at" sortable style="width:11%">
          <template #body="{ data }">
            <span class="date-cell">{{ $utils.niceDate(data.createdAt) }}</span>
          </template>
        </PvColumn>

        <PvColumn field="updated_at" :header="$t('globals.fields.updatedAt')" header-class="cy-updated_at" sortable style="width:11%">
          <template #body="{ data }">
            <span class="date-cell">{{ $utils.niceDate(data.updatedAt) }}</span>
          </template>
        </PvColumn>

        <PvColumn style="width:7rem; text-align:right">
          <template #body="{ data }">
            <div class="row-actions">
              <a :href="`/api/subscribers/${data.id}/export`" class="row-action-btn"
                data-cy="btn-download" v-tooltip.bottom="$t('subscribers.downloadData')">
                <i class="pi pi-download" />
              </a>
              <button v-if="$can('subscribers:manage')" type="button" class="row-action-btn"
                data-cy="btn-edit" v-tooltip.bottom="$t('globals.buttons.edit')" @click="showEditForm(data)">
                <i class="pi pi-pencil" />
              </button>
              <button v-if="$can('subscribers:manage')" type="button" class="row-action-btn row-action-btn--danger"
                data-cy="btn-delete" v-tooltip.bottom="$t('globals.buttons.delete')" @click="onDeleteSubscriber(data)">
                <i class="pi pi-trash" />
              </button>
            </div>
          </template>
        </PvColumn>

        <template #empty v-if="!loading.subscribers">
          <empty-placeholder />
        </template>
      </PvDataTable>
    </div>

    <!-- Manage list modal -->
    <PvDialog v-model:visible="isBulkListFormVisible" :style="{ width: '500px' }" show-header="false" :closable="false" modal>
      <subscriber-bulk-list :num-subscribers="numSelectedSubscribers" @finished="bulkChangeLists" @close="isBulkListFormVisible = false" />
    </PvDialog>

    <!-- Add / edit form modal -->
    <PvDialog v-model:visible="isFormVisible" :style="{ width: '850px' }" class="subscriber-modal" show-header="false" :closable="false" modal @hide="onFormClose">
      <subscriber-form :data="curItem" :is-editing="isEditing" @finished="querySubscribers" @close="isFormVisible = false" />
    </PvDialog>
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import { uris } from '../constants';
import SubscriberBulkList from './SubscriberBulkList.vue';
import SubscriberForm from './SubscriberForm.vue';
import CopyText from '../components/CopyText.vue';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

const { $utils } = useGlobal();
const {
  listSubscribers, getSubscriber, deleteSubscriber, deleteSubscribers,
  blocklistSubscribers, blocklistSubscribersByQuery, deleteSubscribersByQuery,
  manageSubscriberLists, manageSubscriberListsByQuery,
} = subscribersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const {
  refreshTick, subscribers, lists, loading,
} = storeToRefs(store);

const curItem = ref<any>(null);
const isSearchAdvanced = ref(false);
const isEditing = ref(false);
const isFormVisible = ref(false);
const isBulkListFormVisible = ref(false);
const queryEl = ref<any>(null);
const bulk = reactive({ checked: [] as any[], all: false });
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

const numSelectedSubscribers = computed(() => (bulk.all ? (subscribers.value as any).total : bulk.checked.length));

const currentList = computed(() => {
  if (!queryParams.listID || !(lists.value as any).results) return null;
  return (lists.value as any).results.find((l: any) => l.id === queryParams.listID);
});

function listCount(ls: any[]) {
  return ls.reduce((n: number, item: any) => n + (item.subscriptionStatus !== 'unsubscribed' ? 1 : 0), 0);
}

function toggleAdvancedSearch() {
  isSearchAdvanced.value = !isSearchAdvanced.value;
  queryParams.search = '';
  if (!isSearchAdvanced.value) {
    queryInput.value = '';
    queryParams.queryExp = '';
    queryParams.page = 1;
    querySubscribers();
    nextTick(() => { queryEl.value?.$el?.focus(); });
    return;
  }
  const q = queryInput.value.replace(/'/, "''").trim();
  if (q) {
    if ($utils.validateEmail(q)) {
      queryParams.queryExp = `email = '${q.toLowerCase()}'`;
    } else {
      queryParams.queryExp = `(name ~* '${q}' OR email ~* '${q.toLowerCase()}')`;
    }
  }
}

function selectAllSubscribers() { bulk.all = true; }

function onTableCheck() {
  if (bulk.checked.length !== (subscribers.value as any).total) bulk.all = false;
}

function showEditForm(sub: any) {
  curItem.value = sub;
  isFormVisible.value = true;
  isEditing.value = true;
}

function showNewForm() {
  curItem.value = {};
  isFormVisible.value = true;
  isEditing.value = false;
}

function showBulkListForm() { isBulkListFormVisible.value = true; }

function onFormClose() {
  if (route.params.id) router.push({ name: 'subscribers' });
}

function onPageChange(p: number) { querySubscribers({ page: p }); }
function onSort(field: string, direction: string) { querySubscribers({ orderBy: field, order: direction }); }

function onSimpleQueryInput(v: any) {
  const q = (v.target ? v.target.value : v).replace(/'/, "''").trim();
  queryParams.queryExp = '';
  queryParams.page = 1;
  queryParams.search = q.toLowerCase();
}

function onAdvancedQueryEnter(e: KeyboardEvent) {
  if (e.ctrlKey && e.key === 'Enter') onSubmit();
}

function onSubmit() { querySubscribers({ page: 1 }); }

function querySubscribers(params?: any) {
  if (params) Object.assign(queryParams, params);
  const qp: any = {
    list_id: queryParams.listID,
    search: queryParams.search,
    query: queryParams.queryExp,
    page: queryParams.page,
    subscription_status: queryParams.subStatus,
    order_by: queryParams.orderBy,
    order: queryParams.order,
  };
  if (queryParams.queryExp) {
    delete qp.search;
  } else {
    delete qp.query;
  }
  nextTick(() => {
    store.setLoading({ model: 'subscribers', status: true });
    listSubscribers(qp).then((data: any) => {
      store.setModelResponse({ model: 'subscribers', data });
      bulk.checked = [];
    }).finally(() => {
      store.setLoading({ model: 'subscribers', status: false });
    });
  });
}

function onDeleteSubscriber(sub: any) {
  $utils.confirm(null, () => {
    deleteSubscriber(sub.id).then(() => {
      querySubscribers();
      $utils.toast(t('globals.messages.deleted', { name: sub.name }));
    });
  });
}

function onBlocklistSubscribers() {
  let fn: () => void;
  if (!bulk.all && bulk.checked.length > 0) {
    fn = () => {
      const ids = bulk.checked.map((s: any) => s.id);
      blocklistSubscribers({ ids }).then(() => querySubscribers());
    };
  } else {
    fn = () => {
      blocklistSubscribersByQuery({
        search: queryParams.search,
        query: queryParams.queryExp,
        list_ids: queryParams.listID ? [queryParams.listID] : (null as any),
        subscription_status: queryParams.subStatus,
      }).then(() => querySubscribers());
    };
  }
  $utils.confirm(t('subscribers.confirmBlocklist', { num: numSelectedSubscribers.value }), fn);
}

function exportSubscribers() {
  const num = !bulk.all && bulk.checked.length > 0 ? bulk.checked.length : (subscribers.value as any).total;
  $utils.confirm(t('subscribers.confirmExport', { num }), () => {
    const q = new URLSearchParams();
    if (queryParams.search) q.append('search', queryParams.search);
    else if (queryParams.queryExp) q.append('query', queryParams.queryExp);
    if (queryParams.listID) q.append('list_id', String(queryParams.listID));
    if (queryParams.subStatus) q.append('subscription_status', queryParams.subStatus);
    if (!bulk.all && bulk.checked.length > 0) {
      bulk.checked.forEach((s: any) => q.append('id', s.id));
    }
    document.location.href = `${uris.exportSubscribers}?${q.toString()}`;
  });
}

function onDeleteSubscribers() {
  let fn: () => void;
  if (!bulk.all && bulk.checked.length > 0) {
    fn = () => {
      const ids = bulk.checked.map((s: any) => s.id);
      deleteSubscribers({ id: ids }).then(() => {
        querySubscribers();
        $utils.toast(t('subscribers.subscribersDeleted', { num: numSelectedSubscribers.value }));
      });
    };
  } else {
    fn = () => {
      deleteSubscribersByQuery({
        all: queryParams.queryExp.trim() === '' && queryParams.search.trim() === '',
        search: queryParams.search,
        query: queryParams.queryExp,
        list_ids: queryParams.listID ? [queryParams.listID] : (null as any),
        subscription_status: queryParams.subStatus,
      }).then(() => {
        querySubscribers();
        $utils.toast(t('subscribers.subscribersDeleted', { num: numSelectedSubscribers.value }));
      });
    };
  }
  $utils.confirm(t('subscribers.confirmDelete', { num: numSelectedSubscribers.value }), fn);
}

function bulkChangeLists(action: string, preconfirm: boolean, listItems: any[]) {
  const data: any = {
    action,
    search: queryParams.search,
    list_ids: queryParams.listID ? [queryParams.listID] : null,
    target_list_ids: listItems.map((l: any) => l.id),
  };
  if (preconfirm) data.status = 'confirmed';
  let fn: any;
  if (!bulk.all && bulk.checked.length > 0) {
    fn = manageSubscriberLists;
    data.ids = bulk.checked.map((s: any) => s.id);
  } else {
    fn = manageSubscriberListsByQuery;
    data.query = queryParams.queryExp;
    data.subscription_status = queryParams.subStatus;
  }
  fn(data).then(() => {
    querySubscribers();
    $utils.toast(t('subscribers.listChangeApplied'));
  });
}

watch(() => refreshTick.value, () => { querySubscribers(); });

onMounted(() => {
  if (route.params.listID) queryParams.listID = parseInt(route.params.listID as string, 10);
  if (route.query.subscription_status) queryParams.subStatus = route.query.subscription_status as string;
  if (route.params.id) {
    getSubscriber(parseInt(route.params.id as string, 10)).then((data: any) => { showEditForm(data); });
  } else {
    querySubscribers();
  }
});
</script>

<style scoped lang="scss">
.subs-page { display: flex; flex-direction: column; gap: 1.5rem; }

// Page header

.page-header-left { display: flex; flex-direction: column; gap: 0.25rem; }

.page-title-sub { color: var(--lm-text-subtle); font-weight: 400; font-size: 1rem; }

// Table card

// Search toolbar
.search-toolbar {
  display: flex; align-items: center; gap: 0.75rem; padding: 1rem 1rem 0;
  flex-wrap: wrap; border-bottom: 1px solid var(--lm-bg-subtle);
  padding-bottom: 1rem;
}
.search-form { flex: 1; min-width: 220px; max-width: 400px; }
:deep(.search-input) { width: 100%; }
.advanced-toggle {
  font-size: 0.8rem; color: var(--lm-text-muted); text-decoration: none; display: inline-flex; align-items: center; gap: 0.3rem;
}
.toolbar-btn {
  display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.4rem 0.75rem;
  border: 1px solid var(--lm-border); border-radius: 7px; background: var(--lm-surface); color: var(--lm-text-muted);
  font-size: 0.8rem; cursor: pointer; white-space: nowrap;
}

// Advanced panel
.advanced-panel {
  padding: 1rem; background: var(--lm-bg); border-bottom: 1px solid var(--lm-border);
}
.advanced-footer { display: flex; align-items: center; justify-content: space-between; margin-top: 0.5rem; }
.advanced-help { font-size: 0.78rem; color: var(--lm-text-subtle); a { color: var(--lm-text-muted); } }

// Bulk bar
.bulk-bar {
  display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap;
}
.bulk-count { font-size: 0.85rem; color: var(--lm-text-muted); a { color: var(--lm-primary); } }
.bulk-btn {
  display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.3rem 0.65rem;
  border-radius: 6px; font-size: 0.8rem; font-weight: 500; border: 1px solid; cursor: pointer; background: var(--lm-surface);
  color: var(--lm-text-muted); border-color: var(--lm-border);
  &--danger { color: var(--lm-danger); border-color: var(--lm-danger-border); &:hover { background: var(--lm-danger-bg); } }
  &--warn { color: var(--lm-warn); border-color: var(--lm-warn-border); &:hover { background: var(--lm-warn-bg); } }
}

// Make secondary PvTags visible
:deep(.p-tag-secondary) {
  background: var(--lm-bg-subtle);
  color: var(--lm-text-secondary);
  border: 1px solid var(--lm-border);
}

.toolbar-btn:hover {
  background: var(--lm-bg-subtle);
  border-color: var(--lm-text-muted);
  color: var(--lm-text);
}

// Row cells
.email-cell { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.row-name { font-weight: 500; color: var(--lm-primary); text-decoration: none; &:hover { text-decoration: underline; } }
.row-name--blocked { color: var(--lm-danger); text-decoration: line-through; }
.list-tags { display: flex; flex-wrap: wrap; gap: 0.25rem; margin-top: 0.3rem; a { text-decoration: none; } }
.list-count { font-weight: 600; color: var(--lm-text); }
.date-cell { font-size: 0.82rem; color: var(--lm-text-muted); }

// Row actions
:deep(tr:has(.row-name--blocked)) { opacity: 0.6; }
</style>
