<template>
  <div class="ln-page users">
    <users-tabs :members="loaded ? active.length : null">
      <PvButton v-if="$can('users:manage')" icon="pi pi-plus" :label="$t('users.invite')" data-cy="btn-new" @click="showNewForm" />
    </users-tabs>

    <!-- The one banner: members who can sign in with just a password. -->
    <PvMessage
      v-if="noTwofa.length && !twofaDismissed"
      severity="warn"
      icon="pi pi-shield"
      closable
      data-cy="twofa-banner"
      @close="dismissTwofa"
    >
      {{ $t('users.twofaBanner', { n: noTwofa.length }, noTwofa.length) }}
      <router-link v-if="$can('settings:manage')" :to="{ name: 'settings', params: { section: 'security' } }" class="users-fix">
        {{ $t('users.requireForEveryone') }}
      </router-link>
    </PvMessage>

    <div class="users-toolbar">
      <form class="ln-search" role="search" @submit.prevent>
        <PvIconField>
          <PvInputIcon class="pi pi-search" />
          <PvInputText v-model="query" name="query" :placeholder="$t('users.search')" :aria-label="$t('users.search')" data-cy="query" />
        </PvIconField>
      </form>
    </div>

    <table-card :error="error" @retry="getUsers">
      <PvDataTable :value="shown" data-key="id" :loading="loading.users && rows.length > 0">
        <template #empty>
          <table-skeleton v-if="loading.users || !loaded" :cols="5" />
          <empty-state
            v-else-if="query"
            bare
            icon="pi-search"
            :title="$t('states.noMatches')"
            :text="$t('states.noMatchesText', { query })"
            :action-label="$t('states.clearFilters')"
            secondary
            @action="query = ''"
          />
          <empty-state v-else bare icon="pi-user" :title="$t('users.emptyTitle')" :text="$t('users.emptyText')" />
        </template>

        <PvColumn field="username" :header="$t('users.user')" sortable header-class="cy-username">
          <template #body="{ data }">
            <span class="users-who" :class="{ 'is-off': data.status === 'disabled' }">
              <span class="users-avatar" aria-hidden="true">
                <img v-if="data.avatar" :src="data.avatar" alt="">
                <template v-else>{{ initials(data) }}</template>
              </span>
              <span class="users-names">
                <span class="users-name-line">
                  <a class="ln-row-title" :href="$router.resolve(`/users/${data.id}`).href" @click.prevent="showEditForm(data)">
                    {{ data.name || data.username }}
                  </a>
                  <span v-if="isSelf(data)" class="ln-row-sub">{{ $t('users.you') }}</span>
                  <span v-if="data.status === 'disabled'" class="ln-attr">{{ $t('users.status.disabled') }}</span>
                </span>
                <span class="ln-row-sub">{{ data.email || data.username }}</span>
              </span>
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('users.role', 1)" header-style="width:240px">
          <template #body="{ data }">
            <span class="ln-attrs">
              <PvSelect
                v-if="canChangeRole(data)"
                :key="`${data.id}-${roleRev}`"
                :model-value="data.userRole?.id"
                :options="roleOptions"
                option-label="name"
                option-value="id"
                size="small"
                class="users-role-select"
                :aria-label="$t('users.roleFor', { name: data.name || data.username })"
                data-cy="role-select"
                @update:model-value="(id: number) => setRole(data, id)"
              />
              <router-link v-else-if="data.userRole" :to="{ name: 'userRoles', query: { id: data.userRole.id } }" class="ln-attr users-role">
                {{ data.userRole.name }}
              </router-link>
              <router-link v-if="data.listRole" :to="{ name: 'listRoles', query: { id: data.listRole.id } }" class="ln-attr users-role">
                <i class="pi pi-list" aria-hidden="true" /> {{ data.listRole.name }}
              </router-link>
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('users.twofa')" header-style="width:100px" header-class="users-wide" body-class="users-wide">
          <template #body="{ data }">
            <span v-if="data.twofaType && data.twofaType !== 'none'">{{ $t('users.on') }}</span>
            <span v-else-if="!data.passwordLogin" v-tooltip.bottom="$t('users.ssoTwofa')" class="ln-small">{{ $t('users.sso') }}</span>
            <span v-else :class="{ 'users-off': data.status === 'enabled' }">{{ $t('users.off') }}</span>
          </template>
        </PvColumn>

        <PvColumn field="loggedinAt" :header="$t('users.lastActive')" sortable header-style="width:140px" header-class="users-wide" body-class="users-wide">
          <template #body="{ data }">
            <span v-if="data.loggedinAt" v-tooltip.bottom="$utils.niceDate(data.loggedinAt, true)" class="ln-date">{{ ago(data.loggedinAt) }}</span>
            <span v-else class="ln-date">{{ $t('users.never') }}</span>
          </template>
        </PvColumn>

        <PvColumn header-style="width:56px" body-style="text-align:right">
          <template #body="{ data }">
            <button
              v-if="$can('users:manage')"
              type="button"
              class="ln-row-menu"
              :aria-label="$t('globals.buttons.rowActions')"
              aria-haspopup="true"
              data-cy="btn-row-menu"
              @click="(e) => openMenu(e, data)"
            >
              <i class="pi pi-ellipsis-h" />
            </button>
          </template>
        </PvColumn>
      </PvDataTable>
    </table-card>
    <button
      v-if="off.length && !query"
      type="button"
      class="users-off-toggle"
      :aria-expanded="showOff"
      data-cy="toggle-deactivated"
      @click="showOff = !showOff"
    >
      <i :class="showOff ? 'pi pi-chevron-down' : 'pi pi-chevron-right'" aria-hidden="true" />
      {{ $t('users.deactivated', { n: off.length }, off.length) }}
    </button>
    <PvMenu ref="menu" :model="menuItems" popup />

    <user-form v-if="isFormVisible" :data="curItem || {}" :is-editing="isEditing" @finished="getUsers" @close="onFormClose" />

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
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
import { useRoute, useRouter } from 'vue-router';
import type { MenuItem } from 'primevue/menuitem';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import { useTime } from '../composables/useTime';
import UsersTabs from '../components/UsersTabs.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import UserForm from './UserForm.vue';
import { userPayload } from './userPayload';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

