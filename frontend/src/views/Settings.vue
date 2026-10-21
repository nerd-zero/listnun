<template>
  <div class="ln-split settings-page">
    <aside class="settings-side">
      <h1 class="ln-page-title settings-side-title">{{ $t('settings.title') }}</h1>
      <sub-nav :items="navItems" :active="section" :label="$t('settings.title')" />
      <template v-if="$can('settings:maintain')">
        <hr class="settings-side-sep">
        <sub-nav :items="maintenanceItems" :active="section" :label="$t('maintenance.title')" />
      </template>
      <p class="ln-small settings-version">{{ version }}</p>
    </aside>

    <main class="settings-main">
      <header class="settings-head">
        <h2 class="ln-page-title">{{ current.label }}</h2>
        <p class="settings-head-text">{{ current.help }}</p>
      </header>

      <maintenance v-if="section === 'maintenance'" />

      <template v-else>
        <table-skeleton v-if="!form" :rows="6" :cols="2" />
        <form v-else class="ln-settings" @submit.prevent="onSubmit">
          <general-settings v-if="section === 'general'" :key="key" :form="form" />
          <performance-settings v-else-if="section === 'performance'" :key="key" :form="form" />
          <privacy-settings v-else-if="section === 'privacy'" :key="key" :form="form" />
          <security-settings v-else-if="section === 'security'" :key="key" :form="form" />
          <media-settings v-else-if="section === 'media'" :key="key" :form="form" />
          <smtp-settings v-else-if="section === 'smtp'" :key="key" :form="form" />
          <bounce-settings v-else-if="section === 'bounces'" :key="key" :form="form" />
          <messenger-settings v-else-if="section === 'messengers'" :key="key" :form="form" />
          <appearance-settings v-else-if="section === 'appearance'" :key="key" :form="form" />
          <scrub-settings v-else-if="section === 'scrub'" :key="key" :form="form" />
          <button type="submit" hidden aria-hidden="true" tabindex="-1" />
        </form>
      </template>
    </main>

    <!-- Unsaved changes -->
    <div v-if="hasFormChanged && $can('settings:manage')" class="settings-unsaved" role="status">
      <span>{{ $t('settings.unsavedIn', { sections: dirtyLabels }) }}</span>
      <PvButton :label="$t('settings.discard')" size="small" severity="secondary" outlined class="settings-unsaved-discard" @click="discard" />
      <PvButton :label="$t('globals.buttons.save')" size="small" :loading="isLoading" class="settings-unsaved-save" data-cy="btn-save" @click="onSubmit" />
    </div>

    <confirm-action
      v-model:visible="leave.open"
      :title="$t('campaigns.leaveTitle')"
      :text="$t('settings.leaveText', { sections: dirtyLabels })"
      :confirm-label="$t('developers.templates.leave')"
      @confirm="leave.go"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, nextTick, onMounted, reactive, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import {
  getSettings, updateSettings, getServerConfig, getHealth,
} from '../api';
import SubNav, { type SubNavItem } from '../components/ui/SubNav.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
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
import Maintenance from './Maintenance.vue';

// The settings object is the server's flat, dotted-key map.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface Section {
  key: string;
  label: string;
  help: string;
  // The settings keys a section edits, by exact key or "prefix." match,
  // for its unsaved-changes dot.
  keys: string[];
}

const PERFORMANCE_KEYS = [
  'app.concurrency', 'app.message_rate', 'app.batch_size', 'app.max_send_errors', 'app.message_sliding_window',
  'app.message_sliding_window_duration', 'app.message_sliding_window_rate', 'app.cache_slow_queries', 'app.cache_slow_queries_interval',
];

const { $utils, $can } = useGlobal();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { serverConfig } = storeToRefs(useMainStore());

// :key re-renders the sections every time settings are fetched.
const key = ref(0);
const isLoading = ref(false);
// formCopy is the settings as last fetched, to detect changes.
const formCopy = ref('');
const form = ref<Loose | null>(null);
const leave = reactive({ open: false, go: () => {} });
const version = computed(() => (serverConfig.value as Loose).version || '');

