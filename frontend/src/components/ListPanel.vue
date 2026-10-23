<template>
  <section class="ln-card list-panel" :aria-labelledby="`list-panel-${list.id}`" data-cy="list-panel">
    <header class="list-panel-head">
      <div class="list-panel-title">
        <h2 :id="`list-panel-${list.id}`">{{ list.name }}</h2>
        <span v-if="list.description" class="list-panel-desc">{{ list.description }}</span>
        <span class="ln-small">{{ $t(`lists.about.${kindKey}`) }}</span>
        <span v-if="list.tags?.length || scrub" class="ln-attrs">
          <status-tag
            v-if="scrub?.activeJobRequestId"
            status="pending"
            :label="$t('settings.scrub.validatingProgress', { count: $utils.niceNumber(scrubProgress || 0) })"
          />
          <span v-else-if="scrub?.lastResult" class="ln-attr ln-attr--filled">
            {{ $t('settings.scrub.lastValidatedCounts', {
              date: shortDate(scrub.lastResult.completedAt),
              valid: $utils.niceNumber(scrub.lastResult.validCount || 0),
              invalid: $utils.niceNumber(scrub.lastResult.invalidCount || 0),
            }) }}
          </span>
          <span v-for="tag in list.tags" :key="tag" class="ln-attr ln-attr--filled">{{ tag }}</span>
        </span>
      </div>
      <div class="list-panel-actions">
        <PvButton v-if="canManage" :label="$t('globals.buttons.edit')" severity="secondary" outlined data-cy="btn-edit" @click="emit('edit')" />
        <PvButton
          v-if="list.type === 'public'"
          :label="$t('lists.getSignupForm')"
          severity="secondary"
          outlined
          @click="$router.push({ name: 'forms' })"
        />
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

    <dl class="list-panel-stats">
      <div>
        <dt>{{ $t('lists.stat.subscribers') }}</dt>
        <dd data-cy="stat-subscribers">{{ $utils.niceNumber(receiving) }}</dd>
        <span class="ln-small">{{ list.optin === 'double' ? $t('lists.stat.receivingDouble') : $t('lists.stat.receivingSingle') }}</span>
      </div>
      <div>
        <dt>{{ $t('lists.stat.joined') }}</dt>
        <dd :class="{ 'is-ok': joined > 0 }" data-cy="stat-joined">{{ activity ? signed(joined) : '–' }}</dd>
        <span class="ln-small">{{ $t('lists.stat.last30') }}</span>
      </div>
      <div>
        <dt>{{ $t('lists.stat.left') }}</dt>
        <dd data-cy="stat-left">{{ activity ? $utils.niceNumber(left) : '–' }}</dd>
        <span class="ln-small">{{ $t('lists.stat.last30') }}</span>
      </div>
      <div v-if="list.optin === 'double'">
        <dt>{{ $t('lists.stat.notConfirmed') }}</dt>
        <dd :class="{ 'is-warn': unconfirmed > 0 }" data-cy="stat-unconfirmed">{{ $utils.niceNumber(unconfirmed) }}</dd>
        <span class="ln-small">{{ $t('lists.stat.waiting') }}</span>
      </div>
      <div v-else>
        <dt>{{ $t('lists.stat.unsubscribed') }}</dt>
        <dd>{{ $utils.niceNumber(list.subscriberStatuses?.unsubscribed || 0) }}</dd>
        <span class="ln-small">{{ $t('lists.stat.allTime') }}</span>
      </div>
    </dl>

    <div class="list-panel-chart">
      <div class="list-panel-chart-head">
        <h3>{{ $t('lists.chart.title') }}</h3>
        <span class="list-panel-legend" aria-hidden="true">
          <span><span class="is-joined" />{{ $t('lists.chart.joined') }}</span>
          <span><span class="is-left" />{{ $t('lists.chart.left') }}</span>
        </span>
      </div>
      <div class="list-panel-bars" role="img" :aria-label="$t('lists.chart.label', { joined, left })">
        <span v-for="d in days" :key="d.day" class="list-panel-bar" :title="`${shortDate(d.day)}: +${d.joined} −${d.left}`">
          <span class="is-joined" :style="{ height: `${(d.joined / peak) * 100}%` }" />
          <span class="is-left" :style="{ height: `${(d.left / peak) * 100}%` }" />
        </span>
      </div>
      <div class="list-panel-axis ln-small" aria-hidden="true">
        <span>{{ days.length ? shortDate(days[0].day) : '' }}</span>
        <span>{{ days.length > 2 ? shortDate(days[Math.floor(days.length / 2)].day) : '' }}</span>
        <span>{{ $t('lists.chart.today') }}</span>
      </div>
    </div>

    <div class="list-panel-foot">
      <div class="list-panel-recent">
        <div class="list-panel-sub-head">
          <h3>{{ $t('lists.recent.title') }}</h3>
          <router-link :to="{ name: 'campaigns' }" class="ln-small">{{ $t('lists.recent.all') }}</router-link>
        </div>
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
          <span class="list-panel-camp-num">
            <span>{{ rate(c.opened, c.sent) }}</span>
            <span class="ln-small">{{ $t('lists.recent.opened') }}</span>
          </span>
          <span class="list-panel-camp-num">
            <span>{{ rate(c.clicked, c.sent) }}</span>
            <span class="ln-small">{{ $t('lists.recent.clicked') }}</span>
          </span>
        </router-link>
      </div>

      <div class="list-panel-form">
        <h3>{{ $t('lists.signupForm') }}</h3>
        <template v-if="list.type === 'public'">
          <span class="ln-small">{{ $t('lists.form.text') }}</span>
          <span class="list-panel-link">
            <span class="ln-mono">{{ formURL }}</span>
            <PvButton :label="copied ? $t('states.copied') : $t('globals.buttons.copy')" size="small" severity="secondary" outlined @click="copyLink" />
          </span>
          <router-link :to="{ name: 'forms' }" class="list-panel-customise">{{ $t('lists.form.customise') }}</router-link>
        </template>
        <span v-else class="ln-small">{{ $t('lists.form.private') }}</span>
      </div>
    </div>
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
const copied = ref(false);

