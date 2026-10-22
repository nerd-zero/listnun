<template>
  <PvDrawer
    :visible="!!subscriber"
    position="right"
    :modal="false"
    :dismissable="false"
    :show-close-icon="false"
    class="msg-drawer sub-drawer"
    @update:visible="(v) => { if (!v) $emit('close'); }"
  >
    <template #container>
      <div v-if="subscriber" class="sub-d">
        <header class="sub-d-head">
          <span class="sub-d-avatar" aria-hidden="true">{{ initials }}</span>
          <div class="sub-d-who">
            <h2>{{ subscriber.name || subscriber.email }}</h2>
            <span class="ln-small">{{ subscriber.email }}</span>
            <span class="sub-d-state" :class="`is-${tone}`">
              <span>{{ $t(`subscribers.status.${subscriber.status}`) }}</span><span class="ln-mono sub-d-id">#{{ subscriber.id }}</span>
            </span>
          </div>
          <button type="button" class="ln-icon-btn" :aria-label="$t('globals.buttons.close')" @click="$emit('close')">
            <i class="pi pi-times" />
          </button>
        </header>

        <PvTabs v-model:value="tab" class="sub-d-tabs">
          <PvTabList>
            <PvTab value="activity">{{ $t('subscribers.activity') }}</PvTab>
            <PvTab value="lists">{{ $t('globals.terms.lists') }}</PvTab>
            <PvTab value="attribs">{{ $t('globals.terms.attribs') }}</PvTab>
          </PvTabList>
          <PvTabPanels>
            <PvTabPanel value="activity">
              <div class="sub-d-stats">
                <div><span>{{ $t('subscribers.campaignsViewed') }}</span><strong>{{ views.length }}</strong></div>
                <div><span>{{ $t('dashboard.views') }}</span><strong>{{ totalViews }}</strong></div>
                <div><span>{{ $t('dashboard.clicks') }}</span><strong>{{ totalClicks }}</strong></div>
              </div>

              <table-skeleton v-if="loading" :rows="4" :cols="1" />
              <ol v-else-if="timeline.length" class="msg-timeline sub-d-timeline ln-enter">
                <li v-for="(e, i) in timeline" :key="i" :class="`is-${e.tone}`">
                  <span class="msg-timeline-dot" aria-hidden="true" />
                  <div class="msg-timeline-body">
                    <div class="msg-timeline-row">
                      <strong>{{ e.title }}</strong>
                      <time class="ln-small" :datetime="e.at">{{ when(e.at) }}</time>
                    </div>
                    <p class="msg-timeline-detail">{{ e.text }}</p>
                  </div>
                </li>
              </ol>
              <p v-else class="ln-small">{{ $t('subscribers.noActivity') }}</p>

              <h3 class="ln-label sub-d-section">{{ $t('globals.terms.subscriptions', 2) }}</h3>
              <ul class="sub-d-subs">
                <li v-for="l in subscriber.lists" :key="l.id">
                  <span>{{ l.name }}</span>
                  <span :class="`sub-d-sub is-${subTone(l.subscriptionStatus)}`">
                    {{ $t(`subscribers.status.${l.subscriptionStatus}`) }}
                    <template v-if="l.subscriptionStatus === 'unsubscribed'"> {{ shortDate(l.subscriptionCreatedAt) }}</template>
                  </span>
                </li>
              </ul>
            </PvTabPanel>

            <PvTabPanel value="lists">
              <ul class="sub-d-subs">
                <li v-for="l in subscriber.lists" :key="l.id">
                  <span>
                    <router-link v-if="!l.restricted" :to="`/lists/${l.id}`">{{ l.name }}</router-link>
                    <span v-else>{{ l.name }}</span>
                    <span class="ln-row-sub">{{ $t(`lists.optins.${l.optin}`) }}&emsp;{{ shortDate(l.subscriptionCreatedAt) }}
                      <template v-if="l.subscriptionMeta?.optinIp">&emsp;{{ l.subscriptionMeta.optinIp }}</template>
                    </span>
                  </span>
                  <span :class="`sub-d-sub is-${subTone(l.subscriptionStatus)}`">{{ $t(`subscribers.status.${l.subscriptionStatus}`) }}</span>
                </li>
              </ul>
              <p v-if="!subscriber.lists?.length" class="ln-small">{{ $t('subscribers.noLists') }}</p>
            </PvTabPanel>

            <PvTabPanel value="attribs">
              <pre class="ln-code">{{ attribsText }}</pre>
            </PvTabPanel>
          </PvTabPanels>
        </PvTabs>

        <footer class="msg-foot sub-d-foot">
          <PvButton
            v-if="$can('subscribers:manage') && subscriber.status !== 'blocklisted'"
            :label="$t('subscribers.blocklist')"
            severity="danger"
            outlined
            @click="$emit('blocklist', subscriber)"
          />
          <span class="ln-dialog-form-spacer" />
          <a :href="`/api/subscribers/${subscriber.id}/export`" class="p-button p-component p-button-secondary p-button-outlined sub-d-export" data-cy="btn-download">
            {{ $t('subscribers.exportData') }}
          </a>
          <PvButton v-if="$can('subscribers:manage')" :label="$t('globals.buttons.edit')" data-cy="btn-edit" @click="$emit('edit', subscriber)" />
        </footer>
      </div>
    </template>
  </PvDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';
