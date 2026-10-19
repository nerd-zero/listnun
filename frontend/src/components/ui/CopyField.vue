<template>
  <PvInputGroup class="ln-copy-field">
    <PvInputGroupAddon v-if="prefix" class="ln-mono">{{ prefix }}</PvInputGroupAddon>
    <PvInputText :id="id" :model-value="value" readonly class="ln-mono" @focus="(e) => (e.target as HTMLInputElement).select()" />
    <PvButton
      type="button"
      :icon="copied ? 'pi pi-check' : 'pi pi-copy'"
      :label="iconOnly ? undefined : (copied ? $t('states.copied') : $t('globals.buttons.copy'))"
      :aria-label="$t('globals.buttons.copy')"
      :severity="strong ? undefined : 'secondary'"
      :outlined="!strong"
      @click="copy"
    />
  </PvInputGroup>
</template>

<script setup lang="ts">
import { ref } from 'vue';

// CopyField: a read-only value with a copy button (InputGroup).
const props = withDefaults(defineProps<{
  value: string;
  id?: string;
  prefix?: string;
  iconOnly?: boolean;
  // Ink copy button, used for secrets shown once.
  strong?: boolean;
}>(), {
  id: undefined,
  prefix: '',
  iconOnly: false,
  strong: false,
});

const emit = defineEmits(['copied']);

const copied = ref(false);

function copy() {
  navigator.clipboard?.writeText(props.value).then(() => {
    copied.value = true;
    emit('copied');
    setTimeout(() => { copied.value = false; }, 1500);
  });
}
</script>
