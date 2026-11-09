<template>
  <nav class="ln-subnav" :aria-label="label">
    <router-link
      v-for="item in items"
      :key="item.key"
      v-tooltip.right="item.label.length > 22 ? item.label : null"
      :to="item.to"
      class="ln-subnav-item"
      :class="{ 'is-active': item.key === active }"
      :aria-current="item.key === active ? 'page' : undefined"
      :data-cy="`subnav-${item.key}`"
    >
      <i v-if="item.icon" :class="['pi', item.icon]" aria-hidden="true" />
      <span>{{ item.label }}</span>
      <span v-if="item.badge" class="ln-subnav-badge">{{ item.badge }}</span>
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';

export interface SubNavItem {
  key: string;
  label: string;
  to: RouteLocationRaw;
  icon?: string;
  badge?: string | number;
}

// SubNav: a vertical list of router links, e.g. the Settings sections.
// Every section is its own URL so it can be linked to.
defineProps<{
  items: SubNavItem[];
  active: string;
  label?: string;
}>();
</script>
