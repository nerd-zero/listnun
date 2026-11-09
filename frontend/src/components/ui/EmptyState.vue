<template>
  <div class="ln-empty" :class="{ 'ln-empty--bare': bare }">
    <span class="ln-empty-icon" aria-hidden="true"><i :class="['pi', icon]" /></span>
    <strong class="ln-empty-title">{{ title }}</strong>
    <span v-if="text" class="ln-empty-text">{{ text }}</span>
    <div v-if="$slots.default || actionLabel" class="ln-empty-action">
      <slot>
        <PvButton
          :label="actionLabel"
          :severity="secondary ? 'secondary' : undefined"
          :outlined="secondary"
          @click="$emit('action')"
        />
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
// EmptyState: icon, title, one sentence and exactly one primary action.
// `bare` drops the card chrome for use inside a TableCard or DataTable #empty.
withDefaults(defineProps<{
  icon?: string;
  title: string;
  text?: string;
  actionLabel?: string;
  // "No matches" uses a secondary Clear filters button.
  secondary?: boolean;
  bare?: boolean;
}>(), {
  icon: 'pi-inbox',
  text: '',
  actionLabel: '',
  secondary: false,
  bare: false,
});

defineEmits(['action']);
</script>
