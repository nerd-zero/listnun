<template>
  <section class="camp-ed">
    <!-- Status bar -->
    <header class="camp-bar">
      <div class="camp-bar-top">
        <div class="camp-bar-title">
          <h1 class="ln-page-title">{{ isNew ? $t('campaigns.newCampaign') : (form.name || data.name) }}</h1>
          <status-tag
            v-if="isEditing && data.status"
            v-tooltip.bottom="data.pauseReason ? $t(`campaigns.pauseReason.${data.pauseReason}`) : null"
            :status="data.status"
            :label="$t(`campaigns.status.${data.status}`)"
          />
          <span v-if="data.type === 'optin'" class="ln-attr">{{ $t('lists.optin') }}</span>
          <span v-if="savedAt" class="ln-small">{{ $t('campaigns.savedAgo', { when: ago(savedAt).toLowerCase() }) }}</span>
          <span v-if="isEditing" class="ln-small ln-muted camp-ids" :data-campaign-id="data.id">
            #{{ data.id }} <copy-text :text="data.uuid" />
          </span>
        </div>
        <div class="camp-bar-actions">
          <PvButton
            v-if="isEditing"
            icon="pi pi-eye"
            :label="$t('campaigns.preview')"
            severity="secondary"
            outlined
            data-cy="btn-preview"
            @click="isPreviewing = true"
          />
          <PvButton
            v-if="isEditing && canManage"
            :label="$t('campaigns.sendTest')"
            severity="secondary"
            outlined
            data-cy="btn-send-test"
            @click="testOpen = true"
          />
          <PvButton
            :label="primaryLabel"
            :icon="primaryIcon"
            :loading="busy"
            :disabled="primaryDisabled"
            :data-cy="primaryCy"
            aria-keyshortcuts="ctrl+s"
            @click="onPrimary"
          />
        </div>
      </div>

      <nav class="camp-steps" :aria-label="$t('campaigns.steps')">
        <button
          v-for="(s, i) in steps"
          :key="s.key"
          type="button"
          class="camp-step"
          :class="{ 'is-current': s.key === step, 'is-done': stepIndex > i }"
          :aria-current="s.key === step ? 'step' : undefined"
          :disabled="isNew && s.key !== 'settings'"
          :data-cy="`step-${s.key}`"
          @click="goStep(s.key)"
        >
          <span class="camp-step-mark" aria-hidden="true">
            <i v-if="stepIndex > i" class="pi pi-check" />
            <template v-else>{{ i + 1 }}</template>
          </span>
          {{ s.label }}
        </button>
      </nav>
    </header>

    <div v-if="loading.campaigns && isEditing && !data.id" class="camp-body">
      <table-skeleton :rows="6" :cols="1" />
    </div>

    <div v-else class="camp-body">
      <!-- 1. Settings -->
      <form v-show="step === 'settings'" class="camp-settings" @submit.prevent="onPrimary">
        <section class="ln-card ln-card-pad camp-fields">
          <div class="ln-field">
            <label for="camp-name">{{ $t('campaigns.name') }}</label>
            <PvInputText id="camp-name" ref="focusEl" v-model="form.name" name="name" :maxlength="200" :disabled="!canEdit" required autofocus />
          </div>
          <div class="ln-field">
            <label for="camp-subject">{{ $t('campaigns.subject') }}</label>
            <PvInputText id="camp-subject" v-model="form.subject" name="subject" :maxlength="5000" :disabled="!canEdit" required />
          </div>
          <div class="ln-field">
            <label for="camp-from">{{ $t('campaigns.fromAddress') }}</label>
            <PvInputText id="camp-from" v-model="form.fromEmail" name="from_email" :maxlength="200" :disabled="!canEdit"
              :placeholder="$t('campaigns.fromAddressPlaceholder')" required />
          </div>

          <list-selector
            v-model="form.lists"
            :selected="form.lists"
            :all="lists.results"
            :disabled="!canEdit"
            :label="$t('globals.terms.lists')"
            :placeholder="$t('campaigns.sendToLists')"
          />

          <div class="ln-field">
            <label for="camp-tags">{{ $t('globals.terms.tags') }} <span class="ln-muted">{{ $t('campaigns.optional') }}</span></label>
            <PvAutoComplete id="camp-tags" v-model="form.tags" name="tags" data-cy="tags" :disabled="!canEdit" :typeahead="false" multiple />
          </div>

          <!-- Mail validation for the target lists -->
          <div v-if="scrubEnabled && campaignScrubLists.length > 0" class="ln-field" data-cy="scrub-status">
            <span class="ln-field-label">{{ $t('settings.scrub.name') }}</span>
            <div v-for="l in campaignScrubLists" :key="l.id" class="camp-scrub-row">
              <span>{{ l.name }}</span>
              <status-tag v-if="l.scrub?.activeJobRequestId" status="pending" :label="$t('settings.scrub.validating')" />
              <span v-else-if="l.scrub?.lastResult" class="ln-small">
                {{ $t('settings.scrub.lastValidated') }}: {{ shortDate(l.scrub.lastResult.completedAt) }}
              </span>
              <PvButton
                v-if="$can('settings:manage')"
                size="small"
                severity="secondary"
                outlined
                :disabled="!!l.scrub?.activeJobRequestId"
                :label="$t('settings.scrub.scrubList')"
                @click="askScrub(l)"
              />
            </div>
            <p v-if="hasActiveScrubJob" class="ln-help">{{ $t('campaigns.scrubValidationInProgressHelp') }}</p>
          </div>
          <div v-if="scrubEnabled && form.lists.length > 0" class="ln-field" data-cy="scrub-history">
            <span class="ln-field-label">{{ $t('settings.scrub.history') }}</span>
            <scrub-history-list />
          </div>

          <PvAccordion :value="advancedOpen ? 'advanced' : null" class="camp-advanced">
            <PvAccordionPanel value="advanced">
              <PvAccordionHeader data-cy="btn-advanced">{{ $t('campaigns.advanced') }}</PvAccordionHeader>
              <PvAccordionContent>
                <div class="camp-fields">
                  <div class="ln-fields-2">
                    <div class="ln-field">
                      <label for="camp-messenger">{{ $t('globals.terms.messenger') }}</label>
                      <PvSelect v-model="form.messenger" input-id="camp-messenger" :options="allMessengers" :disabled="!canEdit" />
                    </div>
                    <div class="ln-field">
                      <label for="camp-format">{{ $t('campaigns.format') }}</label>
                      <PvSelect
                        v-model="form.content.contentType"
                        input-id="camp-format"
                        :options="contentTypeOptions"
                        option-label="label"
                        option-value="value"
                        :disabled="!canEdit || isEditing"
                      />
                      <p v-if="isEditing" class="ln-help">{{ $t('campaigns.formatInContent') }}</p>
                    </div>
                  </div>
                  <div class="ln-field" data-cy="btn-headers">
                    <label for="camp-headers">{{ $t('settings.smtp.setCustomHeaders') }}</label>
                    <PvTextarea
                      id="camp-headers"
                      v-model="form.headersStr"
                      name="headers"
                      class="ln-mono"
                      rows="3"
                      placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}]"
                      :disabled="!canEdit"
                    />
                    <p class="ln-help">{{ $t('campaigns.customHeadersHelp') }}</p>
                  </div>
                  <div v-if="isEditing" class="ln-field">
                    <label for="camp-attribs">{{ $t('globals.terms.attribs') }}</label>
                    <PvTextarea id="camp-attribs" v-model="form.attribsStr" class="ln-mono" rows="6" :disabled="!canEdit" />
                    <p class="ln-help">{{ $t('campaigns.attribsHelp') }}</p>
                  </div>
                </div>
              </PvAccordionContent>
            </PvAccordionPanel>
          </PvAccordion>
        </section>

        <aside class="ln-card ln-card-pad camp-audience">
          <span class="ln-label">{{ $t('campaigns.audience') }}</span>
          <strong class="ln-display">{{ audience.toLocaleString() }}</strong>
          <span class="ln-small">{{ $t('campaigns.audienceHelp', { n: form.lists.length }) }}</span>
        </aside>
      </form>

      <!-- 2. Content -->
      <section v-if="data.id" v-show="step === 'content'" class="camp-content">
        <editor
          v-model="form.content"
          :id="data.id"
          :title="data.name"
          :disabled="!canEdit"
          :templates="(templates as Loose[])"
          :content-types="contentTypes"
        />

        <div class="camp-content-extras">
          <div class="ln-field" data-cy="media">
            <span class="ln-field-label">{{ $t('campaigns.attachments') }}</span>
            <PvAutoComplete
              v-if="isAttachFieldVisible"
              v-model="form.media"
              name="media"
              option-label="filename"
              :disabled="!canEdit"
              multiple
              @focus="isAttachModalOpen = true"
            />
            <button v-else type="button" class="ln-link-btn" data-cy="btn-attach" @click="isAttachFieldVisible = true">
              {{ $t('campaigns.addAttachments') }}
            </button>
          </div>
          <a href="https://listmonk.app/docs/templating/#template-expressions" target="_blank" rel="noopener noreferrer" class="ln-small">
            {{ $t('campaigns.templatingRef') }} <i class="pi pi-external-link" aria-hidden="true" />
          </a>
        </div>

        <div v-if="form.content.contentType !== 'plain'" class="ln-field camp-alt">
          <div class="camp-alt-head">
            <span class="ln-field-label">{{ $t('campaigns.plainTextVersion') }}</span>
            <template v-if="canEdit">
              <button v-if="form.altbody === null" type="button" class="ln-link-btn" @click="form.altbody = htmlToPlainText(form.content.body)">
                {{ $t('campaigns.addAltText') }}
              </button>
              <button v-else type="button" class="ln-link-btn" @click="form.altbody = null">{{ $t('campaigns.removeAltText') }}</button>
            </template>
          </div>
          <PvTextarea v-if="form.altbody !== null" v-model="form.altbody" :disabled="!canEdit" rows="8" class="ln-mono" />
          <p v-else class="ln-help">{{ $t('campaigns.plainTextNone') }}</p>
        </div>
      </section>

      <!-- 3. Schedule -->
      <section v-show="step === 'schedule'" class="camp-schedule">
        <h2 class="ln-section-title">{{ $t('campaigns.whenTitle') }}</h2>
        <div class="camp-when" role="radiogroup" :aria-label="$t('campaigns.whenTitle')">
          <label class="camp-when-card" :class="{ 'is-on': !form.sendLater }">
            <PvRadioButton v-model="form.sendLater" :value="false" :disabled="!canEdit" input-id="when-now" />
            <span>
              <strong>{{ $t('campaigns.sendNow') }}</strong>
              <span class="ln-small">{{ $t('campaigns.sendNowHelp') }}</span>
            </span>
          </label>
          <label class="camp-when-card" :class="{ 'is-on': form.sendLater }" data-cy="btn-send-later">
            <PvRadioButton v-model="form.sendLater" :value="true" :disabled="!canEdit" input-id="when-later" />
            <span>
              <strong>{{ $t('campaigns.sendLater') }}</strong>
              <span class="ln-small">{{ $t('campaigns.sendLaterHelp') }}</span>
            </span>
          </label>
        </div>

        <div v-if="form.sendLater" class="ln-card ln-card-pad camp-when-pick" data-cy="send_at">
          <div class="ln-field">
            <label for="camp-send-at">{{ $t('campaigns.dateAndTime') }}</label>
            <PvDatePicker
              v-model="form.sendAtDate"
              input-id="camp-send-at"
              show-time
              hour-format="24"
              show-icon
              icon-display="input"
              :min-date="new Date()"
              :disabled="!canEdit"
            />
            <p class="ln-help">{{ $t('campaigns.timezone', { tz: timezone }) }}</p>
          </div>
          <div v-if="form.sendAtDate" class="camp-when-note">
            <i class="pi pi-clock" aria-hidden="true" />
            <span class="ln-small">{{ $t('campaigns.sendsIn', { when: $utils.duration(new Date(), form.sendAtDate) }) }}</span>
          </div>
        </div>

        <!-- Public archive -->
        <section class="ln-card camp-archive">
          <div class="ln-card-head">
            <div>
              <h3 class="ln-card-title">{{ $t('campaigns.archive') }}</h3>
              <p class="ln-small">{{ $t('campaigns.archiveHelp') }}</p>
            </div>
            <PvToggleSwitch v-model="form.archive" :disabled="!canArchive" data-cy="btn-archive" :aria-label="$t('campaigns.archiveEnable')" />
          </div>
          <div v-if="form.archive" class="ln-card-pad camp-fields">
            <div class="ln-fields-2">
              <div class="ln-field">
                <label for="camp-archive-tpl">{{ $t('globals.terms.template') }}</label>
                <PvSelect
                  v-model="form.archiveTemplateId"
                  input-id="camp-archive-tpl"
                  :options="campaignTemplates"
                  option-label="name"
                  option-value="id"
                  :disabled="!canArchive || form.content.contentType === 'visual'"
                />
              </div>
              <div class="ln-field">
                <label for="camp-archive-slug">{{ $t('campaigns.archiveSlug') }}</label>
                <PvInputText id="camp-archive-slug" v-model="form.archiveSlug" name="archive_slug" data-cy="archive-slug" :maxlength="200" :disabled="!canArchive" />
              </div>
            </div>
            <div class="ln-field">
              <div class="camp-alt-head">
                <label for="camp-archive-meta">{{ $t('campaigns.archiveMeta') }}</label>
                <button v-if="!form.archiveMetaStr || form.archiveMetaStr === '{}'" type="button" class="ln-link-btn" @click="onFillArchiveMeta">
                  {{ $t('campaigns.fillSample') }}
                </button>
              </div>
              <PvTextarea id="camp-archive-meta" v-model="form.archiveMetaStr" name="archive_meta" data-cy="archive-meta" class="ln-mono" rows="6" :disabled="!canArchive" />
              <p class="ln-help">{{ $t('campaigns.archiveMetaHelp') }}</p>
            </div>
            <div class="camp-archive-actions">
              <a v-if="data.uuid" :href="`${rootUrl}/archive/${data.uuid}`" target="_blank" rel="noopener noreferrer" class="ln-small">
                {{ $t('campaigns.openArchive') }} <i class="pi pi-external-link" aria-hidden="true" />
              </a>
              <PvButton
                v-if="isEditing"
                :label="$t('campaigns.previewArchive')"
                icon="pi pi-eye"
                severity="secondary"
                outlined
                size="small"
                @click="isPreviewingArchive = true"
              />
              <PvButton
                v-if="!canEdit && canArchive"
                :label="$t('globals.buttons.saveChanges')"
                size="small"
                :loading="busy"
                data-cy="btn-save"
                @click="onUpdateCampaignArchive"
              />
            </div>
          </div>
        </section>
      </section>

      <!-- 4. Review -->
      <section v-show="step === 'review'" class="camp-review">
        <h2 class="ln-section-title">{{ $t('campaigns.reviewTitle') }}</h2>
        <ul class="ln-card camp-checks">
          <li v-for="c in checks" :key="c.key" :class="`is-${c.tone}`">
            <span class="camp-check-mark" aria-hidden="true">
              <i :class="['pi', c.tone === 'ok' ? 'pi-check' : (c.tone === 'warn' ? 'pi-exclamation-triangle' : 'pi-info')]" />
            </span>
            <span class="camp-check-text">
              <strong>{{ c.title }}</strong>
              <span class="ln-small">{{ c.text }}</span>
            </span>
            <button v-if="c.fix" type="button" class="ln-link-btn" @click="c.fix.run">{{ c.fix.label }}</button>
          </li>
        </ul>
      </section>

      <footer class="camp-foot">
        <PvButton v-if="stepIndex > 0" :label="$t('campaigns.back')" severity="secondary" outlined @click="goStep(steps[stepIndex - 1].key)" />
        <span class="ln-dialog-form-spacer" />
        <PvButton
          :label="primaryLabel"
          :loading="busy"
          :disabled="primaryDisabled"
          :data-cy="`${primaryCy}-foot`"
          @click="onPrimary"
        />
      </footer>
    </div>

    <media-picker-dialog v-model:visible="isAttachModalOpen" @select="onAttachSelect" />

    <campaign-preview v-if="isPreviewing" type="campaign" :id="data.id" :title="data.name" @close="isPreviewing = false" />
    <campaign-preview
      v-if="isPreviewingArchive"
      type="campaign"
      :id="data.id"
      :archive-meta="form.archiveMetaStr"
      :title="data.name"
      :content-type="data.contentType"
      :template-id="form.archiveTemplateId"
      is-post
      is-archive
      @close="isPreviewingArchive = false"
    />

    <!-- Send a test -->
    <dialog-form
      v-model:visible="testOpen"
      :title="$t('campaigns.sendTestTitle')"
      :submit-label="$t('campaigns.sendTest')"
      :busy="busy"
      :submit-disabled="!form.testEmails.length"
      @submit="sendTest"
    >
      <div class="ln-field">
        <label for="camp-test-to">{{ $t('campaigns.sendTo') }}</label>
        <PvAutoComplete
          v-model="form.testEmails"
          input-id="camp-test-to"
          :typeahead="false"
          multiple
          :placeholder="$t('campaigns.testEmails')"
          @blur="onTestEmailBlur"
        />
        <p class="ln-help">{{ $t('campaigns.sendTestHelp') }}</p>
      </div>
      <PvMessage v-if="lastTest" severity="success" icon="pi pi-check">
        {{ $t('campaigns.lastTest', { when: ago(lastTest.at).toLowerCase(), to: lastTest.to }) }}
      </PvMessage>
    </dialog-form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :danger="confirm.danger"
      :busy="busy"
      @confirm="confirm.run"
    >
      <template v-if="confirm.secondary" #footer-start>
        <PvButton :label="confirm.secondary.label" text severity="secondary" @click="confirm.secondary.run" />
      </template>
    </confirm-action>
  </section>
