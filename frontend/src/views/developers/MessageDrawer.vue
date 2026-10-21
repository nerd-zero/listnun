<template>
  <PvDrawer
    :visible="!!messageId"
    position="right"
    :modal="false"
    :dismissable="false"
    class="msg-drawer"
    :show-close-icon="false"
    @update:visible="(v) => { if (!v) $emit('close'); }"
  >
    <template #container>
      <div class="msg-drawer-inner">
        <header class="msg-head">
          <div class="msg-head-text">
            <status-tag v-if="msg" :status="msg.status" :label="statusLabel(msg)" />
            <PvSkeleton v-else width="90px" height="20px" />
            <h2 class="msg-to">{{ msg?.recipient }}</h2>
            <p class="msg-subject">{{ msg?.subject }}</p>
          </div>
          <button type="button" class="ln-icon-btn" :aria-label="$t('globals.buttons.close')" @click="$emit('close')">
            <i class="pi pi-times" />
          </button>
        </header>

        <div v-if="loading && !msg" class="msg-section">
          <table-skeleton :rows="4" :cols="2" :two-line="false" />
        </div>

        <template v-else-if="msg">
          <dl class="msg-meta">
            <dt>{{ $t('developers.log.messageId') }}</dt>
            <dd class="ln-mono">{{ msg.id }}</dd>
            <dt>{{ $t('developers.log.template') }}</dt>
            <dd>
              {{ msg.templateName || '—' }}
              <span v-if="msg.templateId" class="ln-muted">#{{ msg.templateId }}</span>
            </dd>
            <dt>{{ $t('developers.log.key') }}</dt>
            <dd class="ln-mono">{{ msg.sender || '—' }}</dd>
            <template v-if="msg.subscriberId">
              <dt>{{ $t('globals.terms.subscriber') }}</dt>
              <dd>
                <router-link :to="{ name: 'subscriber', params: { id: msg.subscriberId } }">#{{ msg.subscriberId }}</router-link>
                <span v-if="msg.subscriberStatus === 'blocklisted'" class="ln-muted">&emsp;{{ $t('developers.log.nowBlocklisted') }}</span>
              </dd>
            </template>
          </dl>

          <PvTabs v-model:value="tab" class="msg-tabs">
            <PvTabList>
              <PvTab value="events">{{ $t('developers.log.events') }}</PvTab>
              <PvTab value="payload">{{ $t('developers.log.payload') }}</PvTab>
              <PvTab value="rendered" :disabled="!msg.body">{{ $t('developers.log.rendered') }}</PvTab>
            </PvTabList>
            <PvTabPanels>
              <PvTabPanel value="events">
                <ol class="msg-timeline">
                  <li v-for="(e, i) in msg.events" :key="i" :class="`is-${e.type}`">
                    <span class="msg-timeline-dot" aria-hidden="true" />
                    <div class="msg-timeline-body">
                      <div class="msg-timeline-row">
                        <strong>{{ eventLabel(e) }}</strong>
                        <time class="ln-mono" :datetime="e.createdAt">{{ clock(e.createdAt) }}</time>
                      </div>
                      <p v-if="e.detail && e.type !== 'bounced' && e.type !== 'failed'" class="msg-timeline-detail">{{ eventDetail(e) }}</p>
                      <p v-else-if="e.type === 'bounced'" class="msg-timeline-detail">{{ bounceText }}</p>
                    </div>
                  </li>
                </ol>
                <pre v-if="smtpReply" class="ln-code msg-reply">{{ smtpReply }}</pre>
              </PvTabPanel>
              <PvTabPanel value="payload">
                <pre class="ln-code">{{ payloadText }}</pre>
              </PvTabPanel>
              <PvTabPanel value="rendered">
                <iframe
                  v-if="msg.body"
                  class="msg-rendered"
                  sandbox=""
                  :srcdoc="msg.body"
                  :title="$t('developers.log.rendered')"
                />
              </PvTabPanel>
            </PvTabPanels>
          </PvTabs>
        </template>

        <footer v-if="msg" class="msg-foot">
          <PvButton
            :label="copied ? $t('states.copied') : $t('developers.log.copyId')"
            severity="secondary"
            outlined
            @click="copyId"
          />
          <PvButton
            v-if="$can('tx:send') && msg.payload && Object.keys(msg.payload).length"
            :label="$t('developers.log.resend')"
            :loading="resending"
            @click="resend"
          />
        </footer>
      </div>
    </template>
  </PvDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import { useGlobal } from '../../composables/useGlobal';
import {
  getTxMessage, resendTxMessage, type TxLogEvent, type TxLogMessage,
} from '../../api/developers';
import StatusTag from '../../components/ui/StatusTag.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';

