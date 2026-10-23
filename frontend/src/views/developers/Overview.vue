<template>
  <div class="ln-page dev-ov">
    <page-header
      :title="isMobile ? $t('developers.overview.quickstart') : $t('developers.title')"
      :subtitle="isMobile ? '' : $t('developers.overview.intro')"
    >
      <template v-if="isMobile" #before-title>
        <span class="ln-small ln-page-eyebrow">{{ $t('globals.terms.developers') }}</span>
      </template>
      <template v-if="!isMobile">
        <a href="/api/swagger/index.html" target="_blank" rel="noopener noreferrer" class="p-button p-component p-button-secondary p-button-outlined dev-ov-ref">
          {{ $t('developers.overview.apiReference') }} <i class="pi pi-external-link" aria-hidden="true" />
        </a>
        <PvButton v-if="$can('apikeys:manage')" icon="pi pi-plus" :label="$t('developers.overview.createKey')" @click="$router.push({ name: 'devKeys' })" />
      </template>
    </page-header>

    <!-- Quickstart -->
    <section class="ln-card dev-ov-quick">
      <div class="dev-ov-steps">
        <div class="dev-ov-steps-head">
          <h2 v-if="!isMobile" class="ln-section-title">{{ $t('developers.overview.quickstart') }}</h2>
          <strong v-else>{{ $t('developers.overview.done', { n: doneCount, total: steps.length }) }}</strong>
          <span class="ln-small">
            {{ isMobile && current ? $t('developers.overview.next', { step: current.next }) : $t('developers.overview.done', { n: doneCount, total: steps.length }) }}
          </span>
        </div>
        <div class="dev-ov-progress" :aria-label="$t('developers.overview.done', { n: doneCount, total: steps.length })">
          <span v-for="s in steps" :key="s.key" :class="{ 'is-done': s.done, 'is-current': s === current }" />
        </div>
        <ol class="dev-ov-list">
          <li v-for="(s, i) in steps" :key="s.key" :class="{ 'is-done': s.done, 'is-current': s === current }">
            <button type="button" class="dev-ov-step" @click="selected = s.key">
              <span class="dev-ov-step-mark" aria-hidden="true">
                <i v-if="s.done" class="pi pi-check" />
                <template v-else>{{ i + 1 }}</template>
              </span>
              <span class="dev-ov-step-text">
                <strong>{{ s.title }}</strong>
                <span v-if="!isMobile" class="ln-small">{{ s.text }}</span>
              </span>
            </button>
          </li>
        </ol>
      </div>

      <div class="dev-ov-snippet">
        <template v-if="!isMobile && shown">
          <h3 class="ln-card-title">{{ steps.indexOf(shown) + 1 }}. {{ shown.title }}</h3>
          <p class="ln-small dev-ov-snippet-help">{{ shown.help }}</p>
        </template>

        <template v-if="!shown || shown.key === 'test' || shown.key === 'log' || isMobile">
          <PvSelectButton v-model="lang" :options="langOptions" option-label="label" option-value="value" :allow-empty="false" class="dev-ov-langs" />
          <div class="dev-ov-code">
            <pre class="ln-code">{{ code }}</pre>
            <!-- Positioned by this wrapper, not the button: PrimeVue's ripple
                 sets an inline position: relative on the button itself. -->
            <div class="dev-ov-copy-wrap">
              <PvButton
                :label="copied ? $t('states.copied') : (isMobile ? $t('developers.overview.copySnippet') : $t('globals.buttons.copy'))"
                icon="pi pi-copy"
                size="small"
                severity="secondary"
                class="dev-ov-copy"
                @click="copy"
              />
            </div>
          </div>
          <div v-if="!isMobile" class="dev-ov-snippet-foot">
            <span class="ln-small">{{ $t('developers.log.key') }} <span class="ln-attr ln-attr--mono">{{ snippetKey }}</span></span>
            <span v-if="snippetTemplate" class="ln-small">{{ $t('developers.log.template') }}
              <span class="ln-attr ln-attr--mono">{{ snippetTemplate.id }}&emsp;{{ snippetTemplate.name }}</span>
            </span>
            <router-link v-if="$can('tx:send')" :to="{ name: 'devSendTest', query: sendQuery }" class="dev-ov-browser">
              {{ $t('developers.overview.orBrowser') }} →
            </router-link>
          </div>
        </template>

        <div v-else class="dev-ov-action">
          <PvButton :label="shown.action" @click="$router.push(shown.to)" />
        </div>
      </div>
    </section>

    <template v-if="!isMobile">
      <!-- Last 24 hours -->
      <section v-if="$can('tx:logs')" class="ln-tiles" :aria-label="$t('developers.overview.last24h')">
        <div class="ln-tile">
          <span class="ln-tile-label">{{ $t('developers.overview.sent24h') }}</span>
          <span class="ln-tile-value">{{ stats ? stats.sent.toLocaleString() : '—' }}</span>
          <span class="ln-small">{{ trend }}</span>
        </div>
        <div class="ln-tile">
          <span class="ln-tile-label is-success">{{ $t('developers.log.status.delivered') }}</span>
          <span class="ln-tile-value">{{ pct(stats?.delivered) }}</span>
          <span class="ln-small">{{ $t('developers.overview.messages', { n: (stats?.delivered || 0).toLocaleString() }) }}</span>
        </div>
        <div class="ln-tile">
          <span class="ln-tile-label is-warn">{{ $t('developers.log.status.bounced') }}</span>
          <span class="ln-tile-value">{{ pct(stats?.bounced) }}</span>
          <span class="ln-small">{{ $t('developers.overview.messages', { n: (stats?.bounced || 0).toLocaleString() }) }}</span>
        </div>
        <div class="ln-tile">
          <span class="ln-tile-label is-danger">{{ $t('developers.log.status.failed') }}</span>
          <span class="ln-tile-value">{{ stats ? stats.failed.toLocaleString() : '—' }}</span>
          <span class="ln-small dev-ov-err" :title="stats?.topError">{{ stats?.topError || $t('developers.overview.noFailures') }}</span>
        </div>
      </section>

      <div class="dev-ov-grid">
        <section v-if="$can('tx:logs')" class="ln-card dev-ov-recent">
          <div class="ln-card-head">
            <h2 class="ln-card-title">{{ $t('developers.overview.recent') }}</h2>
            <router-link :to="{ name: 'devLog' }" class="dev-ov-link">{{ $t('developers.overview.openLog') }}</router-link>
          </div>
          <table-skeleton v-if="!recent" :rows="5" :cols="4" :two-line="false" />
          <empty-state
            v-else-if="!recent.length"
            bare
            icon="pi-inbox"
            :title="$t('developers.log.emptyTitle')"
            :text="$t('developers.log.emptyText')"
          />
          <ul v-else class="dev-ov-rows">
            <li v-for="m in recent" :key="m.id">
              <router-link :to="{ name: 'devLog', query: { id: m.id } }" class="dev-ov-row">
                <status-tag :status="m.status" :label="$t(`developers.log.status.${m.status}`)" />
                <span class="dev-ov-row-to">{{ m.recipient }}</span>
                <span class="ln-small">{{ m.templateName }}</span>
                <span class="ln-date">{{ ago(m.createdAt) }}</span>
              </router-link>
            </li>
          </ul>
        </section>

        <aside class="dev-ov-side">
          <router-link v-if="$can('apikeys:get')" :to="{ name: 'devKeys' }" class="ln-card ln-card-pad dev-ov-card">
            <h2 class="ln-card-title">{{ $t('developers.keys.title') }}</h2>
            <span class="ln-small">{{ keysLine }}</span>
          </router-link>
          <router-link v-if="$can('templates:get')" :to="{ name: 'devTemplates' }" class="ln-card ln-card-pad dev-ov-card">
            <h2 class="ln-card-title">{{ $t('developers.overview.txTemplates') }}</h2>
            <span class="ln-small">{{ $t('developers.overview.templateCount', { n: templates.length }) }}</span>
          </router-link>
          <router-link v-if="$can('templates:get')" :to="{ name: 'devBranding' }" class="ln-card ln-card-pad dev-ov-card">
            <h2 class="ln-card-title">{{ $t('developers.branding.title') }}</h2>
            <span class="ln-small">{{ brandingLine }}</span>
          </router-link>
          <router-link
            v-if="$can('webhooks:get')"
            :to="{ name: 'devWebhooks' }"
            class="ln-card ln-card-pad dev-ov-card"
            :class="{ 'is-dashed': !hooks.length }"
          >
            <h2 class="ln-card-title">{{ $t('developers.webhooks.title') }}</h2>
            <span class="ln-small">{{ hooks.length ? $t('developers.overview.hookCount', { n: hooks.length }) : $t('developers.webhooks.emptyText') }}</span>
            <span v-if="!hooks.length" class="dev-ov-add">{{ $t('developers.webhooks.add') }} →</span>
          </router-link>
        </aside>
      </div>
    </template>

    <div v-if="isMobile && $can('tx:send')" class="dev-ov-sticky">
      <PvButton :label="$t('developers.overview.browserInstead')" class="w-full" @click="$router.push({ name: 'devSendTest', query: sendQuery })" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { storeToRefs } from 'pinia';
