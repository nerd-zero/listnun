<template>
  <fieldset class="scope-picker">
    <legend class="ln-field-label">{{ $t('developers.keys.scopes') }}</legend>
    <p class="ln-help">{{ $t('developers.keys.scopesHelp') }}</p>
    <label v-for="s in API_KEY_SCOPES" :key="s" class="scope-option" :for="`scope-${s}`">
      <PvCheckbox
        :model-value="model"
        :value="s"
        :input-id="`scope-${s}`"
        :disabled="requiredByOthers.has(s)"
        @update:model-value="(v: APIKeyScope[]) => (model = withRequiredScopes(v))"
      />
      <span>
        <span class="ln-mono">{{ s }}</span>
        <span class="ln-row-sub">{{ $t(`developers.keys.scope.${s.replace(':', '_')}`) }}</span>
      </span>
    </label>
  </fieldset>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  API_KEY_SCOPES, API_KEY_SCOPE_REQUIRES, withRequiredScopes, type APIKeyScope,
} from '../../api/developers';

// ScopePicker: a checkbox card per key scope. None are on by default.
// Ticking a scope ticks the ones it needs, which stay ticked (and locked)
// while something picked needs them.
const model = defineModel<APIKeyScope[]>({ required: true });

const requiredByOthers = computed(() => new Set(
  model.value.flatMap((s) => API_KEY_SCOPE_REQUIRES[s] || []),
));
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
