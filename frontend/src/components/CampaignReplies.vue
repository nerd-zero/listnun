<template>
  <section class="ln-card camp-replies" data-cy="campaign-replies" :aria-labelledby="headingId">
    <header class="ln-card-head">
      <h3 :id="headingId" class="ln-card-title">{{ $t('campaigns.replies.title') }}</h3>
    </header>

    <div class="camp-replies-body">
      <!-- Three figures in one row at every width: a row of bordered tiles
           inside this card stacked to a full screen at 390px. -->
      <dl class="camp-replies-stats">
        <div>
          <dt>{{ $t('campaigns.replies.count') }}</dt>
          <dd data-cy="replies-count">{{ counts.replies }}</dd>
        </div>
        <div>
          <dt>{{ $t('campaigns.replies.people') }}</dt>
          <dd data-cy="repliers-count">{{ counts.repliers }}</dd>
        </div>
        <div>
          <dt>{{ $t('campaigns.replies.auto') }}</dt>
          <dd data-cy="auto-count">{{ counts.auto }}</dd>
        </div>
      </dl>
      <p v-if="counts.auto" class="ln-help camp-replies-auto-help">{{ $t('campaigns.replies.autoHelp') }}</p>

      <p v-if="error" class="ln-field-error" role="alert">{{ $t('campaigns.replies.loadError') }}</p>

      <empty-state
        v-else-if="loaded && !replies.length"
        bare
        icon="pi-inbox"
        :title="$t('campaigns.replies.emptyTitle')"
        :text="$t('campaigns.replies.empty')"
      />

      <ul v-else class="camp-replies-list">
        <li v-for="r in replies" :key="r.uuid" class="camp-reply" data-cy="reply">
          <!-- Name and time on one line, address and tag under it, so the
               time never wraps onto a line of its own on a phone. -->
          <div class="camp-reply-head">
            <strong class="camp-reply-name">{{ r.subscriberName || r.fromName || r.fromEmail }}</strong>
            <time class="ln-small ln-muted camp-reply-when" :datetime="r.receivedAt" :title="r.receivedAt">
              {{ ago(r.receivedAt) }}
            </time>
          </div>
          <div class="camp-reply-meta">
            <span class="ln-small ln-muted camp-reply-addr">{{ r.fromEmail }}</span>
            <span v-if="r.isAuto" class="ln-status" data-cy="reply-auto">{{ $t('campaigns.replies.autoTag') }}</span>
          </div>
          <p class="camp-reply-subject ln-small">{{ r.subject }}</p>
          <!-- Text only, never rendered HTML: a reply is untrusted mail. -->
          <p class="camp-reply-text">{{ snippet(r) }}</p>
          <div class="camp-reply-foot">
            <button
              v-if="fullText(r) !== snippet(r)"
              type="button"
              class="ln-link-btn"
              :aria-expanded="expanded.has(r.uuid)"
              @click="toggle(r.uuid)"
            >
              {{ expanded.has(r.uuid) ? $t('campaigns.replies.hideFull') : $t('campaigns.replies.showFull') }}
            </button>
            <span class="ln-small ln-muted">
              {{ r.forwardedTo ? $t('campaigns.replies.forwardedTo', { address: r.forwardedTo }) : $t('campaigns.replies.notForwarded') }}
            </span>
          </div>
          <pre v-if="expanded.has(r.uuid)" class="camp-reply-full">{{ fullText(r) }}</pre>
        </li>
      </ul>

      <PvButton
        v-if="replies.length < total"
        :label="$t('campaigns.replies.loadMore')"
        severity="secondary"
        outlined
        :loading="loading"
        data-cy="replies-more"
        @click="load(page + 1)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  onMounted, reactive, ref, useId,
} from 'vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';
import { useTime } from '../composables/useTime';
import EmptyState from './ui/EmptyState.vue';

// One reply as the API returns it, after the mutator camel-cases keys
// (models.CampaignReply).
interface Reply {
  uuid: string;
  subscriberName: string | null;
  fromEmail: string;
  fromName: string;
  subject: string;
  textBody: string;
  htmlBody: string;
  strippedText: string;
  isAuto: boolean;
  forwardedTo: string;
  receivedAt: string;
}

