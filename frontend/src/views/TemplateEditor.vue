<template>
  <!-- TemplateEditor: one campaign template on its own page. Code on the
       left, the rendered preview on the right, Save and Send test on top. -->
  <div class="tpe" :class="{ 'is-visual': form.type === 'campaign_visual' }">
    <header class="tpe-head">
      <router-link :to="{ name: 'templates' }" class="tpe-back" :aria-label="$t('globals.terms.templates')">
        <i class="pi pi-chevron-left" aria-hidden="true" />
      </router-link>
      <PvInputText
        v-model="form.name"
        class="tpe-name"
        name="name"
        :maxlength="200"
        :placeholder="$t('templates.namePlaceholder')"
        :aria-label="$t('globals.fields.name')"
        :disabled="!canManage"
        data-cy="tpl-name"
      />
      <status-tag v-if="form.isDefault" status="active" :label="$t('templates.default')" />
      <!-- A new template picks HTML or Visual once; it can't change later. -->
      <PvSelectButton
        v-if="!isEditing"
        v-model="form.type"
        :options="typeOptions"
        option-label="label"
        option-value="value"
        :allow-empty="false"
        :aria-label="$t('globals.fields.type')"
      />
      <span v-else-if="form.type === 'campaign_visual'" class="ln-attr">{{ $t('templates.typeCampaignVisual') }}</span>
      <span v-if="isEditing" class="ln-small tpe-used">{{ usedLabel }}</span>
      <span class="tpe-spacer" />
      <PvButton
        v-if="canManage && form.type === 'campaign'"
        :label="$t('templates.sendTest')"
        severity="secondary"
        outlined
        data-cy="btn-send-test"
        @click="openTest"
      />
      <PvButton
        v-if="canManage"
        :label="$t('templates.save')"
        :loading="saving"
        :disabled="!dirty || !form.name.trim()"
        data-cy="btn-save"
        @click="save"
      />
    </header>

    <PvSkeleton v-if="loadingTpl" height="70vh" class="tpe-skel" />

    <div v-else class="tpe-body">
      <section class="tpe-code">
        <visual-editor
          v-if="form.type === 'campaign_visual'"
          name="body"
          :source="form.bodySource"
          height="calc(100vh - 190px)"
          :dark="isDark"
          @change="onChangeVisualEditor"
        />
        <template v-else>
          <div class="tpe-code-head">
            <span class="tpe-file">template.html</span>
            <span class="ln-mono tpe-lang">Go template</span>
          </div>
          <code-editor v-model="form.body" lang="html" theme="ink" :autofocus="false" name="body" class="tpe-editor" />
          <details class="tpe-vars">
            <summary>
              <strong>{{ $t('templates.variables') }}</strong>
              <span class="ln-small">{{ $t('templates.variablesHelp') }}</span>
            </summary>
            <dl>
              <template v-for="v in VARIABLES" :key="v.code">
                <dt><code class="ln-mono">{{ v.code }}</code></dt>
                <dd>{{ $t(v.label) }}</dd>
              </template>
            </dl>
          </details>
          <p class="ln-small tpe-must">
            <i18n-t keypath="templates.placeholderHint" tag="span">
              <template #tag><code class="ln-mono">{{ CONTENT_TAG }}</code></template>
            </i18n-t>
          </p>
          <p v-if="form.type === 'campaign' && !hasUnsubLink(form.body)" class="ln-small tpe-must tpe-warn" role="status">
            <i class="pi pi-exclamation-triangle" aria-hidden="true" />
            <i18n-t keypath="templates.noUnsubLink" tag="span">
              <template #tag><code class="ln-mono">{{ UNSUB_TAG }}</code></template>
            </i18n-t>
          </p>
        </template>
      </section>

      <section v-if="form.type !== 'campaign_visual'" class="tpe-preview" :aria-label="$t('templates.preview')">
        <div class="tpe-preview-head">
          <strong>{{ $t('templates.preview') }}</strong>
          <PvSelectButton
            v-model="device"
            :options="deviceOptions"
            option-label="label"
            option-value="value"
            :allow-empty="false"
            :aria-label="$t('templates.previewWidth')"
          />
        </div>
        <div class="tpe-stage">
          <PvMessage v-if="previewError" severity="error" class="tpe-error">{{ previewError }}</PvMessage>
          <iframe
            v-else
            class="tpe-frame"
            :class="`is-${device}`"
            sandbox=""
            :srcdoc="previewHtml"
            :title="$t('templates.preview')"
          />
        </div>
      </section>
    </div>

    <confirm-action
      v-model:visible="leave.open"
      :title="$t('templates.leaveTitle')"
      :text="$t('templates.leaveText', { name: form.name || $t('templates.newTemplate') })"
      :confirm-label="$t('templates.leaveWithoutSaving')"
      danger
      @confirm="leave.go"
    />

    <PvPopover ref="testPop" class="tpe-test">
      <form class="tpe-test-form" @submit.prevent="sendTest">
        <label for="tpe-test-to" class="ln-field-label">{{ $t('templates.sendTestTo') }}</label>
        <PvInputText id="tpe-test-to" v-model="testTo" type="text" autocomplete="email" data-cy="test-to" />
        <p class="ln-small">{{ $t('templates.sendTestHelp') }}</p>
        <PvButton type="submit" :label="$t('templates.sendTest')" :loading="testing" :disabled="!testTo.trim()" data-cy="btn-send-test-go" />
      </form>
    </PvPopover>
  </div>
