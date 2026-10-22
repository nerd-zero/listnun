<template>
  <div class="ln-page users">
    <users-tabs>
      <PvButton v-if="$can('users:manage')" icon="pi pi-plus" :label="$t('users.invite')" data-cy="btn-new" @click="showNewForm" />
    </users-tabs>

    <div class="users-toolbar">
      <form class="ln-search" role="search" @submit.prevent>
        <PvIconField>
          <PvInputIcon class="pi pi-search" />
          <PvInputText v-model="query" name="query" :placeholder="$t('users.search')" :aria-label="$t('users.search')" data-cy="query" />
        </PvIconField>
      </form>
    </div>

    <table-card :error="error" @retry="getUsers">
      <PvDataTable :value="shown" data-key="id" sort-field="id" :sort-order="1" :loading="loading.users && rows.length > 0">
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
            <span class="users-who">
              <span class="users-avatar" aria-hidden="true">
                <img v-if="data.avatar" :src="data.avatar" alt="">
                <template v-else>{{ initials(data) }}</template>
              </span>
              <span class="users-names">
                <a class="ln-row-title" :class="{ 'is-off': data.status === 'disabled' }" :href="$router.resolve(`/users/${data.id}`).href" @click.prevent="showEditForm(data)">
                  {{ data.name || data.username }}
                </a>
                <span class="ln-row-sub">{{ data.email || data.username }}</span>
              </span>
            </span>
          </template>
        </PvColumn>

        <PvColumn :header="$t('users.role', 1)" header-style="width:220px">
          <template #body="{ data }">
            <span class="ln-attrs">
              <router-link v-if="data.userRole" :to="{ name: 'userRoles', query: { id: data.userRole.id } }" class="ln-attr users-role">
                {{ data.userRole.name }}
              </router-link>
              <router-link v-if="data.listRole" :to="{ name: 'listRoles', query: { id: data.listRole.id } }" class="ln-attr users-role">
                <i class="pi pi-list" aria-hidden="true" /> {{ data.listRole.name }}
              </router-link>
            </span>
          </template>
        </PvColumn>

        <PvColumn field="status" :header="$t('globals.fields.status')" sortable header-class="cy-status" header-style="width:120px">
          <template #body="{ data }">
            <status-tag :status="data.status === 'enabled' ? 'active' : 'disabled'" :label="$t(`users.status.${data.status}`)" />
          </template>
        </PvColumn>

        <PvColumn :header="$t('users.twofa')" header-style="width:80px">
          <template #body="{ data }">
            <span v-if="data.twofaType && data.twofaType !== 'none'">{{ $t('users.on') }}</span>
            <span v-else class="users-off">{{ $t('users.off') }}</span>
          </template>
        </PvColumn>

        <PvColumn field="loggedinAt" :header="$t('users.lastLogin')" sortable header-style="width:140px">
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
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import UserForm from './UserForm.vue';
import { userPayload } from './userPayload';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

// Users are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

const { $utils } = useGlobal();
const {
  listUsers, getUser, deleteUser, updateUser,
} = usersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { ago } = useTime();
const { refreshTick, loading, profile } = storeToRefs(useMainStore());

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

// API users have their own tab; this one lists people.
const rows = computed(() => all.value.filter((u) => u.type !== 'api'));
const shown = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return rows.value;
  return rows.value.filter((u) => [u.name, u.username, u.email].some((v) => String(v || '').toLowerCase().includes(q)));
});

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
  const isSelf = (profile.value as Loose)?.id === u.id;
  const items: MenuItem[] = [
    { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => showEditForm(u) },
  ];
  if (!isSelf) {
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
.ln-row-title.is-off { color: var(--ln-text-3); }
.users-role { text-decoration: none; color: var(--ln-text); &:hover { text-decoration: none; background: var(--ln-hover); } }
.users-off { color: var(--ln-warn-text, #b45309); }
</style>
