<template>
  <div class="richtext-editor" v-if="isRichtextReady">
    <tiny-mce v-model="computedValue" :disabled="disabled" :init="richtextConf" />

    <PvDialog v-model:visible="isRichtextSourceVisible" :style="{ width: '1200px' }" :closable="true" modal :aria-modal="true">
      <div class="richtext-dialog">
        <div class="richtext-dialog-body">
          <code-editor lang="html" v-model="richTextSourceBody" key="richtext-source" />
        </div>
        <div class="richtext-dialog-footer">
          <PvButton severity="secondary" @click="onFormatRichtextHTML" :label="$t('campaigns.formatHTML')" />
          <PvButton severity="secondary" @click="isRichtextSourceVisible = false" :label="$t('globals.buttons.close')" />
          <PvButton @click="onSaveRichTextSource" severity="primary" :label="$t('globals.buttons.save')" />
        </div>
      </div>
    </PvDialog>

    <PvDialog v-model:visible="isInsertHTMLVisible" :style="{ width: '750px' }" :closable="true" modal :aria-modal="true">
      <div class="richtext-dialog">
        <div class="richtext-dialog-body">
          <code-editor lang="html" v-model="insertHTMLSnippet" key="richtext-snippet" />
        </div>
        <div class="richtext-dialog-footer">
          <PvButton severity="secondary" @click="onFormatRichtextHTMLSnippet" :label="$t('campaigns.formatHTML')" />
          <PvButton severity="secondary" @click="isInsertHTMLVisible = false" :label="$t('globals.buttons.close')" />
          <PvButton @click="onInsertHTML" severity="primary" :label="$t('globals.buttons.insert')" />
        </div>
      </div>
    </PvDialog>

    <!-- image picker -->
    <media-picker-dialog v-model:visible="isMediaVisible" @select="onMediaSelect" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import TinyMce from '@tinymce/tinymce-vue';
import { html } from 'js-beautify';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import 'tinymce';
import 'tinymce/icons/default';
import 'tinymce/plugins/anchor';
import 'tinymce/plugins/autolink';
import 'tinymce/plugins/autoresize';
import 'tinymce/plugins/charmap';
import 'tinymce/plugins/colorpicker';
import 'tinymce/plugins/contextmenu';
import 'tinymce/plugins/emoticons';
import 'tinymce/plugins/emoticons/js/emojis';
import 'tinymce/plugins/fullscreen';
import 'tinymce/plugins/help';
import 'tinymce/plugins/hr';
import 'tinymce/plugins/image';
import 'tinymce/plugins/imagetools';
import 'tinymce/plugins/link';
import 'tinymce/plugins/lists';
import 'tinymce/plugins/paste';
import 'tinymce/plugins/searchreplace';
import 'tinymce/plugins/table';
import 'tinymce/plugins/textcolor';
import 'tinymce/plugins/visualblocks';
import 'tinymce/plugins/visualchars';
import 'tinymce/plugins/wordcount';
import 'tinymce/skins/ui/oxide/skin.css';
import 'tinymce/themes/silver';

import { colors, uris } from '../constants';
import CodeEditor from './CodeEditor.vue';
import MediaPickerDialog from './MediaPickerDialog.vue';

const LANGS: Record<string, string> = {
  cs: 'cs',
  de: 'de',
  es: 'es_419',
  fr: 'fr_FR',
  it: 'it_IT',
  pl: 'pl',
  pt: 'pt_PT',
  'pt-BR': 'pt_BR',
  ro: 'ro',
  tr: 'tr',
};

const TRACK_LINK = 'trackLink';
const TRACK_SUFFIX = '@TrackLink';
const EMBED_IMAGE = 'embedImage';

const props = withDefaults(defineProps<{
  disabled?: boolean;
  modelValue?: string;
}>(), { disabled: false, modelValue: '' });

const emit = defineEmits(['update:modelValue']);
const { $events } = useGlobal();
const { t } = useI18n();
const { serverConfig } = storeToRefs(useMainStore());

