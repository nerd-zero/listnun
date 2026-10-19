<template>
  <div class="ln-skeleton" :aria-label="$t('states.loading')" aria-busy="true">
    <div v-for="r in rows" :key="r" class="ln-skeleton-row">
      <div v-for="c in cols" :key="c" class="ln-skeleton-cell">
        <PvSkeleton :width="widths[(r + c) % widths.length]" height="12px" />
        <PvSkeleton v-if="c === 1 && twoLine" width="40%" height="8px" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// TableSkeleton: placeholder rows for DataTable #loading / #empty while the
// first page loads, matching the real column layout.
withDefaults(defineProps<{
  rows?: number;
  cols?: number;
  twoLine?: boolean;
}>(), {
  rows: 5,
  cols: 4,
  twoLine: true,
});

const widths = ['70%', '55%', '85%', '60%', '75%'];
</script>
