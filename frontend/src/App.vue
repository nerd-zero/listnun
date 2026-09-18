<template>
  <div class="app-shell" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'app-dark': isDark }">
    <PvToast position="bottom-right" />
    <PvConfirmDialog />

    <template v-if="isLoaded">
      <!-- ─── Sidebar ─── -->
      <aside class="app-sidebar">
        <div class="sidebar-header">
          <router-link :to="{ name: 'dashboard' }" class="sidebar-brand">
            <img src="@/assets/listnun-mark.png" alt="listnun" class="sidebar-logo" />
            <span class="sidebar-brand-text">
              <span class="sidebar-brand-name">listnun</span>
              <span
                v-if="serverConfig.organization_name"
                class="sidebar-org-name"
                :title="serverConfig.organization_name"
              >{{ serverConfig.organization_name }}</span>
            </span>
          </router-link>
        </div>

        <nav class="sidebar-nav">
          <navigation :active-item="activeItem" />
        </nav>
</aside>

      <!-- ─── Main ─── -->
      <div class="app-main">
        <!-- Topbar -->
        <header class="app-topbar">
          <div class="topbar-start">
            <button type="button" class="topbar-btn" @click="sidebarCollapsed = !sidebarCollapsed">
              <i class="pi pi-bars" />
            </button>
          </div>

          <div class="topbar-end">
            <PvButton
              v-if="serverConfig.needs_restart"
              severity="danger"
              size="small"
              icon="pi pi-exclamation-triangle"
              :label="$t('settings.needsRestart')"
              @click="$utils.confirm($t('settings.confirmRestart'), reloadApp)"
            />

            <button
              type="button"
              class="topbar-btn"
              v-tooltip.bottom="$t('globals.buttons.refresh')"
              data-cy="btn-refresh"
              @click="triggerRefresh"
            >
              <i class="pi pi-refresh" />
            </button>

            <button type="button" class="topbar-btn" @click="toggleDark" v-tooltip.bottom="isDark ? 'Light mode' : 'Dark mode'">
              <i :class="['pi', isDark ? 'pi-sun' : 'pi-moon']" />
            </button>

            <button
              v-if="profile.username"
              type="button"
              class="topbar-user-btn"
              @click="(e) => userMenuRef.show(e)"
            >
              <PvAvatar
                :label="profile.username[0].toUpperCase()"
                class="topbar-avatar"
                shape="circle"
                size="small"
              />
              <span class="topbar-username">{{ profile.username }}</span>
              <i class="pi pi-chevron-down topbar-chevron" />
            </button>
            <PvMenu ref="userMenuRef" :model="userMenuItems" popup />
          </div>
        </header>

        <!-- System notices -->
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

        <!-- Page content -->
        <div class="app-content">
          <router-view :key="$route.fullPath" />
        </div>
      </div>

      <!-- Mobile overlay -->
      <div class="sidebar-overlay" @click="sidebarCollapsed = true" />
    </template>

    <div v-if="!isLoaded" class="app-loading">
      <PvProgressSpinner style="width:48px;height:48px" stroke-width="3" />
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref, computed, watch, onMounted,
} from 'vue';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { uris } from './constants';
import { useMainStore } from './store';
import { setToastInstance, setConfirmInstance } from './toastService';
import { useGlobal } from './composables/useGlobal';
import Navigation from './components/Navigation.vue';

setToastInstance(useToast());
setConfirmInstance(useConfirm());

const { $api, $utils } = useGlobal();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const { serverConfig, profile } = storeToRefs(useMainStore());

const isLoaded = ref(false);
const isDark = ref(localStorage.getItem('lm-dark') === '1');
if (isDark.value) document.documentElement.classList.add('app-dark');
const sidebarCollapsed = ref(window.innerWidth < 992);
const activeItem = ref<Record<string, boolean>>({});
const userMenuRef = ref<any>(null);

const userMenuItems = computed(() => [
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

watch(() => route.name, (name) => {
  activeItem.value = { [name as string]: true };
  if (window.innerWidth < 992) sidebarCollapsed.value = true;
});

onMounted(() => {
  isLoaded.value = true;
  $api.getLists({ minimal: true, per_page: 'all', status: 'active' });
  listenEvents();
  activeItem.value = { [route.name as string]: true };
  window.addEventListener('resize', () => {
    if (window.innerWidth >= 992) sidebarCollapsed.value = false;
  });
});
</script>

<style lang="scss">
@import "assets/style.scss";
@import "assets/icons/fontello.css";
</style>
