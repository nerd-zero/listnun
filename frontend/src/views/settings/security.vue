<template>
  <div class="set-cards">
    <set-card :title="$t('settings.sec.sso')" :help="$t('settings.sec.ssoHelp')">
      <set-row :label="$t('settings.sec.ssoOn')" :help="$t('settings.sec.ssoOnHelp')" toggle>
        <PvToggleSwitch v-model="oidc.enabled" name="security.oidc" :aria-label="$t('settings.sec.ssoOn')" />
      </set-row>
      <template v-if="oidc.enabled">
        <set-row :label="$t('settings.security.OIDCURL')" label-for="sec-oidc-url">
          <template #help>
            <span class="sec-presets">
              {{ $t('settings.smtpx.fillFor') }}
              <a href="#" @click.prevent="setProvider('google')">Google</a>
              <a href="#" @click.prevent="setProvider('microsoft')">Microsoft</a>
              <a href="#" @click.prevent="setProvider('apple')">Apple</a>
            </span>
          </template>
          <PvInputText
            id="sec-oidc-url"
            v-model="oidc.provider_url"
            name="oidc.provider_url"
            placeholder="https://login.yoursite.com"
            :maxlength="300"
            required
            type="url"
            pattern="https?://.*"
            class="ln-mono"
          />
        </set-row>
        <set-row :label="$t('settings.sec.buttonName')" :help="$t('settings.sec.buttonNameHelp')" label-for="sec-oidc-name">
          <PvInputText id="sec-oidc-name" v-model="oidc.provider_name" name="oidc.provider_name" :maxlength="200" />
        </set-row>
        <set-row :label="$t('settings.security.OIDCClientID')" label-for="sec-oidc-client">
          <PvInputText id="sec-oidc-client" ref="clientIdEl" v-model="oidc.client_id" name="oidc.client_id" :maxlength="200" required class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.security.OIDCClientSecret')">
          <PvPassword v-model="oidc.client_secret" name="oidc.client_secret" :maxlength="200" required :feedback="false" :aria-label="$t('settings.security.OIDCClientSecret')" />
        </set-row>
        <set-row :label="$t('settings.security.OIDCRedirectURL')" :help="$t('settings.sec.redirectHelp')">
          <code class="sec-redirect"><copy-text :text="`${serverConfig.root_url}/auth/oidc`" /></code>
          <template v-if="!isURLOk" #note>{{ $t('settings.sec.redirectWarn') }}</template>
        </set-row>
      </template>
    </set-card>

    <set-card :title="$t('settings.sec.twofa')">
      <set-row :label="$t('settings.sec.twofaAll')" :help="$t('settings.sec.twofaAllHelp')" toggle>
        <PvToggleSwitch v-model="data['security.require_2fa']" input-id="require-2fa" name="security.require_2fa" :aria-label="$t('settings.sec.twofaAll')" />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.sec.captcha')" :help="$t('settings.sec.captchaHelp')">
      <set-row :label="$t('settings.sec.captchaOn')" :help="legacyCaptcha ? $t('settings.security.captchaLegacy', { name: legacyCaptcha }) : $t('settings.sec.captchaOnHelp')" toggle>
        <PvToggleSwitch v-model="captchaEnabled" input-id="captcha-forms" name="security.captcha" :aria-label="$t('settings.sec.captchaOn')" />
      </set-row>
      <template v-if="captchaEnabled">
        <set-row :label="$t('settings.sec.siteKey')" :help="$t('settings.sec.siteKeyHelp')" label-for="turnstile-key">
          <PvInputText id="turnstile-key" v-model="data['security.captcha']['turnstile']['key']" name="turnstile_key" :maxlength="200" required class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.sec.secretKey')">
          <PvPassword
            v-model="data['security.captcha']['turnstile']['secret']"
            input-id="turnstile-secret"
            name="turnstile_secret"
            :maxlength="200"
            required
            :feedback="false"
            :aria-label="$t('settings.sec.secretKey')"
          />
        </set-row>
        <set-row :label="$t('settings.security.captchaLogin')" :help="$t('settings.sec.loginHelp')" toggle>
          <PvToggleSwitch v-model="data['security.captcha'].login" input-id="captcha-login" name="captcha_login" :aria-label="$t('settings.security.captchaLogin')" />
        </set-row>
      </template>
    </set-card>

    <set-card :title="$t('settings.ws.advanced')" :help="$t('settings.sec.advancedHelp')" collapsible :summary="advancedSummary">
      <template v-if="oidc.enabled">
        <set-row :label="$t('settings.security.OIDCAutoCreateUsers')" :help="$t('settings.sec.autoCreateHelp')" toggle>
          <PvToggleSwitch v-model="oidc.auto_create_users" name="oidc.auto_create_users" :aria-label="$t('settings.security.OIDCAutoCreateUsers')" />
        </set-row>
        <template v-if="oidc.auto_create_users">
          <set-row :label="$t('settings.security.OIDCDefaultUserRole')">
            <PvSelect
              v-model="oidc.default_user_role_id"
              name="oidc.default_user_role_id"
              :options="userRoles"
              option-label="name"
              option-value="id"
              :aria-label="$t('settings.security.OIDCDefaultUserRole')"
            />
          </set-row>
          <set-row :label="$t('settings.security.OIDCDefaultListRole')">
            <PvSelect
              v-model="oidc.default_list_role_id"
              name="oidc.default_list_role_id"
              :options="listRoleOptions"
              option-label="name"
              option-value="id"
              :aria-label="$t('settings.security.OIDCDefaultListRole')"
            />
          </set-row>
        </template>
      </template>
      <set-row :label="$t('settings.sec.trusted')" :help="$t('settings.sec.trustedHelp')" label-for="sec-trusted" wide>
        <PvTextarea id="sec-trusted" v-model="trustedURLs" name="trusted_urls" rows="3" placeholder="https://example.com" class="ln-mono" />
      </set-row>
    </set-card>
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
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';
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
const oidc = computed(() => data['security.oidc']);

// What the closed Advanced card says.
const advancedSummary = computed(() => {
  const n = (data['security.trusted_urls'] || []).filter((u: string) => u && u.trim()).length;
  return t('settings.sec.advancedSummary', { n }, n);
});

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

<style scoped lang="scss">
.sec-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;

  a { color: var(--ln-text-2); text-decoration: underline; }
}
.sec-redirect {
  font-size: 12px;
  overflow-wrap: anywhere;

  :deep(.pi) { margin-left: 6px; }
}
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
