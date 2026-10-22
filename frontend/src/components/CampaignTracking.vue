<template>
  <section class="ln-card camp-track" data-cy="tracking">
    <header class="ln-card-head">
      <h3 class="ln-card-title">{{ $t('campaigns.tracking.title') }}</h3>
    </header>

    <div class="camp-track-body">
      <label class="ln-check-row" for="camp-track-links">
        <PvToggleSwitch v-model="links" input-id="camp-track-links" :disabled="disabled" data-cy="track-links" />
        <span>
          <strong>{{ $t('campaigns.tracking.links') }}</strong>
          <small>{{ $t('campaigns.tracking.linksHelp') }}</small>
        </span>
      </label>

      <div class="ln-field">
        <label for="camp-track-pixel" class="ln-field-label">{{ $t('campaigns.tracking.pixels') }}</label>
        <TransitionGroup v-if="modelValue.pixels.length" name="ln-list" tag="ul" class="camp-track-pixels">
          <li v-for="p in modelValue.pixels" :key="p" class="camp-track-pixel">
            <span class="ln-mono camp-track-url" :title="p">{{ p }}</span>
            <button
              v-if="!disabled"
              type="button"
              class="ln-icon-btn camp-track-remove"
              :aria-label="`${$t('globals.buttons.remove')} ${p}`"
              @click="remove(p)"
            >
              <i class="pi pi-times" aria-hidden="true" />
            </button>
          </li>
        </TransitionGroup>

        <div v-if="!disabled && modelValue.pixels.length < MAX_PIXELS" class="camp-track-add">
          <PvTextarea
            id="camp-track-pixel"
            v-model="draft"
            rows="2"
            auto-resize
            :placeholder="$t('campaigns.tracking.pixelPlaceholder')"
            :invalid="!!error"
            data-cy="track-pixel-input"
            @keydown.enter.exact.prevent="add"
          />
          <PvButton :label="$t('campaigns.tracking.addPixel')" severity="secondary" outlined :disabled="!draft.trim()" @click="add" />
        </div>
        <p v-if="error" class="ln-field-error" role="alert">{{ error }}</p>
        <p class="ln-help">{{ $t('campaigns.tracking.pixelsHelp') }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { parsePixelInput } from '../composables/trackingPixel';

// A campaign's tracking (models.CampaignTracking): one switch that tracks
// every link, and third-party pixels, so nobody needs template tags.
export interface Tracking {
  links: boolean;
  pixels: string[];
}

// Matches models.MaxTrackingPixels.
const MAX_PIXELS = 10;

const props = defineProps<{ modelValue: Tracking; disabled?: boolean }>();
const emit = defineEmits<{ 'update:modelValue': [Tracking] }>();
const { t } = useI18n();

const draft = ref('');
const error = ref('');

const links = computed({
  get: () => props.modelValue.links,
  set: (v: boolean) => emit('update:modelValue', { ...props.modelValue, links: v }),
});

function add() {
  const r = parsePixelInput(draft.value);
  if ('error' in r) {
    error.value = r.error === 'script' ? t('campaigns.tracking.pixelScript') : t('campaigns.tracking.pixelInvalid');
    return;
  }
  error.value = '';
  draft.value = '';
  if (props.modelValue.pixels.includes(r.url)) return;
  emit('update:modelValue', { ...props.modelValue, pixels: [...props.modelValue.pixels, r.url] });
}

function remove(url: string) {
  emit('update:modelValue', { ...props.modelValue, pixels: props.modelValue.pixels.filter((p) => p !== url) });
}
</script>

<style scoped lang="scss">
.camp-track-body { display: flex; flex-direction: column; gap: 18px; padding: 16px 18px; }
.camp-track small { display: block; font-size: 12px; line-height: 16px; color: var(--ln-text-3); font-weight: 400; margin-top: 2px; }
.camp-track strong { font-weight: 500; }
.camp-track-pixels { list-style: none; margin: 0 0 8px; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.camp-track-pixel {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 4px 4px 10px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
  background: var(--ln-muted);
}
.camp-track-url { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
.camp-track-remove { width: 26px; height: 26px; .pi { font-size: 10px; } }
.camp-track-add {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  :deep(.p-textarea) { flex: 1; font-family: var(--ln-mono); font-size: 12px; }
}
</style>
