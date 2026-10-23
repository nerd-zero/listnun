<template>
  <div class="ln-page lists">
    <page-header
      v-if="!(isMobile && selected)"
      :title="$t('globals.terms.lists')"
      :subtitle="rows.length ? $t('lists.summary', { subs: $utils.niceNumber(totalSubs), n: rows.length }, rows.length) : ''"
    >
      <template #after-title>
        <span v-if="isArchived" class="ln-small">/ {{ $t('lists.archived') }}</span>
      </template>
      <PvButton v-if="$can('lists:manage_all')" icon="pi pi-plus" :label="$t('lists.newList')" data-cy="btn-new" @click="showNewForm" />
    </page-header>

    <div v-if="pending.total && !isArchived && !(isMobile && selected)" role="status" class="lists-banner" data-cy="optin-banner">
      <span class="lists-banner-text">
        <i class="pi pi-envelope" aria-hidden="true" />
        <span>{{ $t('lists.unconfirmedBanner', { n: $utils.niceNumber(pending.total) }, pending.total) }}</span>
      </span>
      <PvButton
        v-if="$can('subscribers:manage')"
        :label="$t('lists.resendConfirmation')"
        size="small"
        severity="secondary"
        outlined
        data-cy="btn-resend"
        @click="askResend"
      />
    </div>

    <div class="lists-layout" :class="{ 'has-panel': selected }">
      <div v-if="!(isMobile && selected)" class="lists-side">
        <form v-if="!isMobile" class="ln-search" role="search" @submit.prevent>
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="search" name="query" :placeholder="$t('lists.find')" :aria-label="$t('lists.find')" data-cy="query" />
          </PvIconField>
        </form>

        <table-skeleton v-if="!loaded" :cols="1" />
        <empty-state
          v-else-if="!rows.length"
          icon="pi-list"
          :title="isArchived ? $t('states.noMatches') : $t('lists.emptyTitle')"
          :text="isArchived ? '' : $t('lists.emptyText')"
          :action-label="!isArchived && $can('lists:manage_all') ? $t('lists.newList') : ''"
          @action="showNewForm"
        />
        <p v-else-if="!shown.length" class="ln-small">{{ $t('states.noMatchesText', { query: search }) }}</p>

        <ul class="lists-cards ln-enter">
          <li v-for="l in shown" :key="l.id">
            <router-link
              :to="{ name: 'list', params: { id: l.id } }"
              class="lists-card"
              :class="{ 'is-current': selected?.id === l.id && !isMobile }"
              :aria-current="selected?.id === l.id ? 'page' : undefined"
              :data-id="l.id"
              data-cy="list-card"
              :replace="!isMobile"
            >
              <span class="lists-card-row">
                <strong class="lists-card-name" data-cy="list-name">{{ l.name }}</strong>
                <span class="lists-card-n">{{ $utils.niceNumber(receiving(l)) }}</span>
              </span>
              <span class="lists-card-row ln-small">
                <span class="lists-card-kind">{{ $t(`lists.kind.${kindKey(l)}`) }}</span>
                <span :class="changeClass(l)">{{ changeLabel(l) }}</span>
              </span>
              <span v-if="awaiting(l)" class="lists-card-unconf ln-small">{{ $t('lists.notConfirmedYet', { n: $utils.niceNumber(awaiting(l)) }) }}</span>
            </router-link>
          </li>
        </ul>

        <router-link :to="isArchived ? { name: 'lists' } : { name: 'lists', query: { status: 'archived' } }" class="ln-small lists-archived-link">
          {{ isArchived ? $t('lists.viewActive') : $t('lists.viewArchived') }}
        </router-link>
      </div>

      <div v-if="selected" class="lists-main">
        <router-link v-if="isMobile" :to="{ name: 'lists' }" class="lists-back ln-small">
          <i class="pi pi-angle-left" aria-hidden="true" /> {{ $t('lists.back') }}
        </router-link>
        <list-panel
          ref="panel"
          :list="selected"
          :scrub="scrubStatus[selected.id]"
          :scrub-progress="scrubProgress[selected.id]"
          @edit="showEditForm(selected)"
          @menu="(e) => openMenu(e, selected!)"
        />
      </div>
    </div>
    <PvMenu ref="menu" :model="menuItems" popup />

    <p v-if="settings['app.cache_slow_queries']" class="ln-small">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferrer">{{ $t('globals.buttons.learnMore') }}</a>
    </p>

    <list-form v-if="isFormVisible" :data="curItem || {}" :is-editing="isEditing" @finished="onSaved" @close="closeForm" />

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
import { useMedia } from '../composables/useMedia';
import PageHeader from '../components/ui/PageHeader.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import ListPanel from '../components/ListPanel.vue';
import ListForm from './ListForm.vue';
import { getLists as listsApi } from '../api/generated/endpoints/lists/lists';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';
import { getSettings as settingsApi } from '../api/generated/endpoints/settings/settings';

