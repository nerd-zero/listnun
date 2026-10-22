<template>
  <div class="ln-page dev-keys">
    <page-header :title="$t('developers.keys.title')" :count="keys.length || null">
      <template #search>
        <form class="ln-search" role="search" @submit.prevent="fetchKeys">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="query" :placeholder="$t('developers.keys.search')" :aria-label="$t('developers.keys.search')" data-cy="query" />
          </PvIconField>
        </form>
      </template>
      <PvButton
        v-if="$can('apikeys:manage')"
        icon="pi pi-plus"
        :label="$t('developers.keys.create')"
        data-cy="btn-new"
        @click="openCreate"
      />
      <template #subtitle>{{ $t('developers.keys.intro') }}</template>
    </page-header>

    <table-card :error="loadError" @retry="fetchKeys">
      <PvDataTable :value="keys" data-key="id" :loading="loading && keys.length > 0">
        <template #empty>
          <table-skeleton v-if="loading" :cols="5" />
          <empty-state
            v-else-if="query"
            bare
            icon="pi-search"
            :title="$t('states.noMatches')"
            :text="$t('states.noMatchesText', { query })"
            :action-label="$t('states.clearFilters')"
            secondary
            @action="clearQuery"
          />
          <empty-state
            v-else
            bare
            icon="pi-key"
            :title="$t('developers.keys.emptyTitle')"
            :text="$t('developers.keys.emptyText')"
            :action-label="$can('apikeys:manage') ? $t('developers.keys.create') : ''"
            @action="openCreate"
          />
        </template>

        <PvColumn :header="$t('developers.keys.name')">
          <template #body="{ data }">
            <span class="ln-row-title">{{ data.name }}</span>
            <span class="ln-row-sub">
              {{ data.createdByName
                ? $t('developers.keys.createdBy', { date: shortDate(data.createdAt), name: data.createdByName })
                : $t('developers.keys.createdOn', { date: shortDate(data.createdAt) }) }}
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('developers.keys.key')">
          <template #body="{ data }">
            <span class="ln-mono">{{ maskedKey(data) }}</span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('developers.keys.scopes')">
          <template #body="{ data }">
            <span class="ln-attrs">
              <span v-for="s in data.scopes" :key="s" class="ln-attr ln-attr--mono">{{ s }}</span>
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('developers.keys.lastUsed')">
          <template #body="{ data }">
            <span class="ln-date">{{ data.lastUsedAt ? ago(data.lastUsedAt) : $t('developers.keys.never') }}</span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('globals.fields.status')">
          <template #body="{ data }">
            <status-tag
              v-if="data.status === 'revoked'"
              status="revoked"
              :label="$t('developers.keys.revoked')"
            />
            <status-tag
              v-else-if="isRotating(data)"
              status="pending"
              :label="$t('developers.keys.rotating', { left: timeLeft(data.prevExpiresAt) })"
            />
            <status-tag v-else status="active" :label="$t('developers.keys.active')" />
          </template>
        </PvColumn>

        <PvColumn v-if="$can('apikeys:manage')" header-style="width:56px" body-style="text-align:right">
          <template #body="{ data }">
            <button
              type="button"
              class="ln-row-menu"
              :aria-label="$t('globals.buttons.rowActions')"
              aria-haspopup="true"
              @click="(e) => openRowMenu(e, data)"
            >
              <i class="pi pi-ellipsis-h" />
            </button>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
    <PvMenu ref="rowMenu" :model="rowMenuItems" popup />

    <!-- API users from before keys existed. They still authenticate. -->
    <section v-if="legacy.length" class="ln-card dev-keys-legacy" data-cy="legacy-api-users">
      <header class="ln-card-head">
        <div>
          <h2 class="ln-card-title">{{ $t('developers.keys.legacyTitle') }}</h2>
          <p class="ln-small">{{ $t('developers.keys.legacyText') }}</p>
        </div>
      </header>
      <ul class="dev-keys-legacy-list">
        <li v-for="u in legacy" :key="u.id">
          <span class="dev-keys-legacy-who">
            <span class="ln-mono ln-row-title">{{ u.username }}</span>
            <span class="ln-row-sub">{{ u.userRole?.name }}</span>
          </span>
          <status-tag :status="u.status === 'enabled' ? 'active' : 'disabled'" :label="$t(`users.status.${u.status}`)" />
          <PvButton
            v-if="$can('apikeys:manage')"
            :label="$t('developers.keys.replace')"
            size="small"
            severity="secondary"
            outlined
            data-cy="btn-replace"
            @click="openReplace(u)"
          />
        </li>
      </ul>
    </section>

    <!-- Replace a legacy API user with a key -->
    <dialog-form
      v-model:visible="replace.open"
      :title="$t('developers.keys.replaceTitle', { name: replace.user?.username })"
      :submit-label="$t('developers.keys.create')"
      :busy="saving"
      :submit-disabled="!validName || form.scopes.length === 0"
      width="560px"
      @submit="onReplace"
    >
      <div class="ln-field">
        <label for="replace-name">{{ $t('developers.keys.name') }}</label>
        <PvInputText id="replace-name" v-model="form.name" class="ln-mono" autocomplete="off" spellcheck="false" :invalid="!!form.name && !validName" />
      </div>
      <scope-picker v-model="form.scopes" />
      <PvMessage v-if="uncovered.length" severity="warn" icon="pi pi-exclamation-triangle">
        <template v-if="uncovered[0] === '*'">{{ $t('developers.keys.replaceAdmin') }}</template>
        <template v-else>
          {{ $t('developers.keys.replaceUncovered') }}
          <span class="ln-attrs dev-keys-uncovered">
            <span v-for="p in uncovered" :key="p" class="ln-attr ln-attr--mono">{{ p }}</span>
          </span>
        </template>
      </PvMessage>
      <label class="ln-check-row" for="replace-disable">
        <PvCheckbox v-model="replace.disable" binary input-id="replace-disable" />
        {{ $t('developers.keys.replaceDisable', { name: replace.user?.username }) }}
      </label>
      <p class="ln-help">{{ $t('developers.keys.replaceDisableHelp') }}</p>
    </dialog-form>

    <div class="dev-keys-help">
      <section class="ln-card ln-card-pad">
        <h2 class="ln-card-title">{{ $t('developers.keys.rotate') }}</h2>
        <p class="ln-small">{{ $t('developers.keys.rotateHelp') }}</p>
      </section>
      <section class="ln-card ln-card-pad">
        <h2 class="ln-card-title">{{ $t('developers.keys.revoke') }}</h2>
        <p class="ln-small">{{ $t('developers.keys.revokeHelp') }}</p>
      </section>
      <section class="ln-card ln-card-pad">
        <h2 class="ln-card-title">{{ $t('developers.keys.scopes') }}</h2>
        <p class="ln-small">
          {{ $t('developers.keys.scopesHelp') }}
          <template v-for="s in API_KEY_SCOPES" :key="s">
            <br><code class="ln-mono">{{ s }}</code> {{ scopeHelp(s).toLowerCase() }}.
          </template>
        </p>
      </section>
    </div>

    <!-- Create: step 1 (details) -->
    <dialog-form
      v-model:visible="createOpen"
      :title="$t('developers.keys.newTitle')"
      :submit-label="$t('developers.keys.create')"
      :busy="saving"
      :submit-disabled="!form.name || form.scopes.length === 0"
      width="520px"
      @submit="onCreate"
    >
      <span class="ln-help">{{ $t('developers.keys.step', { n: 1 }) }} · {{ $t('developers.keys.stepDetails') }}</span>
      <div class="ln-field">
        <label for="key-name">{{ $t('developers.keys.name') }}</label>
        <PvInputText
          id="key-name"
          v-model="form.name"
          class="ln-mono"
          placeholder="checkout-svc"
          autocomplete="off"
          spellcheck="false"
          :invalid="!!form.name && !validName"
          data-cy="key-name"
        />
        <p class="ln-help">{{ $t('developers.keys.nameHelp') }}</p>
      </div>
      <scope-picker v-model="form.scopes" />
    </dialog-form>

    <!-- Edit scopes -->
    <dialog-form
      v-model:visible="scopesOpen"
      :title="$t('developers.keys.editScopes')"
      :submit-label="$t('globals.buttons.save')"
      :busy="saving"
      :submit-disabled="form.scopes.length === 0"
      width="520px"
      @submit="onSaveScopes"
    >
      <scope-picker v-model="form.scopes" />
    </dialog-form>

    <!-- Create: step 2 / rotate: copy the secret once -->
    <PvDialog
      v-model:visible="secretOpen"
      modal
      :closable="stored"
      :draggable="false"
      class="ln-dialog-form"
      :style="{ width: '560px', maxWidth: 'calc(100vw - 32px)' }"
    >
      <template #header>
        <div>
          <div class="p-dialog-title">{{ rotated ? $t('developers.keys.rotatedTitle') : $t('developers.keys.copyTitle') }}</div>
          <span v-if="!rotated" class="ln-help">
            {{ $t('developers.keys.step', { n: 2 }) }} · {{ $t('developers.keys.stepDetails') }} ✓ · {{ $t('developers.keys.stepCopy') }}
          </span>
        </div>
      </template>
      <div v-if="newKey" class="ln-dialog-form-body">
        <PvMessage severity="warn" icon="pi pi-exclamation-triangle">{{ $t('developers.keys.onlyOnce') }}</PvMessage>
        <div class="ln-field">
          <span class="ln-field-label">{{ $t('developers.keys.username') }}</span>
          <span class="ln-mono">{{ newKey.name }}</span>
        </div>
        <div class="ln-field">
          <label for="key-secret">{{ $t('developers.keys.secret') }}</label>
          <copy-field id="key-secret" :value="newKey.secret" strong />
        </div>
        <div class="ln-check-row">
          <span class="ln-small">{{ $t('developers.keys.scopes') }}</span>
          <span class="ln-attrs">
            <span v-for="s in newKey.scopes" :key="s" class="ln-attr ln-attr--mono">{{ s }}</span>
          </span>
        </div>
        <div class="ln-check-row">
          <PvCheckbox v-model="stored" binary input-id="key-stored" />
          <label for="key-stored">{{ $t('developers.keys.stored') }}</label>
        </div>
      </div>
      <template #footer>
        <PvButton
          v-if="newKey && newKey.scopes.includes('tx:send') && hasSendTest"
          :label="$t('developers.keys.sendTest')"
          severity="secondary"
          outlined
          :disabled="!stored"
          @click="goSendTest"
        />
        <span class="ln-dialog-form-spacer" />
        <PvButton :label="$t('developers.keys.done')" :disabled="!stored" data-cy="btn-done" @click="secretOpen = false" />
      </template>
    </PvDialog>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :confirm-text="confirm.typed"
      :danger="confirm.danger"
      :busy="saving"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, reactive, ref, watch, onMounted,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import { useTime } from '../../composables/useTime';
