<template>
  <!-- Two-way Data-Binding -->
  <section class="editor">
    <div class="editor-toolbar">
      <div class="editor-toolbar-left">
        <div class="field">
          <label class="block mb-1 text-sm font-medium">{{ $t('campaigns.format') }}</label>
          <PvSelect v-model="contentTypeSel" :options="contentTypeOptions" option-label="label" option-value="value"
            :disabled="disabled" name="content_type" data-cy="check-format" />
        </div>

        <div class="field" v-if="self.contentType !== 'visual'">
          <label class="block mb-1 text-sm font-medium">{{ $t('globals.terms.template') }}</label>
          <PvSelect v-model="templateId" :options="templateOptions" option-label="name" option-value="id"
            :disabled="disabled" name="template" />
        </div>

        <div v-else class="field">
          <PvButton v-if="!isVisualTplSelector" @click="onShowVisualTplSelector" severity="secondary"
            icon="pi pi-search" data-cy="btn-select-visual-tpl"
            :label="$t('campaigns.importVisualTemplate')" />
          <template v-else>
            <label class="block mb-1 text-sm font-medium">{{ $t('globals.terms.template') }}</label>
            <div class="flex items-center gap-2">
              <PvSelect v-model="visualTemplateId" :options="templateOptions" option-label="name" option-value="id"
                @change="() => isVisualTplDisabled = false" name="template" :disabled="disabled"
                class="copy-visual-template-list" />
              <PvButton :disabled="disabled || isVisualTplDisabled || !visualTemplateId"
                @click="onImportVisualTpl" severity="primary" icon="pi pi-save"
                data-cy="btn-save-visual-tpl" :label="$t('globals.terms.import')">
                <PvProgressSpinner v-if="loading.templates" style="width:1rem;height:1rem" />
              </PvButton>
            </div>
          </template>
        </div>
      </div>

      <div class="editor-toolbar-right">
        <PvButton @click="onTogglePreview" severity="secondary" outlined data-cy="btn-preview"
          aria-keyshortcuts="F9">
          <i class="pi pi-eye" /><span class="has-kbd">{{ $t('campaigns.preview') }} <span class="kbd">F9</span></span>
        </PvButton>
      </div>
    </div>

    <!-- wsywig //-->
    <richtext-editor v-if="self.contentType === 'richtext'" :disabled="disabled" v-model="self.body" />

    <!-- visual editor //-->
    <visual-editor v-if="self.contentType === 'visual'" :source="self.bodySource" @change="onVisualEditorChange"
      height="65vh" ref="visualEditorRef" :dark="isDark" />

    <!-- raw html editor //-->
    <code-editor lang="html" v-if="self.contentType === 'html'" v-model="self.body" key="editor-html" />

    <!-- markdown editor //-->
    <code-editor lang="markdown" v-if="self.contentType === 'markdown'" v-model="self.body" key="editor-markdown" />

    <!-- plain text //-->
    <PvTextarea v-if="self.contentType === 'plain'" v-model="self.body" name="content" ref="plainEditorRef"
      class="plain-editor" />

    <!-- campaign preview //-->
    <campaign-preview v-if="isPreviewing" is-post @close="onTogglePreview" type="campaign" :id="id" :title="title"
      :content-type="self.contentType" :template-id="templateId" :body="self.body" />
  </section>
</template>

<script setup lang="ts">
import {
  ref, computed, watch, nextTick, onMounted, onBeforeUnmount,
} from 'vue';
import { html as beautifyHTMLLib } from 'js-beautify';
import TurndownService from 'turndown';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useDarkMode } from '../composables/useDarkMode';
import CampaignPreview from './CampaignPreview.vue';
import VisualEditor from './VisualEditor.vue';
import RichtextEditor from './RichtextEditor.vue';
import markdownToVisualBlock from './editor';
import CodeEditor from './CodeEditor.vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';
import { getTemplates as templatesApi } from '../api/generated/endpoints/templates/templates';

const turndown = new TurndownService();

const props = withDefaults(defineProps<{
  contentTypes?: Record<string, string>;
  id?: number;
  title?: string;
  disabled?: boolean;
  templates?: any[];
  modelValue?: { body: string; bodySource: string | null; contentType: string; templateId: number | null };
}>(), {
  contentTypes: () => ({}),
  id: 0,
  title: '',
  disabled: false,
  templates: () => [],
  modelValue: () => ({
    body: '', bodySource: null, contentType: '', templateId: null,
  }),
});

const emit = defineEmits(['update:modelValue']);
const { $utils, $events } = useGlobal();
const { isDark } = useDarkMode();
const { setCampaignContent } = campaignsApi();
const { getTemplate } = templatesApi();
const { t } = useI18n();
const { loading } = storeToRefs(useMainStore());

const isPreviewing = ref(false);
const isVisualTplSelector = ref(false);
const isVisualTplDisabled = ref(false);
const contentTypeSel = ref(props.modelValue.contentType);
const templateId = ref<number | null>(null);
const visualTemplateId = ref<number | null>(null);
const visualEditorRef = ref<any>(null);
const plainEditorRef = ref<any>(null);

const self = computed({
  get: () => props.modelValue,
  set: (val: any) => emit('update:modelValue', val),
});

const validTemplates = computed(() => {
  const typ = self.value.contentType === 'visual' ? 'campaign_visual' : 'campaign';
  return (props.templates || []).filter((tpl: any) => tpl.type === typ);
});

