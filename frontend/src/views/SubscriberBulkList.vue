<template>
  <form class="lm-form" @submit.prevent="onSubmit">
    <div class="lm-form-header">
      <h3 class="lm-form-title">{{ $t('subscribers.manageLists') }}</h3>
    </div>

    <div class="lm-form-body">
      <div class="lm-field">
        <label class="lm-label">Action</label>
        <div class="radio-group">
          <label class="radio-item">
            <PvRadioButton v-model="form.action" name="action" value="add" input-id="action-add" data-cy="check-list-add" />
            <span>{{ $t('globals.buttons.add') }}</span>
          </label>
          <label class="radio-item">
            <PvRadioButton v-model="form.action" name="action" value="remove" input-id="action-remove" data-cy="check-list-remove" />
            <span>{{ $t('globals.buttons.remove') }}</span>
          </label>
          <label class="radio-item">
            <PvRadioButton v-model="form.action" name="action" value="unsubscribe" input-id="action-unsubscribe" data-cy="check-list-unsubscribe" />
            <span>{{ $t('subscribers.markUnsubscribed') }}</span>
          </label>
        </div>
      </div>

      <list-selector label="Target lists" placeholder="Lists to apply to" v-model="form.lists" :selected="form.lists"
        :all="lists.results" />

      <div class="lm-field">
        <div class="check-row">
          <PvCheckbox v-model="form.preconfirm" data-cy="preconfirm" :binary="true" :true-value="true" :false-value="false" :disabled="!hasOptinList" input-id="preconfirm" />
          <label for="preconfirm" class="check-label">{{ $t('subscribers.preconfirm') }}</label>
        </div>
        <small class="lm-help">{{ $t('subscribers.preconfirmHelp') }}</small>
      </div>
    </div>

    <div class="lm-form-footer">
      <PvButton @click="$emit('close')" :label="$t('globals.buttons.close')" severity="secondary" />
      <PvButton type="submit" severity="primary" :disabled="form.lists.length === 0" :label="$t('globals.buttons.save')" />
    </div>
  </form>
</template>

<script setup lang="ts">
import { reactive, computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../store';
import ListSelector from '../components/ListSelector.vue';

withDefaults(defineProps<{ numSubscribers?: number }>(), { numSubscribers: 0 });
const emit = defineEmits(['finished', 'close']);

const { lists } = storeToRefs(useMainStore());

const form = reactive({ action: 'add', lists: [] as any[], preconfirm: false });

const hasOptinList = computed(() => form.lists.some((l: any) => l.optin === 'double'));

function onSubmit() {
  emit('finished', form.action, form.preconfirm, form.lists);
  emit('close');
}
</script>

<style scoped lang="scss">
.lm-field { display: flex; flex-direction: column; gap: 0.35rem; }
.lm-label { font-size: 0.8rem; font-weight: 600; color: var(--ln-text); }
.lm-help { font-size: 0.75rem; color: var(--ln-text-4); line-height: 1.4; }
.check-row { display: flex; align-items: center; gap: 0.5rem; }
.check-label { font-size: 0.875rem; color: var(--ln-text); cursor: pointer; }

.radio-group { display: flex; flex-direction: column; gap: 0.5rem; }
.radio-item { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; font-size: 0.875rem; color: var(--ln-text); }
</style>
