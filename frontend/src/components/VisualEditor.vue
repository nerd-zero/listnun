<template>
  <div class="visual-editor-wrapper">
    <iframe ref="visualEditorEl" id="visual-editor" class="visual-editor email-builder-container"
      :title="$t('templates.visualEditorTitle')" />

    <!-- image picker -->
    <media-picker-dialog v-model:visible="isMediaVisible" @select="onMediaSelect" />
  </div>
</template>

<script setup lang="ts">
import {
  ref, onMounted, onUnmounted, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import MediaPickerDialog from './MediaPickerDialog.vue';
import { useGlobal } from '../composables/useGlobal';

const props = withDefaults(defineProps<{
  source?: string;
  height?: string;
  dark?: boolean;
}>(), { source: '', height: 'auto', dark: false });

const emit = defineEmits(['change']);

const { $utils } = useGlobal();
const { t } = useI18n();

// How many times (every RESET_POLL_MS) to check that the builder has drawn
// its container before loading the source into it: about a second.
const RESET_MAX_TRIES = 10;
const RESET_POLL_MS = 100;

const visualEditorEl = ref<HTMLIFrameElement | null>(null);
const isMediaVisible = ref(false);

// The builder's chrome follows the app's light/dark mode. The first mode is
// set directly before the first render, so there's no light flash (the
// srcdoc frame has no URL to carry ?theme=). Toggles are posted as the
// handover's listnun:theme message, which the builder checks the origin of.
const THEME_MESSAGE = 'listnun:theme';

function themeMode(): 'light' | 'dark' {
  return props.dark ? 'dark' : 'light';
}

function applyTheme() {
  const em = (visualEditorEl.value?.contentWindow as any)?.EmailBuilder;
  em?.setThemeMode?.(themeMode());
}

function postTheme() {
  visualEditorEl.value?.contentWindow?.postMessage({ type: THEME_MESSAGE, mode: themeMode() }, window.location.origin);
}

function loadScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    const iframe = visualEditorEl.value!;
    if ((iframe.contentWindow as any).EmailBuilder) { resolve(); return; }
    const script = document.createElement('script');
    script.id = 'email-builder-script';
    script.src = '/admin/static/email-builder/email-builder.umd.js';
    script.onload = () => { resolve(); };
    script.onerror = reject;
    iframe.contentDocument!.head.appendChild(script);
  });
}

function render(source: any) {
  const iframe = visualEditorEl.value!;
  const em = (iframe.contentWindow as any).EmailBuilder;
  if (!em || !em.isRendered('visual-editor-container')) {
    (iframe.contentWindow as any).EmailBuilder.render('visual-editor-container', {
      data: {},
      onChange: (data: any, body: string) => {
        const tpl = body.replace(/\{\{[^}]*\}\}/g, (match) => match.replace(/&quot;/g, '"'));
        emit('change', { source: JSON.stringify(data), body: tpl });
      },
    });
  }
  if (!source) return;
  let n = 0;
  const timer = window.setInterval(() => {
    const container = iframe.contentWindow!.document.getElementById('visual-editor-container');
    if (container && container.hasChildNodes()) {
      em.resetDocument(source);
      window.clearInterval(timer);
      return;
    }
    n += 1;
    if (n >= RESET_MAX_TRIES) { window.clearInterval(timer); }
  }, RESET_POLL_MS);
}

function onMediaSelect(media: any) {
  const iframe = visualEditorEl.value!;
  const input = iframe.contentDocument!.querySelector('.image-url input') as HTMLInputElement;
  if (input) {
    const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')!.set;
    nativeInputValueSetter!.call(input, media.url);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }
}

function onSidebarMount(msg: MessageEvent) {
  if (!msg.data) return;
  if (msg.data === 'visualeditor.select-media') { isMediaVisible.value = true; }
}

onMounted(() => {
  const iframe = visualEditorEl.value!;
  iframe.style.height = props.height!;
  iframe.srcdoc = `
    <!DOCTYPE html>
    <html>
      <head>
        <link rel="stylesheet" href="/public/static/fonts/fonts.css">
        <style>
          body { margin: 0; padding: 0; }
          #visual-editor-container { width: 100%; height: 100%; }
        </style>
      </head>
      <body>
        <div id="visual-editor-container"></div>
      </body>
    </html>
  `;
  iframe.onload = () => {
    loadScript().then(() => {
      const source = props.source ? JSON.parse(props.source) : null;
      applyTheme();
      render(source);
    }).catch(() => {
      $utils.toast(t('templates.visualEditorLoadError'), 'is-danger');
    });
  };
  window.addEventListener('message', onSidebarMount, false);
});

// The app's theme can be toggled while the editor is open.
watch(() => props.dark, postTheme);

onUnmounted(() => {
  window.removeEventListener('message', onSidebarMount, false);
});

defineExpose({ render });
</script>

<style lang="css">
.visual-editor-wrapper {
  width: 100%;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius-card);
  overflow: hidden;
  max-width: 100vw;
}

#visual-editor {
  position: relative;
  border: none;
  width: 100%;
  min-height: 500px;
}
</style>
