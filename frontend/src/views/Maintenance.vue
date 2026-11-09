<template>
  <div class="maint">
    <!-- Subscribers -->
    <section class="ln-card ln-card-pad maint-card">
      <h3 class="ln-card-title">{{ $t('globals.terms.subscribers') }}</h3>
      <p class="ln-small">{{ $t('maintenance.subscribersHelp') }}</p>
      <div class="maint-row">
        <PvSelect v-model="subscriberType" :options="subscriberTypeOptions" option-label="label" option-value="value" :aria-label="$t('maintenance.which')" />
        <span class="maint-spacer" />
        <PvButton severity="danger" outlined :label="$t('globals.buttons.delete')" :loading="busy === 'subscribers'" @click="askSubscribers" />
      </div>
    </section>

    <!-- Subscriptions -->
    <section class="ln-card ln-card-pad maint-card">
      <h3 class="ln-card-title">{{ $t('globals.terms.subscriptions', 2) }}</h3>
      <p class="ln-small">{{ $t('maintenance.subscriptionsHelp') }}</p>
      <div class="maint-row">
        <span class="ln-small">{{ $t('maintenance.olderThan') }}</span>
        <PvDatePicker v-model="subscriptionDate" :max-date="new Date()" date-format="d M yy" show-icon icon-display="input" :aria-label="$t('maintenance.olderThan')" />
        <span class="maint-spacer" />
        <PvButton severity="danger" outlined :label="$t('globals.buttons.delete')" :loading="busy === 'subscriptions'" @click="askSubscriptions" />
      </div>
    </section>

    <!-- Analytics -->
    <section class="ln-card ln-card-pad maint-card">
      <h3 class="ln-card-title">{{ $t('globals.terms.analytics') }}</h3>
      <p class="ln-small">{{ $t('maintenance.analyticsHelp') }}</p>
      <div class="maint-row">
        <PvSelect v-model="analyticsType" :options="analyticsTypeOptions" option-label="label" option-value="value" :aria-label="$t('maintenance.which')" />
        <span class="ln-small">{{ $t('maintenance.before') }}</span>
        <PvDatePicker v-model="analyticsDate" :max-date="new Date()" date-format="d M yy" show-icon icon-display="input" :aria-label="$t('maintenance.before')" />
        <span class="maint-spacer" />
        <PvButton severity="danger" outlined :label="$t('globals.buttons.delete')" :loading="busy === 'analytics'" @click="askAnalytics" />
      </div>
    </section>

    <!-- Export analytics -->
    <section class="ln-card ln-card-pad maint-card">
      <h3 class="ln-card-title">{{ $t('maintenance.exportTitle') }}</h3>
      <p class="ln-small">{{ $t('maintenance.exportHelp') }}</p>
      <div class="maint-row">
        <PvSelect v-model="exportType" :options="exportTypeOptions" option-label="label" option-value="value" :aria-label="$t('maintenance.which')" />
        <span class="ln-small">{{ $t('analytics.fromDate') }}</span>
        <PvDatePicker v-model="exportDate" :max-date="new Date()" date-format="d M yy" show-icon icon-display="input" :aria-label="$t('analytics.fromDate')" />
        <span class="maint-spacer" />
        <a :href="exportURL" class="p-button p-component p-button-secondary p-button-outlined maint-export">
          <i class="pi pi-download" aria-hidden="true" /> {{ $t('subscribers.export') }}
        </a>
      </div>
    </section>

    <!-- Database -->
    <form class="ln-card ln-card-pad maint-card" @submit.prevent="onUpdateDBSettings">
      <h3 class="ln-card-title">{{ $t('maintenance.database.title') }}</h3>
      <p class="ln-small">{{ $t('maintenance.database.vacuumHelp') }}</p>
      <div class="maint-row">
        <label class="maint-toggle" for="maint-vacuum">
          <PvToggleSwitch v-model="dbSettings.vacuum" input-id="maint-vacuum" />
          {{ $t('maintenance.vacuum') }}
        </label>
        <PvInputText
          v-model="dbSettings.vacuum_cron_interval"
          class="ln-mono maint-cron"
          placeholder="0 2 * * *"
          :disabled="!dbSettings.vacuum"
          :aria-label="$t('settings.maintenance.cron')"
          pattern="((\*|[0-9,\-\/]+)\s+){4}(\*|[0-9,\-\/]+)"
        />
        <span class="maint-spacer" />
        <PvButton type="submit" :label="$t('globals.buttons.save')" :loading="busy === 'db'" />
      </div>
    </form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="$t('globals.buttons.delete')"
      :busy="!!busy"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, reactive, ref,
} from 'vue';
import dayjs from 'dayjs';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getMaintenance } from '../api/generated/endpoints/maintenance/maintenance';
import { getSettings as settingsApi } from '../api/generated/endpoints/settings/settings';

