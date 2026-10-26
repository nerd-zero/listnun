<template>
  <div class="ln-page subs" :class="{ 'has-drawer': !!openSub && !isMobile }">
    <page-header :title="$t('globals.terms.subscribers')" :subtitle="summary">
      <template #after-title>
        <span v-if="currentList" class="ln-small">
          / {{ currentList.name }}<template v-if="queryParams.subStatus"> ({{ $t(`subscribers.status.${queryParams.subStatus}`) }})</template>
        </span>
      </template>
      <PvButton
        v-if="!isMobile"
        v-tooltip.bottom="$t('subscribers.export')"
        severity="secondary"
        outlined
        icon="pi pi-download"
        :aria-label="$t('subscribers.export')"
        data-cy="btn-export-subscribers"
        @click="askExport"
      />
      <PvButton
        v-if="$can('subscribers:import')"
        severity="secondary"
        outlined
        icon="pi pi-upload"
        :label="$t('menu.import')"
        @click="$router.push({ name: 'import' })"
      />
      <PvButton
        v-if="$can('subscribers:manage')"
        icon="pi pi-plus"
        :label="$t('subscribers.addPerson')"
        data-cy="btn-new"
        @click="showNewForm"
      />
    </page-header>

    <div class="subs-layout">
      <!-- Segments: built in, then saved by the team. -->
      <nav v-if="!isMobile" class="subs-segs" :aria-label="$t('subscribers.segments.title')">
        <span class="subs-segs-label">{{ $t('subscribers.segments.title') }}</span>
        <a
          v-for="b in builtIns"
          :key="b.key"
          href="#"
          class="subs-seg"
          :aria-current="segKey === b.key ? 'true' : undefined"
          data-cy="segment"
          @click.prevent="pickSegment(b.key)"
        >
          <span>{{ $t(`subscribers.segments.builtin.${b.key}`) }}</span>
          <span class="subs-seg-n">{{ countLabel(b.key) }}</span>
        </a>
        <span class="subs-segs-label">{{ $t('subscribers.segments.saved') }}</span>
        <a
          v-for="g in saved"
          :key="g.id"
          href="#"
          class="subs-seg"
          :aria-current="segKey === `s${g.id}` ? 'true' : undefined"
          data-cy="saved-segment"
          @click.prevent="pickSegment(`s${g.id}`)"
        >
          <span class="subs-seg-name">{{ g.name }}</span>
          <span class="subs-seg-n">{{ countLabel(`s${g.id}`) }}</span>
        </a>
        <button v-if="$can('subscribers:manage')" type="button" class="subs-seg subs-seg-new" @click="newSegment">
          <i class="pi pi-plus" aria-hidden="true" /> {{ $t('subscribers.segments.new') }}
        </button>
      </nav>

      <div class="subs-main">
        <div v-if="isMobile" class="subs-chips" role="tablist" :aria-label="$t('subscribers.segments.title')">
          <button
            v-for="c in chips"
            :key="c.key"
            type="button"
            role="tab"
            class="subs-chip"
            :aria-selected="segKey === c.key"
            @click="pickSegment(c.key)"
          >
            {{ c.name }}<span v-if="segKey === c.key">{{ countLabel(c.key) }}</span>
          </button>
        </div>

        <segment-builder
          v-model="seg"
          :title="currentSaved?.name || ''"
          :count="loading.subscribers && !subscribers.total ? null : (subscribers.total ?? null)"
          :total="counts.everyone"
          :lists="listOptions"
          :can-save="$can('subscribers:manage') && seg.conditions.length > 0"
          :can-sql="$can('subscribers:sql_query')"
          :can-delete="$can('subscribers:manage') && !!currentSaved"
          @save="askSave"
          @sql="editAsSql"
          @delete="askDeleteSegment"
        />

    <div class="subs-filters">
      <form class="subs-search" role="search" @submit.prevent="onSubmit">
        <PvIconField>
          <PvInputIcon class="pi pi-search" />
          <PvInputText
            v-model="queryInput"
            :placeholder="$t('subscribers.findPerson')"
            :aria-label="$t('subscribers.findPerson')"
            :disabled="isSearchAdvanced"
            data-cy="search"
            @input="onSimpleQueryInput"
          />
        </PvIconField>
      </form>
      <PvSelect
        v-if="queryParams.listID"
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
        v-if="isSearchAdvanced"
        :label="$t('subscribers.reset')"
        icon="pi pi-times"
        severity="secondary"
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

        <PvColumn v-if="!isMobile" selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="email" :header="$t('subscribers.col.person')" sortable header-class="cy-email">
          <template #body="{ data }">
            <span class="subs-person">
              <span class="subs-avatar" aria-hidden="true">{{ initials(data) }}</span>
              <span class="subs-person-text">
                <span class="subs-person-name">
                  <a
                    :href="$router.resolve(`/subscribers/${data.id}`).href"
                    class="ln-row-title"
                    data-cy="name"
                    :data-id="data.id"
                    @click.prevent.stop="openDrawer(data)"
                  >{{ data.name || data.email }}</a>
                  <span v-if="data.status === 'blocklisted'" class="subs-blocked-tag" data-cy="blocklisted">{{ $t('subscribers.status.blocklisted') }}</span>
                  <status-tag
                    v-if="data.scrubStatus === 'risky' || data.scrubStatus === 'unchecked_error'"
                    status="pending"
                    :label="$t(`subscribers.scrubStatus.${data.scrubStatus}`)"
                    data-cy="scrub-status"
                  />
                </span>
                <span class="ln-row-sub" data-cy="email">{{ data.email }}</span>
              </span>
              <span v-if="isMobile" class="subs-opened-m" :class="{ 'is-never': !data.lastOpenedAt }">{{ opened(data) }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn v-if="!isMobile" :header="$t('globals.terms.lists')" header-class="cy-lists">
          <template #body="{ data }">
            <!-- Plain text, one list named: chips per list made every row
                 a cluster. Waiting to confirm is the exception worth a mark. -->
            <span class="subs-lists ln-small" data-cy="lists" :title="activeLists(data).map((l) => l.name).join(', ')">
              {{ listsLabel(data) }}
              <span v-if="activeLists(data).some(awaitingOptin)" class="subs-unconf">{{ $t('subscribers.status.unconfirmed').toLowerCase() }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn v-if="!isMobile" :header="$t('subscribers.col.lastOpened')" header-style="width:120px">
          <template #body="{ data }"><span class="ln-date" :class="{ 'subs-never': !data.lastOpenedAt }">{{ opened(data) }}</span></template>
        </PvColumn>

        <PvColumn v-if="!openSub && !isMobile" header-style="width:56px" body-style="text-align:right">
          <template #body="{ data }">
            <button type="button" class="ln-row-menu" data-cy="row-menu" :aria-label="$t('globals.buttons.rowActions')" aria-haspopup="true" @click.stop="(e) => openMenu(e, data)">
              <i class="pi pi-ellipsis-h" />
            </button>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
      </div>
    </div>
    <PvMenu ref="menu" :model="menuItems" popup />

    <!-- Save a segment -->
    <PvDialog v-model:visible="saveDialog.open" :header="$t('subscribers.segments.saveTitle')" :style="{ width: '420px', maxWidth: 'calc(100vw - 32px)' }" modal>
      <form class="subs-save" @submit.prevent="saveSegment">
        <label for="seg-name">{{ $t('subscribers.segments.name') }}</label>
        <PvInputText id="seg-name" v-model="saveDialog.name" :placeholder="$t('subscribers.segments.namePlaceholder')" required autofocus data-cy="segment-name" />
        <div class="subs-save-foot">
          <PvButton type="button" :label="$t('globals.buttons.cancel')" severity="secondary" text @click="saveDialog.open = false" />
          <PvButton type="submit" :label="$t('subscribers.segments.save')" :loading="saveDialog.busy" data-cy="btn-segment-save" />
        </div>
      </form>
    </PvDialog>

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
import SegmentBuilder from '../components/SegmentBuilder.vue';
import type { Seg } from '../components/SegmentBuilder.vue';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

// Subscribers and lists are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

const { $utils, $can } = useGlobal();
const {
  listSubscribers, getSubscriber, deleteSubscriber, deleteSubscribers,
  blocklistSubscribers, blocklistSubscribersByQuery, deleteSubscribersByQuery,
  manageSubscriberLists, manageSubscriberListsByQuery,
  getSegments, createSegment, updateSegment, deleteSegment, countSegments, segmentSQL,
} = subscribersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { refreshTick, loading } = storeToRefs(store);
const subscribers = computed(() => (store.subscribers as Loose) || {});
const lists = computed(() => (store.lists as Loose) || {});
const { shortDate, ago } = useTime();
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

// Built-in segments, as conditions the segment compiler understands.
const builtIns: { key: string; seg: Seg }[] = [
  { key: 'everyone', seg: { match: 'all', conditions: [] } },
  { key: 'engaged', seg: { match: 'all', conditions: [{ field: 'last_opened', op: 'in_last', value: '90' }] } },
  { key: 'new', seg: { match: 'all', conditions: [{ field: 'joined', op: 'in_last', value: '30' }] } },
  { key: 'neverOpened', seg: { match: 'all', conditions: [{ field: 'last_opened', op: 'never' }] } },
  { key: 'unconfirmed', seg: { match: 'all', conditions: [{ field: 'unconfirmed', op: 'is' }] } },
  { key: 'blocklisted', seg: { match: 'all', conditions: [{ field: 'status', op: 'is', value: 'blocklisted' }] } },
];

interface SavedSeg { id: number; name: string; match: 'all' | 'any'; conditions: Seg['conditions'] }

const seg = ref<Seg>({ match: 'all', conditions: [] });
const segKey = ref<string>((route.query.segment as string) || 'everyone');
const saved = ref<SavedSeg[]>([]);
const counts = ref<Record<string, number>>({});
const saveDialog = reactive({ open: false, name: '', busy: false });

const currentSaved = computed(() => saved.value.find((g) => `s${g.id}` === segKey.value) || null);
const chips = computed(() => [
  ...(currentSaved.value ? [{ key: segKey.value, name: currentSaved.value.name }] : []),
  ...builtIns.map((b) => ({ key: b.key, name: t(`subscribers.segments.builtin.${b.key}`) })),
  ...saved.value.filter((g) => `s${g.id}` !== segKey.value).map((g) => ({ key: `s${g.id}`, name: g.name })),
]);

function segFor(key: string): Seg {
  const b = builtIns.find((x) => x.key === key);
  if (b) return b.seg;
  const g = saved.value.find((x) => `s${x.id}` === key);
  return g ? { match: g.match, conditions: g.conditions } : builtIns[0].seg;
}

function countLabel(key: string): string {
  return counts.value[key] === undefined ? '' : String($utils.niceNumber(counts.value[key]));
}

function pickSegment(key: string) {
  segKey.value = key;
  seg.value = JSON.parse(JSON.stringify(segFor(key)));
  router.replace({ query: { ...route.query, segment: key === 'everyone' ? undefined : key } });
}

function newSegment() {
  segKey.value = 'everyone';
  seg.value = { match: 'all', conditions: [] };
}

function fetchCounts() {
  const keys = [...builtIns.map((b) => b.key), ...saved.value.map((g) => `s${g.id}`)];
  countSegments({ segments: keys.map(segFor) } as never).then((d) => {
    const n = (d as unknown as number[]) || [];
    counts.value = Object.fromEntries(keys.map((k, i) => [k, n[i]]));
  }).catch(() => {});
}

function fetchSegments() {
  return getSegments().then((d) => {
    saved.value = (d as unknown as SavedSeg[]) || [];
    fetchCounts();
  });
}

function askSave() {
  if (currentSaved.value) {
    persistSegment(currentSaved.value.name);
    return;
  }
  saveDialog.name = '';
  saveDialog.open = true;
}

function saveSegment() {
  if (saveDialog.name.trim()) persistSegment(saveDialog.name.trim());
}

function persistSegment(name: string) {
  saveDialog.busy = true;
  const body = { name, match: seg.value.match, conditions: seg.value.conditions };
  const req = currentSaved.value ? updateSegment(currentSaved.value.id, body as never) : createSegment(body as never);
  req.then((d) => {
    const g = d as unknown as SavedSeg;
    saveDialog.open = false;
    $utils.toast(t('subscribers.segments.savedToast', { name: g.name }));
    fetchSegments().then(() => { segKey.value = `s${g.id}`; router.replace({ query: { ...route.query, segment: `s${g.id}` } }); });
  }).finally(() => { saveDialog.busy = false; });
}

function askDeleteSegment() {
  const g = currentSaved.value;
  if (!g) return;
  Object.assign(confirm, {
    open: true,
    title: t('subscribers.segments.deleteTitle', { name: g.name }),
    text: t('subscribers.segments.deleteText'),
    label: t('subscribers.segments.delete'),
    typed: '',
    danger: true,
    secondary: null,
    run: () => {
      confirm.busy = true;
      deleteSegment(g.id).then(() => {
        confirm.open = false;
        pickSegment('everyone');
        fetchSegments();
      }).finally(() => { confirm.busy = false; });
    },
  });
}

// editAsSql swaps the conditions for the SQL they compile to, in the
// advanced query box.
function editAsSql() {
  segmentSQL(seg.value as never).then((d) => {
    const { sql } = d as unknown as { sql: string };
    queryParams.queryExp = sql === 'TRUE' ? '' : sql;
    queryParams.search = '';
    queryInput.value = '';
    isSearchAdvanced.value = true;
    seg.value = { match: 'all', conditions: [] };
  });
}

function segmentParam(): string | undefined {
  return seg.value.conditions.length ? JSON.stringify(seg.value) : undefined;
}

function initials(sub: Loose): string {
  const src = (sub.name || sub.email || '').trim();
  const parts = src.split(/[\s@._-]+/).filter(Boolean);
  return ((parts[0]?.[0] || '') + (parts[1]?.[0] || '')).toUpperCase();
}

function opened(sub: Loose): string {
  return sub.lastOpenedAt ? ago(sub.lastOpenedAt) : t('subscribers.never');
}

const rows = computed<Loose[]>(() => subscribers.value.results || []);
const numSelected = computed(() => (bulk.all ? (subscribers.value.total || 0) : bulk.checked.length));
const listOptions = computed(() => lists.value.results || []);
const summary = computed(() => {
  if (counts.value.everyone === undefined) return '';
  return t('subscribers.summary', { n: $utils.niceNumber(counts.value.everyone), lists: listOptions.value.length });
});
const currentList = computed(() => (queryParams.listID ? listOptions.value.find((l: Loose) => l.id === queryParams.listID) : null));
const isFiltered = computed(() => !!(queryParams.search || queryParams.queryExp || queryParams.listID || queryParams.subStatus));

const advancedHint = computed(() => {
  const help = t('subscribers.advancedQueryHelp');
  if (!queryParams.queryExp || subscribers.value.total === undefined) return help;
  return `${help}\u2003${t('subscribers.matchCount', { n: (subscribers.value.total || 0).toLocaleString() })}`;
});

const rangeLabel = computed(() => {
  const total = subscribers.value.total || 0;
  const per = subscribers.value.perPage || 20;
  if (!total) return '';
  const from = (queryParams.page - 1) * per + 1;
  return t('developers.log.range', { from, to: Math.min(queryParams.page * per, total), total: total.toLocaleString() });
});

// listsLabel names the first list and counts the rest ("Weekly digest
// +2 more").
function listsLabel(sub: Loose): string {
  const ls = activeLists(sub);
  if (!ls.length) return '';
  return ls.length > 1 ? `${ls[0].name} ${t('subscribers.moreLists', { n: ls.length - 1 })}` : ls[0].name;
}

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
    segment: segmentParam(),
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
    segment: seg.value.conditions.length ? seg.value : undefined,
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
      const sp = segmentParam();
      if (sp) q.append('segment', sp);
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
    if (seg.value.conditions.length) data.segment = seg.value;
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
watch(seg, () => querySubscribers({ page: 1 }), { deep: true });

// /subscribers and /subscribers/:id share this view (meta.view), so the
// drawer follows the id in the URL instead of the page being rebuilt.
watch(() => route.params.id, (id) => {
  if (!id) {
    openSub.value = null;
    return;
  }
  if (openSub.value && String(openSub.value.id) === id) return;
  getSubscriber(parseInt(id as string, 10)).then((data) => { openSub.value = data as unknown as Loose; });
});

onMounted(() => {
  if (route.params.listID) queryParams.listID = parseInt(route.params.listID as string, 10);
  if (route.query.subscription_status) queryParams.subStatus = route.query.subscription_status as string;
  fetchSegments().then(() => {
    // A saved segment in the URL loads once the segments have.
    if (segKey.value !== 'everyone') seg.value = JSON.parse(JSON.stringify(segFor(segKey.value)));
  });
  if (builtIns.some((b) => b.key === segKey.value)) seg.value = JSON.parse(JSON.stringify(segFor(segKey.value)));
  querySubscribers();
  if (route.params.id) {
    getSubscriber(parseInt(route.params.id as string, 10)).then((data) => { openSub.value = data as unknown as Loose; });
  }
});
</script>

<style scoped lang="scss">
// A double opt-in subscription still waiting for confirmation says so in
// words (the handover allows no dots).
.subs-unconf { margin-left: 6px; color: var(--ln-warn-fg); }
.subs { transition: padding-right var(--ln-dur-base) var(--ln-ease); }
.subs.has-drawer { padding-right: 440px; }

.subs-layout { display: flex; gap: 24px; align-items: flex-start; }
.subs-main { flex: 1 1 0; min-width: 0; display: flex; flex-direction: column; gap: 14px; }

.subs-segs {
  flex: none;
  width: 232px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  position: sticky;
  top: 72px;
}
.subs-segs-label {
  padding: 0 10px 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--ln-text-3);

  &:not(:first-child) { margin-top: 18px; }
}
.subs-seg {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--ln-radius);
  background: transparent;
  color: var(--ln-text-2);
  font: inherit;
  font-size: 14px;
  text-align: left;
  text-decoration: none;
  cursor: pointer;

  &:hover { background: var(--ln-hover); color: var(--ln-text); text-decoration: none; }
  &[aria-current] { background: var(--ln-muted); color: var(--ln-text); font-weight: 600; }
}
.subs-seg-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.subs-seg-n {
  font-size: 12px;
  font-weight: 400;
  color: var(--ln-text-3);
  font-variant-numeric: tabular-nums;
  // Only the open segment shows its count; the rest on hover or focus.
  opacity: 0;
  transition: opacity var(--ln-dur-fast) var(--ln-ease);

  .subs-seg:hover &, .subs-seg:focus-visible &, .subs-seg[aria-current] & { opacity: 1; }
}
.subs-seg-new { margin-top: 4px; justify-content: flex-start; font-weight: 500; i { font-size: 12px; } }

