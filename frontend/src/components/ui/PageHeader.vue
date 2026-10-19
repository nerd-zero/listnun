<template>
  <div class="ln-page-header">
    <div class="ln-page-header-text">
      <div class="ln-page-header-title">
        <slot name="before-title" />
        <h1 class="ln-page-title">{{ title }}</h1>
        <span v-if="count !== undefined && count !== null && count !== ''" class="ln-page-count">{{ count }}</span>
        <slot name="after-title" />
      </div>
      <p v-if="subtitle || $slots.subtitle" class="ln-page-subtitle">
        <slot name="subtitle">{{ subtitle }}</slot>
      </p>
    </div>
    <div v-if="$slots.default || $slots.search" class="ln-page-header-actions">
      <slot name="search" />
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
// PageHeader: title, count, optional subtitle, then a search slot and the
// page's primary action. Actions wrap under the title on narrow screens;
// buttons never truncate.
defineProps<{
  title: string;
  count?: number | string | null;
  subtitle?: string;
}>();
</script>