const { $utils } = useGlobal();
const { gcSubscribers, gcSubscriptions, gcCampaignAnalytics } = getMaintenance();
const { getSettings, getServerConfig, updateSettingByKey } = settingsApi();
const { t } = useI18n();
const { shortDate } = useTime();

const busy = ref<'' | 'subscribers' | 'subscriptions' | 'analytics' | 'db'>('');
const subscriberType = ref('orphan');
const analyticsType = ref('all');
const analyticsDate = ref(dayjs().subtract(7, 'day').toDate());
const subscriptionDate = ref(dayjs().subtract(7, 'day').toDate());
const exportType = ref('views');
const exportDate = ref(dayjs().subtract(30, 'day').toDate());
const dbSettings = ref({ vacuum: false, vacuum_cron_interval: '0 2 * * *' });

const confirm = reactive({
  open: false, title: '', text: '', run: () => {},
});

const exportURL = computed(() => {
  const since = encodeURIComponent(dayjs(exportDate.value).toISOString());
  return `/api/maintenance/analytics/${exportType.value}/export?since=${since}`;
});

const subscriberTypeOptions = computed(() => [
  { label: t('maintenance.orphans'), value: 'orphan' },
  { label: t('subscribers.status.blocklisted'), value: 'blocklisted' },
]);

const analyticsTypeOptions = computed(() => [
  { label: t('maintenance.viewsAndClicks'), value: 'all' },
  { label: t('dashboard.campaignViews'), value: 'views' },
  { label: t('dashboard.linkClicks'), value: 'clicks' },
]);

const exportTypeOptions = computed(() => [
  { label: t('dashboard.campaignViews'), value: 'views' },
  { label: t('dashboard.linkClicks'), value: 'clicks' },
]);

function ask(title: string, text: string, run: () => void) {
  Object.assign(confirm, {
    open: true, title, text, run,
  });
}

function askSubscribers() {
  const which = subscriberTypeOptions.value.find((o) => o.value === subscriberType.value)?.label || '';
  ask(t('maintenance.confirmSubscribersTitle', { which: which.toLowerCase() }), t('maintenance.confirmSubscribersText'), () => {
    busy.value = 'subscribers';
    gcSubscribers(subscriberType.value as never).then((data) => {
      confirm.open = false;
      $utils.toast(t('globals.messages.deletedCount', { name: t('globals.terms.subscribers', 2), num: (data as unknown as { count: number }).count }));
    }).finally(() => { busy.value = ''; });
  });
}

function askSubscriptions() {
  ask(t('maintenance.confirmSubscriptionsTitle', { date: shortDate(subscriptionDate.value) }), t('maintenance.confirmSubscriptionsText'), () => {
    busy.value = 'subscriptions';
    gcSubscriptions({ before_date: dayjs(subscriptionDate.value).toISOString() } as never).then((data) => {
      confirm.open = false;
      $utils.toast(t('globals.messages.deletedCount', { name: t('globals.terms.subscriptions', 2), num: (data as unknown as { count: number }).count }));
    }).finally(() => { busy.value = ''; });
  });
}

function askAnalytics() {
  ask(t('maintenance.confirmAnalyticsTitle', { date: shortDate(analyticsDate.value) }), t('maintenance.confirmAnalyticsText'), () => {
    busy.value = 'analytics';
    gcCampaignAnalytics(analyticsType.value as never, { before_date: dayjs(analyticsDate.value).toISOString() } as never).then(() => {
      confirm.open = false;
      $utils.toast(t('globals.messages.done'));
    }).finally(() => { busy.value = ''; });
  });
}

function loadDBSettings() {
  getSettings().then((data) => {
    const d = data as unknown as Record<string, typeof dbSettings.value>;
    if (d['maintenance.db'] !== undefined) dbSettings.value = { ...d['maintenance.db'] };
  });
}

async function onUpdateDBSettings() {
  busy.value = 'db';
  try {
    await updateSettingByKey('maintenance.db', dbSettings.value as never);
    await getServerConfig();
    $utils.toast(t('globals.messages.updated', { name: t('maintenance.database.title') }));
  } finally {
    busy.value = '';
  }
}

onMounted(loadDBSettings);
</script>

<style scoped lang="scss">
.maint { display: flex; flex-direction: column; gap: 16px; }
.maint-card { display: flex; flex-direction: column; gap: 6px; .ln-small { margin: 0 0 8px; } }
.maint-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.maint-spacer { flex: 1; }
.maint-toggle { display: inline-flex; align-items: center; gap: 10px; font-size: 14px; cursor: pointer; }
.maint-cron { width: 160px; }
.maint-export { text-decoration: none; display: inline-flex; gap: 8px; &:hover { text-decoration: none; } }
</style>
