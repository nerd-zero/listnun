<template>
  <PvDialog
    :visible="true"
    modal
    maximizable
    :draggable="false"
    :header="isEditing ? $t('templates.editTemplate') : $t('templates.newTemplate')"
    class="ln-dialog-form tpl-form-dialog"
    :style="{ width: '1100px', maxWidth: 'calc(100vw - 32px)' }"
    @update:visible="(v) => { if (!v) $emit('close'); }"
  >
    <form :id="formId" class="tpl-form" :class="{ 'is-visual': form.type === 'campaign_visual' }" @submit.prevent="onSubmit">
      <div class="tpl-form-edit">
        <div class="tpl-form-row">
          <div class="ln-field">
            <label for="tpl-name">{{ $t('globals.fields.name') }}</label>
            <PvInputText id="tpl-name" ref="focusEl" v-model="form.name" name="name" :maxlength="200" required />
          </div>
          <div class="ln-field">
            <label for="tpl-type">{{ $t('globals.fields.type') }}</label>
            <PvSelect
              v-model="form.type"
              input-id="tpl-type"
              :disabled="isEditing"
              :options="typeOptions"
              option-label="label"
              option-value="value"
            />
          </div>
        </div>

        <div v-if="form.type === 'tx'" class="ln-field">
          <label for="tpl-subject">{{ $t('templates.subject') }}</label>
          <PvInputText id="tpl-subject" v-model="form.subject" name="subject" :maxlength="200" required />
          <p class="ln-help">
            {{ $t('templates.txEditorHint') }}
            <router-link :to="{ name: 'devTemplates' }">{{ $t('developers.templates.title') }}</router-link>
          </p>
        </div>

        <template v-if="form.body !== null">
          <visual-editor
            v-if="form.type === 'campaign_visual'"
            name="body"
            :source="form.bodySource"
            height="65vh"
            :dark="isDark"
            @change="onChangeVisualEditor"
          />
          <div v-else class="ln-field">
            <span class="ln-field-label">{{ $t('templates.rawHTML') }}</span>
            <code-editor v-model="form.body" lang="html" theme="ink" :autofocus="false" name="body" class="tpl-form-code" />
          </div>
        </template>

        <p class="ln-help">
          <template v-if="form.type === 'campaign'">
            <i18n-t keypath="templates.placeholderHint" tag="span">
              <template #tag><code class="ln-mono">{{ egPlaceholder }}</code></template>
            </i18n-t>
          </template>
          <a target="_blank" rel="noopener noreferrer" href="https://listmonk.app/docs/templating">{{ $t('globals.buttons.learnMore') }}</a>
        </p>
      </div>

      <aside v-if="form.type !== 'campaign_visual'" class="tpl-form-preview">
        <span class="ln-field-label">{{ $t('templates.preview') }}</span>
        <PvMessage v-if="previewError" severity="error">{{ previewError }}</PvMessage>
        <iframe v-else class="tpl-form-frame" sandbox="" :srcdoc="previewHtml" :title="$t('templates.preview')" />
      </aside>
    </form>

    <template #footer>
      <PvButton
        v-if="isEditing && form.type === 'campaign' && !data.isDefault && $can('templates:manage')"
        :label="$t('templates.makeDefault')"
        severity="secondary"
        outlined
        data-cy="btn-set-default"
        @click="$emit('make-default', data)"
      />
      <span class="ln-dialog-form-spacer" />
      <PvButton :label="$t('globals.buttons.cancel')" severity="secondary" outlined @click="$emit('close')" />
      <PvButton
        v-if="$can('templates:manage')"
        type="submit"
        :form="formId"
        :label="$t('globals.buttons.save')"
        :loading="saving"
        data-cy="btn-save"
      />
    </template>
  </PvDialog>
</template>

<script setup lang="ts">
import {
  computed, nextTick, onMounted, reactive, ref, watch,
} from 'vue';
import axios from 'axios';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import { useDarkMode } from '../composables/useDarkMode';
import CodeEditor from '../components/CodeEditor.vue';
import VisualEditor from '../components/VisualEditor.vue';
import { getTemplates as templatesApi } from '../api/generated/endpoints/templates/templates';