</template>

<script setup lang="ts">
import {
  computed, onBeforeUnmount, onMounted, reactive, ref, watch,
} from 'vue';
import axios from 'axios';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import { useGlobal } from '../composables/useGlobal';
import { useDarkMode } from '../composables/useDarkMode';
import CodeEditor from '../components/CodeEditor.vue';
import VisualEditor from '../components/VisualEditor.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getTemplates as templatesApi } from '../api/generated/endpoints/templates/templates';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';
import { hasUnsubLink } from '../composables/unsubLink';

// A template as the API returns it, camelCased.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface Popover { show: (e: Event) => void; hide: () => void }

// How long to wait after typing before re-rendering the preview.
const PREVIEW_DEBOUNCE_MS = 500;
const CONTENT_TAG = '{{ template "content" . }}';
const UNSUB_TAG = '<a href="{{ UnsubscribeURL }}">Unsubscribe</a>';
// What a campaign template can use, shown one click away under the code.
const VARIABLES = [
  { code: CONTENT_TAG, label: 'templates.var.content' },
  { code: '{{ .Subscriber.Name }}', label: 'templates.var.name' },
  { code: '{{ .Subscriber.Email }}', label: 'templates.var.email' },
  { code: '{{ .Subscriber.Attribs.city }}', label: 'templates.var.attrib' },
  { code: '{{ .Campaign.Subject }}', label: 'templates.var.subject' },
  { code: '{{ UnsubscribeURL }}', label: 'templates.var.unsubscribe' },
  { code: '{{ ManageURL }}', label: 'templates.var.manage' },
  { code: '{{ MessageURL }}', label: 'templates.var.message' },
  { code: '{{ TrackView }}', label: 'templates.var.trackView' },
  { code: '{{ TrackLink "https://…" }}', label: 'templates.var.trackLink' },
];

// A new HTML template starts from a small working layout rather than an
// empty editor: it has the required content tag and an unsubscribe link.
const STARTER_HTML = `<!doctype html>
<html>
  <body style="margin:0;background:#f4f4f4;font-family:Helvetica,Arial,sans-serif;">
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
      <tr><td align="center" style="padding:24px 12px;">
        <table role="presentation" width="100%" style="max-width:600px;background:#ffffff;">
          <tr><td style="padding:32px;color:#222;font-size:15px;line-height:1.6;">
            {{ template "content" . }}
          </td></tr>
        </table>
        <p style="font-size:12px;color:#777;">
          <a href="{{ UnsubscribeURL }}" style="color:#777;">Unsubscribe</a>
          &middot; <a href="{{ MessageURL }}" style="color:#777;">View in browser</a>
        </p>
        {{ TrackView }}
      </td></tr>
    </table>
  </body>
</html>
`;

const route = useRoute();
const router = useRouter();
const { $utils, $can } = useGlobal();
const { isDark } = useDarkMode();
const {
  getTemplate, createTemplate, updateTemplate, testTemplate,
} = templatesApi();
const { getUserProfile } = usersApi();
const { t, n } = useI18n();

