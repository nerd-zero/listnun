<template>
  <section class="ln-no-access">
    <span class="ln-empty-icon" aria-hidden="true"><i class="pi pi-lock" /></span>
    <h1 class="ln-section-title">{{ $t('states.noAccessTitle', { page: page }) }}</h1>
    <p class="ln-no-access-text">{{ $t('states.noAccessText', { role: roleName }) }}</p>
    <router-link :to="{ name: 'dashboard' }" class="p-button p-component ln-no-access-btn">
      {{ $t('states.backToDashboard') }}
    </router-link>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../store';

// NoAccess: shown in place of a page the user's role can't open. The URL
// stays put so the user can share it with an admin.
defineProps<{ page: string }>();

const { profile } = storeToRefs(useMainStore());

const roleName = computed(() => {
  const p = profile.value as { userRole?: { name?: string } } | undefined;
  return p?.userRole?.name ?? '';
});
</script>
