<template>
  <div class="ln-page imp">
    <page-header :title="$t('import.title')" :subtitle="$t('import.intro')" />

    <PvSkeleton v-if="isLoading" height="320px" />

    <!-- Steps 1 and 2: pick a file, then set options -->
    <template v-else-if="isFree()">
      <ol class="imp-steps" :aria-label="$t('import.steps')">
        <li v-for="(label, i) in stepLabels" :key="label" :class="{ 'is-on': step === i + 1, 'is-done': step > i + 1 }">
          <span class="imp-step-num">
            <i v-if="step > i + 1" class="pi pi-check" aria-hidden="true" />
            <template v-else>{{ i + 1 }}</template>
          </span>
          {{ label }}
        </li>
      </ol>

      <div v-if="step === 1" class="imp-layout" :class="stepAnim">
        <section class="ln-card ln-card-pad imp-main">
          <h2 class="ln-card-title">{{ $t('import.csvFile') }}</h2>
          <button
            type="button"
            class="imp-drop"
            :class="{ 'is-over': dragOver }"
            @dragover.prevent="dragOver = true"
            @dragleave="dragOver = false"
            @drop.prevent="onFileDrop"
            @click="fileInputEl?.click()"
          >
            <i class="pi pi-cloud-upload" aria-hidden="true" />
            <span class="imp-drop-title">{{ $t('import.csvFileHelp') }}</span>
            <span class="ln-small">{{ $t('import.fileTypes') }}</span>
          </button>
          <input ref="fileInputEl" type="file" :aria-label="$t('import.csvFile')" accept=".csv,.zip,text/csv,application/zip" class="imp-file-input" @change="onFileSelect">
        </section>

        <aside class="ln-card ln-card-pad imp-help">
          <h2 class="ln-card-title">{{ $t('import.instructions') }}</h2>
          <p class="ln-small">{{ $t('import.instructionsHelp') }}</p>
          <code class="ln-mono imp-headers">email, name, attributes</code>
          <h3 class="ln-field-label">{{ $t('import.csvExample') }}</h3>
          <pre class="ln-code imp-example" v-text="example" />
        </aside>
      </div>

      <form v-else class="ln-card imp-options" :class="stepAnim" @submit.prevent="onUpload">
        <div class="imp-file">
          <i class="pi pi-file" aria-hidden="true" />
          <span class="imp-file-name">
            <strong>{{ form.file?.name }}</strong>
            <span class="ln-small">{{ fileSize }}</span>
          </span>
          <button type="button" class="ln-link-btn" @click="changeFile">{{ $t('import.changeFile') }}</button>
        </div>

        <div class="imp-body">
          <div class="ln-field">
            <span id="imp-mode" class="ln-field-label">{{ $t('import.mode') }}</span>
            <div class="imp-modes" role="radiogroup" aria-labelledby="imp-mode">
              <label class="imp-mode" :class="{ 'is-on': form.mode === 'subscribe' }">
                <PvRadioButton v-model="form.mode" name="mode" value="subscribe" data-cy="check-subscribe" />
                <span>
                  <strong>{{ $t('import.subscribe') }}</strong>
                  <span class="ln-small">{{ $t('import.subscribeHelp') }}</span>
                </span>
              </label>
              <label class="imp-mode" :class="{ 'is-on': form.mode === 'blocklist' }">
                <PvRadioButton v-model="form.mode" name="mode" value="blocklist" data-cy="check-blocklist" />
                <span>
                  <strong>{{ $t('import.blocklist') }}</strong>
                  <span class="ln-small">{{ $t('import.blocklistHelp') }}</span>
                </span>
              </label>
            </div>
          </div>

          <div class="ln-fields-2">
            <div class="ln-field">
              <span id="imp-status" class="ln-field-label">{{ $t('import.subStatus') }}</span>
              <div class="imp-radios" role="radiogroup" aria-labelledby="imp-status">
                <template v-if="form.mode === 'subscribe'">
                  <label class="ln-check-row">
                    <PvRadioButton v-model="form.subStatus" name="subStatus" value="unconfirmed" data-cy="check-unconfirmed" />
                    {{ $t('subscribers.status.unconfirmed') }}
                  </label>
                  <label class="ln-check-row">
                    <PvRadioButton v-model="form.subStatus" name="subStatus" value="confirmed" data-cy="check-confirmed" />
                    {{ $t('subscribers.status.confirmed') }}
                  </label>
                </template>
                <label v-else class="ln-check-row">
                  <PvRadioButton v-model="form.subStatus" name="subStatus" value="unsubscribed" data-cy="check-unsubscribed" />
                  {{ $t('subscribers.status.unsubscribed') }}
                </label>
              </div>
            </div>
            <div class="ln-field">
              <label for="imp-delim">{{ $t('import.csvDelim') }}</label>
              <PvInputText id="imp-delim" v-model="form.delim" name="delim" placeholder="," :maxlength="1" required class="ln-mono imp-delim" />
              <p class="ln-help">{{ $t('import.csvDelimHelp') }}</p>
            </div>
          </div>

          <template v-if="form.mode === 'subscribe'">
            <list-selector
              v-model="form.lists"
              :label="$t('globals.terms.lists')"
              :placeholder="$t('import.listSubHelp')"
              :message="$t('import.listSubHelp')"
              :selected="form.lists"
              :all="allLists"
            />

            <label class="imp-toggle">
              <PvToggleSwitch v-model="form.overwriteUserInfo" name="overwriteUserInfo" data-cy="overwrite-user-info" />
              <span>
                <strong>{{ $t('import.overwriteUserInfo') }}</strong>
                <span class="ln-small">{{ $t('import.overwriteUserInfoHelp') }}</span>
              </span>
            </label>
            <label class="imp-toggle">
              <PvToggleSwitch v-model="form.overwriteSubStatus" name="overwriteSubStatus" data-cy="overwrite-sub-status" />
              <span>
                <strong>{{ $t('import.overwriteSubStatus') }}</strong>
                <span class="ln-small">{{ $t('import.overwriteSubStatusHelp') }}</span>
              </span>
            </label>
          </template>
        </div>

        <footer class="imp-foot">
          <PvButton :label="$t('globals.buttons.back')" severity="secondary" outlined @click="changeFile" />
          <PvButton
            type="submit"
            icon="pi pi-upload"
            data-cy="btn-import"
            :disabled="!form.file || (form.mode === 'subscribe' && form.lists.length === 0)"
            :loading="isProcessing"
            :label="$t('import.start')"
          />
        </footer>
      </form>
    </template>

    <!-- Step 3: progress and log -->
    <section v-else-if="isRunning() || isDone()" class="ln-card imp-run ln-step-next">
      <div class="ln-card-head">
        <h2 class="ln-card-title">{{ runTitle }}</h2>
        <status-tag :status="status.status" :label="statusLabel" data-cy="import-status" />
      </div>
      <div class="imp-body">
        <PvProgressBar :value="progress" :show-value="false" class="imp-bar" />
        <p class="ln-small imp-count">
          <span v-if="status.name" class="ln-mono">{{ status.name }}</span>
          {{ $t('import.recordsCount', { num: fmt(written), total: fmt(status.total) }) }}
        </p>

        <div class="ln-tiles">
          <div class="ln-tile">
            <span class="ln-tile-label">{{ $t('import.added') }}</span>
            <span class="ln-tile-value">{{ fmt(written) }}</span>
          </div>
          <div class="ln-tile">
            <span class="ln-tile-label">{{ $t('import.skipped') }}</span>
            <span class="ln-tile-value">{{ fmt(status.skipped) }}</span>
            <span v-if="status.skipped" class="ln-small">{{ $t('import.skippedHelp') }}</span>
          </div>
          <div class="ln-tile">
            <span class="ln-tile-label">{{ $t('import.failed') }}</span>
            <span class="ln-tile-value" :class="{ 'imp-bad': status.failed }">{{ fmt(status.failed) }}</span>
            <span v-if="status.failed" class="ln-small">{{ $t('import.failedHelp') }}</span>
          </div>
        </div>

        <div class="imp-log-head">
          <span class="ln-field-label">{{ $t('import.log') }}</span>
          <button type="button" class="ln-link-btn" :disabled="!logs.length" @click="downloadLog">
            <i class="pi pi-download" aria-hidden="true" /> {{ $t('import.downloadLog') }}
          </button>
        </div>
        <div id="import-log" class="ln-code imp-log">
          <log-view :lines="logs" :loading="false" />
        </div>
      </div>
      <footer class="imp-foot">
        <PvButton
          v-if="isDone()"
          :label="$t('import.importDone')"
          :loading="isProcessing"
          data-cy="btn-import-done"
          @click="onStopImport"
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
      </footer>
    </section>

    <confirm-action
      v-model:visible="confirmStop"
      :title="$t('import.stopTitle')"
      :text="$t('import.stopText')"
      :confirm-label="$t('import.stopImport')"
      danger
      :busy="isProcessing"
      @confirm="onConfirmStop"
    />
    <confirm-action
      v-model:visible="confirmOverwrite"
      :title="$t('import.overwriteSubStatus')"
      :text="$t('import.subscribeWarning')"
      :confirm-label="$t('import.start')"
      @confirm="onConfirmOverwrite"
    />
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import ListSelector from '../components/ListSelector.vue';
import LogView from '../components/LogView.vue';
import PageHeader from '../components/ui/PageHeader.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getImport } from '../api/generated/endpoints/import/import';