.subs-chips {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  margin: 0 -16px;
  padding: 0 16px 2px;
  scrollbar-width: none;

  &::-webkit-scrollbar { display: none; }
}
.subs-chip {
  flex: none;
  display: inline-flex;
  gap: 8px;
  align-items: center;
  height: 36px;
  padding: 0 12px;
  border-radius: var(--ln-radius);
  border: 1px solid var(--ln-border);
  background: var(--ln-card);
  color: var(--ln-text);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  span { color: var(--ln-text-3); font-weight: 400; }
  &[aria-selected="true"] { border-color: var(--ln-ink); background: var(--ln-muted); }
}

.subs-person { display: flex; align-items: center; gap: 12px; min-width: 0; }
.subs-avatar {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 99px;
  background: var(--ln-muted);
  font-size: 11px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.subs-person-text { flex: 1; min-width: 0; display: flex; flex-direction: column; .ln-row-sub { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } }
.subs-person-name { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.subs-blocked-tag {
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--ln-danger-bg);
  color: var(--ln-danger-fg);
  font-size: 11px;
  font-weight: 500;
}
.subs-opened-m { flex: none; font-size: 12px; color: var(--ln-text-2); &.is-never { color: var(--ln-text-4); } }
.subs-never { color: var(--ln-text-4); }
.subs-lists { color: var(--ln-text-2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: block; .subs-unconf { margin-left: 8px; } }

.subs-save { display: flex; flex-direction: column; gap: 8px; label { font-size: 13px; font-weight: 500; } }
.subs-save-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }

.subs-filters { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.subs-search {
  flex: 1 1 280px;
  max-width: 360px;

  .p-inputtext { width: 100%; }
  @media (max-width: 767px) { max-width: none; }
}
.subs-list { width: 200px; }
@media (max-width: 991px) {
  .subs-segs { display: none; }
  // Phones show people as a list, as in the mobile artboard.
  .subs-main :deep(.p-datatable-thead) { display: none; }
}
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

  &.is-blocklisted { color: var(--ln-danger-fg); }
  &.is-disabled { color: var(--ln-text-3); }
}
:deep(tr.is-open) { background: var(--ln-hover) !important; box-shadow: inset 3px 0 0 var(--ln-ink); }
:deep(.p-datatable-tbody > tr) { cursor: pointer; }
</style>
