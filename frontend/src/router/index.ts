import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

// meta.group marks the nav group a page belongs to. meta.perms lists the
// permissions that open it (any one is enough); without them App.vue
// shows NoAccess in place of the page.
const routes: RouteRecordRaw[] = [
  {
    path: '/404',
    name: '404_page',
    meta: { title: '404' },
    component: () => import('../views/404.vue'),
  },
  {
    path: '/',
    name: 'dashboard',
    meta: { title: '' },
    component: () => import('../views/Dashboard.vue'),
  },
  {
    path: '/lists',
    name: 'lists',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import('../views/Lists.vue'),
  },
  {
    path: '/lists/forms',
    name: 'forms',
    meta: { title: 'forms.title', group: 'lists' },
    component: () => import('../views/Forms.vue'),
  },
  {
    path: '/lists/:id',
    name: 'list',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import('../views/Lists.vue'),
  },
  {
    path: '/subscribers',
    name: 'subscribers',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers', perms: ['subscribers:get_all', 'subscribers:get'] },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/import',
    name: 'import',
    meta: { title: 'import.title', group: 'subscribers', perms: ['subscribers:import'] },
    component: () => import('../views/Import.vue'),
  },
  {
    path: '/subscribers/bounces',
    name: 'bounces',
    meta: { title: 'globals.terms.bounces', group: 'subscribers', perms: ['bounces:get'] },
    component: () => import('../views/Bounces.vue'),
  },
  {
    path: '/subscribers/lists/:listID',
    name: 'subscribers_list',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers', perms: ['subscribers:get_all', 'subscribers:get'] },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/:id',
    name: 'subscriber',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers', perms: ['subscribers:get_all', 'subscribers:get'] },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/campaigns',
    name: 'campaigns',
    meta: { title: 'globals.terms.campaigns', group: 'campaigns', perms: ['campaigns:get_all', 'campaigns:get'] },
    component: () => import('../views/Campaigns.vue'),
  },
  {
    path: '/campaigns/media',
    name: 'media',
    meta: { title: 'globals.terms.media', group: 'campaigns', perms: ['media:get'] },
    component: () => import('../views/Media.vue'),
  },
  {
    path: '/campaigns/templates',
    name: 'templates',
    meta: { title: 'globals.terms.templates', group: 'campaigns', perms: ['templates:get'] },
    component: () => import('../views/Templates.vue'),
  },
  {
    path: '/campaigns/analytics',
    name: 'campaignAnalytics',
    meta: { title: 'analytics.title', group: 'campaigns', perms: ['campaigns:get_analytics'] },
    component: () => import('../views/CampaignAnalytics.vue'),
  },
  {
    path: '/campaigns/:id',
    name: 'campaign',
    meta: { title: 'globals.terms.campaign', group: 'campaigns', perms: ['campaigns:get_all', 'campaigns:get'] },
    component: () => import('../views/Campaign.vue'),
  },
  {
    path: '/developers',
    name: 'devOverview',
    meta: {
      title: 'developers.overview.title',
      section: 'developers.title',
      group: 'developers',
      perms: ['apikeys:get', 'tx:send', 'tx:logs', 'templates:get', 'webhooks:get'],
    },
    component: () => import('../views/developers/Overview.vue'),
  },
  {
    path: '/developers/keys',
    name: 'devKeys',
    meta: {
      title: 'developers.keys.title', section: 'developers.title', group: 'developers', perms: ['apikeys:get'],
    },
    component: () => import('../views/developers/ApiKeys.vue'),
  },
  {
    path: '/developers/log',
    name: 'devLog',
    meta: {
      title: 'developers.log.title', section: 'developers.title', group: 'developers', perms: ['tx:logs'],
    },
    component: () => import('../views/developers/MessageLog.vue'),
  },
  {
    path: '/developers/templates',
    name: 'devTemplates',
    meta: {
      title: 'developers.templates.title', section: 'developers.title', group: 'developers', perms: ['templates:get'],
    },
    component: () => import('../views/developers/TxTemplates.vue'),
  },
  {
    path: '/developers/templates/:id',
    name: 'devTemplate',
    meta: {
      title: 'developers.templates.title', section: 'developers.title', group: 'developers', perms: ['templates:get'],
    },
    component: () => import('../views/developers/TxTemplates.vue'),
  },
  {
    path: '/developers/send',
    name: 'devSendTest',
    meta: {
      title: 'developers.send.title', section: 'developers.title', group: 'developers', perms: ['tx:send'],
    },
    component: () => import('../views/developers/SendTest.vue'),
  },
  {
    path: '/developers/branding',
    name: 'devBranding',
    meta: {
      title: 'developers.branding.title', section: 'developers.title', group: 'developers', perms: ['templates:get'],
    },
    component: () => import('../views/developers/Branding.vue'),
  },
  {
    path: '/developers/webhooks',
    name: 'devWebhooks',
    meta: {
      title: 'developers.webhooks.title', section: 'developers.title', group: 'developers', perms: ['webhooks:get'],
    },
    component: () => import('../views/developers/Webhooks.vue'),
  },
  {
    path: '/user/profile',
    name: 'userProfile',
    meta: { title: 'users.profile', group: 'profile' },
    component: () => import('../views/UserProfile.vue'),
  },
  {
    path: '/settings',
    name: 'settings',
    meta: { title: 'globals.terms.settings', group: 'settings', perms: ['settings:get'] },
    component: () => import('../views/Settings.vue'),
  },
  {
    path: '/users',
    name: 'users',
    meta: { title: 'globals.terms.users', group: 'users', perms: ['users:get'] },
    component: () => import('../views/Users.vue'),
  },
  {
    path: '/users/roles/users',
    name: 'userRoles',
    meta: { title: 'users.userRoles', group: 'users', perms: ['roles:get'] },
    component: () => import('../views/Roles.vue'),
  },
  {
    path: '/users/roles/lists',
    name: 'listRoles',
    meta: { title: 'users.listRoles', group: 'users', perms: ['roles:get'] },
    component: () => import('../views/Roles.vue'),
  },
  {
    path: '/settings/maintenance',
    name: 'maintenance',
    meta: { title: 'maintenance.title', group: 'settings', perms: ['settings:maintain'] },
    component: () => import('../views/Maintenance.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,

  scrollBehavior(to) {
    if (to.hash) {
      return { el: to.hash };
    }
    return { top: 0, left: 0 };
  },
});

export default router;
