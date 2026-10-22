<template>
  <div class="ln-page forms">
    <page-header :title="$t('forms.title')" :subtitle="$t('forms.intro')" />

    <div v-if="loading.lists && !publicLists.length" class="forms-grid">
      <PvSkeleton height="260px" />
      <PvSkeleton height="420px" />
    </div>

    <empty-state
      v-else-if="!publicLists.length"
      icon="pi-list"
      :title="$t('forms.noPublicListsTitle')"
      :text="$t('forms.noPublicLists')"
      :action-label="$can('lists:manage_all') ? $t('forms.goToLists') : ''"
      @action="$router.push({ name: 'lists' })"
    />

    <div v-else class="forms-grid">
      <!-- Options -->
      <div class="forms-side">
        <section class="ln-card forms-lists" data-cy="lists">
          <header class="forms-head">
            <h2 class="ln-card-title">{{ $t('forms.publicLists') }}</h2>
            <p class="ln-small">{{ $t('forms.privateHelp') }}</p>
          </header>
          <label v-for="l in publicLists" :key="l.id" class="forms-list" :for="`list-${l.id}`" data-cy="list">
            <PvCheckbox v-model="checked" :value="l.id" :input-id="`list-${l.id}`" />
            <span>
              <strong>{{ l.name }}</strong>
              <span v-if="l.optin" class="ln-small">{{ $t(`lists.optins.${l.optin}`) }}</span>
            </span>
          </label>
          <div class="forms-opts">
            <label class="ln-check-row" for="forms-name">
              <PvCheckbox v-model="askName" binary input-id="forms-name" />
              {{ $t('forms.askName') }}
            </label>
            <label class="ln-check-row" for="forms-captcha" :class="{ 'is-off': !captchaOn }">
              <PvCheckbox v-model="addCaptcha" binary input-id="forms-captcha" :disabled="!captchaOn" />
              {{ $t('forms.addCaptcha') }}
            </label>
            <p v-if="!captchaOn" class="ln-help">
              {{ $t('forms.captchaOff') }}
              <router-link v-if="$can('settings:get')" :to="{ name: 'settings', params: { section: 'security' } }">
                {{ $t('settings.security.name') }}
              </router-link>
            </p>
          </div>
        </section>

        <section v-if="redirectURLs.length" class="ln-card ln-card-pad forms-redirect" data-cy="redirect-urls">
          <h2 class="ln-card-title">{{ $t('forms.redirectURL') }}</h2>
          <p class="ln-small">{{ $t('forms.redirectURLHelp') }}</p>
          <label class="ln-check-row" for="redirect-none">
            <PvRadioButton v-model="selectedRedirectURL" value="" input-id="redirect-none" />
            {{ $t('globals.terms.none') }}
          </label>
          <label v-for="url in redirectURLs" :key="url" class="ln-check-row" :for="`redirect-${url}`">
            <PvRadioButton v-model="selectedRedirectURL" :value="url" :input-id="`redirect-${url}`" />
            <span class="ln-mono forms-url">{{ url }}</span>
          </label>
        </section>

        <section v-if="sc.public_subscription?.enabled" class="ln-card ln-card-pad forms-page-link">
          <h2 class="ln-card-title">{{ $t('forms.publicSubPage') }}</h2>
          <a :href="publicURL" target="_blank" rel="noopener noreferrer" class="ln-mono forms-url" data-cy="url">
            {{ publicURL }} <i class="pi pi-external-link" aria-hidden="true" />
          </a>
        </section>
      </div>

      <!-- Preview and HTML -->
      <div class="forms-main" data-cy="form">
        <section class="ln-card forms-preview">
          <header class="ln-card-head">
            <h2 class="ln-card-title">{{ $t('templates.preview') }}</h2>
            <span class="ln-small">{{ $t('forms.unstyled') }}</span>
          </header>
          <div class="forms-stage">
            <iframe class="forms-frame" data-cy="preview" sandbox="" :srcdoc="previewDoc" :title="$t('templates.preview')" />
          </div>
        </section>

        <section class="ln-card forms-html">
          <header class="ln-card-head">
            <h2 class="ln-card-title">HTML</h2>
            <PvButton :class="{ 'ln-copied': copied }"
              :icon="copied ? 'pi pi-check' : 'pi pi-copy'"
              :label="copied ? $t('states.copied') : $t('forms.copyHTML')"
              size="small"
              :disabled="!checked.length"
              @click="copyHTML"
            />
          </header>
          <p v-if="!checked.length" class="forms-none ln-small" data-cy="no-selection">{{ $t('forms.selectHelp') }}</p>
          <code-editor v-else data-cy="html" :model-value="html" lang="html" theme="ink" :autofocus="false" disabled class="forms-code" />
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import CodeEditor from '../components/CodeEditor.vue';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';

// Lists and server config are the API's loosely typed objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// How long the Copy HTML button shows its confirmation.
const COPIED_MS = 1600;

const { $utils } = useGlobal();
const { t } = useI18n();
const { loading, lists, serverConfig } = storeToRefs(useMainStore());

const checked = ref<number[]>([]);
const askName = ref(true);
const addCaptcha = ref(true);
const selectedRedirectURL = ref('');
const copied = ref(false);

const sc = computed<Loose>(() => serverConfig.value as Loose);
const publicLists = computed<Loose[]>(() => ((lists.value as Loose).results || []).filter((l: Loose) => l.type === 'public'));
const publicURL = computed(() => `${sc.value.root_url}/subscription/form`);
const captchaOn = computed(() => !!sc.value.public_subscription?.captcha_enabled);

