<template>
  <div class="ln-page dev-hooks" :class="{ 'has-drawer': !!openId }">
    <page-header :title="$t('developers.webhooks.title')" :count="endpoints.length || null" :subtitle="$t('developers.webhooks.intro')">
      <PvButton v-if="canManage" icon="pi pi-plus" :label="$t('developers.webhooks.add')" data-cy="btn-new" @click="openAdd" />
    </page-header>

    <error-state v-if="loadError" :title="$t('states.couldntLoad')" :error="loadError" @retry="fetchAll" />

    <template v-else>
      <table-skeleton v-if="loading && !endpoints.length" :rows="2" :cols="3" />

      <empty-state
        v-else-if="!endpoints.length"
        icon="pi-share-alt"
        :title="$t('developers.webhooks.emptyTitle')"
        :text="$t('developers.webhooks.emptyText')"
        :action-label="canManage ? $t('developers.webhooks.add') : ''"
        @action="openAdd"
      />

      <section v-for="e in endpoints" :key="e.id" class="ln-card dev-hook">
        <div class="dev-hook-head">
          <div class="dev-hook-title">
            <div class="dev-hook-url">
              <span class="ln-mono">{{ e.url }}</span>
              <status-tag :status="e.status === 'active' ? 'active' : 'paused'" :label="$t(`developers.webhooks.endpointStatus.${e.status}`)" />
            </div>
            <span class="ln-attrs">
              <span v-for="ev in e.events" :key="ev" class="ln-attr ln-attr--mono">{{ ev }}</span>
            </span>
          </div>
          <div v-if="canManage" class="dev-hook-actions">
            <PvButton :label="$t('developers.webhooks.sendTest')" severity="secondary" outlined :loading="testing === e.id" @click="test(e)" />
            <button type="button" class="ln-row-menu dev-hook-more" :aria-label="$t('globals.buttons.rowActions')" aria-haspopup="true" @click="(ev) => openMenu(ev, e)">
              <i class="pi pi-ellipsis-h" />
            </button>
          </div>
        </div>
        <div class="dev-hook-stats">
          <div>
            <span class="ln-small">{{ $t('developers.webhooks.success24h') }}</span>
            <strong>{{ e.done24h ? `${((e.ok24h / e.done24h) * 100).toFixed(1)}%` : '—' }}</strong>
          </div>
          <div>
            <span class="ln-small">{{ $t('developers.webhooks.median') }}</span>
            <strong>{{ e.medianMs ? `${e.medianMs} ms` : '—' }}</strong>
          </div>
          <div>
            <span class="ln-small">{{ $t('developers.webhooks.retrying') }}</span>
            <strong :class="{ 'is-warn': e.retrying }">{{ e.retrying }}</strong>
          </div>
          <div>
            <span class="ln-small">{{ $t('developers.webhooks.secret') }}</span>
            <span class="dev-hook-secret">
              <span class="ln-mono">{{ secrets[e.id] || `whsec_••••••••${e.secretHint}` }}</span>
              <button v-if="canManage" type="button" class="ln-link-btn" @click="toggleSecret(e)">
                {{ secrets[e.id] ? $t('developers.webhooks.hide') : $t('developers.webhooks.reveal') }}
              </button>
              <button v-if="secrets[e.id]" type="button" class="ln-link-btn" @click="copy(secrets[e.id])">{{ $t('globals.buttons.copy') }}</button>
            </span>
          </div>
        </div>
      </section>

      <div v-if="endpoints.length" class="dev-hooks-grid">
        <table-card>
          <template #toolbar>
            <h2 class="ln-card-title">{{ $t('developers.webhooks.recent') }}</h2>
            <span class="ln-small dev-hooks-note">{{ $t('developers.webhooks.retriesNote') }}</span>
          </template>
          <PvDataTable
            :value="deliveries.results"
            data-key="id"
            lazy
            paginator
            :rows="perPage"
            :total-records="deliveries.total"
            :first="(page - 1) * perPage"
            :row-class="(r) => (r.id === openId ? 'is-open' : '')"
            @page="(ev) => { page = ev.page + 1; fetchDeliveries(); }"
            @row-click="(ev) => openId = ev.data.id"
          >
            <template #empty>
              <empty-state bare icon="pi-send" :title="$t('developers.webhooks.noDeliveries')" :text="$t('developers.webhooks.noDeliveriesText')" />
            </template>
            <PvColumn :header="$t('developers.webhooks.event')">
              <template #body="{ data }"><span class="ln-mono">{{ data.event }}</span></template>
            </PvColumn>
            <PvColumn :header="$t('developers.log.message')">
              <template #body="{ data }"><span class="ln-mono ln-muted">{{ data.messageId || '—' }}</span></template>
            </PvColumn>
            <PvColumn :header="$t('developers.webhooks.response')">
              <template #body="{ data }">
                <span class="ln-mono" :class="codeClass(data.lastStatusCode, data.status)">{{ data.lastStatusCode || (data.status === 'pending' ? '…' : '—') }}</span>
              </template>
            </PvColumn>
            <PvColumn :header="$t('developers.webhooks.time')">
              <template #body="{ data }"><span class="ln-mono ln-muted">{{ data.lastDurationMs != null ? duration(data.lastDurationMs) : '—' }}</span></template>
            </PvColumn>
            <PvColumn :header="$t('developers.webhooks.when')">
              <template #body="{ data }">
                <span class="ln-date">
                  {{ ago(data.createdAt) }}
                  <template v-if="data.status === 'retrying' && data.nextAttemptAt"> · {{ $t('developers.webhooks.retryIn', { t: timeLeft(data.nextAttemptAt) }) }}</template>
                </span>
              </template>
            </PvColumn>
          </PvDataTable>
        </table-card>

        <section class="ln-card dev-hooks-example">
          <div class="ln-card-pad">
            <h2 class="ln-card-title">{{ $t('developers.webhooks.example') }}</h2>
            <i18n-t keypath="developers.webhooks.verify" tag="p" class="ln-small">
              <template #header><code class="ln-mono">Listnun-Signature</code></template>
            </i18n-t>
          </div>
          <pre class="ln-code dev-hooks-code">{{ examplePayload }}</pre>
          <div class="ln-card-pad">
            <p class="ln-small">{{ $t('developers.webhooks.verifyHow') }}</p>
          </div>
        </section>
      </div>
    </template>

    <PvMenu ref="menu" :model="menuItems" popup />

    <dialog-form
      v-model:visible="formOpen"
      :title="editing ? $t('developers.webhooks.edit') : $t('developers.webhooks.add')"
      :submit-label="editing ? $t('globals.buttons.save') : $t('developers.webhooks.add')"
      :busy="saving"
      :submit-disabled="!form.url || !form.events.length"
      width="540px"
      @submit="saveEndpoint"
    >
      <div class="ln-field">
        <label for="hook-url">{{ $t('developers.webhooks.url') }}</label>
        <PvInputText id="hook-url" v-model="form.url" class="ln-mono" placeholder="https://api.example.com/hooks/listnun" autocomplete="off" />
        <p class="ln-help">{{ $t('developers.webhooks.urlHelp') }}</p>
      </div>
      <fieldset class="dev-hooks-events">
        <legend class="ln-field-label">{{ $t('developers.webhooks.events') }}</legend>
        <label v-for="ev in WEBHOOK_EVENTS" :key="ev" class="dev-hooks-event" :for="`ev-${ev}`">
          <PvCheckbox v-model="form.events" :value="ev" :input-id="`ev-${ev}`" />
          <span>
            <span class="ln-mono">{{ ev }}</span>
            <span class="ln-row-sub">{{ $t(`developers.webhooks.eventHelp.${ev.replace('.', '_')}`) }}</span>
          </span>
        </label>
      </fieldset>
    </dialog-form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :danger="confirm.danger"
      :busy="saving"
      @confirm="confirm.run"
    />

    <delivery-drawer :delivery-id="openId" @close="openId = null" @retried="fetchDeliveries" />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onBeforeUnmount, onMounted, reactive, ref, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { storeToRefs } from 'pinia';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import { useTime } from '../../composables/useTime';