// Users are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

const { $utils, $can } = useGlobal();
const {
  listUsers, getUser, deleteUser, updateUser,
} = usersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { ago } = useTime();
const {
  refreshTick, loading, profile, serverConfig, userRoles,
} = storeToRefs(useMainStore());

const {
  curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
} = useFormDialog<Loose>({ newItem: () => ({ type: 'user' }) });
const all = ref<Loose[]>([]);
const loaded = ref(false);
const error = ref<Loose | null>(null);
const query = ref('');
const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Loose | null>(null);
const confirm = reactive({
  open: false, title: '', text: '', label: '', danger: true, busy: false, run: () => {},
});

// API users live under Developers -> API keys; this page lists people.
// Deactivated accounts sit behind a toggle under the table, except when
// searching, so a search still finds them.
const rows = computed(() => all.value.filter((u) => u.type !== 'api').sort((a, b) => a.id - b.id));
const active = computed(() => rows.value.filter((u) => u.status !== 'disabled'));
const off = computed(() => rows.value.filter((u) => u.status === 'disabled'));
const showOff = ref(false);
const shown = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return showOff.value ? [...active.value, ...off.value] : active.value;
  return rows.value.filter((u) => [u.name, u.username, u.email].some((v) => String(v || '').toLowerCase().includes(q)));
});

// Active members who sign in with a password and haven't turned on
// two-step verification. Single sign-on members rely on their identity
// provider. Once the workspace requires it, they're asked at sign-in, so
// there's nothing left to do here.
const TWOFA_DISMISS_KEY = 'ln-team-twofa-dismissed';
const twofaDismissed = ref(readDismissed());
const noTwofa = computed(() => {
  if ((serverConfig.value as Loose)?.require_2fa) return [];
  return active.value.filter((u) => u.passwordLogin && (!u.twofaType || u.twofaType === 'none'));
});

function readDismissed(): boolean {
  try { return localStorage.getItem(TWOFA_DISMISS_KEY) === '1'; } catch { return false; }
}

function dismissTwofa() {
  twofaDismissed.value = true;
  try { localStorage.setItem(TWOFA_DISMISS_KEY, '1'); } catch { /* private mode */ }
}

/** The built-in role holding every permission (auth.SuperAdminRoleID). */
const SUPER_ADMIN_ROLE_ID = 1;

const roleOptions = computed(() => ((userRoles.value as Loose[]) || []).map((r) => ({ id: r.id, name: r.name })));

function isSelf(u: Loose): boolean {
  return (profile.value as Loose)?.id === u.id;
}

// Changing your own role from here could lock you out mid-click; the edit
// form is still there for that. Deactivated accounts keep theirs as is.
function canChangeRole(u: Loose): boolean {
  return $can('users:manage') && !isSelf(u) && u.status !== 'disabled' && roleOptions.value.length > 0;
}

function saveRole(u: Loose, id: number): Promise<void> {
  return updateUser(u.id, { ...userPayload(u), user_role_id: id } as never).then(() => {
    getUsers();
    $utils.toast(t('globals.messages.updated', { name: u.name || u.username }));
  });
}