const redirectURLs = computed<string[]>(() => {
  const urls = sc.value.public_subscription?.redirect_urls;
  return Array.isArray(urls) ? urls : [];
});

// Preselect every public list the first time they load.
watch(publicLists, (ls) => {
  if (!checked.value.length && ls.length) checked.value = ls.map((l) => l.id);
}, { immediate: true });

function esc(value: unknown) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function captchaHTML(): string {
  const ps = sc.value.public_subscription || {};
  if (!captchaOn.value || !addCaptcha.value) return '';
  if (ps.captcha_provider === 'altcha') {
    return '\n'
      + `    <altcha-widget challengeurl="${sc.value.root_url}/api/public/captcha/altcha"></altcha-widget>\n`
      + `    <${'script'} type="module" src="${sc.value.root_url}/public/static/altcha.umd.js" async defer></${'script'}>\n`;
  }
  if (ps.captcha_provider === 'turnstile') {
    return '\n'
      + `    <div class="cf-turnstile" data-sitekey="${esc(ps.captcha_key)}"></div>\n`
      + `    <${'script'} src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></${'script'}>\n`;
  }
  if (ps.captcha_provider === 'hcaptcha') {
    return '\n'
      + `    <div class="h-captcha" data-sitekey="${esc(ps.captcha_key)}"></div>\n`
      + `    <${'script'} src="https://js.hcaptcha.com/1/api.js" async defer></${'script'}>\n`;
  }
  return '';
}

const html = computed(() => {
  let h = `<form method="post" action="${publicURL.value}" class="listnun-form">\n`
    + '  <div>\n'
    + `    <h3>${esc(t('public.sub'))}</h3>\n`
    + '    <input type="hidden" name="nonce" />\n';

  if (selectedRedirectURL.value) {
    h += `    <input type="hidden" name="next" value="${esc(selectedRedirectURL.value)}" />\n`;
  }

  h += '\n'
    + `    <p><input type="email" name="email" required placeholder="${esc(t('subscribers.email'))}" /></p>\n`;
  if (askName.value) {
    h += `    <p><input type="text" name="name" placeholder="${esc(t('public.subName'))}" /></p>\n`;
  }
  h += '\n';

  publicLists.value.filter((l) => checked.value.includes(l.id)).forEach((l) => {
    const id = l.uuid.substring(0, 5);
    h += '    <p>\n'
      + `      <input id="${id}" type="checkbox" name="l" checked value="${l.uuid}" />\n`
      + `      <label for="${id}">${esc(l.name)}</label>\n`;
    if (l.description) {
      h += `      <br />\n      <span>${esc(l.description)}</span>\n`;
    }
    h += '    </p>\n';
  });

  h += captchaHTML();
  h += '\n'
    + `    <input type="submit" value="${esc(t('public.sub'))}" />\n`
    + '  </div>\n'
    + '</form>';
  return h;
});

// The preview frame is sandboxed, so captcha scripts never run and the form
// can't be submitted from here.
const previewDoc = computed(() => '<!doctype html><html><head><meta charset="utf-8">'
  + '<style>body{margin:0;padding:24px;font:16px Georgia,serif;color:#111;background:#fff}'
  + 'h3{margin:0 0 12px}input[type=email],input[type=text]{width:100%;box-sizing:border-box;padding:8px}'
  + '</style></head><body>'
  + `${checked.value.length ? html.value : `<p>${esc(t('forms.selectHelp'))}</p>`}</body></html>`);

function copyHTML() {
  navigator.clipboard?.writeText(html.value).then(() => {
    copied.value = true;
    $utils.toast(t('forms.copied'));
    window.setTimeout(() => { copied.value = false; }, COPIED_MS);
  });
}
</script>

<style scoped lang="scss">
.forms-grid {
  display: grid;
  grid-template-columns: minmax(0, 400px) minmax(0, 1fr);
  gap: 20px;
  align-items: start;
  @media (max-width: 960px) { grid-template-columns: minmax(0, 1fr); }
}
.forms-side, .forms-main { display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.forms-head {
  padding: 14px 18px;
  border-bottom: 1px solid var(--ln-border);
  .ln-small { margin: 2px 0 0; }
}
.forms-list {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--ln-border);
  cursor: pointer;
  > span { display: flex; flex-direction: column; gap: 2px; }
  strong { font-weight: 500; }
  &:hover { background: var(--ln-hover); }
}
.forms-opts {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px;
  .is-off { color: var(--ln-text-3); cursor: default; }
  .ln-help { margin: -4px 0 0; }
}
.forms-redirect, .forms-page-link {
  display: flex;
  flex-direction: column;
  gap: 10px;
  .ln-small { margin: 0; }
}
.forms-url { font-size: 12px; overflow-wrap: anywhere; }
.forms-page-link a { color: var(--ln-text); text-decoration: none; &:hover { text-decoration: underline; } }
.forms-preview { overflow: hidden; }
.forms-stage {
  padding: 24px;
  background: repeating-linear-gradient(135deg, var(--ln-muted) 0 12px, var(--ln-bg) 12px 24px);
}
.forms-frame {
  display: block;
  width: 100%;
  max-width: 420px;
  height: 340px;
  margin: 0 auto;
  border: 0;
  border-radius: 6px;
  background: #fff;
}
.forms-html { overflow: hidden; }
.forms-none { margin: 0; padding: 24px 18px; }
.forms-code :deep(.cm-editor) { border-radius: 0; max-height: 420px; }
</style>
