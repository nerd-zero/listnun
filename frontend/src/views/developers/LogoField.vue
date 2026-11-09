<template>
  <div class="logo-field">
    <div v-if="modelValue" class="logo-current">
      <div class="logo-tile" :class="{ 'is-dark': dark }">
        <img :src="modelValue.url" :alt="modelValue.filename">
      </div>
      <div class="logo-meta">
        <span class="ln-mono">{{ modelValue.filename }}</span>
        <span class="ln-small">{{ metaLine }}</span>
      </div>
    </div>
    <div v-if="modelValue && !disabled" class="logo-actions">
      <PvButton :label="$t('developers.branding.replace')" severity="secondary" outlined size="small" :loading="uploading" @click="pick" />
      <PvButton :label="$t('developers.branding.remove')" severity="danger" outlined size="small" @click="$emit('update:modelValue', null)" />
    </div>

    <div
      v-if="!modelValue && !disabled"
      class="logo-drop"
      :class="{ 'is-over': over }"
      role="button"
      tabindex="0"
      :aria-label="$t('developers.branding.dropLabel')"
      @click="pick"
      @keydown.enter.prevent="pick"
      @keydown.space.prevent="pick"
      @dragover.prevent="over = true"
      @dragleave.prevent="over = false"
      @drop.prevent="onDrop"
    >
      <span class="logo-drop-icon" aria-hidden="true"><i :class="uploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" /></span>
      <strong>
        <i18n-t keypath="developers.branding.drop" tag="span">
          <template #browse><span class="logo-browse">{{ $t('developers.branding.browse') }}</span></template>
        </i18n-t>
      </strong>
      <span class="ln-small">{{ $t('developers.branding.dropHelp', { px: LOGO_MIN_WIDTH }) }}</span>
    </div>

    <PvMessage v-if="error" severity="error" icon="pi pi-exclamation-circle" class="logo-error">{{ error }}</PvMessage>
    <input ref="input" type="file" class="logo-input" :accept="LOGO_TYPES.join(',')" tabindex="-1" aria-hidden="true" @change="onPick">
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { uploadMedia } from '../../api/index';
import {
  LOGO_MAX_BYTES, LOGO_MIN_WIDTH, LOGO_TYPES, type BrandLogo,
} from '../../api/developers';
import { useTime } from '../../composables/useTime';

interface UploadedMedia {
  id: number;
  url: string;
  filename: string;
  contentType: string;
  createdAt: string;
}

// LogoField: one logo slot. Checks the file in the browser, uploads it to
// the media library and hands back what Email branding stores.
const props = defineProps<{
  modelValue: BrandLogo | null;
  // Shows the current logo on a dark tile, for the dark-mode slot.
  dark?: boolean;
  disabled?: boolean;
}>();
const emit = defineEmits(['update:modelValue']);

const { t } = useI18n();
const { shortDate } = useTime();

const input = ref<HTMLInputElement | null>(null);
const over = ref(false);
const uploading = ref(false);
const error = ref('');

const metaLine = computed(() => {
  const l = props.modelValue;
  if (!l) return '';
  const type = (l.contentType.split('/')[1] || '').replace('svg+xml', 'svg').toUpperCase();
  const kb = l.size ? `${Math.max(1, Math.round(l.size / 1024))} KB` : '';
  const dims = l.width && l.height ? `${l.width} × ${l.height}` : '';
  const when = l.uploadedAt ? t('developers.branding.uploaded', { date: shortDate(l.uploadedAt) }) : '';
  return [type, kb, dims, when].filter(Boolean).join(' ');
});

function pick() {
  if (!uploading.value) input.value?.click();
}

// imageSize reads a file's pixel size in the browser.
function imageSize(file: File): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => {
      resolve({ width: img.naturalWidth, height: img.naturalHeight });
      URL.revokeObjectURL(url);
    };
    img.onerror = () => {
      reject(new Error('unreadable'));
      URL.revokeObjectURL(url);
    };
    img.src = url;
  });
}

async function upload(file: File) {
  error.value = '';
  const ext = (file.name.split('.').pop() || '').toUpperCase();
  if (!LOGO_TYPES.includes(file.type)) {
    error.value = t('developers.branding.badType', { name: file.name, type: ext || file.type });
    return;
  }
  if (file.size > LOGO_MAX_BYTES) {
    error.value = t('developers.branding.tooBig', { name: file.name });
    return;
  }

  let size = { width: 0, height: 0 };
  try {
    size = await imageSize(file);
  } catch {
    error.value = t('developers.branding.unreadable', { name: file.name });
    return;
  }
  if (file.type !== 'image/svg+xml' && size.width < LOGO_MIN_WIDTH) {
    error.value = t('developers.branding.tooNarrow', { name: file.name, px: LOGO_MIN_WIDTH });
    return;
  }

  const form = new FormData();
  form.append('file', file);
  uploading.value = true;
  try {
    const m = await uploadMedia(form) as unknown as UploadedMedia;
    const logo: BrandLogo = {
      mediaId: m.id,
      url: m.url,
      filename: m.filename,
      contentType: file.type,
      size: file.size,
      width: size.width,
      height: size.height,
      uploadedAt: m.createdAt || new Date().toISOString(),
    };
    emit('update:modelValue', logo);
  } catch {
    // The API layer has already shown the server's error.
  } finally {
    uploading.value = false;
  }
}

function onPick(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0];
  if (f) upload(f);
  (e.target as HTMLInputElement).value = '';
}

function onDrop(e: DragEvent) {
  over.value = false;
  const f = e.dataTransfer?.files?.[0];
  if (f) upload(f);
}
</script>

<style scoped lang="scss">
.logo-field { display: flex; flex-direction: column; gap: 12px; }
.logo-current { display: flex; gap: 14px; align-items: center; }
.logo-tile {
  width: 152px;
  height: 66px;
  flex: none;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;

  img { max-width: 100%; max-height: 100%; object-fit: contain; }
  &.is-dark { background: #1b1f26; border-color: #2b2f36; }
}
.logo-meta { display: flex; flex-direction: column; gap: 4px; min-width: 0; overflow-wrap: anywhere; }
.logo-actions { display: flex; gap: 8px; }
.logo-drop {
  border: 1px dashed var(--ln-border-strong);
  border-radius: 8px;
  background: var(--ln-muted);
  padding: 28px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
  cursor: pointer;

  &:hover, &.is-over { border-color: var(--ln-ink); }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}
.logo-drop-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: var(--ln-card);
  border: 1px solid var(--ln-border);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.logo-browse { text-decoration: underline; }
.logo-input { display: none; }
</style>