</template>

<script setup lang="ts">
import {
  computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch,
} from 'vue';
import dayjs from 'dayjs';
import htmlToPlainText from 'textversionjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { useTime } from '../composables/useTime';
import CampaignPreview from '../components/CampaignPreview.vue';
import CopyText from '../components/CopyText.vue';
import Editor from '../components/Editor.vue';
import ListSelector from '../components/ListSelector.vue';
import MediaPickerDialog from '../components/MediaPickerDialog.vue';
import ScrubHistoryList from '../components/ScrubHistoryList.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import TableSkeleton from '../components/ui/TableSkeleton.vue';
import DialogForm from '../components/ui/DialogForm.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getSettings as settingsApi } from '../api/generated/endpoints/settings/settings';

// Campaigns, lists and templates are the API's loose camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

type StepKey = 'settings' | 'content' | 'schedule' | 'review';

interface Check {
  key: string;
  tone: 'ok' | 'warn' | 'info';
  title: string;
  text: string;
  fix?: { label: string; run: () => void };
}

// Old tab anchors map onto the new steps, so existing links still land.
const HASH_TO_STEP: Record<string, StepKey> = {
  campaign: 'settings', settings: 'settings', attribs: 'settings', content: 'content', schedule: 'schedule', archive: 'schedule', review: 'review',
};