import {
  API_KEY_SCOPES, createAPIKey, deleteAPIKey, isRotating, listAPIKeys, maskedKey,
  revokeAPIKey, rotateAPIKey, updateAPIKeyScopes,
  type APIKey, type APIKeyScope, type APIKeyWithSecret,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import TableCard from '../../components/ui/TableCard.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import EmptyState from '../../components/ui/EmptyState.vue';
import StatusTag from '../../components/ui/StatusTag.vue';
import DialogForm from '../../components/ui/DialogForm.vue';
import ConfirmAction from '../../components/ui/ConfirmAction.vue';
import CopyField from '../../components/ui/CopyField.vue';
import type { RequestError } from '../../components/ui/ErrorState.vue';
import ScopePicker from './ScopePicker.vue';
import { keyNameFor, suggestScopes, uncoveredPermissions } from './legacyKeys';
import { userPayload } from '../userPayload';
import { getUsers as usersApi } from '../../api/generated/endpoints/users/users';

interface PopupMenu { toggle: (e: Event) => void }
// Users are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const { t } = useI18n();
const router = useRouter();
const { $utils, $can } = useGlobal();
const { shortDate, ago, timeLeft } = useTime();
const { refreshTick } = storeToRefs(useMainStore());

const keys = ref<APIKey[]>([]);
const loading = ref(true);
const loadError = ref<RequestError | null>(null);
const query = ref('');
const saving = ref(false);

const createOpen = ref(false);
const scopesOpen = ref(false);
const secretOpen = ref(false);
const stored = ref(false);
const rotated = ref(false);
const newKey = ref<APIKeyWithSecret | null>(null);
const editing = ref<APIKey | null>(null);
const form = reactive({ name: '', scopes: [] as APIKeyScope[] });

const rowMenu = ref<PopupMenu | null>(null);
const rowKey = ref<APIKey | null>(null);

const confirm = reactive({
  open: false, title: '', text: '', label: '', typed: '', danger: true, run: () => {},
});

const { listUsers, updateUser } = usersApi();
const legacy = ref<Loose[]>([]);
const replace = reactive({ open: false, disable: false, user: null as Loose | null });
const uncovered = computed(() => uncoveredPermissions(replace.user?.userRole, form.scopes));

const validName = computed(() => /^[a-z0-9][a-z0-9._-]{1,62}$/.test(form.name));
const hasSendTest = computed(() => router.hasRoute('devSendTest'));

function scopeHelp(s: APIKeyScope): string {
  return t(`developers.keys.scope.${s.replace(':', '_')}`);
}

function toRequestError(err: unknown): RequestError {
  const e = err as { response?: { status?: number; data?: { message?: string } }; config?: { method?: string; url?: string } };
  return {
    status: e.response?.status,
    method: e.config?.method,
    url: e.config?.url,
    message: e.response?.data?.message,
  };
}

function fetchKeys() {
  loading.value = true;
  listAPIKeys(query.value.trim())
    .then((data) => { keys.value = data; loadError.value = null; })
    .catch((err) => { loadError.value = toRequestError(err); })
    .finally(() => { loading.value = false; });
}

function clearQuery() {
  query.value = '';
  fetchKeys();
}

function openCreate() {
  form.name = '';
  form.scopes = [];
  createOpen.value = true;
}

function showSecret(k: APIKeyWithSecret, isRotation: boolean) {
  newKey.value = k;
  rotated.value = isRotation;
  stored.value = false;
  secretOpen.value = true;
}

function onCreate() {
  if (!validName.value) return;
  saving.value = true;
  createAPIKey(form.name.trim(), form.scopes)
    .then((k) => {
      createOpen.value = false;
      showSecret(k, false);
      fetchKeys();
    })
    .finally(() => { saving.value = false; });
}

function onSaveScopes() {
  if (!editing.value) return;
  saving.value = true;
  updateAPIKeyScopes(editing.value.id, form.scopes)
    .then((k) => {
      scopesOpen.value = false;
      $utils.toast(t('globals.messages.updated', { name: k.name }));
      fetchKeys();
    })
    .finally(() => { saving.value = false; });
}

function askRotate(k: APIKey) {
  Object.assign(confirm, {
    open: true,
    title: `${t('developers.keys.rotate')} ${k.name}?`,
    text: t('developers.keys.confirmRotate', { name: k.name }),
    label: t('developers.keys.rotate'),
    typed: '',
    danger: false,
    run: () => {
      saving.value = true;
      rotateAPIKey(k.id)
        .then((nk) => { confirm.open = false; showSecret(nk, true); fetchKeys(); })
        .finally(() => { saving.value = false; });
    },
  });
}

function askRevoke(k: APIKey) {
  Object.assign(confirm, {
    open: true,
    title: `${t('developers.keys.revoke')} ${k.name}?`,
    text: t('developers.keys.confirmRevoke', { name: k.name }),
    label: t('developers.keys.revoke'),
    typed: k.name,
    danger: true,
    run: () => {
      saving.value = true;
      revokeAPIKey(k.id)
        .then(() => { confirm.open = false; fetchKeys(); })
        .finally(() => { saving.value = false; });
    },
  });
}

function askDelete(k: APIKey) {
  Object.assign(confirm, {
    open: true,
    title: `${t('globals.buttons.delete')} ${k.name}?`,
    text: t('developers.keys.confirmDelete', { name: k.name }),
    label: t('globals.buttons.delete'),
    typed: k.status === 'active' ? k.name : '',
    danger: true,
    run: () => {
      saving.value = true;
      deleteAPIKey(k.id)
        .then(() => { confirm.open = false; fetchKeys(); })
        .finally(() => { saving.value = false; });
    },
  });
}

const rowMenuItems = computed<MenuItem[]>(() => {
  const k = rowKey.value;
  if (!k) return [];
  const items: MenuItem[] = [
    { label: t('developers.keys.copyName'), icon: 'pi pi-copy', command: () => navigator.clipboard?.writeText(k.name) },
  ];
  if (k.status === 'active') {
    items.push(
      {
        label: t('developers.keys.editScopes'),
        icon: 'pi pi-pencil',
        command: () => {
          editing.value = k;
          form.scopes = [...k.scopes];
          scopesOpen.value = true;
        },
      },
      { label: t('developers.keys.rotate'), icon: 'pi pi-refresh', command: () => askRotate(k) },
      { separator: true },
      {
        label: t('developers.keys.revoke'), icon: 'pi pi-ban', class: 'is-danger', command: () => askRevoke(k),
      },
    );
  } else {
    items.push(
      { separator: true },
      {
        label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(k),
      },
    );
  }
  return items;
});

function openRowMenu(e: Event, k: APIKey) {
  rowKey.value = k;
  rowMenu.value?.toggle(e);
}

function fetchLegacy() {
  if (!$can('users:get')) return;
  listUsers().then((d) => {
    legacy.value = (d as unknown as Loose[]).filter((u) => u.type === 'api');
  });
}

function openReplace(u: Loose) {
  replace.user = u;
  replace.disable = false;
  form.name = keyNameFor(u.username);
  form.scopes = suggestScopes(u.userRole);
  replace.open = true;
}

function onReplace() {
  const u = replace.user;
  if (!u || !validName.value) return;
  saving.value = true;
  createAPIKey(form.name.trim(), form.scopes)
    .then(async (k) => {
      replace.open = false;
      showSecret(k, false);
      fetchKeys();
      // Only after the key exists, so a failed create never locks anyone out.
      if (replace.disable && $can('users:manage')) {
        await updateUser(u.id, { ...userPayload(u), status: 'disabled' } as never);
        fetchLegacy();
      }
    })
    .finally(() => { saving.value = false; });
}

function goSendTest() {
  const name = newKey.value?.name;
  secretOpen.value = false;
  router.push({ name: 'devSendTest', query: { key: name } });
}

watch(refreshTick, () => { fetchKeys(); fetchLegacy(); });
onMounted(() => { fetchKeys(); fetchLegacy(); });
</script>

<style scoped lang="scss">
.dev-keys-legacy .ln-card-head .ln-small { margin: 2px 0 0; }
.dev-keys-legacy-list {
  list-style: none;
  margin: 0;
  padding: 0;
  li {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 18px;
    border-bottom: 1px solid var(--ln-border);
    &:last-child { border-bottom: 0; }
  }
}
.dev-keys-legacy-who { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.dev-keys-uncovered { margin-top: 6px; }
.dev-keys-help {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;

  .ln-small { margin: 6px 0 0; }
  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
}
</style>