// Import status and list rows are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// How often to poll the import status while it runs.
const POLL_MS = 250;

const { $utils } = useGlobal();
const {
  getImportLogs, getImportStatus, stopImport, importSubscribers,
} = getImport();
const { t, n } = useI18n();
const route = useRoute();
const store = useMainStore();
const { lists } = storeToRefs(store);

const form = reactive({
  mode: 'subscribe',
  subStatus: 'unconfirmed',
  delim: ',',
  lists: [] as Loose[],
  overwriteUserInfo: false,
  overwriteSubStatus: false,
  file: null as File | null,
});

const step = ref(1);
// The step that appears slides in from the side the user moved towards
// (motion.scss); nothing on first load, where the page itself animates in.
const stepAnim = ref('');
const dragOver = ref(false);
const isLoading = ref(true);
const isProcessing = ref(false);
const confirmStop = ref(false);
const confirmOverwrite = ref(false);
const status = ref<Loose>({ status: '' });
const logs = ref<string[]>([]);
const example = ref('');
const fileInputEl = ref<HTMLInputElement | null>(null);
let pollID: number | undefined;

function fmt(v: unknown): string {
  return n(Number(v) || 0);
}

function isFree() { return status.value.status === 'none'; }
function isRunning() { return status.value.status === 'importing' || status.value.status === 'stopping'; }
function isDone() {
  return status.value.status === 'finished' || status.value.status === 'stopped' || status.value.status === 'failed';
}