const { getScrubListStatus, scrubList } = settingsApi();
const {
  $api: untypedApi, $utils, $can, $events,
} = useGlobal();
// The legacy client resolves to camelCased API objects.
const $api = untypedApi as unknown as Record<string, (...args: unknown[]) => Promise<Loose>>;
const { t } = useI18n();
const { ago, shortDate } = useTime();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { serverConfig, loading, templates } = storeToRefs(store);
const lists = computed(() => (store.lists as Loose) || {});

const focusEl = ref<Loose | null>(null);
const isNew = ref(false);
const isEditing = ref(false);
const isAttachFieldVisible = ref(false);
const isAttachModalOpen = ref(false);
const isPreviewing = ref(false);
const isPreviewingArchive = ref(false);
const advancedOpen = ref(false);
const testOpen = ref(false);
const busy = ref(false);
const step = ref<StepKey>('settings');
const data = ref<Loose>({});
const savedAt = ref<string | null>(null);
const lastTest = ref<{ at: string; to: string } | null>(null);
const selListIDs = ref<number[]>([]);
const scrubListStatus = ref<Record<number, Loose>>({});
let leaveApproved = false;

const form = reactive<Loose>({
  archiveSlug: null,
  name: '',
  subject: '',
  fromEmail: '',
  headersStr: '[]',
  headers: [],
  attribsStr: '{}',
  messenger: 'email',
  lists: [],
  tags: [],
  sendAt: null,
  content: {
    contentType: 'richtext', body: '', bodySource: null, templateId: null,
  },
  altbody: null,
  media: [],
  sendAtDate: null,
  sendLater: false,
  archive: false,
  archiveTemplateId: null,
  archiveMetaStr: '{}',
  archiveMeta: {},
  testEmails: [],
});

