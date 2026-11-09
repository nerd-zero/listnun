<template>
  <section class="ln-table-card" :class="{ 'has-bulk': selectedCount > 0 }">
    <div v-if="$slots.toolbar" class="ln-table-toolbar">
      <slot name="toolbar" />
    </div>

    <!-- Slides open above the table, which eases down rather than jumping. -->
    <ln-collapse :open="selectedCount > 0">
      <div class="ln-bulk-bar" role="toolbar" :aria-label="$t('globals.messages.numSelected', { num: selectedCount })">
        <strong>{{ $t('globals.messages.numSelected', { num: selectedLabel ?? selectedCount }) }}</strong>
        <slot name="bulk" />
        <span class="ln-bulk-spacer" />
        <button type="button" class="ln-bulk-clear" @click="$emit('clear')">{{ $t('globals.buttons.clear') }}</button>
      </div>
    </ln-collapse>

    <error-state
      v-if="error"
      class="ln-table-error"
      :title="errorTitle || $t('states.couldntLoad')"
      :error="error"
      @retry="$emit('retry')"
    />
    <slot v-else />
  </section>
</template>

<script setup lang="ts">
import ErrorState, { type RequestError } from './ErrorState.vue';
import LnCollapse from './LnCollapse.vue';

// TableCard: a DataTable inside a card, with an optional toolbar row, the
// ink bulk bar shown while rows are selected, and an error state in place
// of the table when loading failed.
withDefaults(defineProps<{
  selectedCount?: number;
  // Shown instead of the count, e.g. "All 48,219".
  selectedLabel?: string | number;
  error?: RequestError | null;
  errorTitle?: string;
}>(), {
  selectedCount: 0,
  selectedLabel: undefined,
  error: null,
  errorTitle: '',
});

defineEmits(['clear', 'retry']);
</script>
