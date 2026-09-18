<template>
  <div class="forms-page">
    <div class="page-header">
      <h1 class="page-title">{{ $t('forms.title') }}</h1>
    </div>

    <div v-if="loading.lists" class="forms-loading">
      <PvProgressSpinner style="width:2rem;height:2rem" />
    </div>

    <PvMessage v-else-if="publicLists.length === 0" severity="info" :closable="false">
      {{ $t('forms.noPublicLists') }}
    </PvMessage>

    <div v-else class="forms-grid">
      <!-- Left panel: controls -->
      <div class="forms-panel" data-cy="lists">
        <div class="forms-panel-title">{{ $t('forms.publicLists') }}</div>
        <p class="forms-panel-subtitle">{{ $t('forms.selectHelp') }}</p>

        <div class="checklist">
          <div v-for="(l, i) in publicLists" :key="l.id" class="checklist-item">
            <PvCheckbox v-model="checked" :value="i" :input-id="`list-${l.id}`" />
            <label :for="`list-${l.id}`" class="checklist-label">{{ l.name }}</label>
          </div>
        </div>

        <template v-if="serverConfig.public_subscription?.enabled">
          <PvDivider />
          <p class="forms-section-label">{{ $t('forms.publicSubPage') }}</p>
          <a :href="`${serverConfig.root_url}/subscription/form`" target="_blank" rel="noopener noreferer"
            class="forms-ext-link" data-cy="url">
            <i class="pi pi-external-link" />
            {{ serverConfig.root_url }}/subscription/form
          </a>
        </template>

        <template v-if="redirectURLs.length > 0">
          <PvDivider />
          <p class="forms-section-label">{{ $t('forms.redirectURL') }}</p>
          <p class="forms-panel-subtitle">{{ $t('forms.redirectURLHelp') }}</p>
          <div class="checklist" data-cy="redirect-urls">
            <div class="checklist-item">
              <PvRadioButton v-model="selectedRedirectURL" value="" input-id="redirect-none" />
              <label for="redirect-none" class="checklist-label">{{ $t('globals.terms.none') }}</label>
            </div>
            <div v-for="url in redirectURLs" :key="url" class="checklist-item">
              <PvRadioButton v-model="selectedRedirectURL" :value="url" :input-id="`redirect-${url}`" />
              <label :for="`redirect-${url}`" class="checklist-label checklist-label--url">{{ url }}</label>
            </div>
          </div>
        </template>
      </div>

      <!-- Right panel: generated HTML -->
      <div class="forms-panel" data-cy="form">
        <div class="forms-panel-title">{{ $t('forms.formHTML') }}</div>
        <p class="forms-panel-subtitle">{{ $t('forms.formHTMLHelp') }}</p>

        <div v-if="checked.length === 0" class="forms-empty">
          <i class="pi pi-code forms-empty-icon" />
          <span>{{ $t('forms.selectHelp') }}</span>
        </div>
        <code-editor v-else lang="html" v-model="html" disabled />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import CodeEditor from '../components/CodeEditor.vue';

const { t } = useI18n();
const { loading, lists, serverConfig } = storeToRefs(useMainStore());

const checked = ref<any[]>([]);
const html = ref('');
const selectedRedirectURL = ref('');

const publicLists = computed(() => {
  if (!(lists.value as any).results) return [];
  return (lists.value as any).results.filter((l: any) => l.type === 'public');
});

const redirectURLs = computed(() => {
  const urls = (serverConfig.value as any).public_subscription
    ? (serverConfig.value as any).public_subscription.redirect_urls
    : [];
  return Array.isArray(urls) ? urls : [];
});

function escapeAttr(value: any) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderHTML() {
  const sc = serverConfig.value as any;
  let h = `<form method="post" action="${sc.root_url}/subscription/form" class="listnun-form">\n`
    + '  <div>\n'
    + `    <h3>${t('public.sub')}</h3>\n`
    + '    <input type="hidden" name="nonce" />\n';

  if (selectedRedirectURL.value) {
    h += `    <input type="hidden" name="next" value="${escapeAttr(selectedRedirectURL.value)}" />\n`;
  }

  h += '\n'
    + `    <p><input type="email" name="email" required placeholder="${t('subscribers.email')}" /></p>\n`
    + `    <p><input type="text" name="name" placeholder="${t('public.subName')}" /></p>\n\n`;

  checked.value.forEach((i) => {
    const l = publicLists.value[parseInt(i, 10)];
    h += '    <p>\n'
      + `      <input id="${l.uuid.substr(0, 5)}" type="checkbox" name="l" checked value="${l.uuid}" />\n`
      + `      <label for="${l.uuid.substr(0, 5)}">${l.name}</label>\n`;
    if (l.description) {
      h += `      <br />\n      <span>${l.description}</span>\n`;
    }
    h += '    </p>\n';
  });

  if (sc.public_subscription.captcha_enabled) {
    if (sc.public_subscription.captcha_provider === 'altcha') {
      h += '\n'
        + `    <altcha-widget challengeurl="${sc.root_url}/api/public/captcha/altcha"></altcha-widget>\n`
        + `    <${'script'} type="module" src="${sc.root_url}/public/static/altcha.umd.js" async defer></${'script'}>\n`;
    } else if (sc.public_subscription.captcha_provider === 'hcaptcha') {
      h += '\n'
        + `    <div class="h-captcha" data-sitekey="${sc.public_subscription.captcha_key}"></div>\n`
        + `    <${'script'} src="https://js.hcaptcha.com/1/api.js" async defer></${'script'}>\n`;
    }
  }

  h += '\n'
    + `    <input type="submit" value="${t('public.sub')} " />\n`
    + '  </div>\n'
    + '</form>';

  html.value = h;
}

watch(() => checked.value, () => { renderHTML(); });
watch(() => selectedRedirectURL.value, () => { renderHTML(); });
</script>

<style scoped lang="scss">
.forms-page { display: flex; flex-direction: column; gap: 1.5rem; }
.forms-loading { display: flex; justify-content: center; padding: 3rem; }

.forms-grid {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 1.5rem;
  align-items: start;

  @media (max-width: 768px) { grid-template-columns: 1fr; }
}

.forms-panel {
  background: var(--lm-surface);
  border: 1px solid var(--lm-border);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.forms-panel-title { font-size: 1rem; font-weight: 600; color: var(--lm-text); margin: 0; }
.forms-panel-subtitle { font-size: 0.85rem; color: var(--lm-text-muted); margin: 0; }
.forms-section-label { font-size: 0.8rem; font-weight: 600; color: var(--lm-text); margin: 0; }

.checklist { display: flex; flex-direction: column; gap: 0.6rem; }
.checklist-item { display: flex; align-items: center; gap: 0.5rem; }
.checklist-label { font-size: 0.9rem; color: var(--lm-text); cursor: pointer; &--url { font-size: 0.8rem; color: var(--lm-text-muted); } }

.forms-ext-link {
  display: inline-flex; align-items: center; gap: 0.35rem;
  font-size: 0.85rem; color: var(--lm-primary); text-decoration: none;
  &:hover { text-decoration: underline; }
}

.forms-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 0.75rem; padding: 3rem 1rem; color: var(--lm-text-subtle); font-size: 0.875rem;
}
.forms-empty-icon { font-size: 2.5rem; opacity: 0.4; }
</style>
