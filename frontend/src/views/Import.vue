<template>
  <div class="ln-page imp">
    <page-header :title="$t('import.title')" :subtitle="$t('import.intro')">
      <a href="#" class="imp-sample" @click.prevent="downloadSample">{{ $t('import.sample') }}</a>
    </page-header>

    <PvSkeleton v-if="isLoading" height="320px" />

    <template v-else>
      <!-- Four steps. On a phone only the current one is named. -->
      <ol class="imp-steps" :aria-label="$t('import.steps')">
        <li
          v-for="(label, i) in stepLabels"
          :key="i"
          :class="{ 'is-on': step === i + 1, 'is-done': step > i + 1 }"
          :aria-current="step === i + 1 ? 'step' : undefined"
        >
          <span class="imp-step-num">
            <i v-if="step > i + 1" class="pi pi-check" aria-hidden="true" />
            <template v-else>{{ i + 1 }}</template>
          </span>
          <span class="imp-step-label">{{ label }}</span>
        </li>
      </ol>

      <!-- 1. Upload -->
      <div v-if="step === 1" class="imp-upload" :class="stepAnim">
        <section class="ln-card imp-drop-card">
          <button
            type="button"
            class="imp-drop"
            :class="{ 'is-over': dragOver }"
            :disabled="uploading"
            @dragover.prevent="dragOver = true"
            @dragleave="dragOver = false"
            @drop.prevent="onFileDrop"
            @click="fileInputEl?.click()"
          >
            <span class="imp-drop-icon" aria-hidden="true">
              <i :class="uploading ? 'pi pi-spin pi-spinner' : 'pi pi-upload'" />
            </span>
            <strong>{{ uploading ? $t('import.uploading') : $t('import.drop') }}</strong>
            <span v-if="!uploading">{{ $t('import.or') }} <span class="imp-drop-link">{{ $t('import.choose') }}</span></span>
            <span class="ln-small">{{ $t('import.fileHelp') }}</span>
          </button>
          <input
            ref="fileInputEl"
            type="file"
            :aria-label="$t('import.choose')"
            accept=".csv,.zip,text/csv,application/zip"
            class="imp-file-input"
            @change="onFileSelect"
          >
        </section>

        <aside v-if="recent.length" class="ln-card imp-recent">
          <h2>{{ $t('import.recent') }}</h2>
          <ul>
            <li v-for="r in recent" :key="r.id">
              <span class="imp-recent-top">
                <span class="ln-mono imp-recent-name">{{ r.name }}</span>
                <span class="ln-small">{{ shortDate(r.createdAt) }}</span>
              </span>
              <span :class="{ 'is-warn': r.skipped || r.failed, 'is-bad': r.status === 'failed' }">{{ recentResult(r) }}</span>
            </li>
          </ul>
        </aside>
      </div>

      <template v-else>
        <!-- The file, on every later step. -->
        <section class="ln-card imp-file">
          <span class="imp-file-badge" aria-hidden="true">{{ fileExt }}</span>
          <span class="imp-file-text">
            <span class="ln-mono">{{ file?.name || status.name }}</span>
            <span v-if="file" class="ln-small">{{ fileSummary }}</span>
          </span>
          <PvButton v-if="!started" :label="$t('import.differentFile')" severity="secondary" outlined size="small" @click="restart" />
        </section>

        <!-- 2. Match columns -->
        <section v-if="step === 2 && file" class="ln-card imp-card" :class="stepAnim">
          <header class="imp-card-head">
            <h2>{{ $t('import.matchTitle') }}</h2>
            <p class="ln-small">{{ $t('import.matchHelp') }}</p>
          </header>
          <div class="imp-match" role="table" :aria-label="$t('import.matchTitle')">
            <div class="imp-match-row imp-match-head" role="row">
              <span role="columnheader">{{ $t('import.colFile') }}</span>
              <span role="columnheader">{{ $t('import.colSamples') }}</span>
              <span role="columnheader">{{ $t('import.colSaveAs') }}</span>
            </div>
            <div v-for="(c, i) in columns" :key="i" class="imp-match-row" :class="{ 'is-skipped': c.as === 'skip' }" role="row">
              <span class="ln-mono imp-match-name" role="cell">{{ c.header || $t('import.noHeader', { n: i + 1 }) }}</span>
              <span class="imp-match-samples" role="cell">{{ samplesText(c) }}</span>
              <span role="cell">
                <PvSelect
                  :model-value="c.as"
                  :options="saveAsOptions(c)"
                  option-label="label"
                  option-value="value"
                  :aria-label="$t('import.saveAsFor', { name: c.header || i + 1 })"
                  class="imp-match-select"
                  @update:model-value="(v: string) => setSaveAs(i, v)"
                />
              </span>
            </div>
          </div>
          <footer class="imp-foot">
            <PvButton :label="$t('globals.buttons.back')" severity="secondary" outlined @click="restart" />
            <span class="imp-foot-end">
              <span class="ln-small">{{ hasEmail ? $t('import.kept', { n: keptCount, total: columns.length }) : $t('import.needEmail') }}</span>
              <PvButton :label="$t('import.continue')" :disabled="!hasEmail" :loading="checking" @click="onMatched" />
            </span>
          </footer>
        </section>

        <!-- 3. Choose options -->
        <section v-else-if="step === 3 && check" class="ln-card imp-card" :class="stepAnim">
          <div class="imp-q">
            <h2 id="imp-q-mode">{{ $t('import.qMode') }}</h2>
            <PvSelectButton
              v-model="opts.mode"
              :options="modeOptions"
              option-label="label"
              option-value="value"
              :allow-empty="false"
              aria-labelledby="imp-q-mode"
              class="imp-modes"
            />
          </div>

          <template v-if="opts.mode === 'subscribe'">
            <div class="imp-q imp-lists">
              <h2>{{ $t('import.qLists') }}</h2>
              <list-selector
                v-model="opts.lists"
                :label="$t('import.qLists')"
                :placeholder="$t('import.qListsPlaceholder')"
                :selected="opts.lists"
                :all="allLists"
              />
            </div>

            <div v-if="hasDoubleOptin" class="imp-q">
              <h2 id="imp-q-consent">{{ $t('import.qConsent') }}</h2>
              <div class="imp-choices" role="radiogroup" aria-labelledby="imp-q-consent">
                <label v-for="o in consentOptions" :key="o.value" class="imp-choice" :class="{ 'is-on': opts.consent === o.value }">
                  <PvRadioButton v-model="opts.consent" :value="o.value" name="consent" />
                  <span><strong>{{ o.title }}</strong><span class="ln-small">{{ o.body }}</span></span>
                </label>
              </div>
            </div>

            <div v-if="check.existing" class="imp-q">
              <h2 id="imp-q-existing">{{ $t('import.qExisting', { n: fmt(check.existing) }, check.existing) }}</h2>
              <div class="imp-choices" role="radiogroup" aria-labelledby="imp-q-existing">
                <label v-for="o in existingOptions" :key="o.value" class="imp-choice" :class="{ 'is-on': opts.existing === o.value }">
                  <PvRadioButton v-model="opts.existing" :value="o.value" name="existing" />
                  <span><strong>{{ o.title }}</strong><span class="ln-small">{{ o.body }}</span></span>
                </label>
              </div>
            </div>
          </template>
          <p v-else class="ln-small imp-q">{{ $t('import.blockHelp') }}</p>

          <footer class="imp-foot">
            <PvButton :label="$t('globals.buttons.back')" severity="secondary" outlined @click="goTo(2)" />
            <PvButton :label="$t('import.review')" :disabled="opts.mode === 'subscribe' && !opts.lists.length" @click="goTo(4)" />
          </footer>
        </section>

        <!-- 4. Review -->
        <section v-else-if="step === 4 && !started && check" class="ln-card imp-card" :class="stepAnim">
          <header class="imp-card-head">
            <h2>{{ $t('import.readyTitle', { n: fmt(check.rows) }, check.rows) }}</h2>
          </header>
          <div class="imp-body">
            <div class="ln-tiles imp-tiles">
              <div v-for="tile in forecast" :key="tile.label" class="ln-tile">
                <span class="ln-tile-label">{{ tile.label }}</span>
                <span class="ln-tile-value" :class="{ 'imp-bad': tile.bad }">{{ fmt(tile.value) }}</span>
              </div>
            </div>
            <dl class="imp-summary">
              <template v-for="row in summary" :key="row.label">
                <dt>{{ row.label }}</dt>
                <dd>{{ row.value }}</dd>
              </template>
            </dl>
            <import-problems
              v-if="check.numProblems"
              :problems="check.problems"
              :total="check.numProblems"
              :title="$t('import.problemsSkip', { n: fmt(check.numProblems) }, check.numProblems)"
              :file-name="file?.name"
            />
          </div>
          <footer class="imp-foot">
            <PvButton :label="$t('globals.buttons.back')" severity="secondary" outlined @click="goTo(3)" />
            <PvButton :label="$t('import.start')" data-cy="btn-import" :loading="starting" @click="onStart" />
          </footer>
        </section>

        <!-- 4. Importing, and done -->
        <template v-else-if="step === 4">
          <section class="ln-card imp-card ln-step-next">
            <header class="imp-card-head imp-run-head">
              <h2 data-cy="import-status" :data-status="status.status">{{ runTitle }}</h2>
              <span v-if="isRunning()" class="ln-small">{{ timeLeft }}</span>
            </header>
            <div class="imp-body">
              <PvProgressBar :value="barValue" :show-value="false" class="imp-bar" />
              <div class="ln-tiles imp-tiles">
                <div v-for="tile in progressTiles" :key="tile.label" class="ln-tile">
                  <span class="ln-tile-label">{{ tile.label }}</span>
                  <span class="ln-tile-value" :class="{ 'imp-bad': tile.bad }">{{ fmt(tile.value) }}</span>
                </div>
              </div>
              <p v-if="status.failed" class="imp-note is-bad" role="alert">{{ $t('import.failedRows', { n: fmt(status.failed) }, status.failed) }}</p>
            </div>
          </section>

          <section v-if="isDone() && needsConfirming" class="ln-card imp-confirm">
            <span>
              <strong>{{ $t('import.confirmTitle', { n: fmt(status.added) }, status.added) }}</strong>
              <span class="ln-small">{{ $can('subscribers:manage') ? $t('import.confirmHelp') : $t('import.confirmNoPerm') }}</span>
            </span>
            <PvButton
              v-if="$can('subscribers:manage')"
              :label="optinsSent ? $t('import.confirmSent') : $t('import.confirmSend')"
              :loading="sendingOptins"
              :disabled="optinsSent"
              @click="sendOptins"
            />
          </section>

          <section v-if="problems.length" class="ln-card">
            <import-problems
              :problems="problems"
              :total="status.skipped"
              :title="$t('import.problemsTitle', { n: fmt(status.skipped) }, status.skipped)"
              :help="$t('import.problemsHelp')"
              :file-name="status.name"
            />
          </section>

          <div class="imp-run-foot">
            <a href="#" class="ln-small" @click.prevent="downloadLog">{{ $t('import.downloadLog') }}</a>
            <!-- Secondary while the confirmation emails are the next thing to do. -->
            <PvButton
              v-if="isDone()"
              :severity="needsConfirming && !optinsSent ? 'secondary' : undefined"
              :outlined="needsConfirming && !optinsSent"
              :label="$t('import.another')"
              :loading="isProcessing"
              data-cy="btn-import-done"
              @click="clearAndRestart"
            />
            <PvButton
              v-else
              :label="$t('import.stopImport')"
              severity="danger"
              outlined
              :loading="isProcessing || status.status === 'stopping'"
              data-cy="btn-stop-import"
              @click="confirmStop = true"
            />
          </div>
        </template>
      </template>
    </template>

    <confirm-action
      v-model:visible="confirmStop"
      :title="$t('import.stopTitle')"
      :text="$t('import.stopText')"
      :confirm-label="$t('import.stopImport')"
      danger
      :busy="isProcessing"
      @confirm="onConfirmStop"
    />
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, onMounted, onBeforeUnmount, nextTick,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import dayjs from 'dayjs';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import ListSelector from '../components/ListSelector.vue';
import PageHeader from '../components/ui/PageHeader.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import ImportProblems from '../components/ImportProblems.vue';
import { getImport } from '../api/generated/endpoints/import/import';
import { getLists } from '../api/generated/endpoints/lists/lists';
import { perceivedProgress } from '../composables/perceivedProgress';

