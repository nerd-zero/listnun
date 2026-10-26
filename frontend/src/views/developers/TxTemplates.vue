<template>
  <div class="tpl-editor">
    <header class="tpl-head">
      <div class="tpl-head-title">
        <PvInputText
          v-if="current"
          v-model="form.name"
          class="tpl-name"
          :aria-label="$t('globals.fields.name')"
          :placeholder="$t('developers.templates.namePlaceholder')"
          :disabled="!canManage"
        />
        <span class="ln-attr ln-attr--filled">{{ $t('developers.templates.transactional') }}</span>
        <span v-if="current && current !== 'new'" class="ln-mono ln-muted">template_id {{ current }}</span>
        <span v-if="dirty" class="ln-small">{{ $t('developers.templates.unsaved') }}</span>
      </div>
      <div class="tpl-head-actions">
        <PvButton
          v-if="hasSendTest && current && current !== 'new'"
          :label="$t('developers.templates.sendTest')"
          severity="secondary"
          outlined
          @click="goSendTest"
        />
        <PvButton
          v-if="canManage"
          :label="$t('globals.buttons.save')"
          :loading="saving"
          :disabled="!dirty || !sampleValid || !form.name.trim()"
          data-cy="btn-save"
          @click="save"
        />
      </div>
    </header>

    <div class="tpl-grid">
      <!-- Template list -->
      <aside class="tpl-list" :aria-label="$t('globals.terms.templates')">
        <span class="ln-label">{{ $t('globals.terms.templates') }}&emsp;{{ templates.length }}</span>
        <table-skeleton v-if="loading" :rows="3" :cols="1" />
        <router-link
          v-for="tpl in templates"
          :key="tpl.id"
          :to="{ name: 'devTemplate', params: { id: tpl.id } }"
          class="tpl-list-item"
          :class="{ 'is-active': String(tpl.id) === current }"
        >
          <span class="tpl-list-name">{{ tpl.name }}</span>
          <span class="ln-row-sub">{{ $t('developers.templates.usedWeek', { n: tpl.usedThisWeek.toLocaleString() }) }}</span>
        </router-link>
        <router-link v-if="canManage" :to="{ name: 'devTemplate', params: { id: 'new' } }" class="tpl-list-new">
          {{ $t('developers.templates.new') }}
        </router-link>
      </aside>

      <!-- Editor -->
      <section v-if="current" class="tpl-edit">
        <div class="ln-field tpl-subject">
          <label for="tpl-subject">{{ $t('developers.templates.subject') }}</label>
          <PvInputText id="tpl-subject" v-model="form.subject" class="ln-mono" :disabled="!canManage" />
        </div>

        <PvTabs v-model:value="tab" class="tpl-tabs">
          <PvTabList>
            <PvTab value="html">{{ $t('developers.templates.bodyHtml') }}</PvTab>
            <PvTab value="text">{{ $t('developers.templates.plainText') }}</PvTab>
            <PvTab value="headers">{{ $t('developers.templates.headers') }}</PvTab>
          </PvTabList>
          <PvTabPanels>
            <PvTabPanel value="html">
              <code-editor
                :key="`html-${current}`"
                v-model="form.body"
                lang="html"
                theme="ink"
                :autofocus="false"
                :disabled="!canManage"
                class="tpl-code"
              />
            </PvTabPanel>
            <PvTabPanel value="text">
              <p class="ln-help">{{ $t('developers.templates.plainTextHelp') }}</p>
              <code-editor
                :key="`text-${current}`"
                v-model="form.altBody"
                lang="markdown"
                theme="ink"
                :autofocus="false"
                :disabled="!canManage"
                class="tpl-code"
              />
            </PvTabPanel>
            <PvTabPanel value="headers">
              <p class="ln-help">{{ $t('developers.templates.headersHelp') }}</p>
              <div v-for="(h, i) in headerRows" :key="i" class="tpl-header-row">
                <PvInputText v-model="h.name" class="ln-mono" placeholder="Reply-To" :aria-label="$t('developers.templates.headerName')" :disabled="!canManage" />
                <PvInputText v-model="h.value" class="ln-mono" placeholder="support@example.com" :aria-label="$t('developers.templates.headerValue')" :disabled="!canManage" />
                <PvButton
                  v-if="canManage"
                  icon="pi pi-times"
                  text
                  severity="secondary"
                  :aria-label="$t('globals.buttons.delete')"
                  @click="headerRows.splice(i, 1)"
                />
              </div>
              <PvButton
                v-if="canManage"
                icon="pi pi-plus"
                :label="$t('developers.templates.addHeader')"
                severity="secondary"
                outlined
                size="small"
                @click="headerRows.push({ name: '', value: '' })"
              />
            </PvTabPanel>
          </PvTabPanels>
        </PvTabs>

        <div class="tpl-sample">
          <div class="tpl-sample-head">
            <span class="ln-field-label">{{ $t('developers.templates.sampleData') }} <span class="ln-mono ln-muted">.Tx.Data</span></span>
            <span :class="sampleValid ? 'tpl-ok' : 'tpl-bad'">
              {{ sampleValid ? $t('developers.templates.validJson') : $t('developers.templates.invalidJson') }}
            </span>
          </div>
          <code-editor
            :key="`sample-${current}`"
            v-model="sampleText"
            lang="json"
            :autofocus="false"
            :disabled="!canManage"
            class="tpl-sample-code"
          />
        </div>
      </section>

      <!-- Preview -->
      <section v-if="current" class="tpl-preview" :class="{ 'is-dark': client === 'dark' }">
        <div class="tpl-preview-head">
          <strong>{{ $t('developers.templates.preview') }}</strong>
          <div class="tpl-preview-toggles">
            <PvSelectButton v-model="client" :options="clientOptions" option-label="label" option-value="value" :allow-empty="false" />
            <PvSelectButton v-model="device" :options="deviceOptions" option-label="label" option-value="value" :allow-empty="false" />
          </div>
        </div>
        <div class="tpl-preview-body">
          <div class="tpl-mail" :style="{ maxWidth: device === 'mobile' ? '375px' : '640px' }">
            <div class="tpl-mail-head">
              <span><strong>{{ sender.name }}</strong> &lt;{{ sender.address }}&gt;</span>
              <span>{{ preview?.subject }}</span>
            </div>
            <PvMessage v-if="previewError" severity="error" class="tpl-preview-error">{{ previewError }}</PvMessage>
            <iframe
              v-else
              class="tpl-mail-frame"
              sandbox=""
              :srcdoc="frameDoc"
              :title="$t('developers.templates.preview')"
            />
          </div>
          <p class="ln-help">{{ $t('developers.templates.previewHelp') }}</p>
        </div>
      </section>

      <section v-if="!current && !loading" class="tpl-empty">
        <empty-state
          icon="pi-file"
          :title="$t('developers.templates.emptyTitle')"
          :text="$t('developers.templates.emptyText')"
          :action-label="canManage ? $t('developers.templates.new') : ''"
          @action="$router.push({ name: 'devTemplate', params: { id: 'new' } })"
        />
      </section>
    </div>

    <confirm-action
      v-model:visible="leave.open"
      :title="$t('developers.templates.leaveTitle')"
      :text="$t('developers.templates.leaveText', { name: form.name })"
      :confirm-label="$t('developers.templates.leave')"
      @confirm="leave.go"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, reactive, ref, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import {
  onBeforeRouteLeave, onBeforeRouteUpdate, useRoute, useRouter,
} from 'vue-router';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import {
  createTxTemplate, listTxTemplates, previewTxTemplate, updateTxTemplate,
  type HeaderSet, type TxPreview, type TxTemplate, type TxTemplateInput,
} from '../../api/developers';
import CodeEditor from '../../components/CodeEditor.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import EmptyState from '../../components/ui/EmptyState.vue';
import ConfirmAction from '../../components/ui/ConfirmAction.vue';

