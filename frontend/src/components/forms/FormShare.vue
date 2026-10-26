<template>
  <!-- FormShare: the three ways to put a live form in front of people. A
       draft says how to publish instead. -->
  <section class="ln-card fs">
    <header class="fs-head">
      <h2>{{ $t('forms.share') }}</h2>
    </header>

    <div v-if="form.status !== 'live'" class="fs-body">
      <p class="ln-small">{{ $t('forms.draftHelp') }}</p>
      <PvButton :label="$t('forms.publish')" :loading="busy" data-cy="btn-publish" @click="emit('publish')" />
    </div>

    <template v-else>
      <PvTabs v-model:value="tab" class="fs-tabs">
        <PvTabList>
          <PvTab value="page">{{ $t('forms.hosted') }}</PvTab>
          <PvTab value="embed">{{ $t('forms.embed') }}</PvTab>
          <PvTab value="popup">{{ $t('forms.popup') }}</PvTab>
        </PvTabList>
      </PvTabs>
      <div class="fs-body">
        <p class="ln-small">{{ help }}</p>
        <div v-if="tab === 'page'" class="fs-copy">
          <span class="ln-mono fs-url">{{ displayURL }}</span>
          <PvButton :label="copied === 'page' ? $t('states.copied') : $t('forms.copy')" size="small" @click="copy('page', form.url)" />
        </div>
        <pre v-else class="ln-code fs-code">{{ code }}</pre>
        <span class="fs-actions">
          <a v-if="tab === 'page'" :href="form.url" target="_blank" rel="noopener noreferrer" data-cy="form-url">{{ $t('forms.openPage') }} <i class="pi pi-external-link" aria-hidden="true" /></a>
          <PvButton v-else :label="copied === tab ? $t('states.copied') : $t('forms.copyCode')" size="small" severity="secondary" outlined @click="copy(tab, code)" />
          <button type="button" class="fs-unpublish" @click="emit('unpublish')">{{ $t('forms.unpublish') }}</button>
        </span>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { SignupForm } from './types';

// How long a Copy button says "Copied".
const COPIED_MS = 1600;

const props = defineProps<{ form: SignupForm; busy: boolean }>();
const emit = defineEmits<{ publish: []; unpublish: [] }>();
const { t } = useI18n();

const tab = ref<'page' | 'embed' | 'popup'>('page');
const copied = ref('');

const displayURL = computed(() => props.form.url.replace(/^https?:\/\//, ''));
const code = computed(() => (tab.value === 'embed' ? props.form.embedHtml : props.form.popupScript));
const help = computed(() => ({
  page: t('forms.hostedHelp'),
  embed: t('forms.embedHelp'),
  popup: t('forms.popupHelp'),
})[tab.value]);

function copy(what: string, text: string) {
  navigator.clipboard.writeText(text).then(() => {
    copied.value = what;
    window.setTimeout(() => { if (copied.value === what) copied.value = ''; }, COPIED_MS);
  });
}
</script>

<style scoped lang="scss">
.fs { overflow: hidden; }
.fs-head { padding: 14px 16px 6px; h2 { margin: 0; font-size: 15px; font-weight: 600; } }
.fs-tabs { padding: 0 16px; border-bottom: 1px solid var(--ln-border); }
.fs-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;

  .ln-small { margin: 0; color: var(--ln-text-2); }
}
.fs-copy {
  display: flex;
  border: 1px solid var(--ln-border);
  border-radius: 6px;
  overflow: hidden;

  .p-button { border-radius: 0; flex: none; }
}
.fs-url {
  flex: 1;
  min-width: 0;
  padding: 8px 10px;
  background: var(--ln-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fs-code {
  max-height: 220px;
  margin: 0;
  padding: 10px 12px;
  border-radius: 6px;
  font-size: 12px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.fs-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;

  a { color: var(--ln-text); font-weight: 500; text-decoration: underline; }
  a .pi { font-size: 10px; }
}
.fs-unpublish {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--ln-text-3);
  font: inherit;
  font-size: 13px;
  cursor: pointer;

  &:hover { color: var(--ln-text); text-decoration: underline; }
}
</style>