const isMediaVisible = ref(false);
const isReady = ref(false);
const isRichtextReady = ref(false);
const isRichtextSourceVisible = ref(false);
const isInsertHTMLVisible = ref(false);
const insertHTMLSnippet = ref('');
const richtextConf = ref<any>({});
const richTextSourceBody = ref('');
let imageCallack: any = null;

const computedValue = computed({
  get: () => props.modelValue,
  set: (newValue: string) => emit('update:modelValue', newValue),
});

function trimLines(str: string, removeEmptyLines: boolean) {
  const out = str.split('\n');
  for (let i = 0; i < out.length; i += 1) {
    const line = out[i].trim();
    if (removeEmptyLines) { out[i] = line; } else if (line === '') { out[i] = ''; }
  }
  return out.join('\n').replace(/\n\s*\n\s*\n/g, '\n\n');
}

function beautifyHTML(str: string) {
  let s = trimLines(str.replace(/(<(?!(\/)?a|span)([^>]+)>)/gi, '\n$1\n'), true);
  s = s.replace(/\n+/g, '\n');
  try { s = html(s).trim(); } catch (error) { console.log('error formatting HTML', error); }
  return s;
}

function withDialogCheckbox(body: any, checkbox: any) {
  if (body.type === 'tabpanel') {
    return {
      ...body,
      tabs: body.tabs.map((tab: any) => (
        tab.name === 'general' || tab.title === 'General'
          ? { ...tab, items: [...tab.items, checkbox] }
          : tab
      )),
    };
  }
  return { ...body, items: [...body.items, checkbox] };
}

function getSelectedImage(editor: any) {
  const node = editor.selection.getNode();
  if (!node) return null;
  if (node.nodeName === 'IMG') return node;
  const figure = editor.dom.getParent(node, 'figure.image');
  return figure ? figure.querySelector('img') : null;
}

function onEditorDialogOpen(editor: any) {
  const ed = editor;
  const oldEd = ed.windowManager.open;
  ed.windowManager.open = (tpl: any, r: any) => {
    const data = tpl.initialData || {};
    const isLink = data.url && 'anchor' in data;
    const isImage = data.src && !isLink;
    if (!isLink && !isImage) { return oldEd.call(ed.windowManager, tpl, r); }
    const { onSubmit } = tpl;
    const checkbox = isLink
      ? { type: 'checkbox', name: TRACK_LINK, label: 'Track link?' }
      : { type: 'checkbox', name: EMBED_IMAGE, label: t('media.embed') };
    const spec = { ...tpl, body: withDialogCheckbox(tpl.body, checkbox) };
    if (isLink) {
      const cleanURL = (data.url.value || '').replace(/@TrackLink$/, '');
      const checked = data.url.value !== cleanURL
        || (!cleanURL && JSON.parse(localStorage.getItem(TRACK_LINK) || 'false'));
      spec.initialData = { ...data, [TRACK_LINK]: checked, url: { ...data.url, value: cleanURL } };
      spec.onSubmit = (api: any) => {
        const d = api.getData();
        const shouldTrack = Boolean(d[TRACK_LINK]);
        const url = (d.url.value || '').replace(/@TrackLink$/, '');
        localStorage.setItem(TRACK_LINK, JSON.stringify(shouldTrack));
        if (shouldTrack && /^https?:\/\//i.test(url)) {
          api.setData({ url: { ...d.url, value: `${url}${TRACK_SUFFIX}` } });
        }
        onSubmit(api);
      };
    } else {
      const img = getSelectedImage(ed);
      spec.initialData = { ...data, [EMBED_IMAGE]: Boolean(img && img.hasAttribute('data-embed')) };
      spec.onSubmit = (api: any) => {
        const d = api.getData();
        const shouldEmbed = d[EMBED_IMAGE] === true || d[EMBED_IMAGE] === 'true';
        onSubmit(api);
        const node = (img && ed.getBody().contains(img)) ? img : getSelectedImage(ed);
        if (!node) return;
        if (shouldEmbed) { ed.dom.setAttrib(node, 'data-embed', 'true'); } else { node.removeAttribute('data-embed'); }
        ed.fire('change');
        ed.save();
        computedValue.value = ed.getContent();
      };
    }
    return oldEd.call(ed.windowManager, spec, r);
  };
}