// Lists and Scrub statuses are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }
interface Growth { listId: number; joined: number; left: number }
interface Pending { total: number; due: number }
interface Refreshable { refresh: () => void }

// How often to refresh validation progress while a job runs.
const SCRUB_POLL_MS = 4000;

const {
  $utils, $api, $can, $canList,
} = useGlobal();
const {
  listLists, deleteList, getListsGrowth, getOptinPending, resendOptins,
} = listsApi();
const { createCampaign } = campaignsApi();
const { getScrubListStatus, getScrubListProgress, scrubList } = settingsApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const isMobile = useMedia('(max-width: 767px)');
const {
  refreshTick, settings, serverConfig,
} = storeToRefs(useMainStore());

const {
  curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
} = useFormDialog<Loose>();
const rows = ref<Loose[]>([]);
const loaded = ref(false);
const growth = ref<Record<number, Growth>>({});
const pending = ref<Pending>({ total: 0, due: 0 });
const search = ref('');
const panel = ref<Refreshable | null>(null);
const scrubStatus = ref<Record<number, Loose>>({});
const scrubProgress = ref<Record<number, number>>({});
let scrubPollTimer: ReturnType<typeof setTimeout> | null = null;

const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Loose | null>(null);
const confirm = reactive({
  open: false, title: '', text: '', label: '', typed: '', danger: true, busy: false, run: () => {},
});

const isArchived = computed(() => route.query.status === 'archived');
const selectedId = computed(() => (route.params.id ? Number(route.params.id) : null));
// The list in the panel: the one in the URL, else (on wide screens) the first.
const selected = computed<Loose | null>(() => {
  const byId = rows.value.find((l) => l.id === selectedId.value);
  if (byId) return byId;
  return !isMobile.value && rows.value.length ? rows.value[0] : null;
});
const shown = computed(() => {
  const q = search.value.trim().toLowerCase();
  return q ? rows.value.filter((l) => l.name.toLowerCase().includes(q)) : rows.value;
});
const totalSubs = computed(() => rows.value.reduce((n, l) => n + (l.subscriberCount || 0), 0));

function kindKey(l: Loose): string {
  return `${l.type === 'public' ? 'public' : 'private'}${l.optin === 'double' ? 'Double' : 'Single'}`;
}

// awaiting is the number of double opt-in subscribers yet to confirm.
function awaiting(l: Loose): number {
  if (l.optin !== 'double') return 0;
  return l.subscriberStatuses?.unconfirmed || 0;
}

// receiving is who gets campaigns: confirmed people, and on single opt-in
// lists the unconfirmed too.
function receiving(l: Loose): number {
  const s = l.subscriberStatuses || {};
  return (s.confirmed || 0) + (l.optin === 'double' ? 0 : (s.unconfirmed || 0));
}

function change(l: Loose): number {
  const g = growth.value[l.id];
  return g ? g.joined - g.left : 0;
}

function changeLabel(l: Loose): string {
  const n = change(l);
  if (n > 0) return t('lists.change.up', { n: $utils.niceNumber(n) });
  if (n < 0) return t('lists.change.down', { n: $utils.niceNumber(-n) });
  return t('lists.change.none');
}

