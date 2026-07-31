<template>
  <form @submit.prevent="onSubmit">
    <section class="settings">
      <div v-if="loading.settings && !form" class="flex justify-center p-8">
        <PvProgressSpinner />
      </div>
      <div class="page-header" style="margin-bottom:1.5rem">
        <h1 class="page-title">
          {{ $t('settings.title') }}
          <span style="font-size:0.85rem;font-weight:400;color:#94a3b8">({{ serverConfig.version }})</span>
        </h1>
        <PvButton v-if="$can('settings:manage')" :disabled="!hasFormChanged || isLoading" :loading="isLoading"
          severity="primary" icon="pi pi-save" type="submit" class="isSaveEnabled" data-cy="btn-save"
          :label="$t('globals.buttons.save')" />
      </div>

      <section class="wrap settings-wrap" v-if="form">
        <PvTabs class="lm-tabs" v-model:value="tab">
          <PvTabList>
            <PvTab value="0">{{ $t('settings.general.name') }}</PvTab><!-- general -->
            <PvTab value="1">{{ $t('settings.performance.name') }}</PvTab><!-- performance -->
            <PvTab value="2">{{ $t('settings.privacy.name') }}</PvTab><!-- privacy -->
            <PvTab value="3">{{ $t('settings.security.name') }}</PvTab><!-- security -->
            <PvTab value="4">{{ $t('settings.media.title') }}</PvTab><!-- media -->
            <PvTab value="5">{{ $t('settings.smtp.name') }}</PvTab><!-- mail servers -->
            <PvTab value="6">{{ $t('settings.bounces.name') }}</PvTab><!-- bounces -->
            <PvTab value="7">{{ $t('settings.messengers.name') }}</PvTab><!-- messengers -->
            <PvTab value="8">{{ $t('settings.appearance.name') }}</PvTab><!-- appearance -->
            <PvTab value="9">{{ $t('settings.scrub.name') }}</PvTab><!-- mail validation -->
          </PvTabList>
          <PvTabPanels>
            <PvTabPanel value="0">
              <general-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="1">
              <performance-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="2">
              <privacy-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="3">
              <security-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="4">
              <media-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="5">
              <smtp-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="6">
              <bounce-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="7">
              <messenger-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="8">
              <appearance-settings :form="form" :key="key" />
            </PvTabPanel>
            <PvTabPanel value="9">
              <scrub-settings :form="form" :key="key" />
            </PvTabPanel>
          </PvTabPanels>
        </PvTabs>
      </section>
    </section>
  </form>
</template>

<script setup lang="ts">
import {
  ref, computed, watch, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import {
  getSettings, updateSettings, getServerConfig, getHealth,
} from '../api';
import AppearanceSettings from './settings/appearance.vue';
import ScrubSettings from './settings/scrub.vue';
import BounceSettings from './settings/bounces.vue';
import GeneralSettings from './settings/general.vue';
import MediaSettings from './settings/media.vue';
import MessengerSettings from './settings/messengers.vue';
import PerformanceSettings from './settings/performance.vue';
import PrivacySettings from './settings/privacy.vue';
import SecuritySettings from './settings/security.vue';
import SmtpSettings from './settings/smtp.vue';

const { $utils } = useGlobal();
const { t } = useI18n();
const { serverConfig, loading } = storeToRefs(useMainStore());

// :key is used to re-render child components every time settings is pulled.
const key = ref(0);
const isLoading = ref(false);
// formCopy is a stringified copy of the original settings to detect changes.
const formCopy = ref('');
const form = ref<any>(null);
const tab = ref('0');

const hasFormChanged = computed(() => {
  if (!formCopy.value) return false;
  return JSON.stringify(form.value) !== formCopy.value;
});

function isDummy(pwd: string) {
  return !pwd || (pwd.match(/•/g) || []).length === pwd.length;
}

function hasDummy(pwd: string) {
  return pwd.includes('•');
}

function fetchSettings() {
  isLoading.value = true;
  getSettings().then((data: any) => {
    let d: any = {};
    try {
      d = JSON.parse(JSON.stringify(data));
    } catch {
      return;
    }
    for (let i = 0; i < d.smtp.length; i += 1) {
      d.smtp[i].strEmailHeaders = JSON.stringify(d.smtp[i].email_headers, null, 4);
    }
    d['privacy.domain_blocklist'] = d['privacy.domain_blocklist'].join('\n');
    d['privacy.domain_allowlist'] = d['privacy.domain_allowlist'].join('\n');
    key.value += 1;
    form.value = d;
    formCopy.value = JSON.stringify(d);
    nextTick(() => { isLoading.value = false; });
  });
}

// Saving settings (when no campaigns are running) makes the backend restart
// itself ~500ms later to apply the change (cmd/settings.go's
// handleSettingsRestart -> syscall.Exec in cmd/init.go). Fetching data right
// after the save races that restart window: the request can land while the
// listener is closed or the process is mid-exec, which a proxy in front of
// the app (e.g. Cloudflare) surfaces as a gateway error. Wait for the health
// endpoint to come back before making any further requests.
function waitForServer(timeoutMs = 30000) {
  const start = Date.now();
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      const poll = setInterval(() => {
        getHealth().then(() => {
          clearInterval(poll);
          resolve(true);
        }).catch(() => {
          if (Date.now() - start > timeoutMs) {
            clearInterval(poll);
            reject(new Error('timed out waiting for the server to restart'));
          }
        });
      }, 500);
    }, 1500);
  });
}