const confirm = reactive({
  open: false,
  title: '',
  text: '',
  label: '',
  danger: false,
  run: () => {},
  secondary: null as { label: string; run: () => void } | null,
});

const steps = computed<{ key: StepKey; label: string }[]>(() => [
  { key: 'settings', label: t('campaigns.stepSettings') },
  { key: 'content', label: t('campaigns.content') },
  { key: 'schedule', label: t('campaigns.stepSchedule') },
  { key: 'review', label: t('campaigns.stepReview') },
]);
const stepIndex = computed(() => steps.value.findIndex((s) => s.key === step.value));

const contentTypes = computed(() => Object.freeze({
  richtext: t('campaigns.richText'),
  html: t('campaigns.rawHTML'),
  markdown: t('campaigns.markdown'),
  plain: t('campaigns.plainText'),
  visual: t('campaigns.visual'),
}));
const contentTypeOptions = computed(() => Object.entries(contentTypes.value).map(([value, label]) => ({ value, label })));

const scrubEnabled = computed(() => !!(serverConfig.value as Loose).scrub_enabled);
const canManage = computed(() => $can('campaigns:manage_all', 'campaigns:manage'));
const canSend = computed(() => $can('campaigns:send'));
const canEdit = computed(() => isNew.value || ['draft', 'scheduled', 'paused'].includes(data.value.status));
const canArchive = computed(() => data.value.status !== 'cancelled' && data.value.type !== 'optin');

// campaignScrubLists pairs each target list with its Scrub status, fetched
// tenant-wide the same way Lists.vue does (Scrub has no per-list status
// endpoint); lists never validated are included so they can be validated.
const campaignScrubLists = computed<Loose[]>(() => (form.lists as Loose[]).map((l) => ({ ...l, scrub: scrubListStatus.value[l.id] })));
// hasActiveScrubJob mirrors the server's checkScrubJobOnCampaign block so
// Start/Schedule disable instead of round-tripping into the same 400; the
// server stays authoritative.
const hasActiveScrubJob = computed(() => campaignScrubLists.value.some((l) => l.scrub?.activeJobRequestId));