// MessageDrawer: one message's details beside the log, with its timeline,
// the request it came from and the rendered email.
const props = defineProps<{ messageId: string | null }>();
const emit = defineEmits(['close', 'resent']);

const { t } = useI18n();
const { $utils } = useGlobal();

const msg = ref<TxLogMessage | null>(null);
const loading = ref(false);
const resending = ref(false);
const copied = ref(false);
const tab = ref('events');

function statusLabel(m: TxLogMessage): string {
  const s = t(`developers.log.status.${m.status}`);
  return m.status === 'bounced' && m.bounceType ? `${s}\u2003${m.bounceType}` : s;
}

function eventLabel(e: TxLogEvent): string {
  const label = t(`developers.log.event.${e.type}`);
  if (e.type === 'bounced' && msg.value?.bounceType) return `${label} (${msg.value.bounceType})`;
  return label;
}

// The sent event records the messenger; "email" goes out over SMTP.
function eventDetail(e: TxLogEvent): string {
  if (e.type === 'sent' && e.detail === 'email') return 'SMTP';
  return e.detail;
}

function clock(stamp: string): string {
  return dayjs(stamp).format('HH:mm:ss.SSS');
}

const bounceEvent = computed(() => msg.value?.events?.find((e) => e.type === 'bounced'));

const bounceText = computed(() => {
  if (msg.value?.subscriberStatus === 'blocklisted') return t('developers.log.bouncedBlocklisted');
  return t('developers.log.bouncedText');
});

// The mail server's or messenger's own words, shown verbatim.
const smtpReply = computed(() => {
  if (!msg.value) return '';
  if (msg.value.status === 'failed') return msg.value.error;
  return bounceEvent.value?.detail || '';
});

const payloadText = computed(() => JSON.stringify(msg.value?.payload ?? {}, null, 2));

function load(id: string) {
  loading.value = true;
  getTxMessage(id)
    .then((m) => { if (props.messageId === id) msg.value = m; })
    .finally(() => { loading.value = false; });
}

function copyId() {
  if (!msg.value) return;
  navigator.clipboard?.writeText(msg.value.id).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 1500);
  });
}

function resend() {
  if (!msg.value) return;
  resending.value = true;
  resendTxMessage(msg.value.id)
    .then((ids) => {
      $utils.toast(t('developers.log.resentToast', { id: ids[0] || '' }));
      emit('resent', ids[0]);
    })
    .finally(() => { resending.value = false; });
}

watch(() => props.messageId, (id) => {
  msg.value = null;
  tab.value = 'events';
  if (id) load(id);
}, { immediate: true });
</script>

<style scoped lang="scss">
.msg-drawer-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--ln-card);
}
.msg-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--ln-border);
}
.msg-head-text { min-width: 0; display: flex; flex-direction: column; align-items: flex-start; gap: 6px; }
.msg-to { margin: 0; font-size: 15px; font-weight: 600; word-break: break-all; }
.msg-subject { margin: 0; font-size: 13px; color: var(--ln-text-2); }
.msg-meta {
  margin: 0;
  padding: 16px 20px;
  display: grid;
  grid-template-columns: 108px minmax(0, 1fr);
  gap: 8px 12px;
  font-size: 13px;
  border-bottom: 1px solid var(--ln-border);

  dt { color: var(--ln-text-3); }
  dd { margin: 0; overflow-wrap: anywhere; }
}
.msg-tabs { flex: 1; overflow-y: auto; padding: 0 20px 20px; }
.msg-timeline {
  list-style: none;
  margin: 0;
  padding: 0;

  li {
    position: relative;
    display: flex;
    gap: 14px;
    padding-bottom: 16px;

    &:not(:last-child)::before {
      content: '';
      position: absolute;
      left: 4px;
      top: 14px;
      bottom: 0;
      width: 1px;
      background: var(--ln-border);
    }
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
.msg-timeline-row {
  display: flex;
  justify-content: space-between;
  gap: 8px;

  time { font-size: 11px; color: var(--ln-text-3); }
}
.msg-timeline-detail { margin: 2px 0 0; font-size: 13px; color: var(--ln-text-2); overflow-wrap: anywhere; }
.msg-reply { white-space: pre-wrap; }
.msg-rendered {
  width: 100%;
  height: 520px;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: #fff;
}
.msg-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
</style>

<style lang="scss">
.p-drawer.msg-drawer {
  width: 460px;
  max-width: 100vw;
  border-left: 1px solid var(--ln-border);
  box-shadow: var(--ln-shadow-pop);
}
</style>