const canManage = computed(() => $can('lists:manage') || $canList(props.list.id, 'list:manage'));
const kindKey = computed(() => `${props.list.type === 'public' ? 'public' : 'private'}${props.list.optin === 'double' ? 'Double' : 'Single'}`);
const unconfirmed = computed(() => props.list.subscriberStatuses?.unconfirmed || 0);
const receiving = computed(() => {
  const s = props.list.subscriberStatuses || {};
  return (s.confirmed || 0) + (props.list.optin === 'double' ? 0 : (s.unconfirmed || 0));
});
const joined = computed(() => activity.value?.joined || 0);
const left = computed(() => activity.value?.left || 0);
const days = computed(() => activity.value?.days || []);
// The tallest day sets the chart's scale; 1 keeps an empty month flat.
const peak = computed(() => Math.max(1, ...days.value.map((d) => d.joined + d.left)));
const formURL = computed(() => `${(serverConfig.value as Loose).root_url}/subscription/form?l=${props.list.uuid}`);

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
  gap: 4px;
  min-width: 0;

  h2 { margin: 0; font-size: 20px; line-height: 26px; font-weight: 600; overflow-wrap: anywhere; }
}
.list-panel-desc { color: var(--ln-text-2); }
.list-panel-actions { display: flex; gap: 8px; flex-wrap: wrap; }

.list-panel-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin: 0;
  border-bottom: 1px solid var(--ln-border);

  > div {
    padding: 16px 22px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;

    & + div { border-left: 1px solid var(--ln-border); }
  }
  dt { font-size: 13px; color: var(--ln-text-2); }
  dd {
    margin: 0;
    font-size: 22px;
    line-height: 28px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;

    &.is-ok { color: var(--ln-success-fg); }
    &.is-warn { color: var(--ln-warn-fg); }
  }

  @media (max-width: 600px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));

    > div { padding: 14px 16px; }
    > div + div { border-left: 0; }
    > div:nth-child(even) { border-left: 1px solid var(--ln-border); }
    > div:nth-child(n + 3) { border-top: 1px solid var(--ln-border); }
  }
}

.list-panel-chart {
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border-bottom: 1px solid var(--ln-border);
}
.list-panel-chart-head, .list-panel-sub-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;

  h3 { margin: 0; font-size: 14px; font-weight: 600; }
  a { color: var(--ln-text-2); }
}
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

.list-panel-foot {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.list-panel-recent {
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-right: 1px solid var(--ln-border);

  @media (max-width: 1100px) { border-right: 0; border-bottom: 1px solid var(--ln-border); }
  p { margin: 6px 0 0; }
}
.list-panel-sub-head { margin-bottom: 6px; }
.list-panel-camp {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 72px 72px;
  gap: 10px;
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
.list-panel-camp-num {
  display: flex;
  flex-direction: column;
  align-items: flex-end;

  > span:first-child { font-variant-numeric: tabular-nums; }
}

.list-panel-form {
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 10px;

  h3 { margin: 0; font-size: 14px; font-weight: 600; }
}
.list-panel-link {
  display: flex;
  gap: 6px;

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
.list-panel-customise { font-size: 13px; font-weight: 500; color: var(--ln-text); }
</style>
