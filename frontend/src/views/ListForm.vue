<template>
  <dialog-form
    :visible="visible"
    :title="isEditing ? data.name : $t('lists.newList')"
    :submit-label="canSave ? (isEditing ? $t('globals.buttons.save') : $t('lists.createList')) : ''"
    :busy="busy"
    :submit-disabled="!form.name.trim()"
    width="520px"
    @update:visible="(v) => { if (!v) $emit('close'); }"
    @submit="onSubmit"
  >
    <p v-if="isEditing" class="ln-small list-form-meta">
      #{{ data.id }} · <copy-text :text="data.uuid" />
    </p>

    <div class="ln-field">
      <label for="list-name">{{ $t('globals.fields.name') }}</label>
      <PvInputText id="list-name" ref="focusEl" v-model="form.name" name="name" :maxlength="200" required />
    </div>

    <fieldset class="list-form-type">
      <legend class="ln-field-label">{{ $t('lists.whoCanJoin') }}</legend>
      <label v-for="opt in typeOptions" :key="opt.value" class="list-form-card" :class="{ 'is-on': form.type === opt.value }">
        <PvRadioButton v-model="form.type" :value="opt.value" :input-id="`list-type-${opt.value}`" name="type" />
        <span>
          <strong>{{ opt.label }}</strong>
          <span class="ln-small">{{ opt.help }}</span>
        </span>
      </label>
    </fieldset>

    <label class="list-form-toggle" for="list-optin">
      <PvToggleSwitch v-model="doubleOptin" input-id="list-optin" />
      <span>
        <strong>{{ $t('lists.optins.double') }}</strong>
        <span class="ln-small">{{ $t('lists.doubleOptinHelp') }}</span>
      </span>
    </label>

    <div class="ln-field">
      <label for="list-desc">{{ $t('globals.fields.description') }} <span class="ln-muted">{{ $t('campaigns.optional') }}</span></label>
      <PvInputText id="list-desc" v-model="form.description" name="description" :maxlength="2000" />
    </div>

    <div class="ln-field">
      <label for="list-tags">{{ $t('globals.terms.tags') }} <span class="ln-muted">{{ $t('campaigns.optional') }}</span></label>
      <PvAutoComplete id="list-tags" v-model="form.tags" name="tags" :typeahead="false" multiple />
    </div>

    <label v-if="isEditing" class="list-form-toggle" for="list-archived">
      <PvToggleSwitch v-model="isArchived" input-id="list-archived" />
      <span>
        <strong>{{ $t('lists.archived') }}</strong>
        <span class="ln-small">{{ $t('lists.archivedHelp') }}</span>
      </span>
    </label>
  </dialog-form>
</template>

<script setup lang="ts">
import {
  computed, nextTick, onMounted, reactive, ref,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import CopyText from '../components/CopyText.vue';
import DialogForm from '../components/ui/DialogForm.vue';
import { getLists as listsApi } from '../api/generated/endpoints/lists/lists';

// A list as the API returns it, camelCased.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = withDefaults(defineProps<{
  data?: Loose;
  isEditing?: boolean;
  visible?: boolean;
}>(), { data: () => ({}), isEditing: false, visible: true });

const emit = defineEmits(['finished', 'close']);

const {
  $utils, $api, $can, $canList,
} = useGlobal();
const { createList, updateList } = listsApi();
const { t } = useI18n();

const focusEl = ref<Loose | null>(null);
const busy = ref(false);

const form = reactive({
  name: '',
  type: 'public',
  optin: 'double',
  status: 'active',
  description: '',
  tags: [] as string[],
});

const canSave = computed(() => $can('lists:manage_all') || (props.isEditing && $canList(props.data.id, 'list:manage')));

const typeOptions = computed(() => [
  { value: 'public', label: t('lists.types.public'), help: t('lists.publicHelp') },
  { value: 'private', label: t('lists.types.private'), help: t('lists.privateHelp') },
]);

const doubleOptin = computed({
  get: () => form.optin === 'double',
  set: (v: boolean) => { form.optin = v ? 'double' : 'single'; },
});

const isArchived = computed({
  get: () => form.status === 'archived',
  set: (v: boolean) => { form.status = v ? 'archived' : 'active'; },
});

function done(name: string, key: string) {
  ($api.getLists as (p: unknown) => void)({ minimal: true, per_page: 'all', status: 'active' });
  emit('finished');
  emit('close');
  $utils.toast(t(key, { name }));
}

function onSubmit() {
  busy.value = true;
  const req = props.isEditing ? updateList(props.data.id, form as never) : createList(form as never);
  req
    .then((data) => done((data as unknown as Loose).name, props.isEditing ? 'globals.messages.updated' : 'globals.messages.created'))
    .finally(() => { busy.value = false; });
}

onMounted(() => {
  Object.assign(form, props.data);
  if (!form.description) form.description = '';
  nextTick(() => { focusEl.value?.$el?.focus(); });
});
</script>

<style scoped lang="scss">
.list-form-meta { margin: -4px 0 0; display: inline-flex; gap: 6px; align-items: center; }
.list-form-type {
  border: 0;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;

  legend { padding: 0; margin-bottom: 6px; }
}
.list-form-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
  cursor: pointer;

  > span { display: flex; flex-direction: column; gap: 2px; }
  &.is-on { border-color: var(--ln-ink); box-shadow: 0 0 0 1px var(--ln-ink); }
}
.list-form-toggle {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;

  > span { display: flex; flex-direction: column; gap: 2px; }
  :deep(.p-toggleswitch) { flex: none; margin-top: 1px; }
}
</style>