const canSchedule = computed(() => ['draft', 'paused'].includes(data.value.status) && form.sendLater && form.sendAtDate && !hasActiveScrubJob.value);
const canUnSchedule = computed(() => data.value.status === 'scheduled');
const canStart = computed(() => ['draft', 'paused'].includes(data.value.status) && !form.sendLater && !hasActiveScrubJob.value);

const audience = computed(() => (form.lists as Loose[]).reduce((n, l) => {
  const full = (lists.value.results || []).find((x: Loose) => x.id === l.id);
  return n + (full?.subscriberCount || l.subscriberCount || 0);
}, 0));

const selectedLists = computed(() => {
  if (selListIDs.value.length === 0 || !lists.value.results) return [];
  return lists.value.results.filter((l: Loose) => selListIDs.value.includes(l.id));
});
const allMessengers = computed(() => {
  const sc = serverConfig.value as Loose;
  const email = ['email', ...(sc.messengers || []).filter((m: string) => m.startsWith('email-'))];
  const others = (sc.messengers || []).filter((m: string) => m !== 'email' && !m.startsWith('email-'));
  return [...email, ...others];
});
const campaignTemplates = computed(() => ((templates.value as Loose[]) || []).filter((tpl) => tpl.type === 'campaign'));
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
const rootUrl = computed(() => (serverConfig.value as Loose).root_url || '');

// The header's main action: create, save and move on, or send.
const primaryLabel = computed(() => {
  if (isNew.value) return t('campaigns.continue');
  if (step.value !== 'review') return t('campaigns.continue');
  if (canSend.value && canUnSchedule.value) return t('campaigns.unSchedule');
  if (canSend.value && canSchedule.value) return t('campaigns.schedule');
  if (canSend.value && canStart.value) return t('campaigns.start');
  return t('globals.buttons.saveChanges');
});
const primaryIcon = computed(() => {
  if (step.value !== 'review' || isNew.value) return undefined;
  if (canSend.value && (canSchedule.value || canUnSchedule.value)) return 'pi pi-clock';
  if (canSend.value && canStart.value) return 'pi pi-send';
  return undefined;
});
const primaryCy = computed(() => {
  if (isNew.value) return 'btn-continue';
  if (step.value !== 'review') return 'btn-next';
  if (canUnSchedule.value) return 'btn-unschedule';
  if (canSchedule.value) return 'btn-schedule';
  if (canStart.value) return 'btn-start';
  return 'btn-save';
});
const primaryDisabled = computed(() => {
  if (isNew.value) return !form.name || !form.subject || !canManage.value;
  if (step.value === 'review' && !canEdit.value && !canUnSchedule.value) return true;
  return false;
});

const bodyText = computed(() => String(form.content.body || '').replace(/<[^>]*>/g, '').trim());

const checks = computed<Check[]>(() => {
  const out: Check[] = [];
  out.push(form.subject && form.fromEmail
    ? {
      key: 'subject', tone: 'ok', title: t('campaigns.checkSubject'), text: `${form.subject} ${form.fromEmail}`,
    }
    : {
      key: 'subject', tone: 'warn', title: t('campaigns.checkSubject'), text: t('campaigns.checkSubjectMissing'), fix: { label: t('campaigns.fix'), run: () => goStep('settings') },
    });
  out.push(form.lists.length
    ? {
      key: 'audience',
      tone: 'ok',
      title: t('campaigns.audience'),
      text: t('campaigns.checkAudience', { n: audience.value.toLocaleString(), lists: (form.lists as Loose[]).map((l) => l.name).join(', ') }),
    }
    : {
      key: 'audience', tone: 'warn', title: t('campaigns.audience'), text: t('campaigns.checkNoLists'), fix: { label: t('campaigns.fix'), run: () => goStep('settings') },
    });
  out.push(bodyText.value || form.content.contentType === 'visual'
    ? {
      key: 'content', tone: 'ok', title: t('campaigns.content'), text: t('campaigns.checkContent', { format: contentTypes.value[form.content.contentType as keyof typeof contentTypes.value] }),
    }
    : {
      key: 'content', tone: 'warn', title: t('campaigns.content'), text: t('campaigns.checkEmpty'), fix: { label: t('campaigns.fix'), run: () => goStep('content') },
    });
  out.push(lastTest.value
    ? {
      key: 'test', tone: 'ok', title: t('campaigns.checkTest'), text: t('campaigns.lastTest', { when: ago(lastTest.value.at).toLowerCase(), to: lastTest.value.to }),
    }
    : {
      key: 'test',
      tone: 'info',
      title: t('campaigns.checkTest'),
      text: t('campaigns.checkNoTest'),
      fix: canManage.value ? { label: t('campaigns.sendTest'), run: () => { testOpen.value = true; } } : undefined,
    });
  if (form.content.contentType !== 'plain') {
    out.push({
      key: 'plain', tone: 'info', title: t('campaigns.plainTextVersion'), text: form.altbody !== null ? t('campaigns.checkPlainCustom') : t('campaigns.plainTextNone'),
    });
  }
  out.push(form.sendLater && form.sendAtDate
    ? {
      key: 'when', tone: 'ok', title: t('campaigns.stepSchedule'), text: t('campaigns.checkLater', { date: shortDate(form.sendAtDate, true) }),
    }
    : {
      key: 'when', tone: 'info', title: t('campaigns.stepSchedule'), text: t('campaigns.checkNow'),
    });
  if (hasActiveScrubJob.value) {
    out.push({
      key: 'scrub', tone: 'warn', title: t('settings.scrub.name'), text: t('campaigns.scrubValidationInProgressHelp'),
    });
  }
  return out;
});