const stepLabels = computed(() => [t('import.stepUpload'), t('import.stepOptions'), t('import.stepImport')]);

// Rows written so far: committed plus those in the open transaction
// (pending), so the count moves within a batch. Once the import ends,
// pending is 0 and this is the committed count.
const written = computed(() => (status.value.imported || 0) + (status.value.pending || 0));

const progress = computed(() => {
  if (!status.value || !(status.value.total > 0)) return 0;
  // Skipped and failed rows are done with too, so they count as progress.
  const done = written.value + (status.value.skipped || 0) + (status.value.failed || 0);
  return Math.min(100, Math.ceil((done / status.value.total) * 100));
});

const allLists = computed<Loose[]>(() => (lists.value as Loose).results || []);

const fileSize = computed(() => {
  const size = form.file?.size || 0;
  if (size < 1024 * 1024) return `${n(Math.max(1, Math.round(size / 1024)))} KB`;
  return `${n(Math.round((size / (1024 * 1024)) * 10) / 10)} MB`;
});

const statusLabel = computed(() => {
  const key = `import.status.${status.value.status}`;
  const label = t(key);
  return label === key ? status.value.status : label;
});

const runTitle = computed(() => (isDone()
  ? t('import.doneTitle')
  : t('import.runningTitle', { num: fmt(written.value), total: fmt(status.value.total) })));

