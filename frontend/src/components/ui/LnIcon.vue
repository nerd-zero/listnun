<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 16 16"
    fill="none"
    stroke="currentColor"
    :stroke-width="stroke"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    class="ln-icon"
  >
    <path :d="PATHS[name]" />
  </svg>
</template>

<script setup lang="ts">
import { PATHS, type LnIconName } from './lnIcons';

// LnIcon draws one of the design's own 16px line icons (see lnIcons.ts).
// It is decorative: put the accessible name on the button or link.
withDefaults(defineProps<{
  name: LnIconName;
  size?: number;
  stroke?: number;
}>(), { size: 16, stroke: 1.4 });
</script>

<style scoped>
.ln-icon { flex: none; display: block; }
</style>
