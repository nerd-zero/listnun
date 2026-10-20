<template>
  <div class="ln-page roles">
    <users-tabs>
      <PvButton v-if="$can('roles:manage')" icon="pi pi-plus" :label="$t('users.newRole')" data-cy="btn-new" @click="startNew" />
    </users-tabs>

    <error-state v-if="error" :title="$t('states.couldntLoad')" :error="error" @retry="load" />

    <div v-else class="roles-split">
      <nav class="ln-card roles-list" :aria-label="isUser ? $t('users.userRoles') : $t('users.listRoles')">
        <template v-if="isLoading && !roles.length">
          <PvSkeleton v-for="n in 3" :key="n" height="22px" class="roles-skel" />
        </template>
        <button
          v-for="r in roles"
          :key="r.id"
          type="button"
          class="roles-item"
          :class="{ 'is-on': selected?.id === r.id }"
          :aria-current="selected?.id === r.id ? 'true' : undefined"
          @click="select(r)"
        >
          <span class="roles-item-name">{{ r.name }}</span>
          <span v-if="counts[r.id] !== undefined" class="ln-small">{{ $t('users.numUsers', { num: counts[r.id] }, counts[r.id]) }}</span>
        </button>
        <button v-if="creating" type="button" class="roles-item is-on">
          <span class="roles-item-name">{{ $t('users.newRole') }}</span>
        </button>
        <p v-if="!isLoading && !roles.length && !creating" class="ln-small roles-none">{{ $t('users.noRoles') }}</p>
      </nav>

      <role-form
        v-if="creating || selected"
        :key="formKey"
        :data="creating ? draft : selected || {}"
        :is-editing="!creating"
        :type="isUser ? 'user' : 'list'"
        @saved="onSaved"
        @cancel="creating = false"
        @delete="askDelete"
        @clone="askClone"
      />
      <empty-state
        v-else-if="!isLoading"
        icon="pi-shield"
        :title="$t('users.pickRole')"
        :text="isUser ? $t('users.userRolesHelp') : $t('users.listRolesHelp')"
        :action-label="$can('roles:manage') ? $t('users.newRole') : ''"
        @action="startNew"
      />
    </div>

    <dialog-form
      v-model:visible="clone.open"
      :title="$t('globals.buttons.clone')"
      :submit-label="$t('globals.buttons.clone')"
      :busy="clone.busy"
      :submit-disabled="!clone.name.trim()"
      @submit="doClone"
    >
      <div class="ln-field">
        <label for="role-clone-name">{{ $t('globals.fields.name') }}</label>
        <PvInputText id="role-clone-name" v-model="clone.name" autofocus />
      </div>
    </dialog-form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="$t('users.deleteRole')"
      danger
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
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import UsersTabs from '../components/UsersTabs.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ErrorState from '../components/ui/ErrorState.vue';
import DialogForm from '../components/ui/DialogForm.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import RoleForm from './RoleForm.vue';
import { getRoles as rolesApi } from '../api/generated/endpoints/roles/roles';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

// Roles and users are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const { $utils, $can } = useGlobal();
const {
  listUserRoles, listListRoles, createUserRole, createListRole, deleteRole,
} = rolesApi();
const { listUsers } = usersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { loading, userRoles, listRoles } = storeToRefs(store);

const error = ref<Loose | null>(null);
const selected = ref<Loose | null>(null);
const creating = ref(false);
const draft = ref<Loose>({});
const formKey = ref(0);
const counts = ref<Record<number, number>>({});
const clone = reactive({
  open: false, busy: false, name: '', source: null as Loose | null,
});
const confirm = reactive({
  open: false, title: '', text: '', busy: false, run: () => {},
});

const isUser = computed(() => route.name === 'userRoles');
const isLoading = computed(() => (isUser.value ? loading.value.userRoles : loading.value.listRoles));
const roles = computed<Loose[]>(() => ((isUser.value ? userRoles.value : listRoles.value) as Loose[]) || []);

function select(r: Loose | null) {
  creating.value = false;
  selected.value = r;
  formKey.value += 1;
  const id = r ? String(r.id) : undefined;
  if (route.query.id !== id) router.replace({ query: id ? { id } : {} });
}

