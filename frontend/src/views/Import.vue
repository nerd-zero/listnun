<template>
  <div class="import-page">
    <div class="page-header">
      <h1 class="page-title">{{ $t('import.title') }}</h1>
    </div>

    <div v-if="isLoading" class="flex justify-center p-8">
      <PvProgressSpinner />
    </div>

    <div v-if="isFree()" class="import-layout">
      <!-- Main form card -->
      <form @submit.prevent="onUpload" class="box import-form">
        <!-- Mode & Status row -->
        <div class="import-row">
          <div class="import-field">
            <label class="field-label">{{ $t('import.mode') }}</label>
            <div class="radio-group">
              <label class="radio-option">
                <PvRadioButton v-model="form.mode" name="mode" value="subscribe" data-cy="check-subscribe" />
                <span>{{ $t('import.subscribe') }}</span>
              </label>
              <label class="radio-option">
                <PvRadioButton v-model="form.mode" name="mode" value="blocklist" data-cy="check-blocklist" />
                <span>{{ $t('import.blocklist') }}</span>
              </label>
            </div>
          </div>

          <div class="import-field">
            <label class="field-label">{{ $t('globals.fields.status') }}</label>
            <div class="radio-group">
              <template v-if="form.mode === 'subscribe'">
                <label class="radio-option">
                  <PvRadioButton v-model="form.subStatus" name="subStatus" value="unconfirmed" data-cy="check-unconfirmed" />
                  <span>{{ $t('subscribers.status.unconfirmed') }}</span>
                </label>
                <label class="radio-option">
                  <PvRadioButton v-model="form.subStatus" name="subStatus" value="confirmed" data-cy="check-confirmed" />
                  <span>{{ $t('subscribers.status.confirmed') }}</span>
                </label>
              </template>
              <label v-else class="radio-option">
                <PvRadioButton v-model="form.subStatus" name="subStatus" value="unsubscribed" data-cy="check-unsubscribed" />
                <span>{{ $t('subscribers.status.unsubscribed') }}</span>
              </label>
            </div>
          </div>

          <div class="import-field import-field--narrow">
            <label class="field-label">{{ $t('import.csvDelim') }}</label>
            <PvInputText v-model="form.delim" name="delim" placeholder="," :maxlength="1" required style="width:80px" />
            <small class="field-help">{{ $t('import.csvDelimHelp') }}</small>
          </div>
        </div>

        <!-- Overwrite options (subscribe mode only) -->
        <div v-if="form.mode === 'subscribe'" class="import-row">
          <div class="import-field">
            <div class="toggle-field">
              <div class="toggle-field-header">
                <PvToggleSwitch v-model="form.overwriteUserInfo" name="overwriteUserInfo" data-cy="overwrite-user-info" />
                <label class="field-label" style="margin-bottom:0">{{ $t('import.overwriteUserInfo') }}</label>
              </div>
              <small class="field-help">{{ $t('import.overwriteUserInfoHelp') }}</small>
            </div>
          </div>
          <div class="import-field">
            <div class="toggle-field">
              <div class="toggle-field-header">
                <PvToggleSwitch v-model="form.overwriteSubStatus" name="overwriteSubStatus" data-cy="overwrite-sub-status" />
                <label class="field-label" style="margin-bottom:0">{{ $t('import.overwriteSubStatus') }}</label>
              </div>
              <small class="field-help">{{ $t('import.overwriteSubStatusHelp') }}</small>
            </div>
          </div>
          <div class="import-field import-field--narrow" />
        </div>

        <!-- Lists selector -->
        <div v-if="form.mode === 'subscribe'" class="field" style="margin-bottom:1.25rem">
          <list-selector :label="$t('globals.terms.lists')"
            :placeholder="$t('import.listSubHelp')" :message="$t('import.listSubHelp')" v-model="form.lists"
            :selected="form.lists" :all="lists.results" />
        </div>

        <!-- File upload -->
        <div class="field" style="margin-bottom:0.75rem">
          <label class="field-label">{{ $t('import.csvFile') }}</label>
          <div class="upload-drop-area" @dragover.prevent @drop.prevent="onFileDrop" @click="fileInputEl?.click()">
            <i class="pi pi-cloud-upload upload-icon" />
            <p class="upload-label">{{ $t('import.csvFileHelp') }}</p>
            <input ref="fileInputEl" type="file" style="display:none" @change="onFileSelect" />
          </div>
        </div>

        <div class="import-footer">
          <div class="file-tag" v-if="form.file">
            <PvTag :value="form.file.name" severity="secondary" />
            <PvButton icon="pi pi-times" severity="secondary" size="small" text rounded @click="clearFile" />
          </div>
          <PvButton type="submit" severity="primary" icon="pi pi-upload"
            :disabled="!form.file || (form.mode === 'subscribe' && form.lists.length === 0)"
            :loading="isProcessing" :label="$t('import.upload')" />
        </div>
      </form>

      <!-- Instructions card -->
      <div class="box import-help">
        <h5 class="import-help-title">{{ $t('import.instructions') }}</h5>
        <p class="import-help-text">{{ $t('import.instructionsHelp') }}</p>
        <div class="csv-headers">
          <code><span>email,</span> <span>name,</span> <span>attributes</span></code>
        </div>

        <PvDivider />

        <h5 class="import-help-title">{{ $t('import.csvExample') }}</h5>
        <pre class="csv-example" v-text="example" />
      </div>
    </div>

    <section v-if="isRunning() || isDone()" class="import-status">
      <PvProgressBar :value="progress" style="height:6px;width:100%" />
      <p :class="['import-status-text', { 'import-status-text--success': status.status === 'finished', 'import-status-text--danger': (status.status === 'failed' || status.status === 'stopped') }]">
        {{ status.status }}
      </p>
      <p class="import-count">{{ $t('import.recordsCount', { num: status.imported, total: status.total }) }}</p>
      <p v-if="status.risky" class="import-count import-count--risky">
        {{ $t('import.riskyCount', { num: status.risky }) }}
      </p>
      <PvButton @click="onStopImport" :loading="isProcessing" icon="pi pi-upload" severity="primary"
        :label="isDone() ? $t('import.importDone') : $t('import.stopImport')" />
      <div class="import-logs">
        <log-view :lines="logs" :loading="false" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, onMounted, nextTick,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import ListSelector from '../components/ListSelector.vue';