import type { RouteLocationRaw } from 'vue-router';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import { useTime } from '../../composables/useTime';
import { useMedia } from '../../composables/useMedia';
import {
  getBranding, getTxStats, listAPIKeys, listTxMessages, listTxTemplates, listWebhookEndpoints,
  type APIKey, type Branding, type TxLogMessage, type TxStats, type TxTemplate, type WebhookEndpoint,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import StatusTag from '../../components/ui/StatusTag.vue';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import EmptyState from '../../components/ui/EmptyState.vue';
import {
  SNIPPET_LANGS, snippet, type SnippetLang,
} from './snippets';

// LOG_SEEN_KEY marks, per browser, that the message log has been opened,
// which completes the last quickstart step.
const LOG_SEEN_KEY = 'ln-dev-log-seen';

interface Step {
  key: 'key' | 'template' | 'test' | 'log';
  title: string;
  // The title as it reads after "Next:". Its own string, as lowercasing
  // the title turned "API" into "api".
  next: string;
  text: string;
  help: string;
  done: boolean;
  action: string;
  to: RouteLocationRaw;
}

const { t } = useI18n();
const { $can } = useGlobal();
const { ago } = useTime();
const { serverConfig, profile } = storeToRefs(useMainStore());
const isMobile = useMedia('(max-width: 767px)');

const keys = ref<APIKey[]>([]);
const templates = ref<TxTemplate[]>([]);
const hooks = ref<WebhookEndpoint[]>([]);
const branding = ref<Branding | null>(null);
const stats = ref<TxStats | null>(null);
const recent = ref<TxLogMessage[] | null>(null);
const lang = ref<SnippetLang>('curl');
const selected = ref<Step['key'] | null>(null);
const copied = ref(false);

const logSeen = (() => {
  try { return localStorage.getItem(LOG_SEEN_KEY) === '1'; } catch { return false; }
})();

const sendKeys = computed(() => keys.value.filter((k) => k.status === 'active' && k.scopes.includes('tx:send')));
const snippetKey = computed(() => sendKeys.value[0]?.name || 'your-key');
const snippetTemplate = computed(() => templates.value[0] || null);
const sendQuery = computed(() => ({
  ...(sendKeys.value[0] ? { key: sendKeys.value[0].name } : {}),
  ...(snippetTemplate.value ? { template: String(snippetTemplate.value.id) } : {}),
}));

const steps = computed<Step[]>(() => [
  {
    key: 'key',
    title: t('developers.overview.stepKey'),
    next: t('developers.overview.stepKeyNext'),
    text: sendKeys.value[0] ? t('developers.overview.stepKeyDone', { name: sendKeys.value[0].name }) : t('developers.overview.stepKeyText'),
    help: t('developers.overview.stepKeyHelp'),
    done: sendKeys.value.length > 0,
    action: t('developers.overview.createKey'),
    to: { name: 'devKeys' },
  },
  {
    key: 'template',
    title: t('developers.overview.stepTemplate'),
    next: t('developers.overview.stepTemplateNext'),
    text: snippetTemplate.value ? t('developers.overview.stepTemplateDone', { name: snippetTemplate.value.name }) : t('developers.overview.stepTemplateText'),
    help: t('developers.overview.stepTemplateHelp'),
    done: templates.value.length > 0,
    action: t('developers.templates.new'),
    to: { name: 'devTemplate', params: { id: 'new' } },
  },
  {
    key: 'test',
    title: t('developers.overview.stepTest'),
    next: t('developers.overview.stepTestNext'),
    text: t('developers.overview.stepTestText'),
    help: t('developers.overview.stepTestHelp'),
    done: !!stats.value && (stats.value.sent > 0 || stats.value.sentPrev > 0 || (recent.value?.length || 0) > 0),
    action: t('developers.send.send'),
    to: { name: 'devSendTest' },
  },
  {
    key: 'log',
    title: t('developers.overview.stepLog'),
    next: t('developers.overview.stepLogNext'),
    text: t('developers.overview.stepLogText'),
    help: t('developers.overview.stepLogHelp'),
    done: logSeen && (recent.value?.length || 0) > 0,
    action: t('developers.overview.openLog'),
    to: { name: 'devLog' },
  },
]);

const doneCount = computed(() => steps.value.filter((s) => s.done).length);
const current = computed(() => steps.value.find((s) => !s.done) || null);
const shown = computed(() => steps.value.find((s) => s.key === selected.value) || current.value);

const langOptions = computed(() => SNIPPET_LANGS.map((l) => ({ value: l, label: t(`developers.overview.lang.${l}`) })));

const code = computed(() => snippet(lang.value, {
  rootUrl: (serverConfig.value as { root_url?: string }).root_url || window.location.origin,
  keyName: snippetKey.value,
  templateId: snippetTemplate.value?.id || 1,
  templateName: snippetTemplate.value?.name || '',
  recipient: (profile.value as { email?: string }).email || 'ana@example.com',
  data: (snippetTemplate.value?.sampleData && Object.keys(snippetTemplate.value.sampleData).length)
    ? snippetTemplate.value.sampleData
    : { order_id: '1042' },
}));

const trend = computed(() => {
  if (!stats.value) return '';
  const { sent, sentPrev } = stats.value;
  if (!sentPrev) return t('developers.overview.noPrev');
  const change = Math.round(((sent - sentPrev) / sentPrev) * 100);
  return t('developers.overview.trend', { pct: `${change >= 0 ? '+' : ''}${change}%` });
});

function pct(n?: number): string {
  if (!stats.value || !stats.value.sent) return '—';
  return `${(((n || 0) / stats.value.sent) * 100).toFixed(1)}%`;
}

const keysLine = computed(() => {
  const active = keys.value.filter((k) => k.status === 'active');
  if (!active.length) return t('developers.keys.emptyText');
  const last = active.map((k) => k.lastUsedAt).filter(Boolean).sort().pop();
  return last
    ? t('developers.overview.keysLine', { n: active.length, when: ago(last) })
    : t('developers.overview.keysLineNever', { n: active.length });
});

const brandingLine = computed(() => {
  if (!branding.value?.logo) return t('developers.overview.noLogo');
  const n = templates.value.filter((tpl) => tpl.body.includes('"brand-logo"')).length;
  return t('developers.overview.logoSet', { n });
});

function copy() {
  navigator.clipboard?.writeText(code.value).then(() => {
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 1500);
  });
}