// Import status and list rows are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface FileColumn {
  header: string;
  samples: string[];
  index: number;
  as: string;
  key?: string;
}
interface UploadedFile {
  fileId: string;
  name: string;
  size: number;
  uploadedAt: string;
  delim: string;
  rows: number;
  columns: FileColumn[];
}
interface ImportProblem { row: number; value: string; reason: string }
interface CheckResult {
  rows: number;
  valid: number;
  existing: number;
  numProblems: number;
  problems: ImportProblem[];
}
interface RecentImport {
  id: number;
  name: string;
  mode: string;
  status: string;
  imported: number;
  added: number;
  skipped: number;
  failed: number;
  createdAt: string;
}
// The choices behind a started import, kept so a reload can still label
// its figures and offer the confirmation emails.
interface LastChoices {
  name: string;
  mode: string;
  existing: string;
  consent: string;
  doubleOptin: boolean;
  lists: number[];
}

// How often to poll the import status while it runs.
const POLL_MS = 500;
// How many sample values to show per column.
const SAMPLES_SHOWN = 3;
const LAST_KEY = 'ln-import-last';

const { $utils } = useGlobal();
const {
  getImportLogs, getImportStatus, stopImport, importSubscribers, uploadImportFile, checkImport,
  getRecentImports, getImportProblems,
} = getImport();
const { resendOptins } = getLists();
const { t, n } = useI18n();
const route = useRoute();
const store = useMainStore();
const { lists } = storeToRefs(store);

