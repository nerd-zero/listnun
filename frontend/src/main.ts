import { createApp } from 'vue';
import { createPinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import PrimeVue from 'primevue/config';
import Aura from '@primeuix/themes/aura';
import { definePreset } from '@primeuix/themes';
import ToastService from 'primevue/toastservice';
import ConfirmationService from 'primevue/confirmationservice';
import 'primeicons/primeicons.css';
import 'primeflex/primeflex.css';

import Button from 'primevue/button';
import InputText from 'primevue/inputtext';
import Textarea from 'primevue/textarea';
import Select from 'primevue/select';
import ToggleSwitch from 'primevue/toggleswitch';
import Tag from 'primevue/tag';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Dialog from 'primevue/dialog';
import ProgressBar from 'primevue/progressbar';
import ProgressSpinner from 'primevue/progressspinner';
import Tabs from 'primevue/tabs';
import TabList from 'primevue/tablist';
import Tab from 'primevue/tab';
import TabPanels from 'primevue/tabpanels';
import TabPanel from 'primevue/tabpanel';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import Tooltip from 'primevue/tooltip';
import Badge from 'primevue/badge';
import Chip from 'primevue/chip';
import InputNumber from 'primevue/inputnumber';
import Password from 'primevue/password';
import Checkbox from 'primevue/checkbox';
import RadioButton from 'primevue/radiobutton';
import Paginator from 'primevue/paginator';
import Menu from 'primevue/menu';
import Menubar from 'primevue/menubar';
import PanelMenu from 'primevue/panelmenu';
import Drawer from 'primevue/drawer';
import Message from 'primevue/message';
import FloatLabel from 'primevue/floatlabel';
import AutoComplete from 'primevue/autocomplete';
import MultiSelect from 'primevue/multiselect';
import DatePicker from 'primevue/datepicker';
import Divider from 'primevue/divider';
import Panel from 'primevue/panel';
import Card from 'primevue/card';
import Avatar from 'primevue/avatar';
import IconField from 'primevue/iconfield';
import InputIcon from 'primevue/inputicon';
import FileUpload from 'primevue/fileupload';

import App from './App.vue';
import router from './router';
import * as api from './api';
import Utils from './utils';
import eventBus from './eventBus';

const BlackOnBeigePreset = definePreset(Aura, {
  semantic: {
    primary: {
      // 50-300: cream/beige light tints (hover backgrounds, badges).
      // 400: beige again, used as dark mode's primary color since pure
      // black has no contrast against a dark background. 500+: black,
      // the actual light-mode primary -- 600 lightens slightly for
      // hover since black itself has nowhere darker to go.
      50: '#FAF8F3',
      100: '#F5F0E6',
      200: '#EDE3D0',
      300: '#D8D0C0',
      400: '#D8D0C0',
      500: '#000000',
      600: '#262626',
      700: '#000000',
      800: '#000000',
      900: '#000000',
      950: '#000000',
    },
  },
});

const pinia = createPinia();

const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: 'en',
  messages: {},
});

const app = createApp(App);

app.use(pinia);
app.use(router);
app.use(i18n);

app.use(PrimeVue, {
  theme: {
    preset: BlackOnBeigePreset,
    options: { darkModeSelector: '.app-dark' },
  },
  ripple: true,
});
app.use(ToastService);
app.use(ConfirmationService);

app.component('PvButton', Button);
app.component('PvInputText', InputText);
app.component('PvTextarea', Textarea);
app.component('PvSelect', Select);
app.component('PvToggleSwitch', ToggleSwitch);
app.component('PvTag', Tag);
app.component('PvDataTable', DataTable);
app.component('PvColumn', Column);
app.component('PvDialog', Dialog);
app.component('PvProgressBar', ProgressBar);
app.component('PvProgressSpinner', ProgressSpinner);
app.component('PvTabs', Tabs);
app.component('PvTabList', TabList);
app.component('PvTab', Tab);
app.component('PvTabPanels', TabPanels);
app.component('PvTabPanel', TabPanel);
app.component('PvToast', Toast);
app.component('PvConfirmDialog', ConfirmDialog);
app.component('PvBadge', Badge);
app.component('PvChip', Chip);
app.component('PvInputNumber', InputNumber);
app.component('PvPassword', Password);
app.component('PvCheckbox', Checkbox);
app.component('PvRadioButton', RadioButton);
app.component('PvPaginator', Paginator);
app.component('PvMenu', Menu);
app.component('PvMenubar', Menubar);
app.component('PvPanelMenu', PanelMenu);
app.component('PvDrawer', Drawer);
app.component('PvMessage', Message);
app.component('PvInlineMessage', Message);
app.component('PvFloatLabel', FloatLabel);
app.component('PvAutoComplete', AutoComplete);
app.component('PvMultiSelect', MultiSelect);
app.component('PvDatePicker', DatePicker);
app.component('PvDivider', Divider);
app.component('PvPanel', Panel);
app.component('PvCard', Card);
app.component('PvAvatar', Avatar);
app.component('PvIconField', IconField);
app.component('PvInputIcon', InputIcon);
app.component('PvFileUpload', FileUpload);

app.directive('tooltip', Tooltip);

router.beforeEach((to, _from, next) => {
  if (to.matched.length === 0) {
    next('/404');
  } else {
    next();
  }
});

router.afterEach((to) => {
  const { te, t } = i18n.global;
  const title = to.meta.title && te(to.meta.title as string) ? `${t(to.meta.title as string)} /` : '';
  document.title = `${title} listnun`;
});

async function initConfig(instance: typeof app) {
  let profile: Record<string, unknown>;
  let cfg: Record<string, unknown>;
  try {
    [profile, cfg] = await Promise.all([api.getUserProfile(), api.getServerConfig()]);
  } catch (err: unknown) {
    const axiosErr = err as { response?: { status: number } };
    if (axiosErr.response && axiosErr.response.status === 403) {
      window.location.href = '/admin/login';
    }
    return;
  }

  const lang = await api.getLang(cfg.lang as string);
  (i18n.global.locale as unknown as { value: string }).value = cfg.lang as string;
  i18n.global.setLocaleMessage(cfg.lang as string, lang as Record<string, unknown>);

  const props = instance.config.globalProperties;
  props.$utils = new Utils(i18n.global as ConstructorParameters<typeof Utils>[0]);
  props.$api = api;
  props.$events = eventBus;

  props.$can = (...perms: string[]) => {
    const { userRole } = profile as { userRole: { id: number; permissions: string[] } };
    if (userRole.id === 1) {
      return true;
    }
    return perms.some((perm) => {
      if (perm.endsWith('*')) {
        const group = `${perm.split(':')[0]}:`;
        return userRole.permissions.some((p: string) => p.startsWith(group));
      }
      return userRole.permissions.includes(perm);
    });
  };

  props.$canList = (id: number, perm: string) => {
    const { userRole } = profile as { userRole: { id: number } };
    if (userRole.id === 1) {
      return true;
    }
    const can = props.$can('lists:get_all', 'lists:manage_all');
    if (can) {
      return true;
    }
    const { listRole } = profile as { listRole: { lists: { id: number; permissions: string[] }[] } };
    return listRole.lists.some(
      (list) => list.id === id && list.permissions.includes(perm),
    );
  };

  const currentRoute = router.currentRoute.value;
  const routeTitle = currentRoute.meta.title
    ? `${i18n.global.t(currentRoute.meta.title as string)} /`
    : '';
  document.title = `${routeTitle} listnun`;

  instance.mount('#app');
}

initConfig(app);