const contentTypeOptions = computed(() => Object.entries(props.contentTypes).map(([value, label]) => ({ value, label })));

const templateOptions = computed(() => [{ id: null, name: t('globals.terms.none') }, ...validTemplates.value]);

function trimLines(str: string, removeEmptyLines: boolean) {
  const out = str.split('\n');
  for (let i = 0; i < out.length; i += 1) {
    const line = out[i].trim();
    if (removeEmptyLines) { out[i] = line; } else if (line === '') { out[i] = ''; }
  }
  return out.join('\n').replace(/\n\s*\n\s*\n/g, '\n\n');
}

function beautifyHTML(str: string) {
  let s = trimLines(str.replace(/(<(?!(\/)?a|span)([^>]+)>)/ig, '\n$1\n'), true);
  s = s.replace(/\n+/g, '\n');
  return beautifyHTMLLib(s, {
    indent_size: 4,
    indent_char: ' ',
    max_preserve_newlines: 2,
    inline: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'b', 'strong', 'span', 'em', 'i', 'code', 'a'],
  }).trim();
}

function setDefaultTemplate() {
  if (self.value.contentType === 'visual') {
    visualTemplateId.value = validTemplates.value[0]?.id || null;
  } else {
    if (templateId.value) return;
    const defaultTpl = validTemplates.value.find((tpl: any) => tpl.isDefault === true);
    templateId.value = defaultTpl?.id || validTemplates.value[0]?.id || null;
  }
}

function convertContentType(to: string, from: string) {
  let body = self.value.body ?? '';
  let bodySource: string | null = null;
  let skip = false;
  let isHTML = false;

  if (from === 'richtext' || from === 'html' || from === 'visual') {
    const d = document.createElement('div');
    d.innerHTML = body;
    body = beautifyHTML(d.innerHTML.trim());
    isHTML = true;
  }

  if (isHTML) {
    switch (to) {
      case 'plain': {
        const d = document.createElement('div');
        d.innerHTML = body;
        body = trimLines(d.innerText.trim(), true);
        break;
      }
      case 'markdown': {
        body = turndown.turndown(body).replace(/\n\n+/ig, '\n\n');
        break;
      }
      case 'visual': {
        const md = turndown.turndown(body).replace(/\n\n+/ig, '\n\n');
        bodySource = JSON.stringify(markdownToVisualBlock(md));
        break;
      }
      default:
        break;
    }
  } else if (from === 'markdown' && (to === 'richtext' || to === 'html')) {
    skip = true;
    setCampaignContent(1, { body, from, to }).then((data: any) => {
      nextTick(() => {
        self.value.contentType = to;
        self.value.body = beautifyHTML(data.trim());
      });
    });
  } else if (from === 'plain' && (to === 'richtext' || to === 'html')) {
    body = body.replace(/\n/ig, '<br>\n');
  } else if (to === 'visual') {
    bodySource = JSON.stringify(markdownToVisualBlock(body));
  }

  if (to === 'visual' || from === 'visual') {
    templateId.value = null;
    self.value.templateId = null;
  }

  if (!skip) {
    nextTick(() => {
      self.value.contentType = to;
      self.value.body = body;
      self.value.bodySource = bodySource;
    });
  }
}

function onContentTypeChange(to: string, from: string) {
  if (!self.value.body.trim()) {
    convertContentType(to, from);
    return;
  }
  $utils.confirm(
    t('campaigns.confirmSwitchFormat'),
    () => { convertContentType(to, from); },
    () => { contentTypeSel.value = from; },
  );
}

function onTogglePreview() { isPreviewing.value = !isPreviewing.value; }

function onKeyboardShortcut(e: KeyboardEvent) {
  if (e.key === 'F9') { onTogglePreview(); e.preventDefault(); }
  if (e.ctrlKey && e.key === 's') { $events.$emit('campaign.update'); e.preventDefault(); }
}

function onVisualEditorChange({ body, source }: { body: string; source: string }) {
  self.value.body = body;
  self.value.bodySource = source;
}

function onShowVisualTplSelector() {
  isVisualTplSelector.value = true;
  setDefaultTemplate();
}

function onImportVisualTpl() {
  if (!visualTemplateId.value) return;
  $utils.confirm(t('campaigns.confirmOverwriteContent'), () => {
    getTemplate(visualTemplateId.value!).then((data: any) => {
      self.value.body = data.body;
      self.value.bodySource = data.bodySource;
      isVisualTplDisabled.value = true;
      visualEditorRef.value?.render(JSON.parse(data.bodySource));
    });
  });
}

watch(validTemplates, () => { setDefaultTemplate(); });

watch(contentTypeSel, (to, from) => {
  if (from !== to && to !== self.value.contentType) { onContentTypeChange(to, from); }
});

watch(templateId, (to) => {
  if (self.value.templateId !== to) { self.value.templateId = to; }
});

onMounted(() => {
  contentTypeSel.value = props.modelValue.contentType;
  templateId.value = props.modelValue.templateId;
  window.addEventListener('keydown', onKeyboardShortcut);
  $events.$on('campaign.preview', () => { isPreviewing.value = true; });
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyboardShortcut);
  $events.$off('campaign.preview');
});
</script>

<style scoped lang="scss">
.editor-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;

  &-left {
    display: flex;
    align-items: flex-end;
    gap: 1rem;
    flex-wrap: wrap;

    .field { margin-bottom: 0; }
  }

  &-right {
    flex-shrink: 0;
  }
}
</style>