import LogView from '../components/LogView.vue';
import { getImport } from '../api/generated/endpoints/import/import';

const { $utils } = useGlobal();
const {
  getImportLogs, getImportStatus, stopImport, importSubscribers,
} = getImport();
const { t } = useI18n();
const route = useRoute();
const store = useMainStore();
const { lists } = storeToRefs(store);

const form = reactive({
  mode: 'subscribe',
  subStatus: 'unconfirmed',
  delim: ',',
  lists: [] as any[],
  overwriteUserInfo: false,
  overwriteSubStatus: false,
  file: null as File | null,
});

const isLoading = ref(true);
const isProcessing = ref(false);
const status = ref<any>({ status: '' });
const logs = ref<string[]>([]);
const pollID = ref<any>(null);
const example = ref('');
const fileInputEl = ref<HTMLInputElement | null>(null);

const progress = computed(() => {
  if (!status.value || !status.value.total > 0) return 0;
  return Math.ceil((status.value.imported / status.value.total) * 100);
});

watch(() => form.mode, () => {
  nextTick(() => {
    form.subStatus = form.mode === 'subscribe' ? 'unconfirmed' : 'unsubscribed';
  });
});

function clearFile() { form.file = null; }

function onFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  if (target.files && target.files.length > 0) {
    [form.file] = target.files as any;
  }
}

function onFileDrop(e: DragEvent) {
  if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
    [form.file] = e.dataTransfer.files as any;
  }
}

function isFree() { return status.value.status === 'none'; }
function isRunning() { return status.value.status === 'importing' || status.value.status === 'stopping'; }
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function isSuccessful() { return status.value.status === 'finished'; }
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function isFailed() { return status.value.status === 'stopped' || status.value.status === 'failed'; }
function isDone() {
  return status.value.status === 'finished' || status.value.status === 'stopped' || status.value.status === 'failed';
}

function getLogs() {
  getImportLogs().then((data: any) => {
    logs.value = data.split('\n').map((line: string) => line.replace(/\s+importer\.go:\d+:\s*/, ' *: '));
    nextTick(() => {
      const el = document.getElementById('import-log');
      if (el) el.scrollTop = el.scrollHeight;
    });
  });
}

function pollStatus() {
  clearInterval(pollID.value);
  pollID.value = setInterval(() => {
    getImportStatus().then((data: any) => {
      isProcessing.value = false;
      isLoading.value = false;
      const wasRunning = isRunning();
      status.value = data;
      getLogs();
      if (wasRunning && !isRunning()) {
        clearInterval(pollID.value);
        store.refresh();
      } else if (!isRunning()) {
        clearInterval(pollID.value);
      }
    }, () => {
      isProcessing.value = false;
      isLoading.value = false;
      status.value = { status: 'none' };
      clearInterval(pollID.value);
    });
    return true;
  }, 250);
}

function onStopImport() {
  isProcessing.value = true;
  stopImport().then(() => { pollStatus(); form.file = null; });
}

function renderExample() {
  example.value = 'email,name,attributes\n'
    + 'user1@mail.com,"User One","{""age"": 42, ""planet"": ""Mars""}"\n'
    + 'user2@mail.com,"User Two","{""age"": 24, ""job"": ""Time Traveller""}"';
}