// The role pickers show what was picked even if it isn't saved; closing
// the confirmation rebuilds them so they show the saved role again.
const roleRev = ref(0);
watch(() => confirm.open, (open) => { if (!open) roleRev.value += 1; });

// Super Admin hands over the whole workspace, so that one change asks
// first instead of happening on a stray click.
function setRole(u: Loose, id: number) {
  if (!id || id === u.userRole?.id) return;
  if (id !== SUPER_ADMIN_ROLE_ID) {
    saveRole(u, id);
    return;
  }
  Object.assign(confirm, {
    open: true,
    title: t('users.makeSuperAdminTitle', { name: u.name || u.username }),
    text: t('users.makeSuperAdminText'),
    label: t('users.makeSuperAdmin'),
    danger: true,
    run: () => {
      confirm.busy = true;
      saveRole(u, id).then(() => { confirm.open = false; }).finally(() => { confirm.busy = false; });
    },
  });
}

function initials(u: Loose): string {
  const src = String(u.name || u.username || '?').trim();
  const parts = src.split(/\s+/).filter(Boolean);
  const two = parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : src.slice(0, 2);
  return two.toUpperCase();
}

function getUsers() {
  error.value = null;
  listUsers().then((resp) => { all.value = resp as unknown as Loose[]; })
    .catch((e) => { error.value = e; })
    .finally(() => { loaded.value = true; });
}

function onFormClose() {
  closeForm();
  if (route.params.id) router.push({ name: 'users' });
}

function setStatus(u: Loose, status: string) {
  updateUser(u.id, { ...userPayload(u), status } as never).then(() => {
    getUsers();
    $utils.toast(t('globals.messages.updated', { name: u.name || u.username }));
  });
}

function askDelete(u: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('users.deleteTitle', { name: u.name || u.username }),
    text: t('users.deleteText'),
    label: t('globals.buttons.delete'),
    danger: true,
    run: () => {
      confirm.busy = true;
      deleteUser(u.id).then(() => {
        confirm.open = false;
        getUsers();
        $utils.toast(t('globals.messages.deleted', { name: u.name || u.username }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

const menuItems = computed<MenuItem[]>(() => {
  const u = menuFor.value;
  if (!u) return [];
  const items: MenuItem[] = [
    { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => showEditForm(u) },
  ];
  if (!isSelf(u)) {
    items.push(u.status === 'enabled'
      ? { label: t('users.disable'), icon: 'pi pi-ban', command: () => setStatus(u, 'disabled') }
      : { label: t('users.enable'), icon: 'pi pi-check-circle', command: () => setStatus(u, 'enabled') });
    items.push({ separator: true }, {
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(u),
    });
  }
  return items;
});

function openMenu(e: Event, u: Loose) {
  menuFor.value = u;
  menu.value?.toggle(e);
}

watch(refreshTick, getUsers);

// /users and /users/:id share this view (meta.view), so the form follows
// the id in the URL instead of the page being rebuilt.
watch(() => route.params.id, (id) => {
  if (!id) return;
  getUser(parseInt(id as string, 10)).then((data) => { showEditForm(data as unknown as Loose); });
});

onMounted(() => {
  getUsers();
  if (route.params.id) {
    getUser(parseInt(route.params.id as string, 10)).then((data) => { showEditForm(data as unknown as Loose); });
  }
});
</script>

<style scoped lang="scss">
.users-toolbar { display: flex; gap: 12px; flex-wrap: wrap; }
.users-who { display: inline-flex; align-items: center; gap: 12px; min-width: 0; }
.users-avatar {
  flex: none;
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--ln-muted);
  border: 1px solid var(--ln-border);
  font-size: 12px;
  font-weight: 600;
  img { width: 100%; height: 100%; object-fit: cover; }
}
.users-names { display: flex; flex-direction: column; min-width: 0; }
.users-name-line { display: inline-flex; align-items: center; gap: 6px; min-width: 0; }
.users-who.is-off .ln-row-title, .users-who.is-off .users-avatar { opacity: .6; }
.users-role-select { min-width: 150px; }
@media (max-width: 600px) {
  :deep(.users-wide) { display: none; }
  .users-role-select { min-width: 0; }
}
.users-fix { color: inherit; font-weight: 600; text-decoration: underline; text-underline-offset: 3px; margin-left: 4px; }
.users-off-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  align-self: flex-start;
  padding: 4px 0;
  border: 0;
  background: none;
  color: var(--ln-text-2);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  &:hover { color: var(--ln-text); }
  i { font-size: 11px; }
}
.users-role { text-decoration: none; color: var(--ln-text); &:hover { text-decoration: none; background: var(--ln-hover); } }
.users-off { color: var(--ln-warn-text, #b45309); }
</style>
