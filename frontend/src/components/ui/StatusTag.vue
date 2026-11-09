<template>
  <span class="ln-status" :class="`ln-status--${tone}`" :data-status="status">
    <slot>{{ label || status }}</slot>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { statusTone, type StatusTone } from './status';

// StatusTag: the status word on a light tint with 4px corners. The word
// carries the meaning; colour only supports it (handover: no dots).
const props = defineProps<{
  status: string;
  label?: string;
  // Override the mapped tone when a page knows better.
  tone?: StatusTone;
}>();

const tone = computed(() => props.tone ?? statusTone(props.status));
</script>
