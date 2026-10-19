<template>
  <dialog-form
    :visible="true"
    :title="isEditing ? (data.name || data.username) : $t('users.newUser')"
    :submit-label="$can('users:manage') ? (isEditing ? $t('globals.buttons.save') : $t('users.create')) : ''"
    :busy="busy"
    width="560px"
    @update:visible="(v) => { if (!v) $emit('close'); }"
    @submit="onSubmit"
  >
    <p v-if="isEditing" class="ln-small user-form-meta">
      #{{ data.id }}
      <span v-if="form.type === 'api'" class="ln-attr">{{ $t('users.type.api') }}</span>
    </p>

    <div class="ln-fields-2">
      <div class="ln-field">
        <label for="user-name">{{ $t('globals.fields.name') }}</label>
        <PvInputText id="user-name" ref="focusEl" v-model="form.name" name="name" :maxlength="200" />
      </div>
      <div class="ln-field">
        <label for="user-username">{{ $t('users.username') }}</label>
        <PvInputText
          id="user-username"
          v-model="form.username"
          name="username"
          :maxlength="200"
          required
          autocomplete="off"
          pattern="[a-zA-Z0-9_\-\.@]+$"
        />
      </div>
    </div>
    <p class="ln-help user-form-tight">{{ $t('users.usernameHelp') }}</p>

    <div v-if="form.type !== 'api'" class="ln-field">
      <label for="user-email">{{ $t('subscribers.email') }}</label>
      <PvInputText id="user-email" v-model="form.email" name="email" type="email" :maxlength="200" required />
    </div>

    <div class="ln-fields-2">
      <div class="ln-field">
        <label for="user-role">{{ $t('users.userRole') }}</label>
        <PvSelect v-model="form.userRoleId" input-id="user-role" name="user_role" :options="userRoleOptions" option-label="name" option-value="id" />
      </div>
      <div class="ln-field">
        <label for="user-list-role">{{ $t('users.listRole', 0) }}</label>
        <PvSelect v-model="form.listRoleId" input-id="user-list-role" name="list_role" :options="listRoleOptions" option-label="name" option-value="id" />
      </div>
    </div>

    <div class="ln-field">
      <label for="user-status">{{ $t('globals.fields.status') }}</label>
      <PvSelect v-model="form.status" input-id="user-status" name="status" :options="statusOptions" option-label="label" option-value="value" />
    </div>

    <template v-if="form.type !== 'api'">
      <label class="user-form-toggle" for="user-pw-login">
        <PvToggleSwitch v-model="form.passwordLogin" input-id="user-pw-login" />
        <span>
          <strong>{{ $t('users.passwordEnable') }}</strong>
          <span class="ln-small">{{ $t('users.passwordHelp') }}</span>
        </span>
      </label>
      <div v-if="form.passwordLogin" class="ln-fields-2">
        <div class="ln-field">
          <label for="user-password">{{ $t('users.password') }}</label>
          <PvPassword
            v-model="form.password"
            input-id="user-password"
            name="password"
            :minlength="8"
            :maxlength="200"
            :required="!isEditing"
            :feedback="false"
            toggle-mask
            :placeholder="isEditing ? $t('users.passwordKeep') : ''"
          />
        </div>
        <div class="ln-field">
          <label for="user-password2">{{ $t('users.passwordRepeat') }}</label>
          <PvPassword v-model="form.password2" input-id="user-password2" name="password2" :required="!!form.password" :feedback="false" />
        </div>
      </div>
      <p v-if="mismatch" class="ln-field-error" role="alert">{{ $t('users.passwordMismatch') }}</p>
    </template>
  </dialog-form>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import DialogForm from '../components/ui/DialogForm.vue';
import { userPayload } from './userPayload';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';
import { getRoles } from '../api/generated/endpoints/roles/roles';

// A user as the API returns it, camelCased.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = withDefaults(defineProps<{
  data?: Loose;
  isEditing?: boolean;
}>(), { data: () => ({}), isEditing: false });

const emit = defineEmits(['finished', 'close']);

const { $utils } = useGlobal();
const { createUser, updateUser } = usersApi();
const { listUserRoles, listListRoles } = getRoles();
const { t } = useI18n();
const store = useMainStore();
const { userRoles, listRoles } = storeToRefs(store);

const focusEl = ref<Loose | null>(null);
const busy = ref(false);
const mismatch = ref(false);
const form = reactive<Loose>({
  username: '',
  email: '',
  name: '',
  password: '',
  password2: '',
  passwordLogin: false,
  type: 'user',
  status: 'enabled',
  userRoleId: null,
  listRoleId: '',
});

const userRoleOptions = computed(() => (userRoles.value as Loose[]) || []);
const listRoleOptions = computed(() => [
  { name: t('globals.terms.none'), id: '' },
  ...((listRoles.value as Loose[]) || []),
]);
const statusOptions = computed(() => [
  { label: t('users.status.enabled'), value: 'enabled' },
  { label: t('users.status.disabled'), value: 'disabled' },
]);

function onSubmit() {
  mismatch.value = false;
  if (form.type !== 'api' && form.passwordLogin && form.password && form.password !== form.password2) {
    mismatch.value = true;
    return;
  }

  busy.value = true;
  const payload = userPayload(form);
  const req = props.isEditing ? updateUser(props.data.id, payload as never) : createUser(payload as never);
  req.then((d) => {
    const name = (d as unknown as Loose).name || form.username;
    emit('finished');
    emit('close');
    $utils.toast(t(props.isEditing ? 'globals.messages.updated' : 'globals.messages.created', { name }));
  }).finally(() => { busy.value = false; });
}

onMounted(() => {
  Object.assign(form, props.data, { password: '', password2: '' });
  if (props.data.userRole) form.userRoleId = props.data.userRole.id;
  form.listRoleId = props.data.listRole ? props.data.listRole.id : '';
  listUserRoles().then((data) => {
    store.setModelResponse({ model: 'userRoles', data });
    if (!form.userRoleId && userRoleOptions.value.length) form.userRoleId = userRoleOptions.value[0].id;
  });
  listListRoles().then((data) => { store.setModelResponse({ model: 'listRoles', data }); });
  nextTick(() => { focusEl.value?.$el?.focus(); });
});
</script>

<style scoped lang="scss">
.user-form-meta { margin: 0; display: flex; gap: 8px; align-items: center; }
.user-form-tight { margin-top: -10px; }
.user-form-toggle {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  cursor: pointer;
  > span { display: flex; flex-direction: column; gap: 2px; font-size: 14px; }
  strong { font-weight: 500; }
}
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