// A starter body that shows the template language and the brand block.
const STARTER_BODY = `<table class="wrap" role="presentation">
  <tr><td class="body">
    <h1>Hi {{ .Subscriber.FirstName }}</h1>
    <p>Order #{{ .Tx.Data.order_id }} is confirmed.</p>
  </td></tr>
</table>
`;

// How long to wait after typing before re-rendering the preview.
const PREVIEW_DEBOUNCE_MS = 400;

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { $can, $utils } = useGlobal();
const { serverConfig } = storeToRefs(useMainStore());

const templates = ref<TxTemplate[]>([]);
const loading = ref(true);
const saving = ref(false);
const current = ref<string>('');
const tab = ref('html');
const client = ref<'light' | 'dark'>('light');
const device = ref<'desktop' | 'mobile'>('desktop');
const preview = ref<TxPreview | null>(null);
const previewError = ref('');

const form = reactive<TxTemplateInput>({
  name: '', subject: '', body: '', altBody: '', headers: [], sampleData: {},
});
const sampleText = ref('{}');
const headerRows = ref<{ name: string; value: string }[]>([]);
const savedSnapshot = ref('');

const leave = reactive({ open: false, go: () => {} });

const canManage = computed(() => $can('templates:manage'));
const hasSendTest = computed(() => router.hasRoute('devSendTest'));
const orgName = computed(() => (serverConfig.value as { organization_name?: string }).organization_name || 'listnun');

