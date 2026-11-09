<template>
  <PvDialog
    :visible="visible"
    modal
    :closable="false"
    :draggable="false"
    :dismissable-mask="!busy"
    class="ln-confirm"
    :style="{ width: '440px', maxWidth: 'calc(100vw - 32px)' }"
    @update:visible="(v) => $emit('update:visible', v)"
  >
    <template #container>
      <form class="ln-confirm-body" @submit.prevent="onConfirm">
        <span v-if="danger" class="ln-confirm-icon" aria-hidden="true"><i class="pi pi-exclamation-triangle" /></span>
        <h2 class="ln-confirm-title">{{ title }}</h2>
        <p v-if="text" class="ln-confirm-text">{{ text }}</p>
        <slot />
        <div v-if="confirmText" class="ln-field">
          <label :for="inputId">
            <i18n-t keypath="confirm.typeToConfirm" tag="span">
              <template #value><code class="ln-mono">{{ confirmText }}</code></template>
            </i18n-t>
          </label>
          <PvInputText :id="inputId" v-model="typed" autocomplete="off" spellcheck="false" class="ln-mono" />
        </div>
      </form>
      <div class="ln-confirm-footer">
        <slot name="footer-start" />
        <span class="ln-dialog-form-spacer" />
        <PvButton
          type="button"
          :label="$t('globals.buttons.cancel')"
          severity="secondary"
          outlined
          :disabled="busy"
          @click="$emit('update:visible', false)"
        />
        <PvButton
          type="button"
          :label="confirmLabel"
          :severity="danger ? 'danger' : undefined"
          :loading="busy"
          :disabled="!canConfirm"
          data-cy="btn-confirm"
          @click="onConfirm"
        />
      </div>
    </template>
  </PvDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

// ConfirmAction: a destructive confirmation that names the thing, says what
// happens and puts the action on the right. With confirmText set, the
// action stays disabled until the user types it exactly.
const props = withDefaults(defineProps<{
  visible: boolean;
  title: string;
  text?: string;
  confirmLabel: string;
  confirmText?: string;
  danger?: boolean;
  busy?: boolean;
}>(), {
  text: '',
  confirmText: '',
  danger: true,
  busy: false,
});

const emit = defineEmits(['update:visible', 'confirm']);

const typed = ref('');
const inputId = `ln-confirm-${Math.random().toString(36).slice(2, 9)}`;

const canConfirm = computed(() => !props.confirmText || typed.value.trim() === props.confirmText);

watch(() => props.visible, (v) => { if (v) typed.value = ''; });

function onConfirm() {
  if (canConfirm.value && !props.busy) emit('confirm');
}
</script>
