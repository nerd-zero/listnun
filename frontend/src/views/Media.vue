<template>
  <section class="ln-page media" :class="{ 'is-modal': isModal }">
    <page-header :title="$t('media.title')" :count="countLabel">
      <template #search>
        <form class="ln-search" role="search" @submit.prevent="onQueryMedia">
          <PvIconField>
            <PvInputIcon class="pi pi-search" />
            <PvInputText v-model="queryParams.query" :placeholder="$t('media.search')" :aria-label="$t('media.search')" data-cy="query" />
          </PvIconField>
        </form>
      </template>
      <PvButton v-if="canManage" icon="pi pi-upload" :label="$t('media.upload')" data-cy="btn-toggle-upload" @click="pick" />
    </page-header>

    <div v-if="canManage" class="media-top">
      <div
        class="media-drop"
        :class="{ 'is-over': dragOver }"
        role="button"
        tabindex="0"
        :aria-label="$t('media.dropLabel')"
        data-cy="upload"
        @click="pick"
        @keydown.enter.prevent="pick"
        @keydown.space.prevent="pick"
        @dragover.prevent="dragOver = true"
        @dragleave.prevent="dragOver = false"
        @drop.prevent="onDrop"
      >
        <span class="media-drop-icon" aria-hidden="true"><i class="pi pi-upload" /></span>
        <span class="media-drop-text">
          <strong>{{ $t('media.dropTitle') }}</strong>
          <span class="ln-small">{{ extensionsHelp }}</span>
        </span>
        <span v-if="uploading.total" class="media-progress" aria-live="polite">
          <span class="ln-small">{{ $t('media.uploading', { n: uploading.total, pct: uploadPct }) }}</span>
          <PvProgressBar :value="uploadPct" :show-value="false" />
        </span>
      </div>
      <PvMessage v-for="(e, i) in uploadErrors" :key="i" severity="error" closable @close="uploadErrors.splice(i, 1)">{{ e }}</PvMessage>
      <input ref="fileInput" type="file" multiple class="media-input" tabindex="-1" aria-hidden="true" @change="onPick">
    </div>

    <div v-if="loading.media && !items.length" class="media-grid">
      <div v-for="n in 8" :key="n" class="ln-card media-card"><PvSkeleton height="190px" /></div>
    </div>

    <empty-state
      v-else-if="!items.length"
      :icon="queryParams.query ? 'pi-search' : 'pi-image'"
      :title="queryParams.query ? $t('states.noMatches') : $t('media.emptyTitle')"
      :text="queryParams.query ? $t('states.noMatchesText', { query: queryParams.query }) : $t('media.emptyText')"
      :action-label="queryParams.query ? $t('states.clearFilters') : (canManage ? $t('media.uploadImages') : '')"
      :secondary="!!queryParams.query"
      @action="queryParams.query ? clearQuery() : pick()"
    />

    <div v-else class="media-grid">
      <article
        v-for="item in items"
        :key="item.id"
        class="ln-card media-card"
        :class="{ 'is-selected': selected.has(item.id) }"
      >
        <a
          class="media-thumb"
          :href="item.url"
          target="_blank"
          rel="noopener noreferrer"
          :aria-label="item.filename"
          @click="(e) => onMediaSelect(item, e)"
        >
          <img v-if="item.thumbUrl" :src="item.thumbUrl" :alt="item.filename" loading="lazy">
          <span v-else class="media-ext">{{ ext(item.filename) }}</span>
        </a>
        <PvCheckbox
          v-if="!isModal && canManage"
          class="media-check"
          :model-value="selected.has(item.id)"
          binary
          :aria-label="$t('media.select', { name: item.filename })"
          @update:model-value="toggle(item.id)"
        />
        <div class="media-info">
          <span class="ln-mono media-name" :title="item.filename">{{ item.filename }}</span>
          <span class="ln-small">{{ meta(item) }}</span>
        </div>
        <button
          v-if="!isModal"
          v-tooltip.bottom="$t('media.copyLink')"
          type="button"
          class="ln-row-menu media-copy"
          :aria-label="$t('media.copyLink')"
          @click="copy([item.url])"
        >
          <i class="pi pi-copy" />
        </button>
      </article>
    </div>

    <PvPaginator
      v-if="(media.total || 0) > (media.perPage || 0)"
      :rows="media.perPage"
      :total-records="media.total"
      :first="((media.page || 1) - 1) * (media.perPage || 1)"
      class="media-pager"
      @page="(e) => onPageChange(e.page + 1)"
    />

    <!-- Floating selection bar -->
    <div v-if="selected.size && !isModal" class="media-bar" role="toolbar" :aria-label="$t('globals.messages.numSelected', { num: selected.size })">
      <strong>{{ $t('globals.messages.numSelected', { num: selected.size }) }}</strong>
      <button type="button" class="ln-bulk-btn" @click="copy(selectedUrls)">{{ $t('media.copyLinks') }}</button>
      <button type="button" class="ln-bulk-btn" data-cy="btn-delete" @click="askDelete">{{ $t('globals.buttons.delete') }}</button>
      <button type="button" class="ln-bulk-clear" @click="selected.clear()">{{ $t('globals.buttons.clear') }}</button>
    </div>

    <confirm-action
      v-model:visible="confirm.open"
      :title="$t('media.deleteTitle', { n: selected.size })"
      :text="$t('media.deleteText')"
      :confirm-label="$t('globals.buttons.delete')"
      :busy="confirm.busy"
      @confirm="doDelete"
    />
  </section>
