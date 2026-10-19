<template>
  <section class="empty-placeholder">
    <i :class="'pi ' + (!icon ? 'pi-plus' : icon)" class="empty-icon" />
    <p class="empty-label">{{ !label ? $t('globals.messages.emptyState') : label }}</p>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  icon?: string;
  label?: string;
}>(), {
  icon: '',
  label: '',
});
</script>

<style scoped lang="scss">
.empty-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2.5rem 1rem;
  color: var(--ln-text-4);
}
.empty-icon { font-size: 2rem; opacity: 0.5; }
.empty-label { font-size: 0.875rem; }
</style>