const step = ref(1);
// The step that appears slides in from the side the user moved towards.
const stepAnim = ref('');
const isLoading = ref(true);
const isProcessing = ref(false);
const uploading = ref(false);
const checking = ref(false);
const starting = ref(false);
const started = ref(false);
const dragOver = ref(false);
const confirmStop = ref(false);
const sendingOptins = ref(false);
const optinsSent = ref(false);

const file = ref<UploadedFile | null>(null);
const columns = ref<FileColumn[]>([]);
const check = ref<CheckResult | null>(null);
const recent = ref<RecentImport[]>([]);
const status = ref<Loose>({ status: '' });
const problems = ref<ImportProblem[]>([]);
const last = ref<LastChoices | null>(null);
const fileInputEl = ref<HTMLInputElement | null>(null);
const runStart = ref<{ at: number; done: number } | null>(null);
let pollID: number | undefined;

const opts = reactive({
  mode: 'subscribe',
  lists: [] as Loose[],
  consent: 'ask',
  existing: 'leave',
});

function fmt(v: unknown): string {
  return n(Number(v) || 0);
}

function isRunning() { return status.value.status === 'importing' || status.value.status === 'stopping'; }
function isDone() {
  return status.value.status === 'finished' || status.value.status === 'stopped' || status.value.status === 'failed';
}