</template>

<script setup lang="ts">
import {
  computed, onMounted, reactive, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { makeQueryHandlers } from '../utils';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getMedia as mediaApi } from '../api/generated/endpoints/media/media';

// Media items are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = withDefaults(defineProps<{
  isModal?: boolean;
  type?: string;
}>(), { isModal: false, type: '' });

const emit = defineEmits(['selected', 'close']);
const { $api: untypedApi, $utils, $can } = useGlobal();
const $api = untypedApi as unknown as Record<string, (...args: unknown[]) => Promise<Loose>>;
const { t } = useI18n();
const store = useMainStore();
const { refreshTick, loading, serverConfig } = storeToRefs(store);
const media = computed<Loose>(() => (store.media as Loose) || {});

const fileInput = ref<HTMLInputElement | null>(null);
const dragOver = ref(false);
const selected = reactive(new Set<number>());
const uploading = reactive({ total: 0, done: 0 });
const uploadErrors = ref<string[]>([]);
const queryParams = reactive({ page: 1, query: '' });
const confirm = reactive({ open: false, busy: false });
const usage = ref<{ bytes: number; files: number; unknown: number } | null>(null);
const { getMediaUsage } = mediaApi();

const canManage = computed(() => $can('media:manage'));
const items = computed<Loose[]>(() => media.value.results || []);
const uploadPct = computed(() => (uploading.total ? Math.round((uploading.done / uploading.total) * 100) : 0));
// "7 files · 80 KB": how much space Media uses goes with the count.
const countLabel = computed(() => {
  if (!media.value.total) return null;
  const files = t('media.files', { n: media.value.total.toLocaleString() });
  return usage.value?.bytes ? `${files} · ${formatBytes(usage.value.bytes)}` : files;
});
const selectedUrls = computed(() => items.value.filter((m) => selected.has(m.id)).map((m) => m.url));
// What Media accepts, from this workspace's upload settings: "JPG, PNG,
// GIF, SVG · up to 5 MB each". "*" in the list means any type.
const extensionsHelp = computed(() => {
  const cfg = serverConfig.value as Loose;
  const exts = ((cfg.upload_extensions || []) as string[]).filter((e) => e !== 'jpeg');
  const types = !exts.length || exts.includes('*') ? t('media.anyType') : exts.map((e) => e.toUpperCase()).join(', ');
  const kb = Number(cfg.upload_max_file_size) || 0;
  // The setting is in KB; say it the way it was set ("5000" reads as 5 MB),
  // which never overstates the limit (the server allows kb * 1024 bytes).
  const size = kb >= 1000 ? `${Number((kb / 1000).toFixed(1))} MB` : `${kb} KB`;
  return kb ? t('media.dropLimits', { types, size }) : types;
});

function ext(name: string): string {
  return (name.split('.').pop() || '').toUpperCase();
}

function meta(m: Loose): string {
  const w = m.meta?.width;
  const h = m.meta?.height;
  const dims = w && h ? `${w} × ${h}` : ext(m.filename);
  return m.size ? `${dims}\u2003${formatBytes(m.size)}` : dims;
}

function toggle(id: number) {
  if (selected.has(id)) selected.delete(id);
  else selected.add(id);
}