watch(() => form.mode, () => {
  nextTick(() => {
    form.subStatus = form.mode === 'subscribe' ? 'unconfirmed' : 'unsubscribed';
  });
});

function pickFile(f: File | undefined) {
  if (!f) return;
  form.file = f;
  stepAnim.value = 'ln-step-next';
  step.value = 2;
}

function onFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  pickFile(target.files?.[0]);
  target.value = '';
}

function onFileDrop(e: DragEvent) {
  dragOver.value = false;
  pickFile(e.dataTransfer?.files?.[0]);
}

function changeFile() {
  form.file = null;
  stepAnim.value = 'ln-step-prev';
  step.value = 1;
}

function getLogs() {
  getImportLogs().then((data) => {
    logs.value = String(data).split('\n').map((line: string) => line.replace(/\s+importer\.go:\d+:\s*/, ' *: '));
    nextTick(() => {
      const el = document.getElementById('import-log');
      if (el) el.scrollTop = el.scrollHeight;
    });
  });
}

function pollStatus() {
  window.clearInterval(pollID);
  pollID = window.setInterval(() => {
    getImportStatus().then((data) => {
      isProcessing.value = false;
      isLoading.value = false;
      const wasRunning = isRunning();
      status.value = data as unknown as Loose;
      getLogs();
      if (wasRunning && !isRunning()) {
        window.clearInterval(pollID);
        store.refresh();
      } else if (!isRunning()) {
        window.clearInterval(pollID);
      }
    }, () => {
      isProcessing.value = false;
      isLoading.value = false;
      status.value = { status: 'none' };
      window.clearInterval(pollID);
    });
  }, POLL_MS);
}

// Stops a running import, or clears a finished one so a new one can start.
function onStopImport() {
  isProcessing.value = true;
  stopImport().then(() => {
    pollStatus();
    changeFile();
  });
}

function onConfirmStop() {
  confirmStop.value = false;
  onStopImport();
}