// The preview's From is usually already "Name <address>"; show it as an
// inbox would, falling back to the workspace name for a bare address.
const sender = computed(() => {
  const from = preview.value?.from || '';
  const m = from.match(/^\s*"?([^"<]*?)"?\s*<([^>]+)>\s*$/);
  if (m) return { name: m[1] || orgName.value, address: m[2] };
  return { name: orgName.value, address: from };
});

const clientOptions = computed(() => [
  { value: 'light', label: t('developers.templates.lightClient') },
  { value: 'dark', label: t('developers.templates.darkClient') },
]);
const deviceOptions = computed(() => [
  { value: 'desktop', label: t('developers.templates.desktop') },
  { value: 'mobile', label: t('developers.templates.mobile') },
]);

const parsedSample = computed<Record<string, unknown> | null>(() => {
  try {
    const v = JSON.parse(sampleText.value || '{}');
    return v && typeof v === 'object' && !Array.isArray(v) ? v : null;
  } catch {
    return null;
  }
});
const sampleValid = computed(() => parsedSample.value !== null);

function headersFromRows(): HeaderSet[] {
  return headerRows.value
    .filter((h) => h.name.trim())
    .map((h) => ({ [h.name.trim()]: h.value }));
}

function snapshot(): string {
  return JSON.stringify([form.name, form.subject, form.body, form.altBody, headersFromRows(), sampleText.value]);
}

const dirty = computed(() => !!current.value && snapshot() !== savedSnapshot.value);

// A dark email client: the sample renders on a dark canvas the way clients
// that support dark mode show light emails.
const frameDoc = computed(() => {
  const html = preview.value?.html || '';
  if (client.value !== 'dark') return html;
  const style = '<style>html{color-scheme:dark}body{background:#1b1f26;color:#ece9e1}</style>';
  return html.includes('</head>') ? html.replace('</head>', `${style}</head>`) : style + html;
});

function load(id: string) {
  current.value = id;
  previewError.value = '';
  if (id === 'new') {
    Object.assign(form, {
      name: '', subject: 'Your order #{{ .Tx.Data.order_id }} is confirmed', body: STARTER_BODY, altBody: '', headers: [],
    });
    sampleText.value = JSON.stringify({ order_id: '1042', total: '$49.00' }, null, 2);
    headerRows.value = [];
    savedSnapshot.value = snapshot();
    return;
  }

  const tpl = templates.value.find((x) => String(x.id) === id);
  if (!tpl) return;
  Object.assign(form, {
    name: tpl.name, subject: tpl.subject, body: tpl.body, altBody: tpl.altBody || '',
  });
  sampleText.value = JSON.stringify(tpl.sampleData || {}, null, 2);
  headerRows.value = (tpl.headers || []).flatMap((set) => Object.entries(set).map(([name, value]) => ({ name, value })));
  savedSnapshot.value = snapshot();
}

