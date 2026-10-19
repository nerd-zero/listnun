import type { RouteLocationRaw } from 'vue-router';
import type { LnIconName } from './ui/lnIcons';

export interface NavKid {
  key: string;
  label: string;
  to: RouteLocationRaw;
  // Route names that mark this entry as current.
  names: string[];
  cy?: string;
}

export interface NavItem {
  key: string;
  label: string;
  icon: LnIconName;
  to: RouteLocationRaw;
  // route.meta.group the item owns, for the active state.
  group?: string;
  names?: string[];
  kids?: NavKid[];
  cy?: string;
}

export interface NavGroup {
  key: string;
  label: string;
  items: NavItem[];
}

type Translate = (key: string) => string;
type Can = (...perms: string[]) => boolean;

function kid(key: string, label: string, name: string, cy?: string, extra: string[] = []): NavKid {
  return {
    key, label, to: { name }, names: [name, ...extra], cy,
  };
}

// navGroups builds the sidebar from the user's permissions. Each entry
// keeps the permission check it had before the redesign.
export function navGroups(t: Translate, can: Can): NavGroup[] {
  const main: NavItem[] = [
    {
      key: 'dashboard', label: t('menu.dashboard'), icon: 'dash', to: { name: 'dashboard' }, names: ['dashboard'],
    },
    {
      key: 'lists',
      label: t('globals.terms.lists'),
      icon: 'lists',
      to: { name: 'lists' },
      group: 'lists',
      kids: [
        kid('lists', t('menu.allLists'), 'lists', 'all-lists', ['list']),
        kid('forms', t('menu.forms'), 'forms'),
      ],
    },
  ];

  if (can('subscribers:*')) {
    const kids: NavKid[] = [];
    if (can('subscribers:get_all', 'subscribers:get')) {
      kids.push(kid('subscribers', t('menu.allSubscribers'), 'subscribers', 'all-subscribers', ['subscriber', 'subscribers_list']));
    }
    if (can('subscribers:import')) kids.push(kid('import', t('menu.import'), 'import', 'import'));
    if (can('bounces:get')) kids.push(kid('bounces', t('globals.terms.bounces'), 'bounces', 'bounces'));
    main.push({
      key: 'subscribers', label: t('globals.terms.subscribers'), icon: 'subs', to: { name: 'subscribers' }, group: 'subscribers', kids,
    });
  }

  if (can('campaigns:*')) {
    const kids: NavKid[] = [];
    if (can('campaigns:get')) kids.push(kid('campaigns', t('menu.allCampaigns'), 'campaigns', 'all-campaigns'));
    if (can('campaigns:manage')) {
      kids.push({
        key: 'new', label: t('menu.newCampaign'), to: { name: 'campaign', params: { id: 'new' } }, names: [], cy: 'new-campaign',
      });
    }
    if (can('media:*')) kids.push(kid('media', t('menu.media'), 'media', 'media'));
    if (can('templates:get')) kids.push(kid('templates', t('globals.terms.templates'), 'templates', 'templates'));
    if (can('campaigns:get_analytics')) kids.push(kid('analytics', t('globals.terms.analytics'), 'campaignAnalytics', 'analytics'));
    main.push({
      key: 'campaigns', label: t('globals.terms.campaigns'), icon: 'camp', to: { name: 'campaigns' }, group: 'campaigns', kids,
    });
  }

  // Developers: one entry per page, each gated by the permission that opens it.
  const developers: NavItem[] = [];
  if (can('apikeys:get', 'tx:send', 'tx:logs', 'templates:get', 'webhooks:get')) {
    developers.push({
      key: 'devOverview', label: t('developers.overview.title'), icon: 'code', to: { name: 'devOverview' }, names: ['devOverview'],
    });
  }
  if (can('apikeys:get')) {
    developers.push({
      key: 'devKeys', label: t('developers.keys.title'), icon: 'key', to: { name: 'devKeys' }, names: ['devKeys'],
    });
  }

  if (can('templates:get')) {
    developers.push({
      key: 'devTemplates', label: t('developers.templates.title'), icon: 'file', to: { name: 'devTemplates' }, names: ['devTemplates', 'devTemplate'],
    });
  }
  if (can('tx:send')) {
    developers.push({
      key: 'devSendTest', label: t('developers.send.title'), icon: 'send', to: { name: 'devSendTest' }, names: ['devSendTest'],
    });
  }
  if (can('tx:logs')) {
    developers.push({
      key: 'devLog', label: t('developers.log.title'), icon: 'log', to: { name: 'devLog' }, names: ['devLog'],
    });
  }

  if (can('templates:get')) {
    developers.push({
      key: 'devBranding', label: t('developers.branding.title'), icon: 'image', to: { name: 'devBranding' }, names: ['devBranding'],
    });
  }

  if (can('webhooks:get')) {
    developers.push({
      key: 'devWebhooks', label: t('developers.webhooks.title'), icon: 'webhook', to: { name: 'devWebhooks' }, names: ['devWebhooks'],
    });
  }

  const workspace: NavItem[] = [];
  if (can('users:*', 'roles:*')) {
    const kids: NavKid[] = [];
    if (can('users:get')) kids.push(kid('users', t('globals.terms.users'), 'users', 'users'));
    if (can('roles:get')) {
      kids.push(kid('userRoles', t('users.userRoles'), 'userRoles', 'userRoles'));
      kids.push(kid('listRoles', t('users.listRoles'), 'listRoles', 'listRoles'));
    }
    workspace.push({
      key: 'users', label: t('globals.terms.users'), icon: 'users', to: { name: 'users' }, group: 'users', kids,
    });
  }
  if (can('settings:*')) {
    workspace.push({
      key: 'settings',
      label: t('globals.terms.settings'),
      icon: 'gear',
      to: { name: can('settings:get') ? 'settings' : 'maintenance' },
      group: 'settings',
      cy: 'all-settings',
    });
  }

  return [
    { key: 'main', label: '', items: main },
    { key: 'developers', label: t('globals.terms.developers'), items: developers },
    { key: 'workspace', label: t('menu.workspace'), items: workspace },
  ].filter((g) => g.items.length > 0);
}
