<template>
  <div class="roles-page">
    <div class="page-header">
      <h1 class="page-title">
        {{ $t(isUser ? 'users.userRoles' : 'users.listRoles') }}
        <span v-if="!isNaN(roles.length)" class="page-title-count">{{ roles.length }}</span>
      </h1>
      <PvButton v-if="$can('users:manage')" severity="primary" icon="pi pi-plus"
        data-cy="btn-new" @click="showNewForm('user')" :label="$t('globals.buttons.new')" />
    </div>

    <div class="table-card">
      <PvDataTable :value="roles" :loading="isLoading">
        <PvColumn field="role" :header="$t('users.role')" sortable>
          <template #body="{ data }">
            <div class="role-name-cell">
              <a href="#" class="row-name" @click.prevent="showEditForm(data, 'user')">{{ data.name }}</a>
              <PvTag v-if="data.id === 1" severity="success" size="small" value="Default" />
            </div>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('globals.fields.createdAt')"
          header-class="cy-created_at" sortable style="width:11rem">
          <template #body="{ data }">{{ $utils.niceDate(data.createdAt) }}</template>
        </PvColumn>

        <PvColumn field="updated_at" :header="$t('globals.fields.updatedAt')"
          header-class="cy-updated_at" sortable style="width:11rem">
          <template #body="{ data }">{{ $utils.niceDate(data.updatedAt) }}</template>
        </PvColumn>

        <PvColumn style="width:7rem; text-align:right">
          <template #body="{ data }">
            <div v-if="$can('roles:manage')" class="row-actions">
              <button type="button" class="row-action-btn" data-cy="btn-clone"
                v-tooltip.bottom="$t('globals.buttons.clone')"
                @click="$utils.prompt($t('globals.buttons.clone'), { placeholder: $t('globals.fields.name'), value: $t('campaigns.copyOf', { name: data.name }) }, (name) => onCloneRole(name, data))">
                <i class="pi pi-copy" />
              </button>
              <template v-if="data.id !== 1">
                <button type="button" class="row-action-btn" data-cy="btn-edit"
                  v-tooltip.bottom="$t('globals.buttons.edit')" @click="showEditForm(data, 'user')">
                  <i class="pi pi-pencil" />
                </button>
                <button type="button" class="row-action-btn row-action-btn--danger" data-cy="btn-delete"
                  v-tooltip.bottom="$t('globals.buttons.delete')" @click="onDeleteRole(data)">
                  <i class="pi pi-trash" />
                </button>
              </template>
            </div>
          </template>
        </PvColumn>

        <template #empty v-if="!isLoading">
          <empty-placeholder />
        </template>
      </PvDataTable>
    </div>

    <!-- Add / edit form modal -->
    <PvDialog v-model:visible="isFormVisible" :style="{ width: '700px' }" show-header="false" :closable="false" modal @hide="onFormClose">
      <role-form :data="curItem" :type="curType" :is-editing="isEditing" @finished="formFinished" @close="closeForm" />
    </PvDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useFormDialog } from '../composables/useFormDialog';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import RoleForm from './RoleForm.vue';
import { getRoles as rolesApi } from '../api/generated/endpoints/roles/roles';

const { $utils } = useGlobal();
const {
  listUserRoles, listListRoles, createUserRole, createListRole, deleteRole,
} = rolesApi();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { loading, userRoles, listRoles } = storeToRefs(store);

// Roles keeps its own showEditForm/showNewForm rather than the
// composable's (it also tracks curType, and showNewForm deliberately
// doesn't reset curItem) -- but still shares the curItem/isEditing/
// isFormVisible refs and closeForm helper the other CRUD views use.
const {
  curItem, isEditing, isFormVisible, closeForm,
} = useFormDialog();
const curType = ref<string | null>(null);

const isUser = computed(() => curType.value === 'user');
const isLoading = computed(() => (curType.value === 'user' ? loading.value.userRoles : loading.value.listRoles));
const roles = computed<any[]>(() => (isUser.value ? (userRoles.value as any) : (listRoles.value as any)));

function fetchRoles() {
  if (isUser.value) {
    store.setLoading({ model: 'userRoles', status: true });
    listUserRoles().then((data: any) => {
      store.setModelResponse({ model: 'userRoles', data });
    }).finally(() => { store.setLoading({ model: 'userRoles', status: false }); });
  } else {
    store.setLoading({ model: 'listRoles', status: true });
    listListRoles().then((data: any) => {
      store.setModelResponse({ model: 'listRoles', data });
    }).finally(() => { store.setLoading({ model: 'listRoles', status: false }); });
  }
}

function showEditForm(item: any) {
  curItem.value = item;
  curType.value = isUser.value ? 'user' : 'list';
  isFormVisible.value = true;
  isEditing.value = true;
}

function showNewForm() {
  isEditing.value = false;
  isFormVisible.value = true;
}

function formFinished() {
  fetchRoles();
}

function onFormClose() {
  if (route.params.id) {
    router.push({ name: 'users' });
  }
}

function onCloneRole(name: string, item: any) {
  const form: any = { name };
  let fn: (data: any) => Promise<any>;
  if (isUser.value) {
    fn = createUserRole;
    form.permissions = item.permissions;
  } else {
    fn = createListRole;
    form.lists = item.lists;
  }
  fn(form).then(() => {
    fetchRoles();
    $utils.toast(t('globals.messages.created', { name }));
  });
}

function onDeleteRole(item: any) {
  $utils.confirm(
    t('globals.messages.confirm'),
    () => {
      deleteRole(item.id).then(() => {
        fetchRoles();
        $utils.toast(t('globals.messages.deleted', { name: item.name }));
      });
    },
  );
}

onMounted(() => {
  curType.value = route.name === 'userRoles' ? 'user' : 'list';
  fetchRoles();
});
</script>

<style scoped lang="scss">
.roles-page { display: flex; flex-direction: column; gap: 1.5rem; }

.role-name-cell { display: flex; align-items: center; gap: 0.5rem; }
.row-name { color: var(--lm-text); font-weight: 500; text-decoration: none; &:hover { color: var(--lm-primary); } }
</style>
