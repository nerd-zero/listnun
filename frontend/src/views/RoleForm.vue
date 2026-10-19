<template>
  <form class="ln-card role-ed" data-cy="role-form" @submit.prevent="onSubmit">
    <header class="role-ed-head">
      <div class="role-ed-name">
        <label for="role-name">{{ $t('users.roleName') }}</label>
        <PvInputText id="role-name" ref="focusEl" v-model="form.name" name="name" :maxlength="200" :disabled="disabled" required />
      </div>
      <span class="ln-small">{{ type === 'user' ? $t('users.permsHint') : $t('users.listPermsHint') }}</span>
    </header>

    <!-- User role: permission groups -->
    <template v-if="type === 'user'">
      <div v-for="g in groups" :key="g.group" class="role-ed-group">
        <span class="role-ed-group-name">{{ groupLabel(g.group) }}</span>
        <div class="role-ed-perms">
          <label v-for="p in g.permissions" :key="p" class="role-ed-perm" :for="`perm-${p}`">
            <PvCheckbox v-model="form.permissions" :value="p" :input-id="`perm-${p}`" :disabled="disabled" />
            <span class="ln-mono">{{ p }}</span>
            <a
              v-if="p === 'subscribers:sql_query'"
              v-tooltip.bottom="$t('users.sqlQueryWarning')"
              href="https://listmonk.app/docs/roles-and-permissions/#subscriberssql_query"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="$t('users.sqlQueryWarning')"
              class="role-ed-warn"
            >
              <i class="pi pi-exclamation-triangle" />
            </a>
          </label>
        </div>
      </div>
      <div v-if="!disabled" class="role-ed-toggle">
        <button type="button" class="ln-link-btn" @click="toggleAll">
          {{ allOn ? $t('users.clearAll') : $t('users.selectAll') }}
        </button>
      </div>
    </template>

    <!-- List role: lists × view/manage -->
    <template v-else>
      <table v-if="form.lists.length" class="role-ed-lists">
        <thead>
          <tr>
            <th>{{ $t('globals.terms.list') }}</th>
            <th>{{ $t('users.viewSubscribers') }}</th>
            <th>{{ $t('users.manageSubscribers') }}</th>
            <th><span class="sr-only">{{ $t('globals.buttons.delete') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in form.lists" :key="l.id">
            <td>{{ l.name }}</td>
            <td>
              <PvCheckbox v-model="l.permissions" value="list:get" :input-id="`list-get-${l.id}`" :disabled="disabled" :aria-label="`${$t('users.viewSubscribers')}: ${l.name}`" />
            </td>
            <td>
              <PvCheckbox v-model="l.permissions" value="list:manage" :input-id="`list-manage-${l.id}`" :disabled="disabled" :aria-label="`${$t('users.manageSubscribers')}: ${l.name}`" />
            </td>
            <td class="role-ed-del">
              <button v-if="!disabled" type="button" class="ln-row-menu" :aria-label="$t('globals.buttons.delete')" @click="removeList(l.id)">
                <i class="pi pi-times" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="ln-small role-ed-none">{{ $t('users.noListPerms') }}</p>
      <div v-if="!disabled" class="role-ed-add">
        <PvSelect
          v-model="curList"
          :options="availableLists"
          option-label="name"
          option-value="id"
          filter
          :placeholder="$t('users.addList')"
          :disabled="!availableLists.length"
          :aria-label="$t('users.addList')"
        />
        <PvButton :label="$t('globals.buttons.add')" severity="secondary" outlined :disabled="!curList" @click="addList" />
      </div>
    </template>

    <footer class="role-ed-foot">
      <a href="https://listmonk.app/docs/roles-and-permissions" target="_blank" rel="noopener noreferrer" class="ln-small">
        {{ $t('globals.buttons.learnMore') }}
      </a>
      <span class="ln-dialog-form-spacer" />
      <template v-if="!disabled">
        <PvButton v-if="isEditing" :label="$t('users.deleteRole')" severity="danger" outlined data-cy="btn-delete" @click="$emit('delete', data)" />
        <PvButton v-if="isEditing" :label="$t('globals.buttons.clone')" severity="secondary" outlined data-cy="btn-clone" @click="$emit('clone', data)" />
        <PvButton v-else :label="$t('globals.buttons.cancel')" severity="secondary" outlined @click="$emit('cancel')" />
        <PvButton type="submit" :label="$t('users.saveRole')" :loading="busy" :disabled="!form.name?.trim()" data-cy="btn-save" />
      </template>
      <span v-else class="ln-small">{{ $t('users.adminRoleLocked') }}</span>
    </footer>
  </form>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { getRoles as rolesApi } from '../api/generated/endpoints/roles/roles';
import { translateOr } from '../composables/translateOr';

// Roles and lists are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PermGroup { group: string; permissions: string[] }

// The built-in admin role. It always has every permission and can't change.
const ADMIN_ROLE_ID = 1;

// Permissions a new user role leaves off: anything that reaches every list,
// settings, users or raw SQL is opted into deliberately.
function isDefaultOff(group: string, p: string): boolean {
  return ['admin', 'users'].includes(group) || p === 'subscribers:sql_query'
    || p.startsWith('lists:') || p.startsWith('settings:');
}

const props = withDefaults(defineProps<{
  data?: Loose;
  isEditing?: boolean;
  type?: 'user' | 'list';
}>(), { data: () => ({}), isEditing: false, type: 'user' });

const emit = defineEmits(['saved', 'cancel', 'delete', 'clone']);

const { $utils, $can } = useGlobal();
const {
  createUserRole, updateUserRole, createListRole, updateListRole,
} = rolesApi();
const { t } = useI18n();
const { serverConfig, lists } = storeToRefs(useMainStore());

const focusEl = ref<Loose | null>(null);
const busy = ref(false);
const curList = ref<number | null>(null);
const form = reactive<Loose>({ name: '', permissions: [] as string[], lists: [] as Loose[] });

const disabled = computed(() => props.data.id === ADMIN_ROLE_ID || !$can('roles:manage'));
const groups = computed<PermGroup[]>(() => (serverConfig.value as Loose).permissions || []);
const allPerms = computed(() => groups.value.flatMap((g) => g.permissions));
const allOn = computed(() => allPerms.value.every((p) => form.permissions.includes(p)));
const availableLists = computed<Loose[]>(() => {
  const have = new Set(form.lists.map((l: Loose) => l.id));
  return ((lists.value as Loose).results || []).filter((l: Loose) => !have.has(l.id));
});

function groupLabel(group: string): string {
  const key = `globals.terms.${group}`;
  return translateOr(t, key, group, 2);
}

function toggleAll() {
  form.permissions = allOn.value ? [] : [...allPerms.value];
}

function addList() {
  const l = availableLists.value.find((x) => x.id === curList.value);
  if (!l) return;
  form.lists.push({ id: l.id, name: l.name, permissions: ['list:get', 'list:manage'] });
  curList.value = null;
}

function removeList(id: number) {
  form.lists = form.lists.filter((l: Loose) => l.id !== id);
}

function onSubmit() {
  const payload: Loose = { name: form.name.trim() };
  if (props.type === 'user') {
    payload.permissions = form.permissions;
  } else {
    payload.lists = form.lists.map((l: Loose) => ({ id: l.id, permissions: l.permissions }));
  }

  let req: Promise<unknown>;
  if (props.type === 'user') {
    req = props.isEditing ? updateUserRole(props.data.id, payload as never) : createUserRole(payload as never);
  } else {
    req = props.isEditing ? updateListRole(props.data.id, payload as never) : createListRole(payload as never);
  }

  busy.value = true;
  req.then((d) => {
    const role = d as Loose;
    emit('saved', role);
    $utils.toast(t(props.isEditing ? 'globals.messages.updated' : 'globals.messages.created', { name: role.name || payload.name }));
  }).finally(() => { busy.value = false; });
}

onMounted(() => {
  form.name = props.data.name || '';
  form.permissions = [...(props.data.permissions || [])];
  form.lists = (props.data.lists || []).map((l: Loose) => ({ ...l, permissions: [...(l.permissions || [])] }));

  if (!props.isEditing && props.type === 'user' && !props.data.permissions) {
    form.permissions = groups.value.flatMap((g) => g.permissions.filter((p) => !isDefaultOff(g.group, p)));
  }
  if (!props.isEditing) nextTick(() => { focusEl.value?.$el?.focus(); });
});
</script>

<style scoped lang="scss">
.role-ed { display: flex; flex-direction: column; min-width: 0; }
.role-ed-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 14px 18px;
  border-bottom: 1px solid var(--ln-border);
}
.role-ed-name {
  display: flex;
  align-items: center;
  gap: 10px;
  label { font-size: 14px; font-weight: 500; white-space: nowrap; }
  .p-inputtext { width: 220px; max-width: 100%; }
}
.role-ed-group {
  display: grid;
  grid-template-columns: 170px minmax(0, 1fr);
  gap: 8px 16px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--ln-border);
  @media (max-width: 640px) { grid-template-columns: minmax(0, 1fr); }
}
.role-ed-group-name { font-size: 14px; font-weight: 500; padding-top: 2px; text-transform: capitalize; }
.role-ed-perms { display: flex; flex-wrap: wrap; gap: 8px 20px; }
.role-ed-perm {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  cursor: pointer;
}
.role-ed-warn { color: var(--ln-danger, #b42318); font-size: 12px; }
.role-ed-toggle { padding: 10px 18px 0; }
.role-ed-lists {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
  th, td { padding: 10px 18px; text-align: left; border-bottom: 1px solid var(--ln-border); }
  th { font-size: 12px; font-weight: 500; color: var(--ln-text-2); background: var(--ln-muted); }
  td:nth-child(2), td:nth-child(3), th:nth-child(2), th:nth-child(3) { width: 170px; }
}
.role-ed-del { width: 48px; text-align: right; }
.role-ed-none { margin: 0; padding: 18px; }
.role-ed-add { display: flex; gap: 8px; padding: 14px 18px 0; .p-select { flex: 1; max-width: 360px; } }
.role-ed-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 14px 18px;
  margin-top: 4px;
}
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}
</style>