interface Page {
  results: Reply[];
  counts: { replies: number; repliers: number; auto: number };
  total: number;
}

const PER_PAGE = 20;
// A snippet longer than this is cut, with the rest behind "Show full reply".
const SNIPPET_LEN = 280;

const props = defineProps<{ campaignId: number }>();

const { getCampaignReplies } = campaignsApi();
const { ago } = useTime();
const headingId = useId();

const replies = ref<Reply[]>([]);
const counts = reactive({ replies: 0, repliers: 0, auto: 0 });
const total = ref(0);
const page = ref(0);
const loading = ref(false);
const loaded = ref(false);
const error = ref(false);
const expanded = ref(new Set<string>());

// Block-level closing tags that end a line when an HTML-only reply is
// turned into text.
const BLOCK_END = /<\/(p|div|li|tr|h[1-6]|blockquote)>|<br\s*\/?>/gi;

// fullText is the whole reply as text: the provider's text part, or the
// HTML part as text when the reply was HTML-only. DOMParser builds an
// inert document (nothing runs or loads); script and style contents are
// dropped rather than shown as text.
function fullText(r: Reply): string {
  if (r.textBody.trim()) return r.textBody.trim();
  const doc = new DOMParser().parseFromString(r.htmlBody.replace(BLOCK_END, '$&\n'), 'text/html');
  doc.querySelectorAll('script, style, head').forEach((el) => el.remove());
  return (doc.body.textContent || '').replace(/\n{3,}/g, '\n\n').trim();
}

// snippet is the reply without the quoted original (when the provider
// could separate it), cut to SNIPPET_LEN.
function snippet(r: Reply): string {
  const text = r.strippedText.trim() || fullText(r);
  return text.length > SNIPPET_LEN ? `${text.slice(0, SNIPPET_LEN).trimEnd()}…` : text;
}

function toggle(uuid: string) {
  const next = new Set(expanded.value);
  if (next.has(uuid)) next.delete(uuid);
  else next.add(uuid);
  expanded.value = next;
}

async function load(p: number) {
  loading.value = true;
  error.value = false;
  try {
    const out = (await getCampaignReplies(props.campaignId, { page: p, per_page: PER_PAGE })) as unknown as Page;
    replies.value = p === 1 ? out.results : [...replies.value, ...out.results];
    Object.assign(counts, out.counts);
    total.value = out.total;
    page.value = p;
    loaded.value = true;
  } catch {
    error.value = true;
  } finally {
    loading.value = false;
  }
}

onMounted(() => load(1));
</script>

<style scoped lang="scss">
.camp-replies-body { display: flex; flex-direction: column; gap: 16px; padding: 16px 18px; align-items: stretch; }
.camp-replies-body > .p-button { align-self: flex-start; }
.camp-replies-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
.camp-reply {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 0;
  border-top: 1px solid var(--ln-border);
  &:first-child { border-top: 0; padding-top: 0; }
}
.camp-replies-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin: 0;

  // A label that wraps ("People who replied") mustn't knock its figure
  // out of line with the others.
  > div { display: flex; flex-direction: column; }
  dt { font-size: 13px; line-height: 18px; color: var(--ln-text-2); }
  dd {
    margin: auto 0 0;
    padding-top: 2px;
    font-size: 22px;
    line-height: 28px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
}
.camp-replies-auto-help { margin: -8px 0 0; }
.camp-reply-head { display: flex; align-items: baseline; gap: 12px; }
.camp-reply-name { font-weight: 600; min-width: 0; overflow-wrap: anywhere; }
.camp-reply-when { margin-left: auto; flex: none; }
.camp-reply-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.camp-reply-addr { min-width: 0; overflow-wrap: anywhere; }
.camp-reply-subject { margin: 0; color: var(--ln-text-2); }
.camp-reply-text { margin: 2px 0 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.camp-reply-foot { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.camp-reply-full {
  margin: 6px 0 0;
  padding: 12px;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font: inherit;
  font-size: 13px;
  background: var(--ln-muted);
  border-radius: var(--ln-radius);
}
</style>