function onEditorURLConvert(url: string) { return url; }

function onRichtextViewSource() {
  richTextSourceBody.value = computedValue.value || '';
  isRichtextSourceVisible.value = true;
}

function onOpenInsertHTML() { isInsertHTMLVisible.value = true; }

function onInsertHTML() {
  isInsertHTMLVisible.value = false;
  (window as any).tinymce.editors[0].execCommand('mceInsertContent', false, insertHTMLSnippet.value);
  insertHTMLSnippet.value = '';
}

function onFormatRichtextHTML() { richTextSourceBody.value = beautifyHTML(richTextSourceBody.value); }
function onFormatRichtextHTMLSnippet() { insertHTMLSnippet.value = beautifyHTML(insertHTMLSnippet.value); }

function onSaveRichTextSource() {
  computedValue.value = richTextSourceBody.value;
  (window as any).tinymce.editors[0].setContent(computedValue.value);
  richTextSourceBody.value = '';
  isRichtextSourceVisible.value = false;
}

function onMediaSelect(media: any) { imageCallack(media.url); }

function initRichtextEditor() {
  const { lang } = serverConfig.value as any;
  richtextConf.value = {
    init_instance_callback: () => { isReady.value = true; },
    urlconverter_callback: onEditorURLConvert,
    setup: (editor: any) => {
      editor.addShortcut('ctrl+s', 'Save content', () => { $events.$emit('campaign.update', {}); });
      editor.addShortcut('f9', 'Preview', () => { $events.$emit('campaign.preview', {}); });
      editor.on('init', () => { editor.focus(); onEditorDialogOpen(editor); });
      editor.ui.registry.addButton('html', { icon: 'sourcecode', tooltip: 'Source code', onAction: onRichtextViewSource });
      editor.ui.registry.addButton('insert-html', { icon: 'code-sample', tooltip: 'Insert HTML', onAction: onOpenInsertHTML });
      editor.on('CloseWindow', () => { editor.selection.getNode().scrollIntoView(false); });
    },
    browser_spellcheck: true,
    min_height: 500,
    toolbar_sticky: true,
    entity_encoding: 'raw',
    convert_urls: true,
    relative_urls: false,
    remove_script_host: false,
    extended_valid_elements: 'img[*]',
    plugins: [
      'anchor', 'autoresize', 'autolink', 'charmap', 'emoticons', 'fullscreen',
      'help', 'hr', 'image', 'imagetools', 'link', 'lists', 'paste', 'searchreplace',
      'table', 'visualblocks', 'visualchars', 'wordcount',
    ],
    toolbar: `undo redo | formatselect styleselect fontsizeselect |
              bold italic underline strikethrough forecolor backcolor subscript superscript |
              alignleft aligncenter alignright alignjustify |
              bullist numlist table image insert-html | outdent indent | link hr removeformat |
              html fullscreen help`,
    fontsize_formats: '10px 11px 12px 14px 15px 16px 18px 24px 36px',
    skin: false,
    content_css: false,
    content_style: `
      body { font-family: 'DM Sans', 'Inter', sans-serif; font-size: 15px; }
      img { max-width: 100%; }
      img.img-float-left { float: left; margin: 0 1em 1em 0; }
      img.img-float-right { float: right; margin: 0 0 1em 1em; }
      a { color: ${colors.primary}; }
      table, td { border-color: #ccc;}
    `,
    language: LANGS[lang] || null,
    language_url: LANGS[lang] ? `${uris.static}/tinymce/lang/${LANGS[lang]}.js` : null,
    image_advtab: true,
    image_class_list: [
      { title: 'None', value: '' },
      { title: 'Float left', value: 'img-float-left' },
      { title: 'Float right', value: 'img-float-right' },
    ],
    file_picker_types: 'image',
    file_picker_callback: (callback: any) => {
      isMediaVisible.value = true;
      imageCallack = callback;
    },
  };
  isRichtextReady.value = true;
}

onMounted(() => { initRichtextEditor(); });
</script>

<style scoped lang="scss">
.richtext-dialog {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.richtext-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>
