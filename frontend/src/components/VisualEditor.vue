<template>
  <div class="visual-editor-wrapper">
    <iframe ref="visualEditorEl" id="visual-editor" class="visual-editor email-builder-container"
      title="Visual email editor" />

    <!-- image picker -->
    <media-picker-dialog v-model:visible="isMediaVisible" @select="onMediaSelect" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import MediaPickerDialog from './MediaPickerDialog.vue';

const props = withDefaults(defineProps<{
  source?: string;
  height?: string;
}>(), { source: '', height: 'auto' });

const emit = defineEmits(['change']);

const visualEditorEl = ref<HTMLIFrameElement | null>(null);
const isMediaVisible = ref(false);

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
  let n = 10;
  const timer = window.setInterval(() => {
    const container = iframe.contentWindow!.document.getElementById('visual-editor-container');
    if (container && container.hasChildNodes()) {
      em.resetDocument(source);
      window.clearInterval(timer);
      return;
    }
    n += 1;
    if (n > 10) { window.clearInterval(timer); }
  }, 100);
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
      render(source);
    }).catch((error) => {
      console.error('Failed to load email-builder script:', error);
    });
  };
  window.addEventListener('message', onSidebarMount, false);
});

onUnmounted(() => {
  window.removeEventListener('message', onSidebarMount, false);
});

defineExpose({ render });
</script>

<style lang="css">
.visual-editor-wrapper {
  width: 100%;
  border: 1px solid #eaeaea;
  max-width: 100vw;
}

#visual-editor {
  position: relative;
  border: none;
  width: 100%;
  min-height: 500px;
}
</style>