function fetchScrubListStatus() {
  if (!scrubEnabled.value) return;
  getScrubListStatus().then((res) => {
    const m: Record<number, Loose> = {};
    (Array.isArray(res) ? res : []).forEach((l: Loose) => {
      m[l.id] = { activeJobRequestId: l.activeJobRequestId, lastResult: l.lastResult };
    });
    scrubListStatus.value = m;
  }).catch(() => {});
}

function askScrub(l: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('settings.scrub.scrubList'),
    text: t('settings.scrub.scrubListConfirm', { name: l.name }),
    label: t('settings.scrub.scrubList'),
    danger: false,
    secondary: null,
    run: () => {
      confirm.open = false;
      scrubList(l.id).then(() => {
        $utils.toast(t('settings.scrub.scrubJobStarted'));
        fetchScrubListStatus();
      });
    },
  });
}

function isUnsaved() {
  if (isNew.value) return !!(form.name || form.subject || form.content.body);
  return data.value.body !== form.content.body || data.value.contentType !== form.content.contentType;
}

function onAttachSelect(o: Loose) {
  if (!form.media.some((m: Loose) => m.id === o.id)) form.media.push(o);
}

function onFillArchiveMeta() {
  const sample = { email: 'email@domain.com', name: t('globals.fields.name'), attribs: {} };
  form.archiveMetaStr = ($utils.getPref('campaign.archiveMetaStr') as string) || JSON.stringify(sample, null, 4);
}

function goStep(k: StepKey) {
  if (isNew.value && k !== 'settings') return;
  step.value = k;
  window.history.replaceState(window.history.state, '', `#${k}`);
  if (k === 'content' && (window as Loose).tinymce?.editors?.length > 0) {
    nextTick(() => { (window as Loose).tinymce.editors[0].focus(); });
  }
}

// parseForm reads the JSON fields; false when one is invalid.
function parseForm(): boolean {
  if (form.headersStr && form.headersStr !== '[]') {
    try { form.headers = JSON.parse(form.headersStr); } catch (e) {
      advancedOpen.value = true;
      $utils.toast(String(e), 'is-danger');
      return false;
    }
  } else {
    form.headers = [];
  }
  if (form.archive && form.archiveMetaStr) {
    try { form.archiveMeta = JSON.parse(form.archiveMetaStr); } catch (e) {
      $utils.toast(String(e), 'is-danger');
      return false;
    }
  }
  form.attribs = null;
  if (form.attribsStr && form.attribsStr.trim()) {
    try { form.attribs = JSON.parse(form.attribsStr); } catch (e) {
      advancedOpen.value = true;
      $utils.toast(`${t('subscribers.invalidJSON')}: ${String(e)}`, 'is-danger', 3000);
      return false;
    }
  }
  return true;
}

function fillFrom(d: Loose) {
  data.value = d;
  savedAt.value = d.updatedAt || null;
  Object.assign(form, {
    ...d,
    headersStr: JSON.stringify(d.headers, null, 4),
    archiveMetaStr: d.archiveMeta ? JSON.stringify(d.archiveMeta, null, 4) : '{}',
    attribsStr: d.attribs ? JSON.stringify(d.attribs, null, 4) : '{}',
    content: {
      contentType: d.contentType, body: d.body, bodySource: d.bodySource, templateId: d.templateId,
    },
  });
  isAttachFieldVisible.value = form.media.length > 0;
  form.media = form.media.map((f: Loose) => (!f.id ? { ...f, filename: `❌ ${f.filename}` } : f));
}

function getCampaign(id: string) {
  return $api.getCampaign(id).then((d) => fillFrom(d as Loose));
}

function onTestEmailBlur(e: FocusEvent) {
  const val = (e.target as HTMLInputElement).value?.trim();
  if (val) {
    form.testEmails.push(val);
    (e.target as HTMLInputElement).value = '';
  }
}

function sendTest() {
  if (!parseForm()) return;
  busy.value = true;
  $api.testCampaign({
    id: data.value.id,
    name: form.name,
    subject: form.subject,
    lists: form.lists.map((l: Loose) => l.id),
    from_email: form.fromEmail,
    messenger: form.messenger,
    type: 'regular',
    headers: form.headers,
    tags: form.tags,
    template_id: form.content.templateId,
    content_type: form.content.contentType,
    body: form.content.body,
    altbody: form.content.contentType !== 'plain' ? form.altbody : null,
    subscribers: form.testEmails,
    media: form.media.map((m: Loose) => m.id),
  })
    .then(() => {
      lastTest.value = { at: new Date().toISOString(), to: form.testEmails.join(', ') };
      testOpen.value = false;
      $utils.toast(t('campaigns.testSent'));
    })
    .finally(() => { busy.value = false; });
}

function createCampaign() {
  if (!parseForm()) return;
  busy.value = true;
  $api.createCampaign({
    archiveSlug: form.subject,
    name: form.name,
    subject: form.subject,
    lists: form.lists.map((l: Loose) => l.id),
    from_email: form.fromEmail,
    content_type: form.content.contentType,
    messenger: form.messenger,
    type: 'regular',
    tags: form.tags,
    send_at: form.sendLater ? form.sendAtDate : null,
    headers: form.headers,
    attribs: form.attribs,
    media: form.media.map((m: Loose) => m.id),
  })
    .then((d) => {
      leaveApproved = true;
      router.push({ name: 'campaign', hash: '#content', params: { id: (d as Loose).id } });
    })
    .finally(() => { busy.value = false; });
}