const stepLabels = computed(() => [t('import.stepUpload'), t('import.stepMatch'), t('import.stepOptions'),
  started.value ? t('import.stepImport') : t('import.stepReview')]);

const allLists = computed<Loose[]>(() => (lists.value as Loose).results || []);
const hasDoubleOptin = computed(() => opts.lists.some((l) => l.optin === 'double'));

// --- Upload ---

const fileExt = computed(() => {
  const name = file.value?.name || status.value.name || '';
  return (name.split('.').pop() || 'csv').toUpperCase().slice(0, 4);
});

const fileSummary = computed(() => {
  if (!file.value) return '';
  return t('import.fileSummary', {
    rows: fmt(file.value.rows),
    cols: fmt(file.value.columns.length),
    // The shared relative-time strings read oddly under a minute.
    when: dayjs().diff(file.value.uploadedAt, 'minute') < 1 ? t('import.justNow') : dayjs(file.value.uploadedAt).fromNow(),
  });
});

function goTo(s: number) {
  stepAnim.value = s > step.value ? 'ln-step-next' : 'ln-step-prev';
  step.value = s;
  window.scrollTo({ top: 0 });
}

function upload(f: File | undefined) {
  if (!f || uploading.value) return;
  uploading.value = true;
  uploadImportFile({ file: f }).then((data) => {
    const up = data as unknown as UploadedFile;
    file.value = up;
    columns.value = up.columns.map((c) => ({ ...c }));
    check.value = null;
    goTo(2);
  }).finally(() => { uploading.value = false; });
}

function onFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  upload(target.files?.[0]);
  target.value = '';
}

function onFileDrop(e: DragEvent) {
  dragOver.value = false;
  upload(e.dataTransfer?.files?.[0]);
}

function loadRecent() {
  getRecentImports().then((data) => { recent.value = data as unknown as RecentImport[]; }, () => {});
}

function restart() {
  file.value = null;
  check.value = null;
  started.value = false;
  goTo(1);
  loadRecent();
}

function shortDate(d: string) {
  const day = dayjs(d);
  return day.isSame(dayjs(), 'year') ? day.format('D MMM') : day.format('D MMM YYYY');
}

function recentResult(r: RecentImport) {
  if (r.status === 'failed') return t('import.recentFailed');
  const parts = [r.mode === 'blocklist'
    ? t('import.recentBlocked', { n: fmt(r.imported) }, r.imported)
    : t('import.recentAdded', { n: fmt(r.added) }, r.added)];
  if (r.skipped) parts.push(t('import.recentSkipped', { n: fmt(r.skipped) }, r.skipped));
  return parts.join(', ');
}

function saveFile(body: string, name: string, type: string) {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([body], { type }));
  a.download = name;
  a.click();
  URL.revokeObjectURL(a.href);
}

function downloadSample() {
  saveFile('email,name,city\nana@example.com,Ana Dube,Harare\ntendai@example.com,Tendai Moyo,Bulawayo\n', 'sample.csv', 'text/csv');
}

// --- Match columns ---

const hasEmail = computed(() => columns.value.some((c) => c.as === 'email'));
const keptCount = computed(() => columns.value.filter((c) => c.as !== 'skip').length);