const canManage = computed(() => $can('templates:manage'));
const isEditing = computed(() => route.params.id !== 'new');
const loadingTpl = ref(isEditing.value);
const saving = ref(false);
const testing = ref(false);
const previewHtml = ref('');
const previewError = ref('');
const device = ref<'desktop' | 'mobile'>('desktop');
const testPop = ref<Popover | null>(null);
const testTo = ref('');
const form = reactive<Loose>({
  id: 0, name: '', type: 'campaign', body: '', bodySource: null, isDefault: false, campaignCount: 0,
});
// What was last saved, to tell when there's something to save.
const saved = ref('');
const snapshot = () => JSON.stringify([form.name, form.body, form.bodySource]);
const dirty = computed(() => snapshot() !== saved.value);

const typeOptions = computed(() => [
  { label: t('templates.typeCampaignHTML'), value: 'campaign' },
  { label: t('templates.typeCampaignVisual'), value: 'campaign_visual' },
]);

const deviceOptions = computed(() => [
  { label: t('templates.desktop'), value: 'desktop' },
  { label: t('templates.mobile'), value: 'mobile' },
]);

const usedLabel = computed(() => (form.campaignCount
  ? t('templates.usedBy', { num: n(form.campaignCount) }, form.campaignCount)
  : t('templates.unused')));

let timer: number | undefined;
function schedulePreview() {
  window.clearTimeout(timer);
  timer = window.setTimeout(renderPreview, PREVIEW_DEBOUNCE_MS);
}

// The preview posts the body straight to the server; errors (such as a
// missing content tag while typing) show in place of the preview.
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

// The visual editor reports a change once as it loads (it normalises the
// document); that first one is the saved state, not an edit.
let visualLoaded = false;
function onChangeVisualEditor({ source, body }: { source: unknown; body: string }) {
  const first = !visualLoaded;
  visualLoaded = true;
  form.body = body;
  form.bodySource = source;
  if (first && isEditing.value) saved.value = snapshot();
}

function save() {
  saving.value = true;
  const payload = {
    name: form.name.trim(), type: form.type, body: form.body, body_source: form.bodySource,
  };
  const req = isEditing.value ? updateTemplate(form.id, payload as never) : createTemplate(payload as never);
  req.then((d) => {
    const tpl = d as unknown as Loose;
    saved.value = snapshot();
    $utils.toast(t(isEditing.value ? 'globals.messages.updated' : 'globals.messages.created', { name: tpl.name }));
    if (!isEditing.value) router.replace({ name: 'template', params: { id: String(tpl.id) } });
  }).finally(() => { saving.value = false; });
}

function openTest(e: Event) {
  if (!testTo.value) {
    getUserProfile().then((p) => { testTo.value = testTo.value || String((p as unknown as Loose).email || ''); });
  }
  testPop.value?.show(e);
}

function sendTest() {
  const emails = testTo.value.split(/[,\s]+/).filter(Boolean);
  testing.value = true;
  testTemplate({ type: form.type, body: form.body, emails } as never)
    .then(() => {
      testPop.value?.hide();
      $utils.toast(t('templates.testSent', { to: emails.join(', ') }));
    })
    .finally(() => { testing.value = false; });
}

function load() {
  if (!isEditing.value) {
    form.type = route.query.type === 'campaign_visual' ? 'campaign_visual' : 'campaign';
    if (form.type === 'campaign') form.body = STARTER_HTML;
    saved.value = snapshot();
    renderPreview();
    return;
  }
  getTemplate(Number(route.params.id)).then((d) => {
    Object.assign(form, d as unknown as Loose);
    if (form.body === null || form.body === undefined) form.body = '';
    saved.value = snapshot();
    renderPreview();
  }).finally(() => { loadingTpl.value = false; });
}

