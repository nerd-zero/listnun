<template>
  <section>
    <form class="lm-form" @submit.prevent="onSubmit">
      <div class="lm-form-header">
        <div class="lm-form-title-row">
          <h3 class="lm-form-title">{{ isEditing ? data.name : $t('templates.newTemplate') }}</h3>
          <PvButton severity="secondary" @click="onTogglePreview" icon="pi pi-file" :label="$t('templates.preview') + ' (F9)'" />
        </div>
        <p v-if="isEditing" class="lm-form-meta">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" data-cy="id" />
        </p>
      </div>
      <div class="lm-form-body">
        <div class="name-type-row">
          <div class="lm-field name-field">
            <label class="lm-label">{{ $t('globals.fields.name') }}</label>
            <PvInputText :maxlength="200" ref="focusEl" v-model="form.name" name="name"
              :placeholder="$t('globals.fields.name')" required class="w-full" />
          </div>
          <div class="lm-field type-field">
            <label class="lm-label">{{ $t('globals.fields.type') }}</label>
            <PvSelect v-model="form.type" :disabled="isEditing"
              :options="[
                { label: $t('templates.typeCampaignHTML'), value: 'campaign' },
                { label: $t('templates.typeCampaignVisual'), value: 'campaign_visual' },
                { label: $t('templates.typeTransactional'), value: 'tx' },
              ]"
              option-label="label" option-value="value" class="w-full" />
          </div>
        </div>

        <div v-if="form.type === 'tx'" class="lm-field">
          <label class="lm-label">{{ $t('templates.subject') }}</label>
          <PvInputText :maxlength="200" v-model="form.subject" name="subject"
            :placeholder="$t('templates.subject')" required class="w-full" />
        </div>

        <template v-if="form.body !== null">
          <visual-editor v-if="form.type === 'campaign_visual'" name="body" :source="form.bodySource"
            @change="onChangeVisualEditor" height="70vh" />

          <div v-else class="lm-field">
            <label class="lm-label">{{ $t('templates.rawHTML') }}</label>
            <code-editor lang="html" v-model="form.body" name="body" />
          </div>
        </template>

        <p class="template-help">
          <template v-if="form.type === 'campaign'">
            {{ $t('templates.placeholderHelp', { placeholder: egPlaceholder }) }}
          </template>
          <a target="_blank" rel="noopener noreferer" href="https://listmonk.app/docs/templating">
            {{ $t('globals.buttons.learnMore') }}
          </a>
        </p>
      </div>

      <div class="lm-form-footer">
        <PvButton @click="$emit('close')" :label="$t('globals.buttons.close')" severity="secondary" />
        <PvButton v-if="$can('templates:manage')" type="submit" severity="primary" :loading="loading.templates"
          :label="$t('globals.buttons.save')" />
      </div>
    </form>
    <campaign-preview v-if="previewItem" is-post type="template" :title="previewItem.name"
      :template-type="previewItem.type" :body="form.body" @close="onTogglePreview" />
  </section>
</template>

<script setup lang="ts">
import {
  ref, reactive, nextTick, onMounted, onBeforeUnmount,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import CampaignPreview from '../components/CampaignPreview.vue';
import CodeEditor from '../components/CodeEditor.vue';
import VisualEditor from '../components/VisualEditor.vue';
import CopyText from '../components/CopyText.vue';
import { getTemplates as templatesApi } from '../api/generated/endpoints/templates/templates';

const props = withDefaults(defineProps<{
  data?: any;
  isEditing?: boolean;
}>(), { data: () => ({}), isEditing: false });

const emit = defineEmits(['finished', 'close']);

const { $utils } = useGlobal();
const { createTemplate, updateTemplate } = templatesApi();
const { t } = useI18n();
const { loading } = storeToRefs(useMainStore());

const focusEl = ref<any>(null);
const previewItem = ref<any>(null);
const egPlaceholder = '{{ template "content" . }}';
const form = reactive<any>({
  name: '', subject: '', type: 'campaign', optin: '', body: null, bodySource: null,
});

function onTogglePreview() {
  previewItem.value = !previewItem.value ? { ...form } : null;
}

function onPreviewShortcut(e: KeyboardEvent) {
  if (e.key === 'F9') { onTogglePreview(); e.preventDefault(); }
}

function onCreateTemplate() {
  createTemplate({
    name: form.name, type: form.type, subject: form.subject, body: form.body, body_source: form.bodySource,
  })
    .then((d: any) => { emit('finished'); emit('close'); $utils.toast(t('globals.messages.created', { name: d.name })); });
}

function onUpdateTemplate() {
  updateTemplate(props.data.id, {
    name: form.name, type: form.type, subject: form.subject, body: form.body, body_source: form.bodySource,
  })
    .then((d: any) => { emit('finished'); emit('close'); $utils.toast(`'${d.name}' updated`); });
}

function onSubmit() {
  if (props.isEditing) { onUpdateTemplate(); return; }
  onCreateTemplate();
}

function onChangeVisualEditor({ source, body }: any) {
  form.body = body;
  form.bodySource = source;
}

onMounted(() => {
  Object.assign(form, props.data);
  if (form.body === null || form.body === undefined) { form.body = ''; }
  nextTick(() => { focusEl.value?.$el?.focus(); });
  window.addEventListener('keydown', onPreviewShortcut);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onPreviewShortcut);
});
</script>

<style scoped lang="scss">
.lm-field { display: flex; flex-direction: column; gap: 0.35rem; margin-bottom: 0; }
.lm-label { display: block; font-size: 0.8rem; font-weight: 600; color: var(--lm-text); }

.name-type-row {
  display: grid;
  grid-template-columns: 1fr 200px;
  gap: 1rem;
  align-items: start;
}

.template-help { font-size: 0.78rem; color: var(--lm-text-subtle); margin: 0; }
</style>