onMounted(() => {
  if ($can('apikeys:get')) listAPIKeys().then((x) => { keys.value = x; });
  if ($can('templates:get')) {
    listTxTemplates().then((x) => { templates.value = x; });
    getBranding().then((x) => { branding.value = x; });
  }
  if ($can('webhooks:get')) listWebhookEndpoints().then((x) => { hooks.value = x; });
  if ($can('tx:logs')) {
    getTxStats().then((x) => { stats.value = x; });
    listTxMessages({ page: 1, perPage: 5 }).then((p) => { recent.value = p.results; });
  } else {
    recent.value = [];
  }
});
</script>

<style scoped lang="scss">
.dev-ov-ref { text-decoration: none; display: inline-flex; gap: 8px; &:hover { text-decoration: none; } }

.dev-ov-quick {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.4fr);
  overflow: hidden;

  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
  @media (max-width: 767px) { background: transparent; border: 0; gap: 16px; }
}
.dev-ov-steps {
  padding: 24px;
  border-right: 1px solid var(--ln-border);

  @media (max-width: 991px) { border-right: 0; border-bottom: 1px solid var(--ln-border); }
  @media (max-width: 767px) {
    padding: 16px;
    border: 1px solid var(--ln-border);
    border-radius: var(--ln-radius-card);
    background: var(--ln-card);
  }
}
.dev-ov-steps-head { display: flex; justify-content: space-between; align-items: baseline; gap: 8px; }
.dev-ov-progress {
  display: grid;
  grid-auto-flow: column;
  gap: 4px;
  margin: 12px 0 16px;

  span { height: 4px; border-radius: 9px; background: var(--ln-muted); }
  .is-done { background: var(--ln-success); }
  .is-current { background: var(--ln-ink); }
  .app-dark & span:not(.is-done):not(.is-current) { background: var(--ln-hover); }
}
.dev-ov-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.dev-ov-step {
  width: 100%;
  display: flex;
  gap: 14px;
  align-items: flex-start;
  padding: 10px 10px;
  border: 0;
  border-radius: var(--ln-radius);
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  text-align: left;
  cursor: pointer;

  &:hover { background: var(--ln-hover); }
  .is-current & { background: var(--ln-muted); }
  .app-dark .is-current & { background: var(--ln-hover); }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}
