<template>
  <div class="dash">
    <!-- Header -->
    <div class="dash-header">
      <div>
        <h1 class="dash-greeting">{{ greeting }}, {{ profile.username }}</h1>
        <p class="dash-date">{{ $utils.niceDate(new Date()) }}</p>
      </div>
    </div>

    <!-- Stat cards -->
    <div class="dash-stats">
      <div class="stat-card" data-cy="lists">
        <div class="stat-icon stat-icon--blue">
          <i class="pi pi-list" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isCountsLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(counts.lists.total) }}</span>
          </div>
          <div class="stat-label">{{ $t('globals.terms.list', counts.lists.total) }}</div>
          <div class="stat-breakdown">
            <span>{{ counts.lists.public }} {{ $t('lists.types.public') }}</span>
            <span>{{ counts.lists.private }} {{ $t('lists.types.private') }}</span>
            <span>{{ counts.lists.optinSingle }} Single opt-in</span>
            <span>{{ counts.lists.optinDouble }} Double opt-in</span>
          </div>
        </div>
      </div>

      <div class="stat-card" data-cy="subscribers">
        <div class="stat-icon stat-icon--green">
          <i class="pi pi-users" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isCountsLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(counts.subscribers.total) }}</span>
          </div>
          <div class="stat-label">{{ $t('globals.terms.subscriber', counts.subscribers.total) }}</div>
          <div class="stat-breakdown">
            <span>{{ counts.subscribers.blocklisted || 0 }} {{ $t('subscribers.status.blocklisted') }}</span>
            <span>{{ counts.subscribers.orphans || 0 }} {{ $t('dashboard.orphanSubs') }}</span>
          </div>
        </div>
      </div>

      <div class="stat-card" data-cy="campaigns">
        <div class="stat-icon stat-icon--purple">
          <i class="pi pi-send" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isCountsLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(counts.campaigns.total) }}</span>
          </div>
          <div class="stat-label">{{ $t('globals.terms.campaign', counts.campaigns.total) }}</div>
          <div class="stat-breakdown">
            <span v-for="(num, status) in counts.campaigns.byStatus" :key="status">
              {{ num }} {{ $t(`campaigns.status.${status}`) }}
            </span>
          </div>
        </div>
      </div>

      <div class="stat-card" data-cy="messages">
        <div class="stat-icon stat-icon--orange">
          <i class="pi pi-envelope" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isCountsLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(counts.messages) }}</span>
          </div>
          <div class="stat-label">{{ $t('dashboard.messagesSent') }}</div>
        </div>
      </div>

      <div v-if="(serverConfig as any).scrub_enabled" class="stat-card" data-cy="scrub-risky">
        <div class="stat-icon stat-icon--red">
          <i class="pi pi-shield" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isCountsLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(counts.scrub?.risky_subscribers || 0) }}</span>
          </div>
          <div class="stat-label">{{ $t('dashboard.riskySubscribers') }}</div>
        </div>
      </div>

      <div v-if="(serverConfig as any).scrub_enabled" class="stat-card" data-cy="auto-paused">
        <div class="stat-icon stat-icon--red">
          <i class="pi pi-exclamation-triangle" />
        </div>
        <div class="stat-body">
          <div class="stat-number">
            <PvProgressSpinner v-if="isAutoPausedLoading" style="width:1.5rem;height:1.5rem" stroke-width="4" />
            <span v-else>{{ $utils.niceNumber(autoPaused.length) }}</span>
          </div>
          <div class="stat-label">{{ $t('dashboard.autoPausedCampaigns') }}</div>
          <div class="stat-breakdown">
            <span v-if="!isAutoPausedLoading && autoPaused.length === 0">
              {{ $t('dashboard.noAutoPausedCampaigns') }}
            </span>
            <span v-for="c in autoPaused" :key="c.id">
              {{ c.name }} &mdash; {{ $t(`campaigns.pauseReason.${c.pause_reason}`) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Charts -->
    <div class="dash-charts">
      <div class="chart-card">
        <div class="chart-header">
          <span class="chart-title">{{ $t('dashboard.campaignViews') }}</span>
        </div>
        <div class="chart-body">
          <div v-if="isChartsLoading" class="chart-loading">
            <PvProgressSpinner style="width:2rem;height:2rem" stroke-width="3" />
          </div>
          <chart v-else-if="campaignViews" type="line" :data="campaignViews" />
          <div v-else class="chart-empty">
            <i class="pi pi-chart-line chart-empty-icon" />
            <span>No campaign data yet</span>
          </div>
        </div>
      </div>

      <div class="chart-card">
        <div class="chart-header">
          <span class="chart-title">{{ $t('dashboard.linkClicks') }}</span>
        </div>
        <div class="chart-body">
          <div v-if="isChartsLoading" class="chart-loading">
            <PvProgressSpinner style="width:2rem;height:2rem" stroke-width="3" />
          </div>
          <chart v-else-if="campaignClicks" type="line" :data="campaignClicks" />
          <div v-else class="chart-empty">
            <i class="pi pi-chart-line chart-empty-icon" />
            <span>No campaign data yet</span>
          </div>
        </div>
      </div>
    </div>

    <p v-if="settings['app.cache_slow_queries']" class="dash-cache-note">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferrer">
        <i class="pi pi-external-link" /> {{ $t('globals.buttons.learnMore') }}
      </a>
    </p>
  </div>
</template>

<script setup lang="ts">
import {
  ref, computed, watch, onMounted,
} from 'vue';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../store';
import { colors } from '../constants';
import Chart from '../components/Chart.vue';
import { getDashboard } from '../api/generated/endpoints/dashboard/dashboard';
import { getCampaigns } from '../api/generated/endpoints/campaigns/campaigns';

const { getDashboardCounts, getDashboardCharts } = getDashboard();
const { getAutoPausedCampaigns } = getCampaigns();
const {
  refreshTick, settings, profile, serverConfig,
} = storeToRefs(useMainStore());

const isChartsLoading = ref(true);
const isCountsLoading = ref(true);
const isAutoPausedLoading = ref(true);
const campaignViews = ref<any>(null);
const campaignClicks = ref<any>(null);
const autoPaused = ref<any[]>([]);
const counts = ref<any>({
  lists: {},
  subscribers: {},
  campaigns: {},
  messages: 0,
});

const greeting = computed(() => {
  const h = new Date().getHours();
  if (h < 12) return 'Good morning';
  if (h < 18) return 'Good afternoon';
  return 'Good evening';
});

function makeChart(data: any) {
  if (!data || data.length === 0) return null;
  return {
    labels: data.map((d: any) => dayjs(d.date).format('DD MMM')),
    datasets: [{
      data: data.map((d: any) => d.count),
      borderColor: colors.primary,
      borderWidth: 2,
      pointHoverBorderWidth: 5,
      pointBorderWidth: 0.5,
      fill: true,
      backgroundColor: 'rgba(0,180,216,0.07)',
    }],
  };
}

function fetchData() {
  isCountsLoading.value = true;
  isChartsLoading.value = true;

  getDashboardCounts().then((data: any) => {
    counts.value = data;
    isCountsLoading.value = false;
  });

  getDashboardCharts().then((data: any) => {
    isChartsLoading.value = false;
    campaignViews.value = makeChart(data.campaignViews);
    campaignClicks.value = makeChart(data.linkClicks);
  });

  if ((serverConfig.value as any).scrub_enabled) {
    isAutoPausedLoading.value = true;
    getAutoPausedCampaigns().then((data: any) => {
      autoPaused.value = data || [];
      isAutoPausedLoading.value = false;
    }).catch(() => { isAutoPausedLoading.value = false; });
  }
}

watch(() => refreshTick.value, () => { fetchData(); });

onMounted(() => {
  fetchData();
});
</script>

<style scoped lang="scss">
.dash {
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
}

.dash-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
}

.dash-greeting {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--lm-text);
  margin: 0 0 0.2rem;
  line-height: 1.2;
}