import { listTxMessages, type TxLogMessage } from '../api/developers';
import { statusTone } from '../components/ui/status';
import TableSkeleton from '../components/ui/TableSkeleton.vue';

// Subscribers and their activity are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface TimelineEvent {
  at: string;
  title: string;
  text: string;
  tone: 'delivered' | 'bounced' | 'neutral' | 'failed';
}

// SubscriberDrawer: one subscriber beside the list, with an activity
// timeline built from campaign views and clicks, transactional email and
// subscription changes.
const props = defineProps<{ subscriber: Loose | null }>();
defineEmits(['close', 'edit', 'blocklist']);

const { t } = useI18n();
const { $can } = useGlobal();
const { shortDate } = useTime();
const { getSubscriberActivity } = subscribersApi();

const tab = ref('activity');
const loading = ref(false);
const views = ref<Loose[]>([]);
const clicks = ref<Loose[]>([]);
const txs = ref<TxLogMessage[]>([]);

const initials = computed(() => {
  const s = props.subscriber;
  const src = (s?.name || s?.email || '').trim();
  const parts = src.split(/[\s.@_-]+/).filter(Boolean);
  return (parts.length > 1 ? parts[0][0] + parts[1][0] : src.slice(0, 2)).toUpperCase();
});
const tone = computed(() => statusTone(props.subscriber?.status === 'enabled' ? 'active' : props.subscriber?.status || ''));
const totalViews = computed(() => views.value.reduce((n, v) => n + (v.viewCount || 0), 0));
const totalClicks = computed(() => clicks.value.reduce((n, c) => n + (c.clickCount || 0), 0));
const attribsText = computed(() => JSON.stringify(props.subscriber?.attribs ?? {}, null, 2));

function subTone(s: string): string {
  if (s === 'confirmed') return 'ok';
  if (s === 'unsubscribed') return 'muted';
  return 'warn';
}

function when(stamp: string): string {
  const d = dayjs(stamp);
  if (d.isSame(dayjs(), 'day')) return t('subscribers.todayAt', { time: d.format('HH:mm') });
  if (d.isSame(dayjs().subtract(1, 'day'), 'day')) return t('time.yesterday');
  return shortDate(stamp);
}

const timeline = computed<TimelineEvent[]>(() => {
  const out: TimelineEvent[] = [];
  clicks.value.forEach((c) => out.push({
    at: c.lastClickedAt,
    title: t('subscribers.evClicked'),
    text: [c.campaignName, c.url].filter(Boolean).join('\u2003'),
    tone: 'delivered',
  }));
  views.value.forEach((v) => out.push({
    at: v.lastViewedAt, title: t('subscribers.evViewed'), text: v.name || v.subject, tone: 'neutral',
  }));
  txs.value.forEach((m) => out.push({
    at: m.createdAt,
    title: m.status === 'bounced' ? t('subscribers.evTxBounced') : t('subscribers.evTx'),
    text: `${m.templateName}\u2003${m.id}`,
    tone: m.status === 'bounced' ? 'bounced' : 'neutral',
  }));
  (props.subscriber?.lists || []).forEach((l: Loose) => {
    if (l.subscriptionStatus === 'unsubscribed') {
      out.push({
        at: l.subscriptionCreatedAt, title: t('subscribers.evUnsubscribed'), text: l.name, tone: 'bounced',
      });
    } else {
      out.push({
        at: l.subscriptionCreatedAt,
        title: t('subscribers.evSubscribed'),
        text: l.subscriptionStatus === 'confirmed' && l.optin === 'double' ? t('subscribers.evConfirmed', { name: l.name }) : l.name,
        tone: 'neutral',
      });
    }
  });
  return out.filter((e) => e.at).sort((a, b) => dayjs(b.at).valueOf() - dayjs(a.at).valueOf()).slice(0, 30);
});

