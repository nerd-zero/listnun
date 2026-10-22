<template>
  <div class="ln-page dev-send">
    <page-header :title="$t('developers.send.title')" :subtitle="$t('developers.send.intro')" />

    <div class="dev-send-grid">
      <!-- Form -->
      <form class="ln-card dev-send-form" @submit.prevent="send">
        <div class="dev-send-fields">
          <div class="ln-fields-2">
            <div class="ln-field">
              <label for="send-template">{{ $t('developers.log.template') }}</label>
              <PvSelect
                v-model="templateId"
                input-id="send-template"
                :options="templates"
                option-label="name"
                option-value="id"
                :placeholder="$t('developers.send.pickTemplate')"
                :loading="loadingTemplates"
              />
            </div>
            <div class="ln-field">
              <label for="send-key">{{ $t('developers.send.asKey') }}</label>
              <PvSelect
                v-model="keyName"
                input-id="send-key"
                :options="keyOptions"
                option-label="name"
                option-value="name"
                show-clear
                :placeholder="$t('developers.send.asYou', { name: username })"
                class="ln-mono"
              />
            </div>
          </div>

          <div class="ln-field">
            <label for="send-to">{{ $t('developers.send.recipient') }}</label>
            <PvInputText
              id="send-to"
              v-model="recipient"
              type="email"
              autocomplete="off"
              :invalid="!!recipient && !emailValid"
              placeholder="ana@example.com"
              @blur="lookupSubscriber"
            />
            <p v-if="recipient && !emailValid" class="ln-field-error">{{ $t('developers.send.badEmail') }}</p>
            <p v-else-if="subscriber" class="dev-send-hint is-ok">
              {{ $t('developers.send.existing', { name: subscriber.name, id: subscriber.id }) }}
            </p>
            <p v-else-if="lookedUp && recipient" class="dev-send-hint">
              {{ mode === 'default' ? $t('developers.send.notSubscriber') : $t('developers.send.notSubscriberOk') }}
            </p>
          </div>

          <div class="ln-field">
            <div class="dev-send-label-row">
              <label for="send-data">{{ $t('developers.send.data') }} <span class="ln-muted">JSON</span></label>
              <button v-if="templateSample" type="button" class="ln-link-btn" @click="useSample">{{ $t('developers.send.useSample') }}</button>
            </div>
            <code-editor id="send-data" v-model="dataText" lang="json" :autofocus="false" class="dev-send-data" />
            <p v-if="!dataValid" class="ln-field-error">{{ $t('developers.templates.invalidJson') }}</p>
            <p v-else-if="missing.length" class="dev-send-hint is-warn">
              {{ $t('developers.send.missingVars', { vars: missing.join(', ') }) }}
            </p>
            <p v-else class="dev-send-hint is-ok">
              {{ vars.length ? $t('developers.send.allVars', { n: vars.length }) : $t('developers.templates.validJson') }}
            </p>
          </div>

          <PvAccordion value="">
            <PvAccordionPanel value="options">
              <PvAccordionHeader>{{ $t('developers.send.options') }}</PvAccordionHeader>
              <PvAccordionContent>
                <div class="dev-send-options">
                  <div class="ln-field">
                    <label for="send-mode">{{ $t('developers.send.mode') }}</label>
                    <PvSelect v-model="mode" input-id="send-mode" :options="modeOptions" option-label="label" option-value="value" />
                    <p class="ln-help">{{ $t(`developers.send.modeHelp.${mode}`) }}</p>
                  </div>
                  <div class="ln-fields-2">
                    <div class="ln-field">
                      <label for="send-from">{{ $t('developers.send.from') }}</label>
                      <PvInputText id="send-from" v-model="fromEmail" :placeholder="defaultFrom" />
                    </div>
                    <div class="ln-field">
                      <label for="send-subject">{{ $t('developers.send.subject') }}</label>
                      <PvInputText id="send-subject" v-model="subject" :placeholder="$t('developers.send.subjectDefault')" />
                    </div>
                  </div>
                </div>
              </PvAccordionContent>
            </PvAccordionPanel>
          </PvAccordion>
        </div>

        <footer class="dev-send-foot">
          <span class="ln-small">{{ $t('developers.send.quota') }}</span>
          <PvButton
            type="submit"
            icon="pi pi-send"
            :label="$t('developers.send.send')"
            :loading="sending"
            :disabled="!canSend"
            data-cy="btn-send"
          />
        </footer>
      </form>

      <!-- Result -->
      <div class="dev-send-side">
        <section v-if="message" class="ln-card dev-send-status">
          <div class="dev-send-status-head">
            <status-tag :status="message.status" :label="$t(`developers.log.status.${message.status}`)" />
            <span class="ln-mono">{{ message.id }}</span>
            <router-link :to="{ name: 'devLog', query: { id: message.id } }" class="dev-send-loglink">{{ $t('developers.send.viewInLog') }} →</router-link>
          </div>
          <ol class="dev-send-steps">
            <li v-for="s in steps" :key="s.key" :class="{ 'is-done': s.at, 'is-bad': s.bad }">
              <span class="dev-send-step-bar" />
              <strong>{{ s.label }}</strong>
              <span class="ln-mono ln-muted">{{ s.at || '—' }}</span>
            </li>
          </ol>
        </section>

        <section class="ln-card dev-send-req">
          <div class="dev-send-req-head">
            <span class="ln-attr ln-attr--mono">POST</span>
            <span class="ln-mono">/api/tx</span>
            <PvButton
              :label="copied ? $t('states.copied') : $t('developers.send.copyCurl')"
              size="small"
              severity="secondary"
              outlined
              class="dev-send-copy"
              @click="copyCurl"
            />
          </div>
          <pre class="ln-code dev-send-code">{{ requestText }}</pre>
        </section>

        <section v-if="result" class="ln-card dev-send-req">
          <div class="dev-send-req-head">
            <strong>{{ $t('developers.send.response') }}</strong>
            <span class="ln-mono" :class="result.status < 300 ? 'is-ok' : 'is-bad'">{{ result.status }} {{ result.statusText }}</span>
            <span class="ln-mono ln-muted">{{ result.ms }} ms</span>
          </div>
          <pre class="ln-code dev-send-code">{{ JSON.stringify(result.body, null, 2) }}</pre>
        </section>

        <p class="ln-help">{{ $t('developers.send.footnote') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  computed, onBeforeUnmount, onMounted, ref, watch,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import dayjs from 'dayjs';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import { getSubscribers as subscribersApi } from '../../api/generated/endpoints/subscribers/subscribers';
import {
  getTxMessage, listAPIKeys, listTxTemplates, sendTestTx,
  type APIKey, type TxLogMessage, type TxRequest, type TxResult, type TxTemplate,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import StatusTag from '../../components/ui/StatusTag.vue';
import CodeEditor from '../../components/CodeEditor.vue';
import { missingVars, txDataVars } from './txVars';

// How often to refresh the sent message's status, and for how long.
const POLL_MS = 1000;
const POLL_FOR_MS = 60 * 1000;

const { t } = useI18n();
const route = useRoute();
const { $can } = useGlobal();
const { serverConfig, profile } = storeToRefs(useMainStore());

const templates = ref<TxTemplate[]>([]);
const keys = ref<APIKey[]>([]);
const loadingTemplates = ref(true);

const templateId = ref<number | null>(null);
const keyName = ref<string | null>((route.query.key as string) || null);
const recipient = ref('');
const dataText = ref('{}');
const mode = ref<'default' | 'fallback' | 'external'>('fallback');
const fromEmail = ref('');
const subject = ref('');

const subscriber = ref<{ id: number; name: string } | null>(null);
const lookedUp = ref(false);
const sending = ref(false);
const result = ref<TxResult | null>(null);
const message = ref<TxLogMessage | null>(null);
const copied = ref(false);

const username = computed(() => (profile.value as { username?: string }).username || '');
const defaultFrom = computed(() => (serverConfig.value as { from_email?: string }).from_email || '');
const keyOptions = computed(() => keys.value.filter((k) => k.status === 'active' && k.scopes.includes('tx:send')));
const template = computed(() => templates.value.find((x) => x.id === templateId.value) || null);
const templateSample = computed(() => {
  const s = template.value?.sampleData;
  return s && Object.keys(s).length ? s : null;
});

const emailValid = computed(() => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(recipient.value.trim()));

const parsedData = computed<Record<string, unknown> | null>(() => {
  try {
    const v = JSON.parse(dataText.value || '{}');
    return v && typeof v === 'object' && !Array.isArray(v) ? v : null;
  } catch {
    return null;
  }
});
const dataValid = computed(() => parsedData.value !== null);
const vars = computed(() => (template.value ? txDataVars(template.value.subject, template.value.body) : []));
const missing = computed(() => (parsedData.value ? missingVars(vars.value, parsedData.value) : []));

const canSend = computed(() => $can('tx:send') && !!templateId.value && emailValid.value && dataValid.value);

const modeOptions = computed(() => ['default', 'fallback', 'external'].map((m) => ({ value: m, label: t(`developers.send.modes.${m}`) })));

const request = computed<TxRequest>(() => {
  const r: TxRequest = {
    subscriber_email: recipient.value.trim() || 'ana@example.com',
    template_id: templateId.value || 0,
    data: parsedData.value || {},
  };
  if (mode.value !== 'default') r.subscriber_mode = mode.value;
  if (fromEmail.value.trim()) r.from_email = fromEmail.value.trim();
  if (subject.value.trim()) r.subject = subject.value.trim();
  return r;
});

const authUser = computed(() => keyName.value || username.value);

const requestText = computed(() => [
  `Authorization: Basic base64(${authUser.value}:$LISTNUN_API_KEY)`,
  'Content-Type: application/json',
  '',
  JSON.stringify(request.value, null, 2),
].join('\n'));

function curl(): string {
  const root = ((serverConfig.value as { root_url?: string }).root_url || window.location.origin).replace(/\/$/, '');
  const body = JSON.stringify(request.value, null, 2).replace(/'/g, "'\\''");
  return [
    `curl -X POST ${root}/api/tx \\`,
    `  -u "${authUser.value}:$LISTNUN_API_KEY" \\`,
    '  -H "Content-Type: application/json" \\',
    `  -d '${body}'`,
  ].join('\n');
}

function copyCurl() {
  navigator.clipboard?.writeText(curl()).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 1500);
  });
}

function useSample() {
  if (templateSample.value) dataText.value = JSON.stringify(templateSample.value, null, 2);
}

function lookupSubscriber() {
  subscriber.value = null;
  lookedUp.value = false;
  const email = recipient.value.trim();
  if (!emailValid.value || !$can('subscribers:get_all', 'subscribers:get')) return;
  subscribersApi().listSubscribers({ search: email, per_page: 1 } as never).then((page) => {
    const results = (page as unknown as { results: { id: number; name: string; email: string }[] }).results || [];
    const hit = results.find((s) => s.email.toLowerCase() === email.toLowerCase());
    subscriber.value = hit ? { id: hit.id, name: hit.name } : null;
    lookedUp.value = true;
    if (hit && mode.value === 'fallback') mode.value = 'default';
  });
}

// The status card's three steps, timed from acceptance.
const steps = computed(() => {
  const ev = message.value?.events || [];
  const at = (type: string) => ev.find((e) => e.type === type)?.createdAt;
  const accepted = at('accepted');
  const rel = (stamp?: string) => {
    if (!stamp || !accepted) return '';
    return `+${(dayjs(stamp).diff(dayjs(accepted)) / 1000).toFixed(1)} s`;
  };
  const failed = at('failed') || at('bounced');
  return [
    { key: 'accepted', label: t('developers.log.event.accepted'), at: accepted ? dayjs(accepted).format('HH:mm:ss.SSS') : '' },
    { key: 'sent', label: t('developers.send.sentVia'), at: rel(at('sent')) },
    failed
      ? {
        key: 'end', label: t(`developers.log.status.${message.value?.status}`), at: rel(failed), bad: true,
      }
      : { key: 'end', label: t('developers.log.event.delivered'), at: rel(at('delivered')) },
  ];
});

let pollTimer: number | undefined;
let pollUntil = 0;

function poll(id: string) {
  window.clearTimeout(pollTimer);
  getTxMessage(id).then((m) => {
    message.value = m;
    const done = ['delivered', 'bounced', 'failed'].includes(m.status);
    if (!done && Date.now() < pollUntil) pollTimer = window.setTimeout(() => poll(id), POLL_MS);
  });
}

function send() {
  if (!canSend.value) return;
  sending.value = true;
  message.value = null;
  sendTestTx(request.value, keyName.value || undefined)
    .then((r) => {
      result.value = r;
      if (r.messageIds.length) {
        pollUntil = Date.now() + POLL_FOR_MS;
        poll(r.messageIds[0]);
      }
    })
    .finally(() => { sending.value = false; });
}

watch(templateId, () => {
  // A fresh template starts from its own sample.
  if (templateSample.value) useSample();
});
watch(recipient, () => { lookedUp.value = false; subscriber.value = null; });

onMounted(() => {
  listTxTemplates()
    .then((list) => {
      templates.value = list;
      const want = Number(route.query.template);
      templateId.value = list.find((x) => x.id === want)?.id ?? list[0]?.id ?? null;
    })
    .finally(() => { loadingTemplates.value = false; });
  if ($can('apikeys:get')) listAPIKeys().then((list) => { keys.value = list; });
  recipient.value = (profile.value as { email?: string }).email || '';
});

onBeforeUnmount(() => window.clearTimeout(pollTimer));
</script>

<style scoped lang="scss">
.dev-send-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-send-form { overflow: hidden; }
.dev-send-fields {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.dev-send-label-row { display: flex; justify-content: space-between; align-items: baseline; }
.dev-send-data :deep(.cm-editor) {
  min-height: 180px;
  max-height: 320px;
  border: 1px solid var(--ln-border-strong);
  border-radius: var(--ln-radius);
}
.dev-send-hint {
  margin: 0;
  font-size: 12px;
  color: var(--ln-text-3);

  &.is-ok { color: var(--ln-success-fg); }
  &.is-warn { color: var(--ln-warn-fg); }
}
.dev-send-options { display: flex; flex-direction: column; gap: 14px; }
.dev-send-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
.dev-send-side { display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.dev-send-status { padding: 16px 20px; }
.dev-send-status-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  font-size: 13px;
}
.dev-send-loglink { margin-left: auto; font-size: 13px; text-decoration: underline; color: var(--ln-text); }
.dev-send-steps {
  list-style: none;
  margin: 16px 0 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;

  li { display: flex; flex-direction: column; gap: 4px; font-size: 13px; }
  .ln-mono { font-size: 12px; }
}
.dev-send-step-bar {
  height: 3px;
  border-radius: 9px;
  background: var(--ln-muted);

  .is-done & { background: var(--ln-success); }
  .is-bad & { background: var(--ln-warn); }
}
.dev-send-req { overflow: hidden; }
.dev-send-req-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--ln-border);
  font-size: 13px;

  .is-ok { color: var(--ln-success-fg); }
  .is-bad { color: var(--ln-danger-fg); }
}
.dev-send-copy { margin-left: auto; }
.dev-send-code { border-radius: 0; white-space: pre-wrap; word-break: break-all; }
</style>