// A template as the API returns it, camelCased.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// How long to wait after typing before re-rendering the preview.
const PREVIEW_DEBOUNCE_MS = 500;

const props = withDefaults(defineProps<{
  data?: Loose;
  isEditing?: boolean;
}>(), { data: () => ({}), isEditing: false });

const emit = defineEmits(['finished', 'close', 'make-default']);

const { $utils } = useGlobal();
const { isDark } = useDarkMode();
const { createTemplate, updateTemplate } = templatesApi();
const { t } = useI18n();

const focusEl = ref<Loose | null>(null);
const saving = ref(false);
const previewHtml = ref('');
const previewError = ref('');
const formId = `tpl-form-${Math.random().toString(36).slice(2, 9)}`;
const egPlaceholder = '{{ template "content" . }}';
const form = reactive<Loose>({
  name: '', subject: '', type: 'campaign', body: null, bodySource: null,
});

const typeOptions = computed(() => [
  { label: t('templates.typeCampaignHTML'), value: 'campaign' },
  { label: t('templates.typeCampaignVisual'), value: 'campaign_visual' },
  { label: t('templates.typeTransactional'), value: 'tx' },
]);

// The preview posts the body straight to the server; errors (such as a
// missing content tag while typing) show inline instead of as toasts.
let timer: number | undefined;
function schedulePreview() {
  window.clearTimeout(timer);
  timer = window.setTimeout(renderPreview, PREVIEW_DEBOUNCE_MS);
}

function renderPreview() {
  if (form.type === 'campaign_visual' || !form.body) {
    previewHtml.value = '';
    return;
  }
  const fd = new FormData();
  fd.append('template_type', form.type);
  fd.append('body', form.body);
  axios.post('/api/templates/preview', fd, { responseType: 'text' })
    .then((r) => { previewHtml.value = r.data; previewError.value = ''; })
    .catch((err) => {
      let msg = String(err?.response?.data ?? err);
      try { msg = JSON.parse(msg)?.message || msg; } catch { /* not JSON: show as is */ }
      previewError.value = msg;
    });
}

function payload() {
  return {
    name: form.name, type: form.type, subject: form.subject, body: form.body, body_source: form.bodySource,
  };
}

function onSubmit() {
  saving.value = true;
  const req = props.isEditing ? updateTemplate(props.data.id, payload() as never) : createTemplate(payload() as never);
  req
    .then((d) => {
      emit('finished');
      emit('close');
      $utils.toast(t(props.isEditing ? 'globals.messages.updated' : 'globals.messages.created', { name: (d as unknown as Loose).name }));
    })
    .finally(() => { saving.value = false; });
}

function onChangeVisualEditor({ source, body }: { source: unknown; body: string }) {
  form.body = body;
  form.bodySource = source;
}

watch(() => [form.body, form.type], schedulePreview);

onMounted(() => {
  Object.assign(form, props.data);
  if (form.body === null || form.body === undefined) form.body = '';
  nextTick(() => { focusEl.value?.$el?.focus(); });
  renderPreview();
});
</script>

<style scoped lang="scss">
.tpl-form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.95fr);
  min-height: 60vh;

  &.is-visual { grid-template-columns: minmax(0, 1fr); }
  @media (max-width: 900px) { grid-template-columns: minmax(0, 1fr); }
}
.tpl-form-edit {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  border-right: 1px solid var(--ln-border);
}
.tpl-form-row { display: grid; grid-template-columns: minmax(0, 1fr) 220px; gap: 12px; }
.tpl-form-code :deep(.cm-editor) { min-height: 420px; max-height: 60vh; border-radius: 8px; }
.tpl-form-preview {
  padding: 20px;
  background: var(--ln-muted);
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  .app-dark & { background: var(--ln-hover); }
}
.tpl-form-frame {
  flex: 1;
  min-height: 420px;
  width: 100%;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: #fff;
}
</style>

<style lang="scss">
.p-dialog.tpl-form-dialog .p-dialog-content { padding: 0; }
</style>