function load(s: Loose) {
  loading.value = true;
  views.value = [];
  clicks.value = [];
  txs.value = [];
  const reqs: Promise<unknown>[] = [
    getSubscriberActivity(s.id).then((data) => {
      const a = data as unknown as { campaignViews?: Loose[]; linkClicks?: Loose[] };
      views.value = a.campaignViews || [];
      clicks.value = a.linkClicks || [];
    }).catch(() => {}),
  ];
  if ($can('tx:logs')) {
    reqs.push(listTxMessages({ query: s.email, perPage: 10 }).then((p) => {
      txs.value = p.results.filter((m) => m.recipient.toLowerCase() === s.email.toLowerCase());
    }).catch(() => {}));
  }
  Promise.all(reqs).finally(() => { loading.value = false; });
}

watch(() => props.subscriber?.id, () => {
  tab.value = 'activity';
  if (props.subscriber) load(props.subscriber);
}, { immediate: true });
</script>

<style scoped lang="scss">
.sub-d { height: 100%; display: flex; flex-direction: column; background: var(--ln-card); }
.sub-d-head {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 16px 20px;
  border-bottom: 1px solid var(--ln-border);
}
.sub-d-avatar {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 99px;
  background: var(--ln-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  .app-dark & { background: var(--ln-hover); }
}
.sub-d-who {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;

  h2 { margin: 0; font-size: 15px; font-weight: 600; overflow-wrap: anywhere; }
  .ln-small { overflow-wrap: anywhere; }
}
.sub-d-state {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--ln-text-2);

  &.is-success { color: var(--ln-success-fg); }
  &.is-danger { color: var(--ln-danger-fg); }
  .sub-d-id { color: var(--ln-text-3); font-size: 12px; }
}
.sub-d-tabs {
  flex: 1;
  overflow-y: auto;
  padding: 0 20px 20px;
}
.sub-d-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 16px;

  div {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px 12px;
    border-radius: var(--ln-radius);
    background: var(--ln-muted);
    font-size: 12px;
    color: var(--ln-text-2);
    .app-dark & { background: var(--ln-hover); }
  }
  strong { font-size: 16px; color: var(--ln-text); }
}
.msg-timeline {
  list-style: none;
  margin: 0;
  padding: 0;

  li { position: relative; display: flex; gap: 14px; padding-bottom: 16px; }
  li:not(:last-child)::before {
    content: '';
    position: absolute;
    left: 4px;
    top: 14px;
    bottom: 0;
    width: 1px;
    background: var(--ln-border);
  }
}
.msg-timeline-dot {
  flex: none;
  width: 9px;
  height: 9px;
  margin-top: 5px;
  border-radius: 99px;
  background: var(--ln-text-3);

  .is-delivered & { background: var(--ln-success); }
  .is-bounced & { background: var(--ln-warn); }
  .is-failed & { background: var(--ln-danger); }
}
.msg-timeline-body { flex: 1; min-width: 0; }
.msg-timeline-row { display: flex; justify-content: space-between; gap: 8px; }
.msg-timeline-detail { margin: 2px 0 0; font-size: 13px; color: var(--ln-text-2); overflow-wrap: anywhere; }
.sub-d-section { display: block; margin: 8px 0; }
.sub-d-subs {
  list-style: none;
  margin: 0;
  padding: 0;

  li { display: flex; justify-content: space-between; gap: 12px; padding: 6px 0; font-size: 14px; }
}
.sub-d-sub {
  font-size: 13px;
  white-space: nowrap;

  &.is-ok { color: var(--ln-success-fg); }
  &.is-warn { color: var(--ln-warn-fg); }
  &.is-muted { color: var(--ln-text-3); }
}
.msg-foot {
  display: flex;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
.sub-d-export { text-decoration: none; &:hover { text-decoration: none; } }
</style>
