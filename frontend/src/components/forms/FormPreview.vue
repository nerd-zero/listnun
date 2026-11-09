<template>
  <!-- FormPreview: the hosted page's form as visitors see it, in either
       theme. It mirrors static/public/templates/form-page.html. -->
  <section class="ln-card fp">
    <header class="fp-head">
      <h2>{{ $t('templates.preview') }}</h2>
      <PvSelectButton v-model="theme" :options="themes" option-label="label" option-value="value" :allow-empty="false" :aria-label="$t('forms.previewTheme')" />
    </header>
    <div class="fp-stage" :class="`is-${theme}`">
      <div class="fp-card" inert>
        <p class="fp-site">{{ siteName }}</p>
        <h3>{{ draft.heading || draft.name }}</h3>
        <p v-if="draft.intro" class="fp-intro">{{ draft.intro }}</p>
        <div v-for="f in draft.fields" :key="f.key" class="fp-field">
          <span class="fp-label">{{ f.label }}<span v-if="!f.required" class="fp-opt">{{ $t('forms.optional') }}</span></span>
          <span class="fp-input">{{ f.key === 'email' ? 'you@example.com' : '' }}</span>
        </div>
        <ul v-if="lists.length > 1" class="fp-lists">
          <li v-for="l in lists" :key="l.id"><span class="fp-check"><i class="pi pi-check" /></span>{{ l.name }}</li>
        </ul>
        <div v-if="captcha" class="fp-captcha">{{ $t('forms.turnstile') }}</div>
        <span class="fp-button">{{ draft.buttonLabel || $t('public.sub') }}</span>
        <p class="fp-note">{{ $t('forms.unsubAnytime') }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { FormDraft, PublicList } from './types';

defineProps<{ draft: FormDraft; lists: PublicList[]; siteName: string; captcha: boolean }>();
const { t } = useI18n();

const theme = ref<'light' | 'dark'>('light');
const themes = computed(() => [
  { label: t('forms.light'), value: 'light' },
  { label: t('forms.dark'), value: 'dark' },
]);
</script>

<style scoped lang="scss">
.fp { overflow: hidden; }
.fp-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ln-border);

  h2 { margin: 0; font-size: 15px; font-weight: 600; }
}

// The public pages' own colours (static/public/static/style.css).
.fp-stage {
  --p-bg: #f6f3ec;
  --p-card: #fff;
  --p-text: #1b1e24;
  --p-text-2: #4a4e56;
  --p-text-3: #6b6f76;
  --p-border: #dedad0;
  --p-field: #cbc6b9;
  --p-ink: #1b1e24;
  --p-on-ink: #f6f3ec;

  padding: 32px 24px;
  background: var(--p-bg);
  color: var(--p-text);

  &.is-dark {
    --p-bg: #14171c;
    --p-card: #1b1f26;
    --p-text: #ece9e1;
    --p-text-2: #c7c3ba;
    --p-text-3: #9599a0;
    --p-border: #2b2f36;
    --p-field: #3a3f47;
    --p-ink: #ece9e1;
    --p-on-ink: #14171c;
  }

  @media (max-width: 767px) { padding: 16px 12px; }
}
.fp-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 420px;
  margin: 0 auto;
  padding: 24px;
  border: 1px solid var(--p-border);
  border-radius: 10px;
  background: var(--p-card);

  h3 { margin: 4px 0 0; font-size: 20px; font-weight: 600; overflow-wrap: anywhere; }
}
.fp-site { margin: 0; font-size: 13px; font-weight: 600; color: var(--p-text-2); }
.fp-intro { margin: -6px 0 0; color: var(--p-text-2); overflow-wrap: anywhere; }
.fp-field { display: flex; flex-direction: column; gap: 6px; }
.fp-label { font-size: 14px; font-weight: 600; }
.fp-opt { margin-left: 4px; font-weight: 400; color: var(--p-text-3); }
.fp-input {
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--p-field);
  border-radius: 6px;
  color: var(--p-text-3);
  font-size: 14px;
  line-height: 36px;
}
.fp-lists {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;

  li { display: flex; align-items: center; gap: 8px; font-size: 14px; }
}
.fp-check {
  display: inline-grid;
  place-items: center;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  background: var(--p-ink);
  color: var(--p-on-ink);

  .pi { font-size: 9px; }
}
.fp-captcha {
  padding: 18px;
  border: 1px dashed var(--p-field);
  border-radius: 6px;
  color: var(--p-text-3);
  font-size: 13px;
  text-align: center;
}
.fp-button {
  padding: 10px;
  border-radius: 6px;
  background: var(--p-ink);
  color: var(--p-on-ink);
  font-weight: 600;
  text-align: center;
}
.fp-note { margin: 0; font-size: 12px; color: var(--p-text-3); }
</style>
