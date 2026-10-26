<template>
  <!-- SetCard: a titled group of settings rows. Collapsible cards (like
       "Advanced") start closed and show a one-line summary instead. -->
  <section class="ln-card set-card" :class="{ 'is-closed': collapsible && !open }">
    <header class="set-card-head">
      <span class="set-card-title">
        <h3>{{ title }}</h3>
        <span v-if="help" class="ln-small">{{ help }}</span>
      </span>
      <button
        v-if="collapsible"
        type="button"
        class="set-card-toggle"
        :aria-expanded="open"
        @click="open = !open"
      >
        {{ open ? $t('settings.card.hide') : $t('settings.card.show') }}
        <i :class="`pi ${open ? 'pi-chevron-up' : 'pi-chevron-down'}`" aria-hidden="true" />
      </button>
    </header>
    <p v-if="collapsible && !open && summary" class="set-card-summary ln-small">{{ summary }}</p>
    <div v-show="!collapsible || open" class="set-card-rows">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const props = withDefaults(defineProps<{
  title: string;
  help?: string;
  collapsible?: boolean;
  // What a closed card shows instead of its rows.
  summary?: string;
  // Opens a collapsible card at first, e.g. when it holds an error.
  startOpen?: boolean;
}>(), {
  help: '',
  collapsible: false,
  summary: '',
  startOpen: false,
});

const open = ref(props.startOpen);
</script>

<style scoped lang="scss">
.set-card { overflow: hidden; }
.set-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--ln-border);

  .is-closed & { border-bottom: 0; }
}
.set-card-title {
  display: flex;
  flex-direction: column;
  gap: 2px;

  h3 { margin: 0; font-size: 15px; font-weight: 600; }
}
.set-card-toggle {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px;
  border: 0;
  border-radius: var(--ln-radius);
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  i { font-size: 10px; }
  &:hover { background: var(--ln-hover); }
}
.set-card-summary { margin: -6px 20px 14px; }
</style>