function downloadLog() {
  const blob = new Blob([logs.value.join('\n')], { type: 'text/plain' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `import-${status.value.name || 'log'}.txt`;
  a.click();
  URL.revokeObjectURL(a.href);
}

function renderExample() {
  example.value = 'email,name,attributes\n'
    + 'user1@mail.com,"User One","{""age"": 42, ""planet"": ""Mars""}"\n'
    + 'user2@mail.com,"User Two","{""age"": 24, ""job"": ""Time Traveller""}"';
}

function onSubmit() {
  isProcessing.value = true;
  importSubscribers({
    file: form.file as Blob,
    params: JSON.stringify({
      mode: form.mode,
      subscription_status: form.subStatus,
      delim: form.delim,
      lists: form.lists.map((l) => l.id),
      overwrite_userinfo: form.overwriteUserInfo,
      overwrite_subscription_status: form.overwriteSubStatus,
    }),
  }).then(() => {
    $utils.toast(t('import.importStarted'));
    pollStatus();
  }, () => {
    isProcessing.value = false;
    changeFile();
  });
}

function onConfirmOverwrite() {
  confirmOverwrite.value = false;
  onSubmit();
}

function onUpload() {
  if (form.mode === 'subscribe' && form.overwriteSubStatus) {
    confirmOverwrite.value = true;
    return;
  }
  onSubmit();
}

onMounted(() => {
  renderExample();
  pollStatus();
  const ids = $utils.parseQueryIDs(route.query.list_id);
  if (ids.length > 0 && (lists.value as Loose).results) {
    nextTick(() => {
      form.lists = (lists.value as Loose).results.filter((l: Loose) => ids.indexOf(l.id) > -1);
    });
  }
});

onBeforeUnmount(() => window.clearInterval(pollID));
</script>

<style scoped lang="scss">
.imp-steps {
  display: flex;
  gap: 8px 24px;
  flex-wrap: wrap;
  list-style: none;
  margin: 0;
  padding: 0;

  li {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--ln-text-3);
    &.is-on { color: var(--ln-text); font-weight: 500; }
    &.is-done { color: var(--ln-text-2); }
  }
}
.imp-step-num {
  display: inline-grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1px solid var(--ln-border);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  transition: background-color var(--ln-dur-base) var(--ln-ease), border-color var(--ln-dur-base) var(--ln-ease), color var(--ln-dur-base) var(--ln-ease);
  .pi { font-size: 10px; animation: ln-pop-in var(--ln-dur-base) var(--ln-ease); }
  .is-on & { background: var(--ln-ink); border-color: var(--ln-ink); color: var(--ln-on-ink); }
  .is-done & { background: var(--ln-muted); }
}
.imp-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  @media (max-width: 900px) { grid-template-columns: minmax(0, 1fr); }
}
.imp-main, .imp-help { display: flex; flex-direction: column; gap: 12px; }
.imp-help .ln-small, .imp-help h3 { margin: 0; }
.imp-drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 220px;
  padding: 24px;
  border: 1.5px dashed var(--ln-border);
  border-radius: 10px;
  background: var(--ln-muted);
  color: var(--ln-text);
  font: inherit;
  cursor: pointer;
  transition: border-color var(--ln-dur-fast) var(--ln-ease), background-color var(--ln-dur-fast) var(--ln-ease);

  .pi { font-size: 28px; color: var(--ln-text-3); margin-bottom: 4px; transition: translate var(--ln-dur-base) var(--ln-ease); }
  &:hover, &.is-over { border-color: var(--ln-text-3); background: var(--ln-hover); }
  // Dragging a file over lifts the upload icon.
  &.is-over .pi { translate: 0 -3px; }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
}
.imp-drop-title { font-weight: 500; }
.imp-file-input { display: none; }
.imp-headers { font-size: 13px; padding: 8px 10px; border-radius: 6px; background: var(--ln-muted); }
.imp-example { white-space: pre; font-size: 12px; }
.imp-options { display: flex; flex-direction: column; max-width: 760px; }
.imp-run { display: flex; flex-direction: column; }
.imp-file {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--ln-border);
  > .pi { font-size: 18px; color: var(--ln-text-3); }
}
.imp-file-name {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  strong { overflow-wrap: anywhere; font-weight: 500; }
}
.imp-body { padding: 18px; display: flex; flex-direction: column; gap: 18px; }
.imp-modes { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px; }
.imp-mode, .imp-toggle {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  cursor: pointer;
  > span { display: flex; flex-direction: column; gap: 2px; font-size: 14px; }
  strong { font-weight: 500; }
}
.imp-mode {
  padding: 12px 14px;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  &.is-on { border-color: var(--ln-text-2); background: var(--ln-hover); }
}
.imp-radios { display: flex; gap: 8px 18px; flex-wrap: wrap; min-height: 36px; align-items: center; }
.imp-delim { width: 72px; }
.imp-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
  border-radius: 0 0 var(--ln-radius-card) var(--ln-radius-card);
}
.imp-bar { height: 8px; }
.imp-bad { color: var(--ln-danger-fg); }
.imp-count { margin: -8px 0 0; display: flex; gap: 10px; flex-wrap: wrap; }
.imp-log-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: -10px; }
.imp-log {
  max-height: 320px;
  overflow: auto;
  padding: 10px 14px;
}
</style>