const sections = computed<Section[]>(() => [
  {
    key: 'general', label: t('settings.general.name'), help: t('settings.help.general'), keys: ['app.'],
  },
  {
    key: 'performance', label: t('settings.performance.name'), help: t('settings.help.performance'), keys: PERFORMANCE_KEYS,
  },
  {
    key: 'privacy', label: t('settings.privacy.name'), help: t('settings.help.privacy'), keys: ['privacy.'],
  },
  {
    key: 'security', label: t('settings.security.name'), help: t('settings.help.security'), keys: ['security.'],
  },
  {
    key: 'media', label: t('settings.media.title'), help: t('settings.help.media'), keys: ['upload.'],
  },
  {
    key: 'smtp', label: t('settings.smtp.name'), help: t('settings.help.smtp'), keys: ['smtp'],
  },
  {
    key: 'bounces', label: t('settings.bounces.name'), help: t('settings.help.bounces'), keys: ['bounce.'],
  },
  {
    key: 'messengers', label: t('settings.messengers.name'), help: t('settings.help.messengers'), keys: ['messengers'],
  },
  {
    key: 'appearance', label: t('settings.appearance.name'), help: t('settings.help.appearance'), keys: ['appearance.'],
  },
  {
    key: 'scrub', label: t('settings.scrub.name'), help: t('settings.help.scrub'), keys: ['scrub'],
  },
]);

const maintenanceSection = computed<Section>(() => ({
  key: 'maintenance', label: t('maintenance.title'), help: t('maintenance.help'), keys: [],
}));

const section = computed(() => {
  if (route.name === 'maintenance') return 'maintenance';
  const s = route.params.section as string | undefined;
  return sections.value.some((x) => x.key === s) ? (s as string) : 'general';
});

const current = computed(() => (section.value === 'maintenance'
  ? maintenanceSection.value
  : sections.value.find((s) => s.key === section.value) || sections.value[0]));

function belongs(sec: Section, k: string): boolean {
  if (sec.key === 'general' && PERFORMANCE_KEYS.includes(k)) return false;
  return sec.keys.some((p) => (p.endsWith('.') ? k.startsWith(p) : k === p));
}

// dirtySections are the sections with a key whose value changed.
const dirtySections = computed<Section[]>(() => {
  if (!form.value || !formCopy.value) return [];
  const before = JSON.parse(formCopy.value) as Loose;
  const changed = Object.keys(form.value).filter((k) => JSON.stringify(form.value![k]) !== JSON.stringify(before[k]));
  return sections.value.filter((s) => changed.some((k) => belongs(s, k)));
});
const hasFormChanged = computed(() => dirtySections.value.length > 0);
const dirtyLabels = computed(() => dirtySections.value.map((s) => s.label).join(', '));

const navItems = computed<SubNavItem[]>(() => (!$can('settings:get') ? [] : sections.value.map((s) => ({
  key: s.key,
  label: s.label,
  to: { name: 'settings', params: { section: s.key } },
  badge: dirtySections.value.some((d) => d.key === s.key) ? '●' : undefined,
}))));
const maintenanceItems = computed<SubNavItem[]>(() => [
  { key: 'maintenance', label: t('maintenance.title'), to: { name: 'maintenance' } },
]);

function isDummy(pwd: string) {
  return !pwd || (pwd.match(/•/g) || []).length === pwd.length;
}

function hasDummy(pwd: string) {
  return pwd.includes('•');
}

