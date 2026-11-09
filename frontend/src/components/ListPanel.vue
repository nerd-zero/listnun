<template>
  <section class="ln-card list-panel" :aria-labelledby="`list-panel-${list.id}`" data-cy="list-panel">
    <header class="list-panel-head">
      <div class="list-panel-title">
        <h2 :id="`list-panel-${list.id}`">{{ list.name }}</h2>
        <span class="ln-small">{{ list.description || $t(`lists.about.${kindKey}`) }}</span>
        <!-- Shown only while a check runs: a finished one is history. -->
        <status-tag
          v-if="scrub?.activeJobRequestId"
          status="pending"
          :label="$t('settings.scrub.validatingProgress', { count: $utils.niceNumber(scrubProgress || 0) })"
        />
      </div>
      <div class="list-panel-actions">
        <PvButton v-if="canManage" :label="$t('globals.buttons.edit')" severity="secondary" outlined data-cy="btn-edit" @click="emit('edit')" />
        <PvButton
          v-if="$can('campaigns:manage')"
          icon="pi pi-send"
          :label="$t('lists.sendACampaign')"
          data-cy="btn-campaign"
          @click="$router.push(`/campaigns/new?list_id=${list.id}`)"
        />
        <PvButton
          icon="pi pi-ellipsis-h"
          severity="secondary"
          text
          :aria-label="$t('lists.moreActions')"
          aria-haspopup="true"
          data-cy="btn-row-menu"
          @click="(e) => emit('menu', e)"
        />
      </div>
    </header>

    <!-- At most three figures: who gets campaigns, the month's change, and
         people waiting to confirm only when there are some. -->
    <div class="list-panel-stats">
      <component
        :is="canSeeSubs ? 'router-link' : 'div'"
        :to="subsLink(list.optin === 'double' ? 'confirmed' : undefined)"
        class="list-panel-stat"
        data-cy="stat-subscribers-link"
      >
        <span class="list-panel-stat-label">{{ $t('lists.stat.subscribers') }}<i v-if="canSeeSubs" class="pi pi-arrow-right" aria-hidden="true" /></span>
        <span class="list-panel-stat-value" data-cy="stat-subscribers">{{ $utils.niceNumber(receiving) }}</span>
      </component>
      <div class="list-panel-stat">
        <span class="list-panel-stat-label">{{ $t('lists.stat.last30Title') }}</span>
        <span class="list-panel-stat-value" :class="{ 'is-down': net < 0 }" data-cy="stat-change">{{ activity ? signed(net) : '–' }}</span>
        <span v-if="activity" class="ln-small" data-cy="stat-joined-left">{{ $t('lists.stat.joinedLeft', { joined: $utils.niceNumber(joined), left: $utils.niceNumber(left) }) }}</span>
      </div>
      <component
        :is="canSeeSubs ? 'router-link' : 'div'"
        v-if="unconfirmed > 0 && list.optin === 'double'"
        :to="subsLink('unconfirmed')"
        class="list-panel-stat"
      >
        <span class="list-panel-stat-label">{{ $t('lists.stat.notConfirmed') }}<i v-if="canSeeSubs" class="pi pi-arrow-right" aria-hidden="true" /></span>
        <span class="list-panel-stat-value is-warn" data-cy="stat-unconfirmed">{{ $utils.niceNumber(unconfirmed) }}</span>
      </component>
    </div>

    <!-- Details one click away rather than stacked down the panel. -->
    <PvTabs v-model:value="tab" class="list-panel-tabs">
      <PvTabList>
        <PvTab value="growth">{{ $t('lists.tab.growth') }}</PvTab>
        <PvTab value="campaigns" data-cy="tab-campaigns">{{ $t('lists.tab.campaigns') }}</PvTab>
        <PvTab v-if="list.type === 'public'" value="form" data-cy="tab-form">{{ $t('lists.signupForm') }}</PvTab>
      </PvTabList>
      <PvTabPanels>
        <PvTabPanel value="growth">
          <div class="list-panel-chart">
            <span class="list-panel-legend" aria-hidden="true">
              <span><span class="is-joined" />{{ $t('lists.chart.joined') }}</span>
              <span><span class="is-left" />{{ $t('lists.chart.left') }}</span>
            </span>
            <div class="list-panel-bars" role="img" :aria-label="$t('lists.chart.label', { joined, left })">
              <span v-for="d in days" :key="d.day" class="list-panel-bar" :title="`${shortDate(d.day)}: +${d.joined} −${d.left}`">
                <span class="is-joined" :style="{ height: `${(d.joined / peak) * 100}%` }" />
                <span class="is-left" :style="{ height: `${(d.left / peak) * 100}%` }" />
              </span>
            </div>
            <div class="list-panel-axis ln-small" aria-hidden="true">
              <span>{{ days.length ? shortDate(days[0].day) : '' }}</span>
              <span>{{ $t('lists.chart.today') }}</span>
            </div>
          </div>
        </PvTabPanel>

        <PvTabPanel value="campaigns">
          <div class="list-panel-recent">
            <p v-if="activity && !activity.campaigns.length" class="ln-small">{{ $t('lists.recent.none') }}</p>
            <router-link
              v-for="c in activity?.campaigns || []"
              :key="c.id"
              :to="{ name: 'campaign', params: { id: c.id } }"
              class="list-panel-camp"
            >
              <span class="list-panel-camp-name">
                <strong>{{ c.name }}</strong>
                <span class="ln-small">{{ when(c) }}</span>
              </span>
              <span v-if="c.sent" class="list-panel-camp-num">{{ $t('lists.recent.openedRate', { n: rate(c.opened, c.sent) }) }}</span>
            </router-link>
            <router-link :to="{ name: 'campaigns' }" class="list-panel-more">{{ $t('lists.recent.all') }}</router-link>
          </div>
        </PvTabPanel>

        <PvTabPanel v-if="list.type === 'public'" value="form">
          <div class="list-panel-form">
            <span class="ln-small">{{ $t('lists.form.text') }}</span>
            <span class="list-panel-link">
              <span class="ln-mono">{{ formURL }}</span>
              <PvButton :label="copied ? $t('states.copied') : $t('globals.buttons.copy')" size="small" severity="secondary" outlined @click="copyLink" />
            </span>
            <router-link :to="{ name: 'forms' }" class="list-panel-more">{{ $t('lists.form.customise') }}</router-link>
          </div>
        </PvTabPanel>
      </PvTabPanels>
    </PvTabs>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import StatusTag from './ui/StatusTag.vue';