function samplesText(c: FileColumn) {
  const s = c.samples.slice(0, SAMPLES_SHOWN);
  return s.length ? s.join(', ') : '—';
}

// The attribute a column is saved as: its own key, or one made from its
// header the way the server makes them.
function attrKey(c: FileColumn) {
  return c.key || c.header.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '');
}

function saveAsOptions(c: FileColumn) {
  const out = [
    { label: t('import.asEmail'), value: 'email' },
    { label: t('import.asName'), value: 'name' },
  ];
  if (attrKey(c)) out.push({ label: t('import.asAttribute', { key: attrKey(c) }), value: 'attribute' });
  if (c.as === 'attributes' || /^attrib/i.test(c.header)) out.push({ label: t('import.asAttributes'), value: 'attributes' });
  out.push({ label: t('import.asSkip'), value: 'skip' });
  return out;
}

// Only one column is the email and one the name: choosing either for a
// column saves the one that had it as an attribute.
function setSaveAs(i: number, v: string) {
  columns.value = columns.value.map((c, j) => {
    if (j === i) return { ...c, as: v, key: v === 'attribute' ? attrKey(c) : c.key };
    if ((v === 'email' || v === 'name') && c.as === v) {
      const key = attrKey(c);
      return { ...c, as: key ? 'attribute' : 'skip', key };
    }
    return c;
  });
}

function columnsReq() {
  return columns.value.map((c) => ({ index: c.index, as: c.as, key: c.as === 'attribute' ? attrKey(c) : undefined }));
}

function onMatched() {
  if (!file.value) return;
  checking.value = true;
  checkImport({ file_id: file.value.fileId, columns: columnsReq() }).then((data) => {
    check.value = data as unknown as CheckResult;
    goTo(3);
  }).finally(() => { checking.value = false; });
}

// --- Options ---

const modeOptions = computed(() => [
  { label: t('import.modeSubscribe'), value: 'subscribe' },
  { label: t('import.modeBlock'), value: 'blocklist' },
]);
const consentOptions = computed(() => [
  { value: 'ask', title: t('import.consentAsk'), body: t('import.consentAskHelp') },
  { value: 'yes', title: t('import.consentYes'), body: t('import.consentYesHelp') },
]);
const existingOptions = computed(() => [
  { value: 'leave', title: t('import.existingLeave'), body: t('import.existingLeaveHelp') },
  { value: 'update', title: t('import.existingUpdate'), body: t('import.existingUpdateHelp') },
]);

// --- Review ---

const newCount = computed(() => Math.max(0, (check.value?.valid || 0) - (check.value?.existing || 0)));
const asksConsent = computed(() => opts.mode === 'subscribe' && hasDoubleOptin.value && opts.consent === 'ask');

const forecast = computed(() => {
  const c = check.value;
  if (!c) return [];
  const out = opts.mode === 'blocklist'
    ? [{ label: t('import.willBlock'), value: c.valid, bad: false }]
    : [
      { label: t('import.newSubs'), value: newCount.value, bad: false },
      { label: opts.existing === 'update' ? t('import.alreadyUpdated') : t('import.alreadyUnchanged'), value: c.existing, bad: false },
    ];
  if (asksConsent.value) out.push({ label: t('import.willConfirm'), value: newCount.value, bad: false });
  out.push({ label: t('import.problemRows'), value: c.numProblems, bad: c.numProblems > 0 });
  return out;
});

// "Email, name and 3 attributes".
function columnsSummary() {
  const kept = columns.value.filter((c) => c.as !== 'skip');
  const attrs = kept.filter((c) => c.as === 'attribute').length;
  const parts = [t('import.asEmail')];
  if (kept.some((c) => c.as === 'name')) parts.push(t('import.asName').toLowerCase());
  if (attrs) parts.push(t('import.nAttributes', { n: attrs }, attrs));
  if (kept.some((c) => c.as === 'attributes')) parts.push(t('import.asAttributes').toLowerCase());
  if (parts.length === 1) return parts[0];
  return t('import.listAnd', { items: parts.slice(0, -1).join(', '), last: parts[parts.length - 1] });
}

const summary = computed(() => {
  if (opts.mode === 'blocklist') {
    return [
      { label: t('import.sumAction'), value: t('import.modeBlock') },
      { label: t('import.sumColumns'), value: columnsSummary() },
    ];
  }
  const out = [{ label: t('import.sumLists'), value: opts.lists.map((l) => l.name).join(', ') }];
  if (hasDoubleOptin.value) {
    out.push({ label: t('import.sumConsent'), value: opts.consent === 'ask' ? t('import.consentAskShort') : t('import.consentYesShort') });
  }
  if (check.value?.existing) {
    out.push({ label: t('import.sumExisting'), value: opts.existing === 'update' ? t('import.existingUpdate') : t('import.existingLeave') });
  }
  out.push({ label: t('import.sumColumns'), value: columnsSummary() });
  return out;
});

