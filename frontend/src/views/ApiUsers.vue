<template>
  <div class="ln-page api-users">
    <users-tabs />

    <section class="ln-card ln-card-pad api-users-moved">
      <i class="pi pi-key" aria-hidden="true" />
      <div>
        <h2 class="ln-card-title">{{ $t('users.apiMovedTitle') }}</h2>
        <p class="ln-small">{{ $t('users.apiMovedText') }}</p>
      </div>
      <router-link v-if="$can('apikeys:get')" :to="{ name: 'devKeys' }" class="p-button p-component api-users-open">
        {{ $t('users.openApiKeys') }}
      </router-link>
    </section>

    <template v-if="rows.length || !loaded">
      <h2 class="ln-field-label api-users-label">{{ $t('users.legacyApiUsers') }}</h2>
      <table-card :error="error" @retry="getUsers">
        <PvDataTable :value="rows" data-key="id">
          <template #empty>
            <table-skeleton :cols="4" />
          </template>
          <PvColumn field="username" :header="$t('users.username')">
            <template #body="{ data }">
              <a class="ln-row-title ln-mono" :class="{ 'is-off': data.status === 'disabled' }" href="#" @click.prevent="showEditForm(data)">{{ data.username }}</a>
              <span v-if="data.name" class="ln-row-sub">{{ data.name }}</span>
            </template>
          </PvColumn>
          <PvColumn :header="$t('users.role', 1)" header-style="width:200px">
            <template #body="{ data }">
              <span v-if="data.userRole" class="ln-attr">{{ data.userRole.name }}</span>
            </template>
          </PvColumn>
          <PvColumn :header="$t('globals.fields.status')" header-style="width:120px">
            <template #body="{ data }">
              <status-tag :status="data.status === 'enabled' ? 'active' : 'disabled'" :label="$t(`users.status.${data.status}`)" />
            </template>
          </PvColumn>
          <PvColumn :header="$t('globals.fields.createdAt')" header-style="width:140px">
            <template #body="{ data }"><span class="ln-date">{{ shortDate(data.createdAt) }}</span></template>
          </PvColumn>
          <PvColumn header-style="width:88px" body-style="text-align:right">
            <template #body="{ data }">
              <span v-if="$can('users:manage')" class="api-users-actions">
                <button v-tooltip.bottom="$t('globals.buttons.edit')" type="button" class="ln-row-menu" :aria-label="$t('globals.buttons.edit')" data-cy="btn-edit" @click="showEditForm(data)">
                  <i class="pi pi-pencil" />
                </button>
                <button v-tooltip.bottom="$t('globals.buttons.delete')" type="button" class="ln-row-menu" :aria-label="$t('globals.buttons.delete')" data-cy="btn-delete" @click="askDelete(data)">
                  <i class="pi pi-trash" />
                </button>
              </span>
            </template>
          </PvColumn>
        </PvDataTable>
      </table-card>
    </template>

    <user-form v-if="isFormVisible" :data="curItem || {}" :is-editing="isEditing" @finished="getUsers" @close="closeForm" />

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="$t('users.deleteApiText')"
      :confirm-label="$t('globals.buttons.delete')"
      danger
      :busy="confirm.busy"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, onMounted,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import { useTime } from '../composables/useTime';
import UsersTabs from '../components/UsersTabs.vue';
import TableCard from '../components/ui/TableCard.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import UserForm from './UserForm.vue';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

// Users are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// API users (type "api") predate API keys. They still authenticate, so they
// stay listed here for editing and removal, but new integrations use keys.
const { $utils } = useGlobal();
const { listUsers, deleteUser } = usersApi();
const { t } = useI18n();
const { shortDate } = useTime();
const {
  curItem, isEditing, isFormVisible, showEditForm, closeForm,
} = useFormDialog<Loose>();

const all = ref<Loose[]>([]);
const loaded = ref(false);
const error = ref<Loose | null>(null);
const confirm = reactive({
  open: false, title: '', busy: false, run: () => {},
});

const rows = computed(() => all.value.filter((u) => u.type === 'api'));

function getUsers() {
  error.value = null;
  listUsers().then((d) => { all.value = d as unknown as Loose[]; })
    .catch((e) => { error.value = e; })
    .finally(() => { loaded.value = true; });
}

function askDelete(u: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('users.deleteTitle', { name: u.username }),
    run: () => {
      confirm.busy = true;
      deleteUser(u.id).then(() => {
        confirm.open = false;
        getUsers();
        $utils.toast(t('globals.messages.deleted', { name: u.username }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

onMounted(getUsers);
</script>

<style scoped lang="scss">
.api-users-moved {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  > .pi { font-size: 20px; color: var(--ln-text-3); }
  > div { flex: 1; min-width: 220px; }
  .ln-small { margin: 4px 0 0; }
}
.api-users-open { text-decoration: none; &:hover { text-decoration: none; } }
.api-users-label { margin: 8px 0 -6px; }
.api-users-actions { display: inline-flex; gap: 2px; }
.ln-row-title.is-off { color: var(--ln-text-3); }
</style>
