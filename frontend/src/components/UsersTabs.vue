<template>
  <div class="users-tabs">
    <page-header :title="$t('users.usersAndRoles')">
      <slot />
    </page-header>
    <nav class="users-tabs-nav" :aria-label="$t('users.usersAndRoles')">
      <router-link
        v-for="tab in tabs"
        :key="tab.name"
        :to="{ name: tab.name }"
        class="users-tab"
        :class="{ 'is-on': $route.name === tab.name }"
        :aria-current="$route.name === tab.name ? 'page' : undefined"
      >
        {{ tab.label }}
      </router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import PageHeader from './ui/PageHeader.vue';

// UsersTabs: the shared "Users and roles" header with one tab per route.
const { t } = useI18n();
const { $can } = useGlobal();

const tabs = computed(() => [
  $can('users:get') && { name: 'users', label: t('globals.terms.users') },
  $can('roles:get') && { name: 'userRoles', label: t('users.userRoles') },
  $can('roles:get') && { name: 'listRoles', label: t('users.listRoles') },
  $can('users:get') && { name: 'apiUsers', label: t('users.apiUsers') },
].filter(Boolean) as { name: string; label: string }[]);
</script>

<style scoped lang="scss">
.users-tabs { display: flex; flex-direction: column; gap: 16px; }
.users-tabs-nav {
  display: flex;
  gap: 22px;
  border-bottom: 1px solid var(--ln-border);
  overflow-x: auto;
}
.users-tab {
  padding: 0 0 10px;
  margin-bottom: -1px;
  border-bottom: 2px solid transparent;
  color: var(--ln-text-2);
  font-size: 14px;
  white-space: nowrap;
  text-decoration: none;

  &:hover { color: var(--ln-text); text-decoration: none; }
  &.is-on { color: var(--ln-text); font-weight: 500; border-bottom-color: var(--ln-text); }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; border-radius: 2px; }
}
</style>