function saveLast(c: LastChoices) {
  last.value = c;
  try { localStorage.setItem(LAST_KEY, JSON.stringify(c)); } catch { /* storage off */ }
}

function loadLast(name: string) {
  try {
    const c = JSON.parse(localStorage.getItem(LAST_KEY) || 'null') as LastChoices | null;
    if (c && c.name === name) last.value = c;
  } catch { /* storage off */ }
}

function onStart() {
  if (!file.value) return;
  starting.value = true;
  const subscribe = opts.mode === 'subscribe';
  const listIDs = subscribe ? opts.lists.map((l) => l.id as number) : [];
  importSubscribers({
    params: JSON.stringify({
      file_id: file.value.fileId,
      mode: opts.mode,
      // Single opt-in lists don't ask, so they keep the importer's default.
      subscription_status: subscribe && hasDoubleOptin.value && opts.consent === 'yes' ? 'confirmed' : '',
      lists: listIDs,
      overwrite_userinfo: subscribe && opts.existing === 'update',
      overwrite_subscription_status: false,
      columns: columnsReq(),
    }),
  }).then(() => {
    saveLast({
      name: file.value?.name || '',
      mode: opts.mode,
      existing: opts.existing,
      consent: opts.consent,
      doubleOptin: hasDoubleOptin.value,
      lists: listIDs,
    });
    started.value = true;
    optinsSent.value = false;
    problems.value = [];
    runStart.value = null;
    status.value = { status: 'importing', name: file.value?.name, total: check.value?.rows || 0 };
    pollStatus();
  }).finally(() => { starting.value = false; });
}

// --- Importing ---

// Rows written so far: committed plus those in the open transaction.
const written = computed(() => (status.value.imported || 0) + (status.value.pending || 0));
// Skipped and failed rows are done with too.
const processed = computed(() => written.value + (status.value.skipped || 0) + (status.value.failed || 0));

const progress = computed(() => {
  if (!(status.value.total > 0)) return 0;
  return Math.min(100, Math.ceil((processed.value / status.value.total) * 100));
});
// The bar eases in, which reads as quicker; it's full only once done.
const barValue = computed(() => (isDone() ? 100 : perceivedProgress(progress.value)));

const runTitle = computed(() => {
  const total = fmt(status.value.total);
  if (status.value.status === 'failed') return t('import.failedTitle');
  if (isDone()) return t('import.doneTitle', { num: fmt(written.value), total });
  return t('import.runningTitle', { num: fmt(processed.value), total });
});

// Time left from the pace so far, once there's enough of it to go on.
const timeLeft = computed(() => {
  const s = runStart.value;
  const total = status.value.total || 0;
  if (!s || processed.value <= s.done || processed.value < total * 0.05) return t('import.canLeave');
  const rate = (processed.value - s.done) / (Date.now() - s.at);
  const secs = Math.max(1, Math.round((total - processed.value) / rate / 1000));
  const mins = Math.ceil(secs / 60);
  const left = secs < 60 ? t('import.secondsLeft', { n: secs }, secs) : t('import.minutesLeft', { n: mins }, mins);
  return `${left} ${t('import.canLeave')}`;
});

const progressTiles = computed(() => {
  const s = status.value;
  const notImported = (s.skipped || 0) + (s.failed || 0);
  if (last.value?.mode === 'blocklist') {
    return [
      { label: t('import.blocked'), value: written.value, bad: false },
      { label: t('import.notImported'), value: notImported, bad: notImported > 0 },
    ];
  }
  const already = Math.max(0, (s.imported || 0) - (s.added || 0));
  return [
    { label: t('import.added'), value: s.added || 0, bad: false },
    { label: last.value?.existing === 'update' ? t('import.alreadyUpdated') : t('import.alreadyLeft'), value: already, bad: false },
    { label: t('import.notImported'), value: notImported, bad: notImported > 0 },
  ];
});

const needsConfirming = computed(() => !!last.value && last.value.mode === 'subscribe' && last.value.doubleOptin
  && last.value.consent === 'ask' && (status.value.added || 0) > 0);

function loadProblems() {
  getImportProblems().then((data) => { problems.value = data as unknown as ImportProblem[]; }, () => {});
}