async function onSubmit() {
  const f = JSON.parse(JSON.stringify(form.value));
  let hasDummyField = '';

  for (let i = 0; i < f.smtp.length; i += 1) {
    f.smtp[i].host = f.smtp[i].host?.trim();
    if (isDummy(f.smtp[i].password)) { f.smtp[i].password = ''; } else if (hasDummy(f.smtp[i].password)) { hasDummyField = `smtp #${i + 1}`; }
    if (f.smtp[i].strEmailHeaders && f.smtp[i].strEmailHeaders !== '[]') {
      f.smtp[i].email_headers = JSON.parse(f.smtp[i].strEmailHeaders);
    } else { f.smtp[i].email_headers = []; }
  }

  for (let i = 0; i < f['bounce.mailboxes'].length; i += 1) {
    f['bounce.mailboxes'][i].host = f['bounce.mailboxes'][i].host?.trim();
    if (isDummy(f['bounce.mailboxes'][i].password)) { f['bounce.mailboxes'][i].password = ''; } else if (hasDummy(f['bounce.mailboxes'][i].password)) { hasDummyField = `bounce #${i + 1}`; }
  }

  const checks: [string, string][] = [
    ['upload.s3.aws_secret_access_key', 's3'],
    ['bounce.sendgrid_key', 'sendgrid'],
  ];
  for (const [key2, label] of checks) {
    if (isDummy(f[key2])) { f[key2] = ''; } else if (hasDummy(f[key2])) { hasDummyField = label; }
  }

  const objChecks: [any, string, string][] = [
    [f['bounce.azure'], 'shared_secret', 'azure shared secret'],
    [f['security.captcha'].hcaptcha, 'secret', 'captcha'],
    [f['security.oidc'], 'client_secret', 'oidc'],
    [f['bounce.postmark'], 'password', 'postmark'],
    [f['bounce.forwardemail'], 'key', 'forwardemail'],
    [f['bounce.lettermint'], 'key', 'lettermint'],
    [f.scrub, 'api_key', 'scrub'],
  ];
  for (const [obj, field, label] of objChecks) {
    if (isDummy(obj[field])) { obj[field] = ''; } else if (hasDummy(obj[field])) { hasDummyField = label; }
  }

  for (let i = 0; i < f.messengers.length; i += 1) {
    if (isDummy(f.messengers[i].password)) { f.messengers[i].password = ''; } else if (hasDummy(f.messengers[i].password)) { hasDummyField = `messenger #${i + 1}`; }
  }

  if (hasDummyField) {
    $utils.toast(t('globals.messages.passwordChangeFull', { name: hasDummyField }), 'is-danger');
    return;
  }

  f['privacy.domain_blocklist'] = f['privacy.domain_blocklist'].split('\n').map((v: string) => v.trim().toLowerCase()).filter((v: string) => v !== '');
  f['privacy.domain_allowlist'] = f['privacy.domain_allowlist'].split('\n').map((v: string) => v.trim().toLowerCase()).filter((v: string) => v !== '');

  isLoading.value = true;
  try {
    const result: any = await updateSettings(f);

    // Campaigns are running: the app wasn't auto-restarted, settings are
    // saved but need a manual restart (banner + flow in App.vue).
    if (result && typeof result === 'object' && result.needsRestart) {
      await getServerConfig();
      fetchSettings();
      return;
    }

    $utils.toast(t('settings.savedApplying'));
    try {
      await waitForServer();
      await getServerConfig();
      fetchSettings();
    } catch {
      $utils.toast(t('settings.applyTimeout'), 'is-danger');
    }
  } finally {
    isLoading.value = false;
  }
}

watch(tab, (t2) => { $utils.setPref('settings.tab', t2); });

onBeforeRouteLeave((_to, _from, next) => {
  if (hasFormChanged.value) {
    $utils.confirm(t('globals.messages.confirmDiscard'), () => next(true));
    return;
  }
  next(true);
});

onMounted(() => {
  tab.value = String($utils.getPref('settings.tab') || '0');
  fetchSettings();
});
</script>
