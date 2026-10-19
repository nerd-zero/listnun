<template>
  <span class="ln-status" :class="`ln-status--${tone}`" :data-status="status">
    <span class="ln-status-dot" aria-hidden="true" />
    <slot>{{ label || status }}</slot>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { statusTone, type StatusTone } from './status';

// StatusTag: a pill with a dot, so status never depends on hue alone.
const props = defineProps<{
  status: string;
  label?: string;
  // Override the mapped tone when a page knows better.
  tone?: StatusTone;
}>();

const tone = computed(() => props.tone ?? statusTone(props.status));
</script>
