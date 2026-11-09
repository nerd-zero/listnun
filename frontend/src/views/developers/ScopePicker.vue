<template>
  <fieldset class="scope-picker">
    <legend class="ln-field-label">{{ $t('developers.keys.scopes') }}</legend>
    <p class="ln-help">{{ $t('developers.keys.scopesHelp') }}</p>
    <label v-for="s in API_KEY_SCOPES" :key="s" class="scope-option" :for="`scope-${s}`">
      <PvCheckbox v-model="model" :value="s" :input-id="`scope-${s}`" />
      <span>
        <span class="ln-mono">{{ s }}</span>
        <span class="ln-row-sub">{{ $t(`developers.keys.scope.${s.replace(':', '_')}`) }}</span>
      </span>
    </label>
  </fieldset>
</template>

<script setup lang="ts">
import { API_KEY_SCOPES, type APIKeyScope } from '../../api/developers';

// ScopePicker: a checkbox card per key scope. None are on by default.
const model = defineModel<APIKeyScope[]>({ required: true });
</script>

<style scoped lang="scss">
.scope-picker {
  border: 0;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;

  legend { padding: 0; margin-bottom: 2px; }
}
.scope-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius);
  cursor: pointer;

  &:hover { background: var(--ln-hover); }
}
</style>
