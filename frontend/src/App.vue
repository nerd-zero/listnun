<template>
  <PvToast position="bottom-right" />
  <PvConfirmDialog />

  <app-shell
    v-if="isLoaded"
    :org-name="serverConfig.organization_name"
    :console-url="serverConfig.console_url"
    :needs-restart="serverConfig.needs_restart"
    :dark="isDark"
    :username="profile.username"
    :name="profile.name"
    :account-items="accountItems"
    @refresh="triggerRefresh"
    @toggle-dark="toggleDark"
    @restart="$utils.confirm($t('settings.confirmRestart'), reloadApp)"
  >
    <template #notices>
      <template v-if="serverConfig.update">
        <div
          v-if="serverConfig.update.update && serverConfig.update.update.is_new"
          class="app-notice app-notice--success"
        >
          <i class="pi pi-arrow-up-right" />
          <span>
            {{ $t('settings.updateAvailable', { version: serverConfig.update.update.release_version }) }}
          </span>
          <a :href="serverConfig.update.update.url" target="_blank" rel="noopener noreferrer">View release</a>
        </div>
        <div
          v-for="m in (serverConfig.update.messages || [])"
          :key="m.title"
          class="app-notice"
          :class="m.priority === 'high' ? 'app-notice--danger' : 'app-notice--info'"
        >
          <i class="pi pi-info-circle" />
          <strong v-if="m.title">{{ m.title }}</strong>
          <span>{{ m.description }}</span>
          <a v-if="m.url" :href="m.url" target="_blank" rel="noopener noreferrer">View</a>
        </div>
      </template>
      <div v-if="serverConfig.has_legacy_user" class="app-notice app-notice--danger">
        <i class="pi pi-exclamation-triangle" />
        Remove <code>admin_username</code> / <code>admin_password</code> from config.
        <router-link :to="{ name: 'users' }">Go to Users</router-link>
      </div>
    </template>

    <no-access v-if="denied" :page="pageTitle" />
    <router-view v-else v-slot="{ Component, route: r }">
      <Transition name="ln-page" mode="out-in">
        <div :key="viewKey(r)" class="ln-view">
          <component :is="Component" />
        </div>
      </Transition>
    </router-view>
  </app-shell>

  <div v-else class="app-loading">
    <PvProgressSpinner style="width:48px;height:48px" stroke-width="3" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter, type RouteLocationNormalizedLoaded } from 'vue-router';
import { uris } from './constants';
import { useMainStore } from './store';
import { setToastInstance, setConfirmInstance } from './toastService';
import { useGlobal } from './composables/useGlobal';
import { useDarkMode } from './composables/useDarkMode';
import AppShell from './components/AppShell.vue';
import NoAccess from './views/NoAccess.vue';
import { translateOr } from './composables/translateOr';
import { waitForServer } from './api/waitForServer';

setToastInstance(useToast());
setConfirmInstance(useConfirm());

const { $api, $utils, $can } = useGlobal();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();

// Routes that share a view and update it from the URL themselves (settings
// sections, the open subscriber or user, roles tabs) name it in meta.view,
// so moving between them keeps the page instead of rebuilding it. Anything
// else is rebuilt, and animated in, on every URL change.
function viewKey(r: RouteLocationNormalizedLoaded): string {
  return (r.meta.view as string | undefined) || r.fullPath;
}
const { serverConfig, profile } = storeToRefs(useMainStore());

const isLoaded = ref(false);

const denied = computed(() => {
  const perms = route.meta.perms as string[] | undefined;
  return !!perms?.length && !$can(...perms);
});

const pageTitle = computed(() => {
  const title = route.meta.title as string | undefined;
  return title ? translateOr(t, title, title) : '';
});
const { isDark, toggleDark } = useDarkMode();

const accountItems = computed(() => [
  { label: t('users.profile'), icon: 'pi pi-user', command: () => router.push('/user/profile') },
  { separator: true },
  { label: t('users.logout'), icon: 'pi pi-sign-out', command: () => doLogout() },
]);

function triggerRefresh() { useMainStore().refresh(); }

function doLogout() {
  $api.logout().then(() => { document.location.href = uris.root; });
}

// How long a restart may take before we stop waiting and say so.
const RESTART_TIMEOUT_MS = 60000;

async function reloadApp() {
  try {
    await $api.reloadApp();
  } catch {
    // The API client has already shown the error.
    return;
  }
  $utils.toast(t('settings.restarting'));
  try {
    await waitForServer(RESTART_TIMEOUT_MS);
    document.location.reload();
  } catch {
    $utils.toast(t('settings.restartTimeout'), 'is-danger');
  }
}

// MAX_ERROR_TOASTS caps how many server errors are toasted per page load,
// so a burst of failures doesn't bury the screen.
const MAX_ERROR_TOASTS = 50;

interface ServerEvent {
  type?: string;
  message?: string;
}

function parseEvent(raw: string): ServerEvent | null {
  try {
    const d: unknown = JSON.parse(raw);
    return d !== null && typeof d === 'object' ? d as ServerEvent : null;
  } catch {
    return null;
  }
}

function listenEvents() {
  const re = /(.+?)\.go:\d+:(.+?)$/im;
  const src = new EventSource(uris.errorEvents, { withCredentials: true });
  let n = 0;
  src.onmessage = (e) => {
    if (n > MAX_ERROR_TOASTS) return;
    const d = parseEvent(e.data);
    if (d?.type !== 'error' || typeof d.message !== 'string') return;
    n += 1;
    const m = re.exec(d.message.trim());
    if (m) $utils.toast(m[2], 'is-danger');
  };
}

onMounted(() => {
  isLoaded.value = true;
  $api.getLists({ minimal: true, per_page: 'all', status: 'active' });
  listenEvents();
});
</script>

<style lang="scss">
@import "assets/style.scss";
@import "assets/shell.scss";
@import "assets/patterns.scss";
@import "assets/motion.scss";
@import "assets/tinymce.scss";
</style>