function startNew() {
  draft.value = {};
  creating.value = true;
  selected.value = null;
  formKey.value += 1;
}

// Pick the role named in ?id=, else keep the current one, else the first.
function reselect() {
  const want = parseInt(String(route.query.id || selected.value?.id || 0), 10);
  const r = roles.value.find((x) => x.id === want) || roles.value[0] || null;
  if (r?.id !== selected.value?.id || !selected.value) select(r);
  else selected.value = r;
}

function fetchRoles(): Promise<void> {
  const model = isUser.value ? 'userRoles' : 'listRoles';
  const fn = isUser.value ? listUserRoles : listListRoles;
  error.value = null;
  store.setLoading({ model, status: true });
  return fn().then((data) => { store.setModelResponse({ model, data }); })
    .catch((e) => { error.value = e; })
    .finally(() => { store.setLoading({ model, status: false }); });
}

// How many users hold each role, for the role list.
function fetchCounts() {
  if (!$can('users:get')) return;
  listUsers().then((data) => {
    const c: Record<number, number> = {};
    (data as unknown as Loose[]).forEach((u) => {
      const id = isUser.value ? u.userRole?.id : u.listRole?.id;
      if (id) c[id] = (c[id] || 0) + 1;
    });
    roles.value.forEach((r) => { c[r.id] = c[r.id] || 0; });
    counts.value = c;
  });
}

function load() {
  creating.value = false;
  fetchRoles().then(() => { reselect(); fetchCounts(); });
}

function onSaved(role: Loose) {
  creating.value = false;
  fetchRoles().then(() => {
    const r = roles.value.find((x) => x.id === role?.id);
    select(r || selected.value);
    fetchCounts();
  });
}

// Deleting a user role in use fails (ON DELETE RESTRICT); deleting a list
// role in use clears it from its users (ON DELETE SET NULL), so say which.
function deleteText(r: Loose): string {
  const num = counts.value[r.id];
  if (!num) return t('users.deleteRoleText');
  return t(isUser.value ? 'users.deleteRoleInUse' : 'users.deleteListRoleInUse', { num });
}

function askDelete(r: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('users.deleteRoleTitle', { name: r.name }),
    text: deleteText(r),
    run: () => {
      confirm.busy = true;
      deleteRole(r.id).then(() => {
        confirm.open = false;
        selected.value = null;
        router.replace({ query: {} });
        load();
        $utils.toast(t('globals.messages.deleted', { name: r.name }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askClone(r: Loose) {
  clone.source = r;
  clone.name = t('campaigns.copyOf', { name: r.name });
  clone.open = true;
}

function doClone() {
  const r = clone.source;
  if (!r) return;
  clone.busy = true;
  const req = isUser.value
    ? createUserRole({ name: clone.name.trim(), permissions: r.permissions } as never)
    : createListRole({ name: clone.name.trim(), lists: (r.lists || []).map((l: Loose) => ({ id: l.id, permissions: l.permissions })) } as never);
  req.then((d) => {
    clone.open = false;
    onSaved(d as unknown as Loose);
    $utils.toast(t('globals.messages.created', { name: clone.name }));
  }).finally(() => { clone.busy = false; });
}

watch(() => route.name, (name) => {
  if (name === 'userRoles' || name === 'listRoles') {
    selected.value = null;
    load();
  }
});

onMounted(load);
</script>

<style scoped lang="scss">
.roles-split {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  @media (max-width: 860px) { grid-template-columns: minmax(0, 1fr); }
}
.roles-list { display: flex; flex-direction: column; padding: 6px; gap: 2px; }
.roles-skel { margin: 8px 10px; }
.roles-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 9px 12px;
  border: 0;
  border-radius: 6px;
  background: none;
  color: var(--ln-text);
  font: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;

  &:hover { background: var(--ln-hover); }
  &.is-on { background: var(--ln-hover); font-weight: 500; }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: -2px; }
}
.roles-item-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.roles-none { margin: 0; padding: 10px 12px; }
</style>
