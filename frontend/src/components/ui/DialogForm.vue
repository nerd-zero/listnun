<template>
  <PvDialog
    :visible="visible"
    modal
    :header="title"
    :maximizable="maximizable"
    :draggable="false"
    :dismissable-mask="!busy"
    :closable="!busy"
    class="ln-dialog-form"
    :style="{ width, maxWidth: 'calc(100vw - 32px)' }"
    @update:visible="(v) => $emit('update:visible', v)"
    @hide="$emit('hide')"
  >
    <template v-if="$slots.header" #header>
      <slot name="header" />
    </template>

    <form :id="formId" class="ln-dialog-form-body" novalidate @submit.prevent="$emit('submit')">
      <slot />
    </form>

    <template #footer>
      <slot name="footer">
        <slot name="footer-start" />
        <span class="ln-dialog-form-spacer" />
        <PvButton
          type="button"
          :label="cancelLabel || $t('globals.buttons.cancel')"
          severity="secondary"
          outlined
          :disabled="busy"
          @click="$emit('update:visible', false)"
        />
        <PvButton
          v-if="submitLabel"
          type="submit"
          :form="formId"
          :label="submitLabel"
          :severity="danger ? 'danger' : undefined"
          :loading="busy"
          :disabled="submitDisabled"
          data-cy="btn-save"
        />
      </slot>
    </template>
  </PvDialog>
</template>

<script setup lang="ts">
// DialogForm: a Dialog with a header, a scrolling body that is a <form>,
// and a sticky footer with Cancel and the submit action. Used by the
// list, subscriber, template, role and user forms.
withDefaults(defineProps<{
  visible: boolean;
  title?: string;
  submitLabel?: string;
  cancelLabel?: string;
  busy?: boolean;
  submitDisabled?: boolean;
  danger?: boolean;
  maximizable?: boolean;
  width?: string;
}>(), {
  title: '',
  submitLabel: '',
  cancelLabel: '',
  busy: false,
  submitDisabled: false,
  danger: false,
  maximizable: false,
  width: '480px',
});

defineEmits(['update:visible', 'submit', 'hide']);

const formId = `ln-dialog-form-${Math.random().toString(36).slice(2, 9)}`;
</script>
