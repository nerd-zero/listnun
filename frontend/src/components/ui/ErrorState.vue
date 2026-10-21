<template>
  <div class="ln-error" role="alert">
    <span class="ln-error-icon" aria-hidden="true"><i class="pi pi-exclamation-circle" /></span>
    <strong class="ln-error-title">{{ title }}</strong>
    <span class="ln-error-text">{{ text || $t('states.errorText') }}</span>
    <span v-if="details" class="ln-error-meta ln-mono">{{ details }}</span>
    <div class="ln-error-actions">
      <PvButton :label="$t('states.tryAgain')" size="small" @click="$emit('retry')" />
      <PvButton
        v-if="details"
        :label="copied ? $t('states.copied') : $t('states.copyDetails')"
        size="small"
        severity="secondary"
        outlined
        @click="copy"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

export interface RequestError {
  status?: number;
  method?: string;
  url?: string;
  requestId?: string;
  message?: string;
}

// ErrorState: what went wrong, a retry, and the request details so support
// can find the failing call.
const props = withDefaults(defineProps<{
  title: string;
  text?: string;
  error?: RequestError | null;
}>(), {
  text: '',
  error: null,
});

defineEmits(['retry']);

const copied = ref(false);

const details = computed(() => {
  const e = props.error;
  if (!e) return '';
  return [e.status, [e.method?.toUpperCase(), e.url].filter(Boolean).join(' '), e.requestId ? `req ${e.requestId}` : '']
    .filter(Boolean)
    .join(' ');
});

function copy() {
  const text = [details.value, props.error?.message].filter(Boolean).join('\n');
  navigator.clipboard?.writeText(text).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 1500);
  });
}
</script>