// Leaving with unsaved changes asks first; so does closing the tab.
function beforeUnload(e: BeforeUnloadEvent) {
  if (dirty.value) e.preventDefault();
}
const leave = reactive({ open: false, go: () => {} });
onBeforeRouteLeave((_to, _from, next) => {
  if (!dirty.value) {
    next(true);
    return;
  }
  // Settle exactly once: closing the dialog any other way cancels.
  let settled = false;
  const stop = watch(() => leave.open, (open) => { if (!open) settle(false); });
  function settle(ok: boolean) {
    if (settled) return;
    settled = true;
    stop();
    leave.open = false;
    next(ok);
  }
  leave.go = () => settle(true);
  leave.open = true;
});

watch(() => [form.body, form.type], schedulePreview);
watch(() => form.type, (type) => {
  if (isEditing.value) return;
  if (type === 'campaign_visual' && form.body === STARTER_HTML) form.body = '';
  if (type === 'campaign' && !form.body) form.body = STARTER_HTML;
});
onMounted(() => {
  load();
  window.addEventListener('beforeunload', beforeUnload);
});
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload));
</script>

<style scoped lang="scss">
.tpe {
  display: flex;
  flex-direction: column;
  // The editor uses the whole window below the top bar, like Campaign.vue.
  margin: -28px -32px -48px;
  // Exactly the window below the top bar: code and preview scroll inside,
  // so the variables and the content-tag line stay in view.
  height: calc(100vh - 56px);
  @media (max-width: 991px) { margin: -20px -16px -32px; }
  @media (max-width: 1000px) { height: auto; }
}
.tpe-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 14px;
  padding: 14px 24px;
  border-bottom: 1px solid var(--ln-border);
  background: var(--ln-surface);
}
.tpe-back {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: var(--ln-text);
  &:hover { background: var(--ln-hover); }
}
.tpe-name {
  font-size: 18px;
  font-weight: 600;
  border-color: transparent;
  background: transparent;
  min-width: 0;
  flex: 0 1 360px;
  &:hover, &:focus { border-color: var(--ln-border); background: var(--ln-surface); }
}
.tpe-spacer { flex: 1; }
.tpe-skel { margin: 24px; }
.tpe-body {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  min-height: 0;
  overflow: hidden;
  .is-visual & { grid-template-columns: minmax(0, 1fr); }
  @media (max-width: 1000px) { grid-template-columns: minmax(0, 1fr); }
}
.tpe-code {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: #1b1d22;
  color: #d7d9de;
  .is-visual & { background: none; color: inherit; padding: 16px 24px; }
}
.tpe-code-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #2c2f36;
}
.tpe-file { font-size: 13px; font-weight: 600; padding: 4px 10px; border-radius: 4px; background: #2c2f36; }
.tpe-lang { font-size: 12px; color: #8b8f99; }
.tpe-editor {
  flex: 1;
  min-height: 0;
  overflow: auto;
  :deep(.cm-editor) { height: 100%; border-radius: 0; }
  @media (max-width: 1000px) { :deep(.cm-editor) { min-height: 460px; } }
}
.tpe-vars {
  border-top: 1px solid #2c2f36;
  padding: 12px 16px;
  font-size: 13px;
  summary { cursor: pointer; display: flex; gap: 10px; align-items: baseline; flex-wrap: wrap; }
  summary .ln-small { color: #8b8f99; }
  dl { display: grid; grid-template-columns: max-content minmax(0, 1fr); gap: 6px 16px; margin: 12px 0 4px; }
  dt code { color: #f0a868; }
  dd { margin: 0; color: #b8bcc6; }
}
.tpe-must {
  margin: 0;
  padding: 10px 16px;
  border-top: 1px solid #2c2f36;
  color: #8b8f99;
  code { color: #f0a868; }
}
.tpe-warn {
  color: #f0c36b;
  i { margin-right: 6px; font-size: 12px; }
}
.tpe-preview {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--ln-muted);
  .app-dark & { background: var(--ln-hover); }
}
.tpe-preview-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--ln-border);
}
.tpe-stage { flex: 1; padding: 24px; display: flex; justify-content: center; overflow: auto; }
.tpe-error { align-self: flex-start; width: 100%; }
.tpe-frame {
  width: 100%;
  min-height: 560px;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: #fff;
  transition: width 0.2s ease;
  &.is-mobile { width: 390px; max-width: 100%; }
}
.tpe-test-form { display: flex; flex-direction: column; gap: 8px; width: 300px; max-width: 80vw; }
</style>
