<template>
  <div class="users-page">
    <div class="page-header">
      <h1 class="page-title">
        {{ $t('globals.terms.users') }}
        <span v-if="!isNaN(users.length)" class="page-title-count">{{ users.length }}</span>
      </h1>
      <PvButton v-if="$can('users:manage')" severity="primary" icon="pi pi-plus"
        data-cy="btn-new" @click="showNewForm" :label="$t('globals.buttons.new')" />
    </div>

    <div class="table-card">
      <PvDataTable :value="users" :loading="loading.users" :rows="20"
        sort-field="createdAt" sort-order="1" v-model:selection="checked" data-key="id">
        <template #header>
          <div class="table-toolbar">
            <form class="search-form" @submit.prevent="getUsers()">
              <PvIconField>
                <PvInputIcon class="pi pi-search" />
                <PvInputText v-model="queryParams.query" name="query" ref="query"
                  class="search-input" placeholder="Search users…" data-cy="query" />
              </PvIconField>
            </form>
          </div>
        </template>

        <PvColumn selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="username" :header="$t('users.username')" header-class="cy-username" sortable>
          <template #body="{ data }">
            <div class="user-cell">
              <div class="user-name-row">
                <a class="row-name" :class="{ 'row-name--disabled': data.status === 'disabled' }"
                  :href="`/users/${data.id}`" @click.prevent="showEditForm(data)">
                  {{ data.username }}
                </a>
                <PvTag v-if="data.status === 'disabled'" severity="secondary" size="small"
                  :value="$t(`users.status.${data.status}`)" />
                <PvTag v-if="data.type === 'api'" severity="info" size="small">
                  <i class="pi pi-code" /> {{ $t(`users.type.${data.type}`) }}
                </PvTag>
              </div>
              <span v-if="data.name" class="user-fullname">{{ data.name }}</span>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="status" :header="$t('users.role')" header-class="cy-status" sortable>
          <template #body="{ data }">
            <div class="role-cell">
              <router-link :to="{ name: 'userRoles' }">
                <PvTag :severity="data.userRole.id === 1 ? 'success' : 'info'" size="small">
                  <i class="pi pi-user" /> {{ data.userRole.name }}
                </PvTag>
              </router-link>
              <router-link v-if="data.listRole" :to="{ name: 'listRoles' }">
                <PvTag severity="secondary" size="small">
                  <i class="pi pi-list" /> {{ data.listRole.name }}
                </PvTag>
              </router-link>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="name" :header="$t('subscribers.email')" header-class="cy-name" sortable>
          <template #body="{ data }">
            <a v-if="data.email" class="row-name" :href="`/users/${data.id}`" @click.prevent="showEditForm(data)">
              {{ data.email }}
            </a>
            <span v-else class="text-muted">—</span>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('globals.fields.createdAt')" header-class="cy-created_at" sortable style="width:10rem">
          <template #body="{ data }">{{ $utils.niceDate(data.createdAt) }}</template>
        </PvColumn>

        <PvColumn field="updated_at" :header="$t('globals.fields.updatedAt')" header-class="cy-updated_at" sortable style="width:10rem">
          <template #body="{ data }">{{ $utils.niceDate(data.updatedAt) }}</template>
        </PvColumn>

        <PvColumn field="last_login" :header="$t('users.lastLogin')" header-class="cy-updated_at" sortable style="width:10rem">
          <template #body="{ data }">{{ data.loggedinAt ? $utils.niceDate(data.loggedinAt, true) : '—' }}</template>
        </PvColumn>

        <PvColumn style="width:6rem; text-align:right" align-frozen="right">
          <template #body="{ data }">
            <div class="row-actions">
              <button v-if="$can('users:manage')" type="button" class="row-action-btn"
                data-cy="btn-edit" v-tooltip.bottom="$t('globals.buttons.edit')" @click="showEditForm(data)">
                <i class="pi pi-pencil" />
              </button>
              <button v-if="$can('users:manage')" type="button" class="row-action-btn row-action-btn--danger"
                data-cy="btn-delete" v-tooltip.bottom="$t('globals.buttons.delete')" @click="onDeleteUser(data)">
                <i class="pi pi-trash" />
              </button>
            </div>
          </template>
        </PvColumn>

        <template #empty v-if="!loading.users">
          <empty-placeholder />
        </template>
      </PvDataTable>
    </div>

    <!-- Add / edit form modal -->
    <PvDialog v-model:visible="isFormVisible" :style="{ width: '600px' }" show-header="false" :closable="false" modal @hide="onFormClose">
      <user-form :data="curItem" :is-editing="isEditing" @finished="formFinished" @close="closeForm" />
    </PvDialog>
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, watch, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import UserForm from './UserForm.vue';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

const { $utils } = useGlobal();
const { listUsers, getUser, deleteUser } = usersApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { refreshTick, loading } = storeToRefs(useMainStore());

const {
  curItem, isEditing, isFormVisible, showEditForm, showNewForm, closeForm,
} = useFormDialog();
const users = ref<any[]>([]);
const checked = ref<any[]>([]);
const queryParams = reactive({
  page: 1, query: '', orderBy: 'id', order: 'asc',
});

function formFinished() { getUsers(); }

function onFormClose() {
  if (route.params.id) router.push({ name: 'users' });
}

function getUsers() {
  listUsers().then((resp: any) => { users.value = resp; });
}

function onDeleteUser(item: any) {
  $utils.confirm(t('globals.messages.confirm'), () => {
    deleteUser(item.id).then(() => {
      getUsers();
      $utils.toast(t('globals.messages.deleted', { name: item.name }));
    });
  });
}

watch(() => refreshTick.value, () => { getUsers(); });

onMounted(() => {
  if (route.params.id) {
    getUser(parseInt(route.params.id as string, 10)).then((data: any) => { showEditForm(data); });
  } else {
    getUsers();
  }
});
</script>

<style scoped lang="scss">
.users-page { display: flex; flex-direction: column; gap: 1.5rem; }

:deep(.p-tag-secondary) {
  background: var(--lm-bg-subtle);
  color: var(--lm-text-secondary);
  border: 1px solid var(--lm-border);
}

.table-toolbar { display: flex; align-items: center; gap: 1rem; }
.search-form { flex: 0 0 260px; }
.search-input { width: 100%; }

.user-cell { display: flex; flex-direction: column; gap: 0.2rem; }
.user-name-row { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.user-fullname { font-size: 0.78rem; color: var(--lm-text-subtle); }
.role-cell { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.row-name { color: var(--lm-text); font-weight: 500; text-decoration: none; &:hover { color: var(--lm-primary); } &--disabled { color: var(--lm-text-subtle); } }
.text-muted { color: var(--lm-text-subtle); }
</style>