function pollStatus() {
  window.clearInterval(pollID);
  const tick = () => getImportStatus().then((data) => {
    isLoading.value = false;
    isProcessing.value = false;
    const wasRunning = isRunning();
    status.value = data as unknown as Loose;

    if (isRunning() || isDone()) {
      if (!started.value) {
        started.value = true;
        step.value = 4;
      }
      if (!runStart.value && isRunning()) runStart.value = { at: Date.now(), done: processed.value };
      if (!last.value) loadLast(status.value.name);
      // Problems are listed up to a cap; fetch while the list can grow.
      if ((status.value.skipped || 0) > problems.value.length) loadProblems();
    }
    if (!isRunning()) {
      window.clearInterval(pollID);
      if (wasRunning) store.refresh();
    }
  }, () => {
    isLoading.value = false;
    isProcessing.value = false;
    status.value = { status: 'none' };
    window.clearInterval(pollID);
  });
  tick();
  pollID = window.setInterval(tick, POLL_MS);
}

function onConfirmStop() {
  confirmStop.value = false;
  isProcessing.value = true;
  stopImport().then(() => pollStatus());
}

// Clears the finished import so a new one can start.
function clearAndRestart() {
  isProcessing.value = true;
  stopImport().then(() => {
    isProcessing.value = false;
    status.value = { status: 'none' };
    problems.value = [];
    restart();
  });
}

function sendOptins() {
  sendingOptins.value = true;
  resendOptins({ list_ids: last.value?.lists || [] }).then((data) => {
    const queued = Number((data as unknown as Loose)?.queued) || 0;
    $utils.toast(t('import.confirmSending', { n: fmt(queued) }, queued));
    optinsSent.value = true;
  }).finally(() => { sendingOptins.value = false; });
}

function downloadLog() {
  getImportLogs().then((data) => {
    saveFile(String(data), `import-${status.value.name || 'log'}.txt`, 'text/plain');
  });
}

onMounted(() => {
  loadRecent();
  pollStatus();
  const ids = $utils.parseQueryIDs(route.query.list_id);
  if (ids.length > 0) {
    nextTick(() => {
      opts.lists = allLists.value.filter((l: Loose) => ids.indexOf(l.id) > -1);
    });
  }
});

onBeforeUnmount(() => window.clearInterval(pollID));
</script>

<style scoped lang="scss">
.imp-sample { font-size: 14px; color: var(--ln-text); text-decoration: underline; }

.imp-steps {
  display: flex;
  align-items: center;
  gap: 12px;
  list-style: none;
  margin: 0;
  padding: 0;

  li {
    display: inline-flex;
    flex: 1 1 0;
    align-items: center;
    gap: 10px;
    font-size: 14px;
    color: var(--ln-text-3);
    white-space: nowrap;

    // The line to the next step.
    &:not(:last-child)::after {
      content: '';
      flex: 1;
      height: 2px;
      margin-left: 6px;
      border-radius: 1px;
      background: var(--ln-border);
    }
    &:last-child { flex: none; }
    &.is-on { color: var(--ln-text); font-weight: 600; }
    &.is-done { color: var(--ln-text); }
    &.is-done::after { background: var(--ln-success-fg); }
  }

  // On a phone: the numbers, and the current step's name.
  @media (max-width: 767px) {
    gap: 8px;

    li { flex: none; }
    li::after, .imp-step-label { display: none; }
    li.is-on .imp-step-label { display: inline; }
  }
}
.imp-step-num {
  display: inline-grid;
  place-items: center;
  flex: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 1px solid var(--ln-border);
  background: var(--ln-card);
  font-size: 12px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;

  .pi { font-size: 10px; }
  .is-on & { background: var(--ln-ink); border-color: var(--ln-ink); color: var(--ln-on-ink); }
  .is-done & { background: var(--ln-success-fg); border-color: var(--ln-success-fg); color: var(--ln-card); }
}

.imp-upload {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 16px;
  align-items: start;

  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
}
.imp-drop-card { padding: 20px; }
.imp-drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-height: 248px;
  padding: 24px;
  border: 1.5px dashed var(--ln-text-4, var(--ln-text-3));
  border-radius: 10px;
  background: var(--ln-muted);
  color: var(--ln-text);
  font: inherit;
  text-align: center;
  cursor: pointer;

  strong { font-size: 16px; font-weight: 600; }
  .ln-small { margin-top: 6px; color: var(--ln-text-3); }
  &:hover, &.is-over { border-color: var(--ln-text-3); background: var(--ln-hover); }
  &:disabled { cursor: progress; }
}
.imp-drop-icon {
  display: inline-grid;
  place-items: center;
  width: 44px;
  height: 44px;
  margin-bottom: 10px;
  border-radius: 8px;
  background: var(--ln-card);

  .pi { font-size: 16px; }
}
.imp-drop-link { text-decoration: underline; font-weight: 500; }
.imp-file-input { display: none; }

