<template>
  <div class="users-tabs">
    <page-header :title="$t('users.team')" :subtitle="$t('users.teamHelp')">
      <slot />
    </page-header>
    <nav class="users-tabs-nav" :aria-label="$t('users.team')">
      <router-link
        v-for="tab in tabs"
        :key="tab.name"
        :to="{ name: tab.name }"
        class="users-tab"
        :class="{ 'is-on': $route.name === tab.name }"
        :aria-current="$route.name === tab.name ? 'page' : undefined"
      >
        {{ tab.label }}
        <span v-if="tab.count !== null" class="users-tab-count">{{ tab.count }}</span>
      </router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { getRoles } from '../api/generated/endpoints/roles/roles';
import PageHeader from './ui/PageHeader.vue';

// UsersTabs: the shared Team header with one tab per route. members is the
// number of active members, from the page that already loaded them; role
// counts come from the store, loaded here when no page has yet.
const props = defineProps<{ members?: number | null }>();

const { t } = useI18n();
const { $can } = useGlobal();
const store = useMainStore();
const { userRoles, listRoles } = storeToRefs(store);
const { listUserRoles, listListRoles } = getRoles();

function size(v: unknown): number | null {
  return Array.isArray(v) && v.length ? v.length : null;
}

const tabs = computed(() => [
  $can('users:get') && { name: 'users', label: t('users.members'), count: props.members ?? null },
  $can('roles:get') && { name: 'userRoles', label: t('users.roles'), count: size(userRoles.value) },
  $can('roles:get') && { name: 'listRoles', label: t('users.listAccess'), count: size(listRoles.value) },
].filter(Boolean) as { name: string; label: string; count: number | null }[]);

onMounted(() => {
  if (!$can('roles:get')) return;
  if (!size(userRoles.value)) listUserRoles().then((data) => { store.setModelResponse({ model: 'userRoles', data }); });
  if (!size(listRoles.value)) listListRoles().then((data) => { store.setModelResponse({ model: 'listRoles', data }); });
});
</script>

<style scoped lang="scss">
.users-tabs { display: flex; flex-direction: column; gap: 16px; }
.users-tabs-nav {
  display: flex;
  gap: 22px;
  border-bottom: 1px solid var(--ln-border);
  overflow-x: auto;
  overflow-y: hidden;
}
.users-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
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
.users-tab-count { color: var(--ln-text-3); font-weight: 400; font-size: 13px; }
</style>
