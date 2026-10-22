<template>
  <div class="field list-selector">
    <div :class="['list-tags', ...classes]">
      <div class="tags">
        <PvTag v-for="l in selectedItems" :key="l.id" :class="[l.subscriptionStatus, { 'is-restricted': l.restricted }, 'list']"
          :data-id="l.id" class="list">
          {{ l.name }}
          <sup v-if="l.optin === 'double' && l.subscriptionStatus">
            {{ $t(`subscribers.status.${l.subscriptionStatus}`) }}
          </sup>
          <i v-if="!disabled && !l.restricted" class="pi pi-times ml-1 cursor-pointer" @click="removeList(l.id)" />
        </PvTag>
      </div>
    </div>

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ label + (selectedItems ? ` (${selectedItems.length})` : '') }}</label>
      <PvAutoComplete v-model="query" :placeholder="placeholder" data-cy="list-selector"
        :disabled="all.length === 0 || disabled"
        :suggestions="suggestions" @complete="onSearch" @item-select="onSelect" option-label="name"
        :dropdown="true" force-selection class="w-full" />
      <small v-if="message" class="block mt-1 text-color-secondary">{{ message }}</small>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref, watch, nextTick, onMounted,
} from 'vue';

const props = withDefaults(defineProps<{
  label?: string;
  placeholder?: string;
  message?: string;
  required?: boolean;
  disabled?: boolean;
  classes?: unknown[];
  selected?: unknown[];
  all?: unknown[];
}>(), {
  label: '',
  placeholder: '',
  message: '',
  required: false,
  disabled: false,
  classes: () => [],
  selected: () => [],
  all: () => [],
});

const emit = defineEmits(['update:modelValue']);

const query = ref('');
const selectedItems = ref<any[]>([]);
const suggestions = ref<unknown[]>([]);

function onSearch(event: { query: string }) {
  const q = (event.query || '').toLowerCase();
  const subIDs = selectedItems.value.reduce((obj: Record<number, boolean>, item: any) => ({ ...obj, [item.id]: true }), {});
  suggestions.value = (props.all as any[]).filter(
    (l) => !(l.id in subIDs) && l.name.toLowerCase().includes(q),
  );
}

function onSelect(event: { value: unknown }) {
  selectList(event.value);
}

function selectList(l: any) {
  if (!l) return;
  selectedItems.value.push(l);
  query.value = '';
  nextTick(() => {
    emit('update:modelValue', selectedItems.value);
  });
}

function removeList(id: number) {
  selectedItems.value = selectedItems.value.filter((l: any) => l.id !== id);
  nextTick(() => {
    emit('update:modelValue', selectedItems.value);
  });
}

watch(
  () => props.selected,
  () => {
    selectedItems.value = JSON.parse(JSON.stringify(props.selected));
  },
);

onMounted(() => {
  if (props.selected) {
    selectedItems.value = JSON.parse(JSON.stringify(props.selected));
  }
});
</script>