.imp-recent {
  overflow: hidden;

  h2 { margin: 0; padding: 16px; font-size: 15px; font-weight: 600; border-bottom: 1px solid var(--ln-border); }
  ul { list-style: none; margin: 0; padding: 0; }
  li {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 14px 16px;
    font-size: 14px;

    & + li { border-top: 1px solid var(--ln-border); }
    .is-warn { color: var(--ln-warn-fg); }
    .is-bad { color: var(--ln-danger-fg); }
  }
}
.imp-recent-top { display: flex; justify-content: space-between; gap: 12px; }
.imp-recent-name { font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.imp-file {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
}
@media (max-width: 767px) {
  .imp-file { flex-wrap: wrap; }
  .imp-file .p-button { width: 100%; }
}
.imp-file-badge {
  display: inline-grid;
  place-items: center;
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 6px;
  background: var(--ln-muted);
  font-size: 11px;
  font-weight: 600;
}
.imp-file-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;

  .ln-mono { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ln-small { color: var(--ln-text-3); }
}

.imp-card { overflow: hidden; }
.imp-card-head {
  padding: 18px 20px;
  border-bottom: 1px solid var(--ln-border);

  h2 { margin: 0; font-size: 16px; font-weight: 600; }
  p { margin: 2px 0 0; }
}
.imp-run-head {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: baseline;
  gap: 4px 16px;
  padding-bottom: 0;
  border-bottom: 0;

  .ln-small { color: var(--ln-text-3); }
}

.imp-match-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 2.4fr) 240px;
  gap: 16px;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid var(--ln-border);

  &.is-skipped .imp-match-name, &.is-skipped .imp-match-samples { color: var(--ln-text-3); }

  @media (max-width: 767px) {
    grid-template-columns: minmax(0, 1fr);
    gap: 6px;

    &.imp-match-head { display: none; }
  }
}
.imp-match-head {
  padding-top: 10px;
  padding-bottom: 10px;
  background: var(--ln-muted);
  font-size: 13px;
  font-weight: 600;
}
.imp-match-name { font-weight: 500; overflow-wrap: anywhere; }
.imp-match-samples { overflow: hidden; color: var(--ln-text-2); text-overflow: ellipsis; white-space: nowrap; }
.imp-match-select { width: 100%; }

.imp-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
.imp-match + .imp-foot { border-top: 0; }
.imp-foot-end {
  display: flex;
  align-items: center;
  gap: 16px;

  .ln-small { color: var(--ln-text-3); }
}

.imp-q {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
  padding: 18px 20px;

  & + & { border-top: 1px solid var(--ln-border); }
  h2 { margin: 0; font-size: 15px; font-weight: 600; }
}
.imp-modes { align-self: flex-start; }
// The shared list picker: our heading instead of its label, chosen lists
// under the field.
.imp-lists :deep(.list-selector) {
  display: flex;
  flex-direction: column-reverse;
  gap: 8px;

  label { display: none; }
  .field { margin: 0; }
}
.imp-choices {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;

  @media (max-width: 767px) { grid-template-columns: minmax(0, 1fr); }
}
.imp-choice {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--ln-border);
  border-radius: var(--ln-radius-card, 10px);
  cursor: pointer;

  > span { display: flex; flex-direction: column; gap: 2px; }
  strong { font-weight: 600; }
  .ln-small { color: var(--ln-text-2); }
  // A 2px ring drawn as border + inset shadow, so selecting doesn't shift.
  &.is-on { border-color: var(--ln-ink); box-shadow: inset 0 0 0 1px var(--ln-ink); }
}

.imp-body { display: flex; flex-direction: column; gap: 16px; padding: 18px 20px; }
.imp-tiles { grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); }
.imp-bad { color: var(--ln-danger-fg); }
.imp-summary {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 8px 16px;
  margin: 0;

  dt { color: var(--ln-text-3); }
  dd { margin: 0; }

  @media (max-width: 767px) {
    grid-template-columns: minmax(0, 1fr);
    gap: 2px;

    dd { margin-bottom: 8px; }
  }
}
.imp-bar { height: 8px; }
.imp-note {
  margin: 0;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 13px;

  &.is-bad { background: var(--ln-danger-bg); color: var(--ln-danger-fg); }
}

.imp-confirm {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;

  > span { display: flex; flex-direction: column; gap: 2px; }
  .ln-small { color: var(--ln-text-2); }

  @media (max-width: 767px) { flex-direction: column; align-items: stretch; }
}

.imp-run-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;

  a { color: var(--ln-text-2); text-decoration: underline; }
}
</style>