.dev-ov-step-mark {
  flex: none;
  width: 24px;
  height: 24px;
  border-radius: 99px;
  border: 1px solid var(--ln-border-strong);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--ln-text-2);

  i { font-size: 11px; }
  .is-done & { background: var(--ln-success); border-color: var(--ln-success); color: #fff; }
  .is-current & { background: var(--ln-ink); border-color: var(--ln-ink); color: var(--ln-on-ink); }
}
.dev-ov-step-text { display: flex; flex-direction: column; gap: 2px; strong { font-weight: 500; } .is-current & strong { font-weight: 600; } }

.dev-ov-snippet {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;

  @media (max-width: 767px) { padding: 0; }
}
.dev-ov-snippet-help { margin: -8px 0 0; }
.dev-ov-langs { align-self: flex-start; @media (max-width: 767px) { align-self: stretch; :deep(.p-togglebutton) { flex: 1; } } }
.dev-ov-code {
  position: relative;

  pre { padding-right: 96px; min-height: 160px; white-space: pre; }

  // Phones: the button sits in a strip under the code, inside the same
  // dark box, so it can't scroll under the sticky "send from the browser" bar.
  @media (max-width: 767px) {
    border-radius: 8px;
    overflow: hidden;
    background: #14171c;
    .app-dark & { background: #0c0e11; border: 1px solid var(--ln-border); }

    pre { padding-right: 16px; border: 0; border-radius: 0; }
  }
}
.dev-ov-copy-wrap {
  position: absolute;
  top: 10px;
  right: 10px;

  @media (max-width: 767px) {
    position: static;
    display: flex;
    justify-content: flex-end;
    padding: 10px 12px;
    border-top: 1px solid rgba(236, 233, 225, .12);
  }
}
.dev-ov-copy {
  background: rgba(236, 233, 225, .08) !important;
  border: 1px solid rgba(236, 233, 225, .2) !important;
  color: #ece9e1 !important;
}
.dev-ov-snippet-foot { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.dev-ov-browser { margin-left: auto; font-size: 13px; color: var(--ln-text); text-decoration: underline; }
.dev-ov-action { margin-top: 8px; }

.dev-ov-err { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.dev-ov-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-ov-recent { overflow: hidden; }
.dev-ov-link { font-size: 13px; color: var(--ln-text); text-decoration: underline; }
.dev-ov-rows { list-style: none; margin: 0; padding: 0; }
.dev-ov-row {
  display: grid;
  grid-template-columns: 120px minmax(0, 1.4fr) minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 12px 18px;
  border-top: 1px solid var(--ln-border);
  color: var(--ln-text);
  text-decoration: none;

  li:first-child & { border-top: 0; }
  &:hover { background: var(--ln-hover); text-decoration: none; }
}
.dev-ov-row-to { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dev-ov-side { display: flex; flex-direction: column; gap: 16px; }
.dev-ov-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--ln-text);
  text-decoration: none;

  &:hover { border-color: var(--ln-border-strong); text-decoration: none; }
  &.is-dashed { border-style: dashed; }
}
.dev-ov-add { margin-top: 6px; font-weight: 500; font-size: 13px; }

.dev-ov-sticky {
  position: sticky;
  bottom: 0;
  margin: 0 -16px -32px;
  padding: 12px 16px;
  background: var(--ln-bg);
  border-top: 1px solid var(--ln-border);
}
</style>
