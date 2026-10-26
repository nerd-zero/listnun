<template>
  <PvDrawer
    :visible="!!deliveryId"
    position="right"
    :modal="false"
    :dismissable="false"
    :show-close-icon="false"
    class="msg-drawer"
    @update:visible="(v) => { if (!v) $emit('close'); }"
  >
    <template #container>
      <div class="whd">
        <header class="whd-head">
          <div class="whd-head-text">
            <status-tag v-if="d" :status="tone" :label="statusLabel" />
            <PvSkeleton v-else width="140px" height="20px" />
            <h2 class="ln-mono whd-event">{{ d?.event }}</h2>
            <span v-if="d" class="ln-mono ln-small">→ {{ shortUrl(d.endpointUrl) }}</span>
          </div>
          <button type="button" class="ln-icon-btn" :aria-label="$t('globals.buttons.close')" @click="$emit('close')">
            <i class="pi pi-times" />
          </button>
        </header>

        <div v-if="d" class="whd-body">
          <section>
            <h3 class="ln-label">{{ $t('developers.webhooks.attempts') }}</h3>
            <ol class="whd-attempts">
              <li v-for="a in d.history" :key="`${a.attempt}-${a.createdAt}`">
                <span class="ln-mono">{{ a.attempt }}</span>
                <span class="ln-mono" :class="codeClass(a.statusCode)" :title="a.error || undefined">{{ a.statusCode || (a.error ? $t('developers.webhooks.noResponse') : '—') }}</span>
                <span class="ln-mono">{{ duration(a.durationMs) }}</span>
                <span class="ln-small">{{ clock(a.createdAt) }}</span>
              </li>
              <li v-if="d.status === 'retrying' || d.status === 'pending'" class="is-next">
                <span class="ln-mono">{{ d.attempts + 1 }}</span>
                <span>—</span>
                <span>—</span>
                <span class="ln-small">{{ $t('developers.webhooks.scheduled', { at: d.nextAttemptAt ? clock(d.nextAttemptAt) : '' }) }}</span>
              </li>
            </ol>
            <p class="ln-help">{{ $t('developers.webhooks.backoff') }}</p>
          </section>

          <section>
            <h3 class="ln-label">{{ $t('developers.webhooks.request') }}</h3>
            <pre class="ln-code">{{ requestText }}</pre>
          </section>

          <section v-if="last">
            <h3 class="ln-label">{{ $t('developers.webhooks.response') }}</h3>
            <pre class="ln-code"><span :class="codeClass(last.statusCode)">{{ last.statusCode ? `HTTP ${last.statusCode}` : last.error }}</span>{{ last.response ? `\n${last.response}` : '' }}</pre>
          </section>
        </div>

        <footer v-if="d" class="msg-foot">
          <PvButton :class="{ 'ln-copied': copied }" :label="copied ? $t('states.copied') : $t('developers.webhooks.copyPayload')" severity="secondary" outlined @click="copyPayload" />
          <PvButton v-if="$can('webhooks:manage')" :label="$t('developers.webhooks.resendNow')" :loading="retrying" @click="retry" />
        </footer>
      </div>
    </template>
  </PvDrawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import {
  getWebhookDelivery, retryWebhookDelivery, WEBHOOK_MAX_ATTEMPTS, type WebhookDelivery,
} from '../../api/developers';
import StatusTag from '../../components/ui/StatusTag.vue';

// DeliveryDrawer: one webhook delivery's attempts, request and the
// endpoint's reply.
const props = defineProps<{ deliveryId: string | null }>();
const emit = defineEmits(['close', 'retried']);

const { t } = useI18n();

const d = ref<WebhookDelivery | null>(null);
const retrying = ref(false);
const copied = ref(false);

const last = computed(() => (d.value?.history?.length ? d.value.history[d.value.history.length - 1] : null));

const tone = computed(() => ({
  success: 'delivered', retrying: 'retrying', pending: 'pending', failed: 'failed',
}[d.value?.status || 'pending'] || 'pending'));

const statusLabel = computed(() => {
  if (!d.value) return '';
  if (d.value.status === 'retrying') {
    return t('developers.webhooks.retryingAttempt', { n: d.value.attempts + 1, max: WEBHOOK_MAX_ATTEMPTS });
  }
  return t(`developers.webhooks.status.${d.value.status}`);
});

const payloadText = computed(() => JSON.stringify(d.value?.payload ?? {}));

const requestText = computed(() => {
  if (!d.value) return '';
  let path = d.value.endpointUrl;
  try { path = new URL(d.value.endpointUrl).pathname; } catch { /* keep the URL */ }
  return [
    `POST ${path}`,
    'Content-Type: application/json',
    `Listnun-Event: ${d.value.event}`,
    `Listnun-Delivery-Id: ${d.value.id}`,
    'Listnun-Signature: t=…,v1=…',
    '',
    JSON.stringify(d.value.payload ?? {}, null, 2),
  ].join('\n');
});

function shortUrl(u: string): string {
  return u.replace(/^https?:\/\//, '');
}

function codeClass(code: number | null): string {
  if (!code) return 'whd-bad';
  return code >= 200 && code < 300 ? 'whd-ok' : 'whd-bad';
}

function duration(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)} s` : `${ms} ms`;
}

function clock(stamp: string): string {
  return dayjs(stamp).format('HH:mm:ss');
}

function load(id: string) {
  getWebhookDelivery(id).then((x) => { if (props.deliveryId === id) d.value = x; });
}

function copyPayload() {
  navigator.clipboard?.writeText(payloadText.value).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 1500);
  });
}

function retry() {
  if (!d.value) return;
  retrying.value = true;
  retryWebhookDelivery(d.value.id)
    .then(() => { emit('retried'); load(d.value!.id); })
    .finally(() => { retrying.value = false; });
}

watch(() => props.deliveryId, (id) => {
  d.value = null;
  if (id) load(id);
}, { immediate: true });
</script>

<style scoped lang="scss">
.whd { height: 100%; display: flex; flex-direction: column; background: var(--ln-card); }
.whd-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--ln-border);
}
.whd-head-text { display: flex; flex-direction: column; align-items: flex-start; gap: 6px; min-width: 0; }
.whd-event { margin: 0; font-size: 15px; font-weight: 500; }
.whd-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;

  h3 { margin: 0 0 8px; }
  pre { white-space: pre-wrap; word-break: break-all; }
}
.whd-attempts {
  list-style: none;
  margin: 0 0 8px;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;

  li {
    display: grid;
    grid-template-columns: 32px 56px 72px 1fr;
    gap: 8px;
    align-items: center;
    padding: 8px 12px;
    border-radius: 6px;
    background: var(--ln-muted);
    font-size: 13px;

    &.is-next { background: transparent; border: 1px dashed var(--ln-border-strong); color: var(--ln-text-3); }
  }
}
.whd-ok { color: var(--ln-success-fg); }
.whd-bad { color: var(--ln-danger-fg); }
.ln-code .whd-ok { color: #6cc39b; }
.ln-code .whd-bad { color: #e58a76; }
.msg-foot {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
</style>