import { getLists as listsApi } from '../api/generated/endpoints/lists/lists';

// The list and its activity are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface Day { day: string; joined: number; left: number }
interface Camp { id: number; name: string; status: string; startedAt?: string; sendAt?: string; sent: number; opened: number; clicked: number }
interface Activity { days: Day[]; joined: number; left: number; campaigns: Camp[] }

// How long "Copied" shows on the copy button.
const COPIED_MS = 1500;

const props = defineProps<{
  list: Loose;
  scrub?: Loose;
  scrubProgress?: number;
}>();
const emit = defineEmits<{ edit: []; menu: [e: Event] }>();

const { t } = useI18n();
const { $utils, $can, $canList } = useGlobal();
const { shortDate } = useTime();
const { serverConfig } = storeToRefs(useMainStore());
const { getListActivity } = listsApi();

const activity = ref<Activity | null>(null);
const tab = ref('growth');
const copied = ref(false);

const canSeeSubs = computed(() => $can('subscribers:get_all', 'subscribers:get'));
const canManage = computed(() => $can('lists:manage') || $canList(props.list.id, 'list:manage'));
const kindKey = computed(() => `${props.list.type === 'public' ? 'public' : 'private'}${props.list.optin === 'double' ? 'Double' : 'Single'}`);
const unconfirmed = computed(() => props.list.subscriberStatuses?.unconfirmed || 0);
const receiving = computed(() => {
  const s = props.list.subscriberStatuses || {};
  return (s.confirmed || 0) + (props.list.optin === 'double' ? 0 : (s.unconfirmed || 0));
});
const joined = computed(() => activity.value?.joined || 0);
const left = computed(() => activity.value?.left || 0);
const net = computed(() => joined.value - left.value);
const days = computed(() => activity.value?.days || []);
// The tallest day sets the chart's scale; 1 keeps an empty month flat.
const peak = computed(() => Math.max(1, ...days.value.map((d) => d.joined + d.left)));
const formURL = computed(() => `${(serverConfig.value as Loose).root_url}/subscription/form?l=${props.list.uuid}`);

// subsLink is this list's subscribers, optionally only those in a
// subscription status. The Subscribers figure links to the confirmed on
// a confirm-by-email list, so the page shows the same number.
function subsLink(status?: string) {
  return { name: 'subscribers_list', params: { listID: props.list.id }, query: status ? { subscription_status: status } : {} };
}

function signed(n: number): string {
  return n > 0 ? `+${$utils.niceNumber(n)}` : String($utils.niceNumber(n));
}

function rate(n: number, sent: number): string {
  if (!sent) return '–';
  return `${((n / sent) * 100).toFixed(1)}%`;
}

