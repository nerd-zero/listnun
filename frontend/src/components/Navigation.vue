<template>
  <div class="nav-sections">
    <div class="nav-section">
      <router-link :to="{ name: 'dashboard' }" class="nav-item" exact-active-class="nav-item--active">
        <i class="pi pi-home nav-icon" />
        <span>{{ $t('menu.dashboard') }}</span>
      </router-link>
    </div>

    <div class="nav-section">
      <p class="nav-label">{{ $t('globals.terms.lists') }}</p>
      <router-link :to="{ name: 'lists' }" class="nav-item" exact-active-class="nav-item--active" data-cy="all-lists">
        <i class="pi pi-list nav-icon" />
        <span>{{ $t('menu.allLists') }}</span>
      </router-link>
      <router-link :to="{ name: 'forms' }" class="nav-item" exact-active-class="nav-item--active">
        <i class="pi pi-globe nav-icon" />
        <span>{{ $t('menu.forms') }}</span>
      </router-link>
    </div>

    <div v-if="$can('subscribers:*')" class="nav-section">
      <p class="nav-label">{{ $t('globals.terms.subscribers') }}</p>
      <router-link
        v-if="$can('subscribers:get_all', 'subscribers:get')"
        :to="{ name: 'subscribers' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="all-subscribers"
      >
        <i class="pi pi-users nav-icon" />
        <span>{{ $t('menu.allSubscribers') }}</span>
      </router-link>
      <router-link
        v-if="$can('subscribers:import')"
        :to="{ name: 'import' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="import"
      >
        <i class="pi pi-upload nav-icon" />
        <span>{{ $t('menu.import') }}</span>
      </router-link>
      <router-link
        v-if="$can('bounces:get')"
        :to="{ name: 'bounces' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="bounces"
      >
        <i class="pi pi-times-circle nav-icon" />
        <span>{{ $t('globals.terms.bounces') }}</span>
      </router-link>
    </div>

    <div v-if="$can('campaigns:*')" class="nav-section">
      <p class="nav-label">{{ $t('globals.terms.campaigns') }}</p>
      <router-link
        v-if="$can('campaigns:get')"
        :to="{ name: 'campaigns' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="all-campaigns"
      >
        <i class="pi pi-send nav-icon" />
        <span>{{ $t('menu.allCampaigns') }}</span>
      </router-link>
      <router-link
        v-if="$can('campaigns:manage')"
        :to="{ name: 'campaign', params: { id: 'new' } }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="new-campaign"
      >
        <i class="pi pi-plus-circle nav-icon" />
        <span>{{ $t('menu.newCampaign') }}</span>
      </router-link>
      <router-link
        v-if="$can('media:*')"
        :to="{ name: 'media' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="media"
      >
        <i class="pi pi-image nav-icon" />
        <span>{{ $t('menu.media') }}</span>
      </router-link>
      <router-link
        v-if="$can('templates:get')"
        :to="{ name: 'templates' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="templates"
      >
        <i class="pi pi-file nav-icon" />
        <span>{{ $t('globals.terms.templates') }}</span>
      </router-link>
      <router-link
        v-if="$can('campaigns:get_analytics')"
        :to="{ name: 'campaignAnalytics' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="analytics"
      >
        <i class="pi pi-chart-line nav-icon" />
        <span>{{ $t('globals.terms.analytics') }}</span>
      </router-link>
    </div>

    <div v-if="$can('users:*', 'roles:*')" class="nav-section">
      <p class="nav-label">{{ $t('globals.terms.users') }}</p>
      <router-link
        v-if="$can('users:get')"
        :to="{ name: 'users' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="users"
      >
        <i class="pi pi-user nav-icon" />
        <span>{{ $t('globals.terms.users') }}</span>
      </router-link>
      <router-link
        v-if="$can('roles:get')"
        :to="{ name: 'userRoles' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="userRoles"
      >
        <i class="pi pi-shield nav-icon" />
        <span>{{ $t('users.userRoles') }}</span>
      </router-link>
      <router-link
        v-if="$can('roles:get')"
        :to="{ name: 'listRoles' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="listRoles"
      >
        <i class="pi pi-key nav-icon" />
        <span>{{ $t('users.listRoles') }}</span>
      </router-link>
    </div>

    <div v-if="$can('settings:*')" class="nav-section">
      <p class="nav-label">System</p>
      <router-link
        v-if="$can('settings:get')"
        :to="{ name: 'settings' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="all-settings"
      >
        <i class="pi pi-cog nav-icon" />
        <span>{{ $t('menu.settings') }}</span>
      </router-link>
      <router-link
        v-if="$can('settings:maintain')"
        :to="{ name: 'maintenance' }"
        class="nav-item"
        exact-active-class="nav-item--active"
        data-cy="maintenance"
      >
        <i class="pi pi-wrench nav-icon" />
        <span>{{ $t('menu.maintenance') }}</span>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  activeItem?: Record<string, unknown>;
}>(), {
  activeItem: () => ({}),
});

defineEmits(['toggle-group']);
</script>