function fetchTemplates(select?: string) {
  loading.value = true;
  return listTxTemplates()
    .then((list) => {
      templates.value = list;
      const want = select || (route.params.id as string);
      if (want) {
        load(want);
      } else if (list.length) {
        // /developers/templates and /developers/templates/:id are separate
        // routes sharing this instance, so the update hook won't load it.
        load(String(list[0].id));
        router.replace({ name: 'devTemplate', params: { id: list[0].id } });
      }
    })
    .finally(() => { loading.value = false; });
}

let previewTimer: number | undefined;
function schedulePreview() {
  window.clearTimeout(previewTimer);
  previewTimer = window.setTimeout(renderPreview, PREVIEW_DEBOUNCE_MS);
}

function renderPreview() {
  if (!current.value || !parsedSample.value) return;
  previewTxTemplate({
    subject: form.subject, body: form.body, altBody: form.altBody, data: parsedSample.value,
  })
    .then((p) => { preview.value = p; previewError.value = ''; })
    .catch((err) => { previewError.value = err?.response?.data?.message || String(err); });
}

function save() {
  if (!parsedSample.value) return;
  const input: TxTemplateInput = {
    name: form.name.trim(),
    subject: form.subject,
    body: form.body,
    altBody: form.altBody,
    headers: headersFromRows(),
    sampleData: parsedSample.value,
  };
  saving.value = true;
  const req = current.value === 'new' ? createTxTemplate(input) : updateTxTemplate(Number(current.value), input);
  req
    .then((tpl) => {
      $utils.toast(t('globals.messages.updated', { name: tpl.name }));
      savedSnapshot.value = snapshot();
      return fetchTemplates(String(tpl.id)).then(() => {
        if (current.value !== String(tpl.id) || route.params.id === 'new') {
          router.replace({ name: 'devTemplate', params: { id: tpl.id } });
        }
      });
    })
    .finally(() => { saving.value = false; });
}

function goSendTest() {
  router.push({ name: 'devSendTest', query: { template: current.value } });
}

function guard(next: () => void) {
  if (!dirty.value) {
    next();
    return;
  }
  leave.go = () => {
    leave.open = false;
    savedSnapshot.value = snapshot();
    next();
  };
  leave.open = true;
}

onBeforeRouteUpdate((to) => new Promise<boolean>((resolve) => {
  guard(() => {
    load(to.params.id as string);
    resolve(true);
  });
  watch(() => leave.open, (open) => { if (!open) resolve(false); }, { once: true });
}));

onBeforeRouteLeave(() => new Promise<boolean>((resolve) => {
  guard(() => resolve(true));
  watch(() => leave.open, (open) => { if (!open) resolve(false); }, { once: true });
}));

watch([() => form.subject, () => form.body, () => form.altBody, sampleText, current], schedulePreview);

onMounted(() => fetchTemplates());
</script>

<style scoped lang="scss">
.tpl-editor {
  margin: -28px -32px -48px;
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 56px);

  @media (max-width: 991px) { margin: -20px -16px -32px; }
}
.tpl-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 14px 32px;
  border-bottom: 1px solid var(--ln-border);
}
.tpl-head-title { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; min-width: 0; }
.tpl-name {
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.01em;
  border-color: transparent;
  background: transparent;
  padding: 2px 6px;
  margin-left: -6px;
  // Grows with the name where the browser supports it.
  field-sizing: content;
  min-width: 12ch;
  max-width: min(560px, 50vw);

  &:hover:not(:disabled), &:focus { border-color: var(--ln-border-strong); background: var(--ln-card); }
}
.tpl-head-actions { display: flex; gap: 8px; }