// formatBytes shows a size in the largest unit that keeps it above 1.
function formatBytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${i === 0 ? v : v.toFixed(v < 10 ? 1 : 0)} ${units[i]}`;
}

function fetchUsage() {
  if (props.isModal) return;
  getMediaUsage().then((d) => { usage.value = d as unknown as typeof usage.value; }).catch(() => { usage.value = null; });
}

function getMedia() {
  $api.getMedia({ page: queryParams.page, query: queryParams.query });
  fetchUsage();
}

function onQueryMedia() {
  queryParams.page = 1;
  getMedia();
}

function clearQuery() {
  queryParams.query = '';
  onQueryMedia();
}

function onMediaSelect(m: Loose, e: Event) {
  if (props.isModal) {
    e.preventDefault();
    emit('selected', m);
    emit('close');
  }
}

function pick() {
  fileInput.value?.click();
}

function upload(files: File[]) {
  if (!files.length) return;
  uploading.total = files.length;
  uploading.done = 0;
  const finish = () => {
    uploading.done += 1;
    if (uploading.done >= uploading.total) {
      uploading.total = 0;
      getMedia();
    }
  };
  files.forEach((f) => {
    const fd = new FormData();
    fd.set('file', f);
    $api.uploadMedia(fd).then(finish, (err) => {
      uploadErrors.value.push(t('media.uploadFailed', { name: f.name, error: err?.response?.data?.message || String(err) }));
      finish();
    });
  });
}

function onPick(e: Event) {
  const input = e.target as HTMLInputElement;
  upload(Array.from(input.files || []));
  input.value = '';
}

function onDrop(e: DragEvent) {
  dragOver.value = false;
  upload(Array.from(e.dataTransfer?.files || []));
}

function copy(urls: string[]) {
  navigator.clipboard?.writeText(urls.join('\n')).then(() => $utils.toast(t('states.copied')));
}

function askDelete() {
  confirm.open = true;
}

function doDelete() {
  confirm.busy = true;
  Promise.allSettled([...selected].map((id) => $api.deleteMedia(id)))
    .then(() => {
      selected.clear();
      confirm.open = false;
      getMedia();
    })
    .finally(() => { confirm.busy = false; });
}

const { onPageChange } = makeQueryHandlers(queryParams, getMedia);

watch(refreshTick, getMedia);
onMounted(() => { $api.getMedia(); fetchUsage(); });
</script>

<style scoped lang="scss">
.media-used { font-variant-numeric: tabular-nums; }
.media-top { display: flex; flex-direction: column; gap: 10px; }
.media-drop {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border: 1px dashed var(--ln-border-strong);
  border-radius: var(--ln-radius-card);
  background: var(--ln-muted);
  cursor: pointer;
  transition: border-color var(--ln-dur-fast) var(--ln-ease), background-color var(--ln-dur-fast) var(--ln-ease);

  .media-drop-text { display: flex; flex-direction: column; gap: 2px; }
  &:hover, &.is-over { border-color: var(--ln-ink); }
  // Dragging a file over lifts the upload icon.
  &.is-over { background: var(--ln-hover); }
  &.is-over .media-drop-icon { translate: 0 -3px; }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: 2px; }
  .app-dark & { background: var(--ln-card); }
}
.media-drop-icon {
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: 8px;
  background: var(--ln-card);
  border: 1px solid var(--ln-border);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: translate var(--ln-dur-base) var(--ln-ease);
}
.media-drop-text { flex: 1; min-width: 0; }
.media-progress { display: flex; flex-direction: column; gap: 6px; flex: 0 1 240px; }
.media-input { display: none; }

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 14px;
}
.media-card {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color var(--ln-dur-fast) var(--ln-ease), box-shadow var(--ln-dur-fast) var(--ln-ease);

  &.is-selected { border-color: var(--ln-ink); box-shadow: 0 0 0 1px var(--ln-ink); }
}
.media-thumb {
  height: 150px;
  background: var(--ln-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;

  img { width: 100%; height: 100%; object-fit: cover; }
  .is-modal & { cursor: pointer; }
}
.media-ext { font-family: var(--ln-mono); font-size: 13px; color: var(--ln-text-3); }
.media-check { position: absolute; top: 10px; left: 10px; }
.media-info {
  padding: 10px 44px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.media-name { font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.media-copy { position: absolute; right: 6px; bottom: 10px; }
.media-pager { border: 1px solid var(--ln-border); border-radius: var(--ln-radius-card); }

.media-bar {
  position: fixed;
  left: 50%;
  bottom: 24px;
  transform: translateX(-50%);
  z-index: 30;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 8px 16px;
  border-radius: 10px;
  background: var(--ln-ink);
  color: var(--ln-on-ink);
  font-size: 14px;
  box-shadow: var(--ln-shadow-pop);

  .ln-bulk-btn {
    height: 30px;
    padding: 0 10px;
    border-radius: var(--ln-radius);
    border: 1px solid rgba(127, 127, 127, .5);
    background: transparent;
    color: var(--ln-on-ink);
    font: inherit;
    font-size: 13px;
    cursor: pointer;
  }
  .ln-bulk-clear { border: 0; background: none; color: var(--ln-on-ink); text-decoration: underline; font: inherit; font-size: 13px; cursor: pointer; }
}
</style>
