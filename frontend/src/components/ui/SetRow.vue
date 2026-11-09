<template>
  <!-- SetRow: one setting. Its name and a short help line on the left, the
       control on the right; "wide" stacks them for controls that need the
       full width (editors, lists). -->
  <div class="set-row" :class="{ 'is-wide': wide, 'is-switch': toggle }">
    <div class="set-row-text">
      <label v-if="labelFor" :for="labelFor" class="set-row-label">{{ label }}</label>
      <span v-else class="set-row-label">{{ label }}</span>
      <span v-if="help || $slots.help" class="ln-small set-row-help"><slot name="help">{{ help }}</slot></span>
    </div>
    <div class="set-row-control">
      <slot />
    </div>
    <!-- A warning about this setting, under the whole row. -->
    <span v-if="$slots.note" class="ln-small set-row-note"><slot name="note" /></span>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label: string;
  help?: string;
  // The id of the control, so clicking the name focuses it.
  labelFor?: string;
  wide?: boolean;
  // A toggle: the control hugs the right edge.
  toggle?: boolean;
}>(), {
  help: '',
  labelFor: '',
  wide: false,
  toggle: false,
});
</script>

<style scoped lang="scss">
.set-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 320px);
  gap: 8px 24px;
  align-items: center;
  padding: 14px 20px;

  & + & { border-top: 1px solid var(--ln-border); }
  &.is-switch { grid-template-columns: minmax(0, 1fr) auto; }
  &.is-wide { grid-template-columns: minmax(0, 1fr); }

  @media (max-width: 767px) {
    grid-template-columns: minmax(0, 1fr);

    &.is-switch { grid-template-columns: minmax(0, 1fr) auto; }
  }
}
.set-row-text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.set-row-label { font-weight: 500; }
.set-row-help { color: var(--ln-text-3); }
.set-row-control {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;

  :deep(.p-inputtext),
  :deep(.p-select),
  :deep(.p-autocomplete),
  :deep(.p-inputnumber),
  :deep(.p-textarea) { width: 100%; }

  // Segmented choices fill the column like the other controls.
  :deep(.p-multiselect) { width: 100%; align-items: center; }
  :deep(.p-selectbutton) { display: flex; }
  :deep(.p-selectbutton .p-togglebutton) { flex: 1; }

  .is-switch & { align-items: flex-end; }
  // Read-only values (set elsewhere) look like values, not fields.
  :deep(.p-inputtext[readonly]) { background: var(--ln-muted); color: var(--ln-text-2); }
}
.set-row-note { grid-column: 1 / -1; color: var(--ln-warn-fg); }
</style>
