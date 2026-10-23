<template>
  <div class="ln-shell" :class="{ 'is-rail': showRail && !isMobile, 'is-mobile': isMobile }">
    <!-- Sidebar or rail. The wrapper animates its width between the two
         and the contents cross-fade. -->
    <div v-if="!isMobile" class="ln-side" :class="{ 'is-rail': showRail }">
      <Transition name="ln-side-swap" mode="out-in">
        <aside v-if="!showRail" key="sidebar" class="ln-sidebar">
          <router-link :to="{ name: 'dashboard' }" class="ln-brand">
            <span class="ln-logo">
              <img src="@/assets/listnun-mark.png" alt="" class="ln-logo-mark" />
              <span class="ln-logo-word">listnun</span>
            </span>
            <span v-if="orgName" class="ln-brand-org" :title="orgName">{{ orgName }}</span>
          </router-link>

          <navigation />

          <div class="ln-sidebar-foot">
            <a v-if="consoleUrl" :href="consoleUrl" class="ln-nav-item" target="_blank" rel="noopener noreferrer">
              <ln-icon name="external" class="ln-nav-icon" />
              <span class="ln-nav-text">{{ $t('menu.billing') }}</span>
            </a>
            <button type="button" class="ln-nav-item" @click="setRail(true)">
              <ln-icon name="collapse" class="ln-nav-icon" />
              <span class="ln-nav-text">{{ $t('menu.collapse') }}</span>
            </button>
            <p class="ln-powered">
              {{ $t('public.poweredBy') }}
              <a href="https://n0.rocks" target="_blank" rel="noopener noreferrer">Nerd Zero</a>
            </p>
          </div>
        </aside>
        <aside v-else key="rail" class="ln-rail">
          <router-link :to="{ name: 'dashboard' }" class="ln-rail-brand" aria-label="listnun">
            <img src="@/assets/listnun-mark.png" alt="" class="ln-logo-mark" />
          </router-link>
          <navigation rail />
          <div class="ln-rail-foot">
            <a
              v-if="consoleUrl"
              v-tooltip.right="$t('menu.billing')"
              :href="consoleUrl"
              class="ln-rail-item"
              target="_blank"
              rel="noopener noreferrer"
              :aria-label="$t('menu.billing')"
            >
              <ln-icon name="external" />
            </a>
            <button
              v-if="!forcedRail"
              v-tooltip.right="$t('menu.expand')"
              type="button"
              class="ln-rail-item"
              :aria-label="$t('menu.expand')"
              @click="setRail(false)"
            >
              <ln-icon name="expand" />
            </button>
          </div>
        </aside>
      </Transition>
    </div>

    <!-- Drawer under 992px -->
    <PvDrawer v-if="isMobile" v-model:visible="drawerOpen" position="left" class="ln-drawer" :show-close-icon="false">
      <template #container="{ closeCallback }">
        <div class="ln-sidebar ln-sidebar--drawer">
          <div class="ln-drawer-head">
            <router-link :to="{ name: 'dashboard' }" class="ln-brand">
              <span class="ln-logo">
                <img src="@/assets/listnun-mark.png" alt="" class="ln-logo-mark" />
                <span class="ln-logo-word">listnun</span>
              </span>
              <span v-if="orgName" class="ln-brand-org">{{ orgName }}</span>
            </router-link>
            <button type="button" class="ln-icon-btn" :aria-label="$t('menu.closeMenu')" @click="closeCallback">
              <ln-icon name="close" />
            </button>
          </div>
          <navigation />
          <div class="ln-sidebar-foot">
            <a v-if="consoleUrl" :href="consoleUrl" class="ln-nav-item" target="_blank" rel="noopener noreferrer">
              <ln-icon name="external" class="ln-nav-icon" />
              <span class="ln-nav-text">{{ $t('menu.billing') }}</span>
            </a>
            <p class="ln-powered">
              {{ $t('public.poweredBy') }}
              <a href="https://n0.rocks" target="_blank" rel="noopener noreferrer">Nerd Zero</a>
            </p>
          </div>
        </div>
      </template>
    </PvDrawer>

    <div class="ln-main">
      <header class="ln-topbar">
        <div class="ln-topbar-start">
          <template v-if="isMobile">
            <button type="button" class="ln-icon-btn ln-icon-btn--lg" :aria-label="$t('menu.openMenu')" @click="drawerOpen = true">
              <ln-icon name="menu" />
            </button>
            <router-link :to="{ name: 'dashboard' }" class="ln-logo">
              <img src="@/assets/listnun-mark.png" alt="" class="ln-logo-mark" />
              <span class="ln-logo-word">listnun</span>
            </router-link>
          </template>
          <span v-else class="ln-crumb">
            <template v-if="crumbSection">{{ crumbSection }}<span class="ln-crumb-sep" aria-hidden="true">/</span></template>
            <span class="ln-crumb-current">{{ crumb }}</span>
          </span>
        </div>

        <div class="ln-topbar-end">
          <button
            v-if="needsRestart && !isMobile"
            type="button"
            class="ln-restart"
            @click="$emit('restart')"
          >
            {{ $t('settings.needsRestart') }}
          </button>

          <button
            v-tooltip.bottom="$t('globals.buttons.refresh')"
            type="button"
            class="ln-icon-btn"
            :aria-label="$t('globals.buttons.refresh')"
            data-cy="btn-refresh"
            @click="$emit('refresh')"
          >
            <ln-icon name="refresh" />
          </button>

          <button
            v-tooltip.bottom="$t('menu.switchTheme')"
            type="button"
            class="ln-icon-btn"
            :aria-label="$t('menu.switchTheme')"
            @click="$emit('toggle-dark')"
          >
            <ln-icon :name="dark ? 'sun' : 'moon'" />
          </button>

          <button
            v-if="username"
            type="button"
            class="ln-account"
            data-cy="user-menu"
            :aria-label="$t('menu.account')"
            aria-haspopup="true"
            @click="(e) => accountMenu?.toggle(e)"
          >
            <span class="ln-account-avatar">{{ initials }}</span>
            <ln-icon name="chevronDown" :size="12" :stroke="1.6" />
          </button>
          <PvMenu ref="accountMenu" :model="accountItems" popup />
        </div>
      </header>

      <slot name="notices" />

      <main class="ln-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  computed, onBeforeUnmount, onMounted, ref, watch,
} from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import type { MenuItem } from 'primevue/menuitem';
import Navigation from './Navigation.vue';
import { translateOr } from '../composables/translateOr';
import LnIcon from './ui/LnIcon.vue';