function fetchSettings() {
  isLoading.value = true;
  getSettings().then((data) => {
    let d: Loose = {};
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

function discard() {
  if (!formCopy.value) return;
  form.value = JSON.parse(formCopy.value);
  key.value += 1;
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
  if (!form.value) return;
  const f = JSON.parse(JSON.stringify(form.value));
  let hasDummyField = '';

  for (let i = 0; i < f.smtp.length; i += 1) {
    f.smtp[i].host = f.smtp[i].host?.trim();
    if (isDummy(f.smtp[i].password)) { f.smtp[i].password = ''; } else if (hasDummy(f.smtp[i].password)) { hasDummyField = `smtp #${i + 1}`; }
    if (f.smtp[i].strEmailHeaders && f.smtp[i].strEmailHeaders !== '[]') {
      try {
        f.smtp[i].email_headers = JSON.parse(f.smtp[i].strEmailHeaders);
      } catch (e) {
        $utils.toast(`smtp #${i + 1}: ${String(e)}`, 'is-danger');
        return;
      }
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
  checks.forEach(([key2, label]) => {
    if (isDummy(f[key2])) { f[key2] = ''; } else if (hasDummy(f[key2])) { hasDummyField = label; }
  });

  const objChecks: [Loose, string, string][] = [
    [f['bounce.azure'], 'shared_secret', 'azure shared secret'],
    [f['security.captcha'].hcaptcha, 'secret', 'captcha'],
    [f['security.captcha'].turnstile, 'secret', 'captcha'],
    [f['security.oidc'], 'client_secret', 'oidc'],
    [f['bounce.postmark'], 'password', 'postmark'],
    [f['bounce.forwardemail'], 'key', 'forwardemail'],
    [f['bounce.lettermint'], 'key', 'lettermint'],
    [f.scrub, 'api_key', 'scrub'],
  ];
  objChecks.forEach((check) => {
    const [obj, field, label] = check;
    if (!obj) return;
    if (isDummy(obj[field])) { obj[field] = ''; } else if (hasDummy(obj[field])) { hasDummyField = label; }
  });

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
    const result = await updateSettings(f) as Loose;

    // Campaigns are running: the app wasn't auto-restarted, settings are
    // saved but need a manual restart (the Needs restart pill in the shell).
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

// Moving between sections keeps unsaved edits; leaving Settings asks.
onBeforeRouteLeave((to, _from, next) => {
  if (!hasFormChanged.value || to.name === 'settings' || to.name === 'maintenance') {
    next(true);
    return;
  }
  let settled = false;
  const stop = watch(() => leave.open, (open) => {
    if (!open && !settled) {
      settled = true;
      stop();
      next(false);
    }
  });
  leave.go = () => {
    settled = true;
    stop();
    leave.open = false;
    next(true);
  };
  leave.open = true;
});

watch(section, (s) => { $utils.setPref('settings.section', s); });

onMounted(() => {
  // /settings with no section opens the one last used.
  if (route.name === 'settings' && !route.params.section) {
    const last = $utils.getPref('settings.section') as string;
    if (last && last !== 'general' && sections.value.some((s) => s.key === last)) {
      router.replace({ name: 'settings', params: { section: last } });
    }
  }
  if ($can('settings:get')) fetchSettings();
});
</script>

<style scoped lang="scss">
.settings-page {
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 40px;
  padding-bottom: 80px;
}
.settings-side {
  position: sticky;
  top: 84px;

  @media (max-width: 991px) { position: static; }
}
.settings-side-title { margin-bottom: 16px; }
.settings-side-sep { border: 0; border-top: 1px solid var(--ln-border); margin: 12px 10px; }
.settings-version { padding: 16px 10px 0; color: var(--ln-text-4); }
.settings-main { max-width: 820px; min-width: 0; }
.settings-head { margin-bottom: 20px; }
.settings-head-text { margin: 4px 0 0; color: var(--ln-text-2); }

.settings-unsaved {
  position: fixed;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  z-index: 30;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 8px 8px 16px;
  border-radius: 10px;
  background: var(--ln-ink);
  color: var(--ln-on-ink);
  font-size: 14px;
  box-shadow: var(--ln-shadow-pop);

  .settings-unsaved-discard {
    background: transparent !important;
    color: var(--ln-on-ink) !important;
    border-color: rgba(127, 127, 127, .5) !important;
  }
  .settings-unsaved-save {
    background: var(--ln-on-ink) !important;
    color: var(--ln-ink) !important;
    border-color: var(--ln-on-ink) !important;
  }
}
:deep(.ln-subnav-badge) { color: var(--ln-warn); font-size: 9px; }
</style>