function when(c: Camp): string {
  switch (c.status) {
    case 'scheduled': return t('lists.recent.scheduled', { date: shortDate(c.sendAt) });
    case 'running': return t('lists.recent.sending');
    case 'paused': return t('lists.recent.paused');
    default: return t('lists.recent.sent', { date: shortDate(c.startedAt) });
  }
}

function copyLink() {
  navigator.clipboard.writeText(formURL.value).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, COPIED_MS);
  });
}

function fetchActivity() {
  const { id } = props.list;
  activity.value = null;
  getListActivity(id).then((d) => {
    // A slow response for a list the user has since left is dropped.
    if (props.list.id === id) activity.value = d as unknown as Activity;
  }).catch(() => {});
}

watch(() => props.list.id, fetchActivity, { immediate: true });
// A private list has no signup form tab.
watch(() => props.list.type, (type) => { if (type !== 'public' && tab.value === 'form') tab.value = 'growth'; });
defineExpose({ refresh: fetchActivity });
</script>

<style scoped lang="scss">
.list-panel { display: flex; flex-direction: column; min-width: 0; }

.list-panel-head {
  padding: 20px 22px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
  border-bottom: 1px solid var(--ln-border);
}
.list-panel-title {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  min-width: 0;

  h2 { margin: 0; font-size: 20px; line-height: 26px; font-weight: 600; overflow-wrap: anywhere; }
}
.list-panel-actions { display: flex; gap: 8px; flex-wrap: wrap; }

.list-panel-stats {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(0, 1fr);
  border-bottom: 1px solid var(--ln-border);

  > * {
    padding: 16px 22px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;

    & + * { border-left: 1px solid var(--ln-border); }
  }
  .list-panel-stat-label { font-size: 13px; color: var(--ln-text-2); display: flex; align-items: center; gap: 6px; i { font-size: 10px; } }
  .list-panel-stat-value {
    font-size: 22px;
    line-height: 28px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;

    // Colour only where something needs attention.
    &.is-down { color: var(--ln-danger-fg); }
    &.is-warn { color: var(--ln-warn-fg); }
  }

  @media (max-width: 600px) { > * { padding: 14px 16px; } }
}

.list-panel-stat {
  color: var(--ln-text);
  text-decoration: none;

  &[href]:hover { background: var(--ln-hover); text-decoration: none; }
  &[href]:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: -2px; }
}

.list-panel-tabs :deep(.p-tablist) { padding: 0 12px; }
.list-panel-tabs :deep(.p-tabpanels) { padding: 0; }

.list-panel-chart { padding: 16px 22px 18px; display: flex; flex-direction: column; gap: 10px; }
.list-panel-legend {
  display: flex;
  gap: 14px;
  font-size: 12px;
  color: var(--ln-text-2);

  > span { display: inline-flex; gap: 6px; align-items: center; }
  .is-joined, .is-left { width: 10px; height: 10px; border-radius: 2px; }
}
.is-joined { background: var(--ln-success); }
.is-left { background: var(--ln-text-4); }
.list-panel-bars {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 96px;
  border-bottom: 1px solid var(--ln-border);

  @media (max-width: 600px) { gap: 2px; }
}
.list-panel-bar {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  gap: 1px;

  .is-joined { border-radius: 2px 2px 0 0; }
}
.list-panel-axis { display: flex; justify-content: space-between; }

.list-panel-recent { padding: 12px 22px 18px; display: flex; flex-direction: column; gap: 2px; p { margin: 6px 0; } }
.list-panel-camp {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 6px;
  margin: 0 -6px;
  border-radius: 6px;
  color: var(--ln-text);
  font-size: 13px;
  text-decoration: none;

  &:hover { background: var(--ln-hover); text-decoration: none; }
}
.list-panel-camp-name {
  display: flex;
  flex-direction: column;
  min-width: 0;

  strong { font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
}
.list-panel-camp-num { flex: none; font-size: 13px; color: var(--ln-text-2); font-variant-numeric: tabular-nums; }
.list-panel-more { margin-top: 8px; font-size: 13px; font-weight: 500; color: var(--ln-text); text-decoration: underline; }

.list-panel-form { padding: 16px 22px 18px; display: flex; flex-direction: column; align-items: flex-start; gap: 10px; }
.list-panel-link {
  display: flex;
  gap: 6px;
  width: 100%;

  .ln-mono {
    flex: 1;
    min-width: 0;
    height: 34px;
    box-sizing: border-box;
    padding: 0 10px;
    border-radius: 6px;
    border: 1px solid var(--ln-border-strong);
    background: var(--ln-muted);
    font-size: 12px;
    display: flex;
    align-items: center;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
}
</style>