.dash-date {
  font-size: 0.875rem;
  color: var(--lm-text-muted);
  margin: 0;
}

// Stat cards
.dash-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.25rem;

  @media (max-width: 1100px) { grid-template-columns: repeat(2, 1fr); }
  @media (max-width: 600px)  { grid-template-columns: 1fr; }
}

.stat-card {
  background: var(--lm-surface);
  border: 1px solid var(--lm-border);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  gap: 1rem;
  align-items: flex-start;
  transition: box-shadow 0.18s, transform 0.18s;

  &:hover {
    box-shadow: 0 4px 16px rgba(0,0,0,0.07);
    transform: translateY(-1px);
  }
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  i { font-size: 1.15rem; }
  &--blue   { background: #e0f7fc; color: #0077b6; }
  &--green  { background: var(--lm-success-bg); color: #16a34a; }
  &--purple { background: #f5f3ff; color: #7c3aed; }
  &--orange { background: #fff7ed; color: #ea580c; }
  &--red    { background: #fee2e2; color: #b91c1c; }
}

.stat-body {
  min-width: 0;
  flex: 1;
}

.stat-number {
  font-size: 2rem;
  font-weight: 700;
  color: var(--lm-text);
  line-height: 1;
  margin-bottom: 0.25rem;
}

.stat-label {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--lm-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.75rem;
}

.stat-breakdown {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;

  span {
    font-size: 0.8rem;
    color: var(--lm-text-subtle);
  }
}

// Charts
.dash-charts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.25rem;

  @media (max-width: 768px) { grid-template-columns: 1fr; }
}

.chart-card {
  background: var(--lm-surface);
  border: 1px solid var(--lm-border);
  border-radius: 12px;
  overflow: hidden;
}

.chart-header {
  padding: 1.25rem 1.5rem 0;
}

.chart-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--lm-text);
}

.chart-body {
  padding: 1rem 1.5rem 1.5rem;
  min-height: 220px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chart-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
}

.chart-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  color: var(--lm-text-subtle);
  font-size: 0.875rem;
}

.chart-empty-icon {
  font-size: 1.75rem;
  opacity: 0.4;
}

.dash-cache-note {
  font-size: 0.8rem;
  color: var(--lm-text-subtle);
  margin: 0;

  a { color: var(--lm-text-subtle); text-decoration: underline; }
}
</style>