function resetForm() {
  form.mode = 'subscribe';
  form.overwriteUserInfo = false;
  form.overwriteSubStatus = false;
  form.file = null;
  form.lists = [];
  form.subStatus = 'unconfirmed';
  form.delim = ',';
}

function onSubmit() {
  isProcessing.value = true;
  importSubscribers({
    file: form.file as Blob,
    params: JSON.stringify({
      mode: form.mode,
      subscription_status: form.subStatus,
      delim: form.delim,
      lists: form.lists.map((l: any) => l.id),
      overwrite_userinfo: form.overwriteUserInfo,
      overwrite_subscription_status: form.overwriteSubStatus,
    }),
  }).then(() => {
    $utils.toast(t('import.importStarted'));
    pollStatus();
  }, () => {
    isProcessing.value = false;
    form.file = null;
  });
}

function onUpload() {
  if (form.mode === 'subscribe' && form.overwriteSubStatus) {
    $utils.confirm(t('import.subscribeWarning'), onSubmit, resetForm);
    return;
  }
  onSubmit();
}

onMounted(() => {
  renderExample();
  pollStatus();
  const ids = $utils.parseQueryIDs(route.query.list_id);
  if (ids.length > 0 && (lists.value as any).results) {
    nextTick(() => {
      form.lists = (lists.value as any).results.filter((l: any) => ids.indexOf(l.id) > -1);
    });
  }
});
</script>

<style scoped lang="scss">
.import-page { display: flex; flex-direction: column; gap: 1.5rem; }

.import-layout {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 1.5rem;
  align-items: start;
}

.import-form { display: flex; flex-direction: column; gap: 1.25rem; }

/* Horizontal row of fields */
.import-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 1.5rem;
  padding-bottom: 1.25rem;
  border-bottom: 1px solid var(--lm-border);
}

.import-field { display: flex; flex-direction: column; }
.import-field--narrow { max-width: 160px; }

.field-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--lm-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.6rem;
  display: block;
}
.field-help { font-size: 0.8rem; color: var(--lm-text-muted); margin-top: 0.35rem; }

.radio-group { display: flex; flex-direction: column; gap: 0.6rem; }
.radio-option { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; font-size: 0.9rem; }

.toggle-field-header { display: flex; align-items: center; gap: 0.6rem; margin-bottom: 0.35rem; }

.upload-drop-area {
  border: 2px dashed var(--lm-border);
  border-radius: 10px;
  cursor: pointer;
  padding: 2.5rem 2rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  transition: border-color 0.15s, background 0.15s;

  &:hover {
    border-color: var(--lm-primary);
    background: var(--lm-primary-light);
  }
}
.upload-icon { font-size: 2rem; color: var(--lm-text-muted); }
.upload-label { font-size: 0.875rem; color: var(--lm-text-muted); margin: 0; }

.import-footer {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding-top: 0.25rem;
}
.file-tag { display: flex; align-items: center; gap: 0.25rem; flex: 1; }

/* Instructions card */
.import-help { display: flex; flex-direction: column; gap: 0.75rem; }
.import-help-title { font-size: 0.9rem; font-weight: 600; color: var(--lm-text); margin: 0; }
.import-help-text { font-size: 0.875rem; color: var(--lm-text-muted); margin: 0; line-height: 1.6; }

.csv-headers {
  background: var(--lm-bg-subtle);
  border: 1px solid var(--lm-border);
  border-radius: 6px;
  padding: 0.6rem 0.85rem;
  font-size: 0.8rem;

  code { font-family: 'Fira Code', 'Cascadia Code', monospace; color: var(--lm-primary); }
  span { margin-right: 0.15rem; }
}

.csv-example {
  background: var(--lm-bg-subtle);
  border: 1px solid var(--lm-border);
  border-radius: 6px;
  padding: 0.75rem 1rem;
  font-size: 0.78rem;
  font-family: 'Fira Code', 'Cascadia Code', monospace;
  color: var(--lm-text);
  margin: 0;
  white-space: pre;
  overflow-x: auto;
}

:deep(.p-tag-secondary) {
  background: var(--lm-bg-subtle);
  color: var(--lm-text);
  border: 1px solid var(--lm-border);
}

.import-status {
  background: var(--lm-surface); border: 1px solid var(--lm-border); border-radius: 12px;
  padding: 2rem; display: flex; flex-direction: column; align-items: center; gap: 1rem;
  text-align: center;
}
.import-status-text { font-size: 1.25rem; font-weight: 600; text-transform: capitalize; color: var(--lm-text); margin: 0; }
.import-status-text--success { color: #16a34a; }
.import-status-text--danger { color: #dc2626; }
.import-count { font-size: 0.875rem; color: var(--lm-text-muted); margin: 0; }
.import-count--risky { color: #b91c1c; }
.import-logs { width: 100%; }
</style>