function updateCampaign(typ?: string): Promise<void> {
  if (!parseForm()) return Promise.reject(new Error('invalid'));
  const typMsg = typ === 'start' ? 'campaigns.started' : 'globals.messages.updated';
  if (!form.sendAtDate) form.sendLater = false;
  busy.value = true;
  return $api.updateCampaign(data.value.id, {
    archive_slug: form.archiveSlug,
    name: form.name,
    subject: form.subject,
    lists: form.lists.map((l: Loose) => l.id),
    from_email: form.fromEmail,
    messenger: form.messenger,
    type: 'regular',
    tags: form.tags,
    send_at: form.sendLater ? form.sendAtDate : null,
    headers: form.headers,
    attribs: form.attribs,
    template_id: form.content.templateId,
    content_type: form.content.contentType,
    body: form.content.body,
    body_source: form.content.bodySource,
    altbody: form.content.contentType !== 'plain' ? form.altbody : null,
    archive: form.archive,
    archive_template_id: form.archiveTemplateId,
    archive_meta: form.archiveMeta,
    media: form.media.map((m: Loose) => m.id),
  })
    .then((d) => {
      const c = d as Loose;
      data.value = c;
      savedAt.value = c.updatedAt || new Date().toISOString();
      form.archiveSlug = c.archiveSlug;
      form.attribsStr = c.attribs ? JSON.stringify(c.attribs, null, 4) : '{}';
      if (typ !== 'quiet') $utils.toast(t(typMsg, { name: c.name }));
    })
    .finally(() => { busy.value = false; });
}

function onUpdateCampaignArchive() {
  if (isEditing.value && canEdit.value) return;
  let archiveMeta;
  try {
    archiveMeta = JSON.parse(form.archiveMetaStr);
  } catch (e) {
    $utils.toast(String(e), 'is-danger');
    return;
  }
  busy.value = true;
  $api.updateCampaignArchive(data.value.id, {
    archive: form.archive,
    archive_template_id: form.archiveTemplateId,
    archive_meta: archiveMeta,
    archive_slug: form.archiveSlug,
  })
    .then((d) => {
      form.archiveSlug = (d as Loose).archiveSlug;
      $utils.toast(t('globals.messages.updated', { name: data.value.name }));
    })
    .finally(() => { busy.value = false; });
}

function changeStatus(status: string) {
  busy.value = true;
  return $api.changeCampaignStatus(data.value.id, status).finally(() => { busy.value = false; });
}

function askSend() {
  const scheduling = canSchedule.value;
  Object.assign(confirm, {
    open: true,
    title: t(`campaigns.confirm.${scheduling ? 'scheduled' : 'running'}.title`, { name: form.name }),
    text: t(`campaigns.confirm.${scheduling ? 'scheduled' : 'running'}.text`, { n: audience.value.toLocaleString() }),
    label: t(`campaigns.confirm.${scheduling ? 'scheduled' : 'running'}.action`),
    danger: false,
    secondary: null,
    run: () => {
      updateCampaign('quiet').then(() => changeStatus(scheduling ? 'scheduled' : 'running')).then(() => {
        confirm.open = false;
        leaveApproved = true;
        router.push({ name: 'campaigns' });
      });
    },
  });
}

function askUnschedule() {
  Object.assign(confirm, {
    open: true,
    title: t('campaigns.confirm.draft.title', { name: form.name }),
    text: t('campaigns.confirm.draft.text'),
    label: t('campaigns.confirm.draft.action'),
    danger: false,
    secondary: null,
    run: () => {
      changeStatus('draft').then((d) => {
        confirm.open = false;
        fillFrom(d as Loose);
      });
    },
  });
}

function onPrimary() {
  if (isNew.value) {
    createCampaign();
    return;
  }
  if (step.value !== 'review') {
    const next = steps.value[stepIndex.value + 1].key;
    if (canEdit.value && canManage.value) {
      updateCampaign('quiet').then(() => goStep(next)).catch(() => {});
    } else {
      goStep(next);
    }
    return;
  }
  if (canSend.value && canUnSchedule.value) askUnschedule();
  else if (canSend.value && (canSchedule.value || canStart.value)) askSend();
  else if (canEdit.value) updateCampaign().catch(() => {});
}

watch(selectedLists, (v) => { form.lists = v; });
watch(() => form.lists, fetchScrubListStatus);
watch(() => data.value.sendAt, (v) => {
  if (v !== null && v !== undefined) {
    form.sendLater = true;
    form.sendAtDate = dayjs(v).toDate();
  } else {
    form.sendLater = false;
    form.sendAtDate = null;
  }
});

onBeforeRouteLeave((_to, _from, next) => {
  if (leaveApproved || !isUnsaved()) {
    next(true);
    return;
  }

  // Settle the navigation exactly once: closing the dialog any other way
  // than its actions cancels it.
  let settled = false;
  const stop = watch(() => confirm.open, (open) => {
    if (!open) settle(false);
  });
  function settle(ok: boolean) {
    if (settled) return;
    settled = true;
    stop();
    confirm.open = false;
    next(ok);
  }

  Object.assign(confirm, {
    open: true,
    title: t('campaigns.leaveTitle'),
    text: t('campaigns.leaveText', { name: form.name || t('campaigns.newCampaign') }),
    label: isNew.value ? t('developers.templates.leave') : t('campaigns.saveAndLeave'),
    danger: false,
    secondary: isNew.value ? null : { label: t('developers.templates.leave'), run: () => settle(true) },
    run: () => {
      if (isNew.value) {
        settle(true);
        return;
      }
      updateCampaign('quiet').then(() => settle(true)).catch(() => settle(false));
    },
  });
});