import {
  createWebhookEndpoint, deleteWebhookEndpoint, listWebhookDeliveries, listWebhookEndpoints,
  revealWebhookSecret, rotateWebhookSecret, sendWebhookTest, updateWebhookEndpoint, WEBHOOK_EVENTS,
  type WebhookDeliveryPage, type WebhookEndpoint, type WebhookEvent,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import TableCard from '../../components/ui/TableCard.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import EmptyState from '../../components/ui/EmptyState.vue';
import ErrorState, { type RequestError } from '../../components/ui/ErrorState.vue';
import StatusTag from '../../components/ui/StatusTag.vue';
import DialogForm from '../../components/ui/DialogForm.vue';
import ConfirmAction from '../../components/ui/ConfirmAction.vue';
import DeliveryDrawer from './DeliveryDrawer.vue';

interface PopupMenu { toggle: (e: Event) => void }

// How often to refresh deliveries while the page is open.
const REFRESH_MS = 10 * 1000;

const { t } = useI18n();
const { $can, $utils } = useGlobal();
const { ago, timeLeft } = useTime();
const { refreshTick } = storeToRefs(useMainStore());

const perPage = 20;
const endpoints = ref<WebhookEndpoint[]>([]);
const deliveries = ref<WebhookDeliveryPage>({
  results: [], total: 0, page: 1, perPage,
});
const page = ref(1);
const loading = ref(true);
const loadError = ref<RequestError | null>(null);
const saving = ref(false);
const testing = ref<number | null>(null);
const openId = ref<string | null>(null);
const secrets = reactive<Record<number, string>>({});

const formOpen = ref(false);
const editing = ref<WebhookEndpoint | null>(null);
const form = reactive({ url: '', events: [] as WebhookEvent[] });

const menu = ref<PopupMenu | null>(null);
const menuFor = ref<WebhookEndpoint | null>(null);
const confirm = reactive({
  open: false, title: '', text: '', label: '', danger: true, run: () => {},
});

const canManage = computed(() => $can('webhooks:manage'));

const examplePayload = JSON.stringify({
  type: 'message.bounced',
  created_at: '2026-10-19T10:21:05Z',
  data: {
    message_id: 'msg_01JAD5R9M3TB',
    template_id: 4,
    recipient: 'old-inbox@example.com',
    status: 'bounced',
    bounce: { type: 'hard' },
  },
}, null, 2);

function codeClass(code: number | null, status: string): string {
  if (!code) return status === 'pending' ? 'ln-muted' : 'dev-hooks-bad';
  return code >= 200 && code < 300 ? 'dev-hooks-ok' : 'dev-hooks-bad';
}

function duration(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)} s` : `${ms} ms`;
}

function fetchDeliveries() {
  return listWebhookDeliveries({ page: page.value, perPage }).then((p) => { deliveries.value = p; });
}

function fetchAll() {
  loading.value = true;
  loadError.value = null;
  Promise.all([listWebhookEndpoints(), fetchDeliveries()])
    .then(([eps]) => { endpoints.value = eps; })
    .catch((err) => {
      const e = err as { response?: { status?: number }; config?: { method?: string; url?: string } };
      loadError.value = { status: e.response?.status, method: e.config?.method, url: e.config?.url };
    })
    .finally(() => { loading.value = false; });
}

function openAdd() {
  editing.value = null;
  form.url = '';
  form.events = ['message.delivered', 'message.bounced', 'message.failed'];
  formOpen.value = true;
}

function openEdit(e: WebhookEndpoint) {
  editing.value = e;
  form.url = e.url;
  form.events = [...e.events];
  formOpen.value = true;
}

function saveEndpoint() {
  saving.value = true;
  const input = { url: form.url.trim(), events: form.events, status: editing.value?.status };
  const req = editing.value ? updateWebhookEndpoint(editing.value.id, input) : createWebhookEndpoint(input);
  req
    .then(() => { formOpen.value = false; fetchAll(); })
    .finally(() => { saving.value = false; });
}

function test(e: WebhookEndpoint) {
  testing.value = e.id;
  sendWebhookTest(e.id)
    .then((r) => {
      $utils.toast(t('developers.webhooks.testQueued'));
      fetchDeliveries().then(() => { openId.value = r.deliveryId; });
    })
    .finally(() => { testing.value = null; });
}

function toggleSecret(e: WebhookEndpoint) {
  if (secrets[e.id]) {
    delete secrets[e.id];
    return;
  }
  revealWebhookSecret(e.id).then((r) => { secrets[e.id] = r.secret; });
}

function copy(s: string) {
  navigator.clipboard?.writeText(s).then(() => $utils.toast(t('states.copied')));
}

function setStatus(e: WebhookEndpoint, status: 'active' | 'disabled') {
  updateWebhookEndpoint(e.id, { url: e.url, events: e.events, status }).then(fetchAll);
}

function askRotate(e: WebhookEndpoint) {
  Object.assign(confirm, {
    open: true,
    title: t('developers.webhooks.rotateTitle'),
    text: t('developers.webhooks.rotateText'),
    label: t('developers.webhooks.rotate'),
    danger: false,
    run: () => {
      saving.value = true;
      rotateWebhookSecret(e.id)
        .then((r) => { confirm.open = false; secrets[e.id] = r.secret; fetchAll(); })
        .finally(() => { saving.value = false; });
    },
  });
}

function askDelete(e: WebhookEndpoint) {
  Object.assign(confirm, {
    open: true,
    title: t('developers.webhooks.deleteTitle', { url: e.url }),
    text: t('developers.webhooks.deleteText'),
    label: t('globals.buttons.delete'),
    danger: true,
    run: () => {
      saving.value = true;
      deleteWebhookEndpoint(e.id)
        .then(() => { confirm.open = false; fetchAll(); })
        .finally(() => { saving.value = false; });
    },
  });
}

const menuItems = computed<MenuItem[]>(() => {
  const e = menuFor.value;
  if (!e) return [];
  return [
    { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => openEdit(e) },
    e.status === 'active'
      ? { label: t('developers.webhooks.disable'), icon: 'pi pi-pause', command: () => setStatus(e, 'disabled') }
      : { label: t('developers.webhooks.enable'), icon: 'pi pi-play', command: () => setStatus(e, 'active') },
    { label: t('developers.webhooks.rotate'), icon: 'pi pi-refresh', command: () => askRotate(e) },
    { separator: true },
    {
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(e),
    },
  ];
});

function openMenu(ev: Event, e: WebhookEndpoint) {
  menuFor.value = e;
  menu.value?.toggle(ev);
}

let timer: number | undefined;
onMounted(() => {
  fetchAll();
  timer = window.setInterval(() => { if (!document.hidden) fetchDeliveries(); }, REFRESH_MS);
});
onBeforeUnmount(() => window.clearInterval(timer));
watch(refreshTick, fetchAll);
</script>

<style scoped lang="scss">
.dev-hooks { transition: padding-right .15s; }
.dev-hooks.has-drawer { padding-right: 460px; @media (max-width: 991px) { padding-right: 0; } }

.dev-hook { overflow: hidden; }
.dev-hook-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px;
  flex-wrap: wrap;
}
.dev-hook-title { display: flex; flex-direction: column; gap: 8px; min-width: 0; }
.dev-hook-url {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;

  .ln-mono { font-size: 14px; overflow-wrap: anywhere; }
}
.dev-hook-actions { display: flex; gap: 8px; align-items: flex-start; }
.dev-hook-more { border: 1px solid var(--ln-border-strong); height: 36px; width: 36px; }
.dev-hook-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-top: 1px solid var(--ln-border);

  > div {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 12px 20px;
    border-left: 1px solid var(--ln-border);

    &:first-child { border-left: 0; }
  }
  strong { font-size: 20px; line-height: 28px; font-weight: 600; font-variant-numeric: tabular-nums; }
  .is-warn { color: var(--ln-warn-fg); }
  @media (max-width: 767px) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
.dev-hook-secret { display: flex; gap: 8px; align-items: baseline; flex-wrap: wrap; .ln-mono { font-size: 12px; overflow-wrap: anywhere; } }

.dev-hooks-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-hooks-note { margin-left: auto; }
.dev-hooks-ok { color: var(--ln-success-fg); }
.dev-hooks-bad { color: var(--ln-danger-fg); }
:deep(tr.is-open) { background: var(--ln-hover) !important; box-shadow: inset 3px 0 0 var(--ln-ink); }
:deep(.p-datatable-tbody > tr) { cursor: pointer; }

.dev-hooks-example { overflow: hidden; .ln-small { margin: 4px 0 0; } }
.dev-hooks-code { border-radius: 0; }

.dev-hooks-events {
  border: 0;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;

  legend { padding: 0; margin-bottom: 6px; }
}
.dev-hooks-event {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
  cursor: pointer;

  &:hover { background: var(--ln-hover); }
}
</style>
