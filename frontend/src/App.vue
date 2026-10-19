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
    <router-view v-else :key="$route.fullPath" />
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
import { useRoute, useRouter } from 'vue-router';
import { uris } from './constants';
import { useMainStore } from './store';
import { setToastInstance, setConfirmInstance } from './toastService';
import { useGlobal } from './composables/useGlobal';
import AppShell from './components/AppShell.vue';
import NoAccess from './views/NoAccess.vue';
import { translateOr } from './composables/translateOr';

setToastInstance(useToast());
setConfirmInstance(useConfirm());

const { $api, $utils, $can } = useGlobal();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
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
const isDark = ref(localStorage.getItem('lm-dark') === '1');
if (isDark.value) document.documentElement.classList.add('app-dark');

const accountItems = computed(() => [
  { label: t('users.profile'), icon: 'pi pi-user', command: () => router.push('/user/profile') },
  { separator: true },
  { label: t('users.logout'), icon: 'pi pi-sign-out', command: () => doLogout() },
]);

function toggleDark() {
  isDark.value = !isDark.value;
  document.documentElement.classList.toggle('app-dark', isDark.value);
  localStorage.setItem('lm-dark', isDark.value ? '1' : '');
}

function triggerRefresh() { useMainStore().refresh(); }

function doLogout() {
  $api.logout().then(() => { document.location.href = uris.root; });
}

function reloadApp() {
  $api.reloadApp().then(() => {
    $utils.toast('Reloading…');
    const poll = setInterval(() => {
      $api.getHealth().then(() => { clearInterval(poll); document.location.reload(); });
    }, 500);
  });
}

function listenEvents() {
  const re = /(.+?)\.go:\d+:(.+?)$/im;
  const src = new EventSource(uris.errorEvents, { withCredentials: true });
  let n = 0;
  src.onmessage = (e) => {
    if (n > 50) return;
    n += 1;
    const d = JSON.parse(e.data);
    if (d?.type === 'error') {
      const m = re.exec(d.message.trim());
      if (m) $utils.toast(m[2], 'is-danger', null, true);
    }
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
@import "assets/icons/fontello.css";
</style>