onMounted(() => {
  window.onbeforeunload = () => isUnsaved() || null;
  form.fromEmail = (serverConfig.value as Loose).from_email;

  const { id } = route.params as { id: string };
  if (id === 'new') {
    isNew.value = true;
    if (route.query.list_id) {
      const strIds = (Array.isArray(route.query.list_id) ? route.query.list_id : [route.query.list_id]) as string[];
      selListIDs.value = strIds.map((v) => parseInt(v, 10));
    }
  } else {
    const intID = parseInt(id, 10);
    if (intID <= 0 || Number.isNaN(intID)) {
      $utils.toast(t('campaigns.invalid'));
      return;
    }
    isEditing.value = true;
  }

  $api.getTemplates().then((tpls) => {
    const list = tpls as Loose[];
    if (list.length > 0 && !form.content.templateId) {
      const tpl = list.find((i) => i.isDefault === true);
      if (tpl) form.content.templateId = tpl.id;
    }
  });

  if (isEditing.value) {
    getCampaign(id).then(() => {
      const fromHash = HASH_TO_STEP[route.hash.replace('#', '')];
      if (fromHash) step.value = fromHash;
      if (route.hash === '#attribs') advancedOpen.value = true;
    });
  } else {
    form.messenger = 'email';
  }

  nextTick(() => { focusEl.value?.$el?.focus(); });
  $events.$on('campaign.update', () => {
    if (canEdit.value && !isNew.value) updateCampaign().catch(() => {});
  });
});

onBeforeUnmount(() => {
  $events.$off('campaign.update');
  window.onbeforeunload = null;
});
</script>

<style scoped lang="scss">
.camp-ed {
  margin: -28px -32px -48px;
  min-height: calc(100vh - 56px);
  display: flex;
  flex-direction: column;

  @media (max-width: 991px) { margin: -20px -16px -32px; }
}
.camp-bar {
  position: sticky;
  top: 56px;
  z-index: 10;
  background: var(--ln-bg);
  border-bottom: 1px solid var(--ln-border);
  padding: 16px 32px 0;

  @media (max-width: 991px) { padding: 14px 16px 0; }
}
.camp-bar-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.camp-bar-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;

  h1 { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 520px; }
}
.camp-ids { display: inline-flex; gap: 6px; align-items: center; }
.camp-bar-actions { display: flex; gap: 8px; flex-wrap: wrap; }

.camp-steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 4px;
  margin-top: 14px;
}
.camp-step {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px 12px;
  border: 0;
  border-bottom: 2px solid var(--ln-border);
  background: transparent;
  color: var(--ln-text-3);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
  text-align: left;

  &:disabled { cursor: not-allowed; opacity: .6; }
  &.is-done { color: var(--ln-text); border-bottom-color: var(--ln-success); }
  &.is-current { color: var(--ln-text); font-weight: 600; border-bottom-color: var(--ln-ink); }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
  @media (max-width: 600px) { font-size: 0; gap: 0; justify-content: center; }
}
.camp-step-mark {
  width: 20px;
  height: 20px;
  flex: none;
  border-radius: 99px;
  border: 1px solid var(--ln-border-strong);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;

  i { font-size: 10px; }
  .is-done & { background: var(--ln-success); border-color: var(--ln-success); color: #fff; }
  .is-current & { background: var(--ln-ink); border-color: var(--ln-ink); color: var(--ln-on-ink); }
}

.camp-body {
  padding: 24px 32px 48px;
  max-width: 1200px;
  width: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 20px;

  @media (max-width: 991px) { padding: 20px 16px 32px; }
}

.camp-settings {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 20px;
  align-items: start;

  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
}
.camp-fields { display: flex; flex-direction: column; gap: 16px; }
.camp-fields :deep(.field) { margin-bottom: 0; }
.camp-audience {
  position: sticky;
  top: 180px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.camp-scrub-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; font-size: 14px; }
.camp-advanced { margin-top: 4px; }

.camp-content { display: flex; flex-direction: column; gap: 16px; }
.camp-content-extras { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; flex-wrap: wrap; }
.camp-alt-head { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; }

.camp-schedule { display: flex; flex-direction: column; gap: 16px; max-width: 760px; }
.camp-when {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;

  @media (max-width: 600px) { grid-template-columns: minmax(0, 1fr); }
}
.camp-when-card {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 16px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius-card);
  background: var(--ln-card);
  cursor: pointer;

  > span { display: flex; flex-direction: column; gap: 4px; }
  &.is-on { border-color: var(--ln-ink); box-shadow: 0 0 0 1px var(--ln-ink); }
}
.camp-when-pick { display: flex; flex-direction: column; gap: 12px; }
.camp-when-note {
  display: flex;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--ln-radius);
  background: var(--ln-muted);
  .app-dark & { background: var(--ln-hover); }
}
.camp-archive { overflow: hidden; .ln-card-head { align-items: flex-start; } .ln-small { margin: 4px 0 0; } }
.camp-archive-actions { display: flex; align-items: center; justify-content: flex-end; gap: 12px; flex-wrap: wrap; }

.camp-review { display: flex; flex-direction: column; gap: 16px; max-width: 760px; }
.camp-checks { list-style: none; margin: 0; padding: 0; overflow: hidden; }
.camp-checks li {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 18px;
  border-top: 1px solid var(--ln-border);

  &:first-child { border-top: 0; }
}
.camp-check-mark {
  width: 22px;
  height: 22px;
  flex: none;
  border-radius: 99px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--ln-muted);
  color: var(--ln-text-2);

  i { font-size: 10px; }
  .is-ok & { background: var(--ln-success); color: #fff; }
  .is-warn & { background: var(--ln-warn-bg); color: var(--ln-warn-fg); }
}
.camp-check-text { flex: 1; display: flex; flex-direction: column; gap: 2px; min-width: 0; overflow-wrap: anywhere; }

.camp-foot {
  display: flex;
  gap: 8px;
  padding-top: 16px;
  border-top: 1px solid var(--ln-border);
}
</style>