.tpl-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) minmax(320px, 460px);

  @media (max-width: 1279px) { grid-template-columns: 200px minmax(0, 1fr); }
  @media (max-width: 767px) { grid-template-columns: minmax(0, 1fr); }
}
.tpl-list {
  border-right: 1px solid var(--ln-border);
  padding: 20px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;

  .ln-label { padding: 0 10px 6px; }
  @media (max-width: 767px) { border-right: 0; border-bottom: 1px solid var(--ln-border); }
}
.tpl-list-item {
  display: flex;
  flex-direction: column;
  padding: 8px 10px;
  border-radius: var(--ln-radius);
  text-decoration: none;
  color: var(--ln-text);

  &:hover { background: var(--ln-hover); text-decoration: none; }
  &.is-active { background: var(--ln-muted); .tpl-list-name { font-weight: 600; } }
  .app-dark &.is-active { background: var(--ln-hover); }
}
.tpl-list-new {
  margin-top: 8px;
  padding: 8px 10px;
  border: 1px dashed var(--ln-border-strong);
  border-radius: var(--ln-radius);
  text-align: center;
  color: var(--ln-text-2);
  text-decoration: none;

  &:hover { background: var(--ln-hover); color: var(--ln-text); text-decoration: none; }
}

.tpl-edit {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border-right: 1px solid var(--ln-border);
}
.tpl-subject { padding: 20px 20px 16px; border-bottom: 1px solid var(--ln-border); }
.tpl-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;

  :deep(.p-tablist) { padding: 0 20px; }
  :deep(.p-tabpanels) { padding: 0; flex: 1; }
  :deep(.p-tabpanel) { height: 100%; }
  .ln-help { padding: 12px 20px 0; }
}
.tpl-code {
  height: 100%;
  min-height: 420px;

  :deep(.cm-editor) { height: 100%; min-height: 420px; }
}
.tpl-header-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr) 36px;
  gap: 8px;
  padding: 8px 20px 0;
}
.tpl-tabs :deep(.p-tabpanel) > .p-button { margin: 12px 20px; }

.tpl-sample {
  border-top: 1px solid var(--ln-border);
  padding: 16px 20px 20px;
}
.tpl-sample-head {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
}
.tpl-ok { color: var(--ln-success-fg); font-size: 12px; }
.tpl-bad { color: var(--ln-danger-fg); font-size: 12px; }
// The global .code-editor is a fixed 65vh box; the sample fits its content.
.tpl-sample-code.code-editor { height: auto; min-height: 0; border: 0; }
.tpl-sample-code :deep(.cm-editor) {
  height: auto;
  min-height: 140px;
  max-height: 260px;
  border: 1px solid var(--ln-border-strong);
  border-radius: var(--ln-radius);
}

// On desktop the editor fills the window like the artboard: the code
// scrolls inside its pane and the sample data stays in view beneath it.
@media (min-width: 992px) {
  .tpl-editor { height: calc(100vh - 56px); min-height: 0; }
  .tpl-grid { min-height: 0; }
  .tpl-list, .tpl-preview { overflow-y: auto; }
  .tpl-edit { min-height: 0; }
  .tpl-tabs {
    min-height: 0;
    :deep(.p-tabpanels) { min-height: 0; overflow: hidden; }
    :deep(.p-tabpanel) { overflow-y: auto; }
  }
  .tpl-code {
    min-height: 0;
    :deep(.cm-editor) { min-height: 0; }
  }
  .tpl-sample { flex: none; }
}

.tpl-preview {
  background: var(--ln-muted);
  display: flex;
  flex-direction: column;
  min-width: 0;

  @media (max-width: 1279px) { grid-column: 1 / -1; border-top: 1px solid var(--ln-border); }
}
.tpl-preview-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--ln-border);
}
.tpl-preview-toggles { display: flex; gap: 8px; flex-wrap: wrap; }
.tpl-preview-body { padding: 24px; display: flex; flex-direction: column; gap: 20px; }
.tpl-mail {
  width: 100%;
  margin: 0 auto;
  background: var(--ln-card);
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  overflow: hidden;
}
.tpl-mail-head {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 16px;
  font-size: 13px;
  border-bottom: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
.tpl-mail-frame {
  display: block;
  width: 100%;
  height: 520px;
  border: 0;
  background: #fff;

  .is-dark & { background: #1b1f26; }
}
.tpl-preview-error { margin: 16px; }
.tpl-empty { grid-column: 2 / -1; padding: 32px; }
</style>
