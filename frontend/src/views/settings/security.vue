<template>
  <div class="items">
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['security.oidc']['enabled']" name="security.oidc" />
        <span>{{ $t('settings.security.enableOIDC') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.security.OIDCHelp') }}</small>
    </div>

    <div class="flex flex-col gap-4" :class="{ disabled: !data['security.oidc']['enabled'] }">
      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCURL') }}</label>
        <PvInputText v-model="data['security.oidc']['provider_url']" name="oidc.provider_url"
          placeholder="https://login.yoursite.com" :disabled="!data['security.oidc']['enabled']" :maxlength="300"
          required type="url" pattern="https?://.*" class="w-full" />
        <div class="quick-links mt-2">
          <a href="#" @click.prevent="() => setProvider('google')">Google</a>
          <a href="#" @click.prevent="() => setProvider('microsoft')">Microsoft</a>
          <a href="#" @click.prevent="() => setProvider('apple')">Apple</a>
        </div>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCName') }}</label>
        <PvInputText v-model="data['security.oidc']['provider_name']" name="oidc.provider_name"
          :disabled="!data['security.oidc']['enabled']" :maxlength="200" class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCClientID') }}</label>
        <PvInputText v-model="data['security.oidc']['client_id']" name="oidc.client_id" ref="clientIdEl"
          :disabled="!data['security.oidc']['enabled']" :maxlength="200" required class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCClientSecret') }}</label>
        <PvPassword v-model="data['security.oidc']['client_secret']" name="oidc.client_secret"
          :disabled="!data['security.oidc']['enabled']" :maxlength="200" required :feedback="false" class="w-full" />
      </div>

      <hr class="m-0" />

      <div class="field">
        <div class="flex items-center gap-2">
          <PvToggleSwitch v-model="data['security.oidc']['auto_create_users']" :disabled="!data['security.oidc']['enabled']"
            name="oidc.auto_create_users" />
          <span>{{ $t('settings.security.OIDCAutoCreateUsers') }}</span>
        </div>
        <small class="block mt-1 text-color-secondary">{{ $t('settings.security.OIDCAutoCreateUsersHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCDefaultUserRole') }}</label>
        <PvSelect v-model="data['security.oidc']['default_user_role_id']"
          :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
          name="oidc.default_user_role_id"
          :options="userRoles" option-label="name" option-value="id" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.security.OIDCDefaultRoleHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCDefaultListRole') }}</label>
        <PvSelect v-model="data['security.oidc']['default_list_role_id']"
          :disabled="!data['security.oidc']['enabled'] || !data['security.oidc']['auto_create_users']"
          name="oidc.default_list_role_id"
          :options="listRoleOptions" option-label="name" option-value="id" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.security.OIDCDefaultRoleHelp') }}</small>
      </div>

      <hr class="m-0" />

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.security.OIDCRedirectURL') }}</label>
        <code><copy-text :text="`${serverConfig.root_url}/auth/oidc`" /></code>
      </div>
      <p v-if="data['security.oidc']['enabled'] && !isURLOk" class="text-red-500 text-sm mt-1">
        <i class="pi pi-exclamation-triangle" />
        {{ $t('settings.security.OIDCRedirectWarning') }}
      </p>
    </div>

    <hr class="m-0" />

    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="captchaEnabled" input-id="captcha-forms" name="security.captcha" />
        <label for="captcha-forms">{{ $t('settings.security.captchaForms') }}</label>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.security.captchaFormsHelp') }}</small>
      <small v-if="legacyCaptcha" class="block mt-1 text-color-secondary">{{ $t('settings.security.captchaLegacy', { name: legacyCaptcha }) }}</small>
    </div>

    <div v-if="captchaEnabled" class="flex flex-col gap-4">
      <div class="field">
        <label for="turnstile-key" class="block mb-1 text-sm font-medium">{{ $t('settings.security.turnstileSiteKey') }}</label>
        <PvInputText id="turnstile-key" v-model="data['security.captcha']['turnstile']['key']" name="turnstile_key" :maxlength="200"
          required class="w-full ln-mono" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.security.turnstileKeyHelp') }}</small>
      </div>
      <div class="field">
        <label for="turnstile-secret" class="block mb-1 text-sm font-medium">{{ $t('settings.security.captchaSecret') }}</label>
        <PvPassword v-model="data['security.captcha']['turnstile']['secret']" input-id="turnstile-secret" name="turnstile_secret"
          :maxlength="200" required :feedback="false" class="w-full" />
      </div>
      <div class="field">
        <div class="flex items-center gap-2">
          <PvToggleSwitch v-model="data['security.captcha'].login" input-id="captcha-login" name="captcha_login" />
          <label for="captcha-login">{{ $t('settings.security.captchaLogin') }}</label>
        </div>
        <small class="block mt-1 text-color-secondary">{{ $t('settings.security.captchaLoginHelp') }}</small>
      </div>
    </div>

    <hr class="m-0" />

    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['security.require_2fa']" input-id="require-2fa" name="security.require_2fa" />
        <label for="require-2fa">{{ $t('settings.security.require2fa') }}</label>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.security.require2faHelp') }}</small>
    </div>

    <hr class="m-0" />

    <p class="settings-section-label">{{ $t('settings.security.trustedURLs') }} / CORS</p>
    <div class="field">
      <PvTextarea v-model="trustedURLs" name="trusted_urls" rows="5"
        placeholder="https://example.com" class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.security.trustedURLsHelp') }}</small>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref, computed, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import CopyText from '../../components/CopyText.vue';
import { getRoles } from '../../api/generated/endpoints/roles/roles';

const OIDC_PROVIDERS: Record<string, string> = {
  google: 'https://accounts.google.com',
  github: 'https://token.actions.githubusercontent.com',
  microsoft: 'https://login.microsoftonline.com/{TENANT_HERE}/v2.0',
  apple: 'https://appleid.apple.com',
};

const props = defineProps<{ form?: any }>();
const { $can } = useGlobal();
const { listUserRoles, listListRoles } = getRoles();
const { t } = useI18n();
const store = useMainStore();
const { serverConfig: serverConfigRef, userRoles, listRoles } = storeToRefs(store);
const serverConfig = computed(() => serverConfigRef.value as any);
const clientIdEl = ref<any>(null);
const data = props.form;

const listRoleOptions = computed(() => [
  { id: null, name: `— ${t('globals.terms.none')} —` },
  ...((listRoles.value as any[]) || []),
]);

const trustedURLs = computed({
  get() {
    const domains = data['security.trusted_urls'];
    return domains && Array.isArray(domains) ? domains.join('\n') : '';
  },
  set(value: string) {
    data['security.trusted_urls'] = value.split('\n');
  },
});

// Settings saved before Turnstile existed lack its block until the v6.27
// migration runs; fill it in so the form can bind to it.
if (!data['security.captcha'].turnstile) {
  data['security.captcha'].turnstile = { enabled: false, key: '', secret: '' };
}
if (data['security.captcha'].login === undefined) data['security.captcha'].login = false;

const LEGACY_CAPTCHAS = { altcha: 'ALTCHA', hcaptcha: 'hCaptcha' } as const;

// Captcha is Cloudflare Turnstile only (handover). An ALTCHA or hCaptcha
// setup from before keeps working until Turnstile is switched on, which
// replaces it.
const legacyCaptcha = computed(() => {
  const c = data['security.captcha'];
  if (c.turnstile.enabled) return '';
  const on = (Object.keys(LEGACY_CAPTCHAS) as (keyof typeof LEGACY_CAPTCHAS)[]).find((p) => c[p]?.enabled);
  return on ? LEGACY_CAPTCHAS[on] : '';
});

const captchaEnabled = computed({
  get() {
    return !!data['security.captcha'].turnstile.enabled;
  },
  set(value: boolean) {
    const c = data['security.captcha'];
    c.turnstile.enabled = !!value;
    if (value) {
      c.altcha.enabled = false;
      c.hcaptcha.enabled = false;
    } else {
      c.login = false;
    }
  },
});

const isURLOk = computed(() => {
  try {
    const u = new URL(serverConfig.value.root_url);
    return u.hostname !== 'localhost' && u.hostname !== '127.0.0.1';
  } catch {
    return false;
  }
});

onMounted(() => {
  if ($can('roles:get')) {
    listUserRoles().then((res: any) => { store.setModelResponse({ model: 'userRoles', data: res }); });
    listListRoles().then((res: any) => { store.setModelResponse({ model: 'listRoles', data: res }); });
  }
});

function setProvider(provider: string) {
  data['security.oidc'].provider_url = OIDC_PROVIDERS[provider];
  data['security.oidc'].provider_name = provider.charAt(0).toUpperCase() + provider.slice(1);
  nextTick(() => { clientIdEl.value?.$el?.focus(); });
}
</script>

<style scoped>
:deep(.p-password) { width: 100%; }
:deep(.p-password-input) { width: 100%; }
</style>
