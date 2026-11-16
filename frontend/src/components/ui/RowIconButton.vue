<template>
  <button
    v-tooltip.bottom="label"
    type="button"
    class="ln-row-menu"
    :aria-label="label"
    :aria-haspopup="menu ? 'true' : undefined"
    :data-cy="cy || undefined"
    @click.stop="(e) => emit('click', e)"
  >
    <i :class="['pi', icon]" aria-hidden="true" />
  </button>
</template>

<script setup lang="ts">
// RowIconButton is a row's or card's icon-only action, named by its
// tooltip. It doesn't let the click reach a clickable row behind it.
defineProps<{
  icon: string;
  label: string;
  cy?: string;
  // menu marks a button that opens a menu.
  menu?: boolean;
}>();

const emit = defineEmits<{ click: [e: MouseEvent] }>();
</script>
