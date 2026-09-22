<template>
  <PvDialog :visible="visible" @update:visible="emit('update:visible', $event)"
    :style="{ width: '900px' }" :closable="true" modal :aria-modal="true">
    <media is-modal @selected="emit('select', $event)" @close="emit('update:visible', false)" />
  </PvDialog>
</template>

<script setup lang="ts">
import Media from '../views/Media.vue';

defineProps<{ visible: boolean }>();
const emit = defineEmits<{ 'update:visible': [boolean]; select: [any] }>();

// Media emits `close` the instant a file is picked (in is-modal mode) --
// forwarding it here means picking a file always closes this dialog.
</script>