function changeClass(l: Loose): string {
  const n = change(l);
  if (n > 0) return 'is-up';
  return n < 0 ? 'is-down' : 'is-flat';
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

function fetchPending() {
  if (!$can('subscribers:manage')) return;
  getOptinPending().then((d) => { pending.value = d as unknown as Pending; }).catch(() => {});
}

function fetchLists() {
  listLists({
    per_page: 'all',
    order_by: 'name',
    order: 'asc',
    status: isArchived.value ? 'archived' : 'active',
  } as never).then((resp) => {
    rows.value = (resp as unknown as Loose).results || [];
    loaded.value = true;
  });
  getListsGrowth().then((d) => {
    const m: Record<number, Growth> = {};
    ((d as unknown as Growth[]) || []).forEach((g) => { m[g.listId] = g; });
    growth.value = m;
  }).catch(() => {});
  fetchPending();
  if ((serverConfig.value as Loose).scrub_enabled) fetchScrubStatus();
}

function onSaved() {
  fetchLists();
  panel.value?.refresh();
}

function askResend() {
  const { due } = pending.value;
  if (!due) {
    $utils.toast(t('lists.resendNoneDue'));
    return;
  }
  Object.assign(confirm, {
    open: true,
    title: t('lists.resendTitle'),
    text: t('lists.resendText', { n: $utils.niceNumber(due) }, due),
    label: t('lists.resendConfirmation'),
    typed: '',
    danger: false,
    run: () => {
      confirm.busy = true;
      resendOptins({}).then((d) => {
        confirm.open = false;
        const n = (d as unknown as Loose)?.queued || 0;
        $utils.toast(n ? t('lists.resendStarted', { n: $utils.niceNumber(n) }, n) : t('lists.resendNoneDue'));
        fetchPending();
      }).finally(() => { confirm.busy = false; });
    },
  });
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
        if (selectedId.value === l.id) router.replace({ name: 'lists' });
        fetchLists();
        refreshMinimalLists();
        $utils.toast(t('globals.messages.deleted', { name: l.name }));
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
watch(isArchived, () => {
  loaded.value = false;
  fetchLists();
});

onMounted(fetchLists);

onUnmounted(() => {
  if (scrubPollTimer) clearTimeout(scrubPollTimer);
});
</script>

<style scoped lang="scss">
.lists-banner {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  padding: 12px 16px;
  border-radius: var(--ln-radius-card);
  background: var(--ln-warn-bg);

  @media (max-width: 767px) { flex-direction: column; align-items: flex-start; }

  // A card-coloured button on the tinted banner, as in the artboard.
  :deep(.p-button) { background: var(--ln-card); color: var(--ln-text); flex: none; }
}
.lists-banner-text {
  display: flex;
  gap: 10px;
  align-items: flex-start;

  i { margin-top: 3px; color: var(--ln-warn-fg); }
}

.lists-layout {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}
.lists-side {
  flex: none;
  width: 360px;
  display: flex;
  flex-direction: column;
  gap: 10px;

  .lists-layout:not(.has-panel) & { flex: 1; width: auto; }
  @media (max-width: 991px) { width: 300px; }
  @media (max-width: 767px) { flex: 1; width: auto; }
}
.lists-main {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.lists-side .ln-search { width: 100%; }
.lists-back { display: inline-flex; align-items: center; gap: 4px; color: var(--ln-text-2); }

.lists-cards {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.lists-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: var(--ln-radius-card);
  border: 1px solid var(--ln-border);
  background: var(--ln-card);
  color: var(--ln-text);
  text-decoration: none;

  &:hover { border-color: var(--ln-border-strong); text-decoration: none; }
  &.is-current { border: 2px solid var(--ln-ink); padding: 13px 15px; }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}
.lists-card-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}
.lists-card-name { font-weight: 600; min-width: 0; overflow-wrap: anywhere; }
.lists-card-n { font-size: 17px; font-weight: 600; font-variant-numeric: tabular-nums; }
.lists-card-kind { color: var(--ln-text-3); }
.lists-card-unconf { color: var(--ln-warn-fg); }
.is-up { color: var(--ln-success-fg); font-variant-numeric: tabular-nums; }
.is-down { color: var(--ln-danger-fg); font-variant-numeric: tabular-nums; }
.is-flat { color: var(--ln-text-3); }
.lists-archived-link { color: var(--ln-text-2); }
</style>