// Below this width the sidebar becomes a drawer.
const MOBILE_BREAKPOINT = 992;
const RAIL_KEY = 'ln-rail';

interface PopupMenu { toggle: (e: Event) => void }

const props = defineProps<{
  orgName?: string;
  consoleUrl?: string;
  needsRestart?: boolean;
  dark: boolean;
  username?: string;
  name?: string;
  accountItems: MenuItem[];
}>();

defineEmits(['refresh', 'toggle-dark', 'restart']);

const { t } = useI18n();
const route = useRoute();

const isMobile = ref(window.innerWidth < MOBILE_BREAKPOINT);
const drawerOpen = ref(false);
const rail = ref(localStorage.getItem(RAIL_KEY) === '1');
const accountMenu = ref<PopupMenu | null>(null);

// Full-width pages (the campaign editor) set meta.rail to collapse the
// sidebar to the rail without changing the user's own preference.
const forcedRail = computed(() => !!route.meta.rail);
const showRail = computed(() => rail.value || forcedRail.value);

const crumb = computed(() => {
  const title = route.meta.title as string | undefined;
  if (!title) return t('menu.dashboard');
  return translateOr(t, title, title);
});

// meta.section names the area a page sits in, e.g. "Developers / API keys".
const crumbSection = computed(() => {
  const section = route.meta.section as string | undefined;
  return section ? translateOr(t, section, '') : '';
});

const initials = computed(() => {
  const src = (props.name || props.username || '').trim();
  const parts = src.split(/[\s._-]+/).filter(Boolean);
  const letters = parts.length > 1 ? parts[0][0] + parts[1][0] : src.slice(0, 2);
  return letters.toUpperCase();
});

function setRail(on: boolean) {
  rail.value = on;
  localStorage.setItem(RAIL_KEY, on ? '1' : '');
}

function onResize() {
  isMobile.value = window.innerWidth < MOBILE_BREAKPOINT;
  if (!isMobile.value) drawerOpen.value = false;
}

watch(() => route.fullPath, () => { drawerOpen.value = false; });

onMounted(() => window.addEventListener('resize', onResize));
onBeforeUnmount(() => window.removeEventListener('resize', onResize));
</script>
