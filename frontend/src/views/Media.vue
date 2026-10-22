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
        <span>
          <strong>{{ $t('media.dropTitle') }}</strong>
          <span class="ln-small">{{ extensionsHelp }}</span>
        </span>
      </div>
      <div class="ln-card ln-card-pad media-store">
        <div class="media-store-row">
          <span>{{ $t('media.storage') }}</span>
          <strong class="ln-mono">{{ provider }}</strong>
        </div>
        <template v-if="uploading.total">
          <PvProgressBar :value="uploadPct" :show-value="false" />
          <span class="ln-small">{{ $t('media.uploading', { n: uploading.total, pct: uploadPct }) }}</span>
        </template>
        <span v-else class="ln-small">{{ $t('media.storageHelp') }}</span>
        <PvMessage v-for="(e, i) in uploadErrors" :key="i" severity="error" closable @close="uploadErrors.splice(i, 1)">{{ e }}</PvMessage>
      </div>
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
import { useTime } from '../composables/useTime';
import { makeQueryHandlers } from '../utils';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';

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
const { shortDate } = useTime();
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

const canManage = computed(() => $can('media:manage'));
const items = computed<Loose[]>(() => media.value.results || []);
const provider = computed(() => (serverConfig.value as Loose).media_provider || '');
const uploadPct = computed(() => (uploading.total ? Math.round((uploading.done / uploading.total) * 100) : 0));
const countLabel = computed(() => (media.value.total ? t('media.files', { n: media.value.total.toLocaleString() }) : null));
const selectedUrls = computed(() => items.value.filter((m) => selected.has(m.id)).map((m) => m.url));
const extensionsHelp = computed(() => t('media.dropHelp'));

function ext(name: string): string {
  return (name.split('.').pop() || '').toUpperCase();
}

function meta(m: Loose): string {
  const w = m.meta?.width;
  const h = m.meta?.height;
  const dims = w && h ? `${w} × ${h}` : ext(m.filename);
  return `${dims} · ${shortDate(m.createdAt)}`;
}

function toggle(id: number) {
  if (selected.has(id)) selected.delete(id);
  else selected.add(id);
}

function getMedia() {
  $api.getMedia({ page: queryParams.page, query: queryParams.query });
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
onMounted(() => { $api.getMedia(); });
</script>

<style scoped lang="scss">
.media-top {
  display: grid;
  grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr);
  gap: 16px;

  @media (max-width: 900px) { grid-template-columns: minmax(0, 1fr); }
  .is-modal & { grid-template-columns: minmax(0, 1fr); .media-store { display: none; } }
}
.media-drop {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border: 1px dashed var(--ln-border-strong);
  border-radius: var(--ln-radius-card);
  background: var(--ln-muted);
  cursor: pointer;

  > span:last-child { display: flex; flex-direction: column; gap: 2px; }
  &:hover, &.is-over { border-color: var(--ln-ink); }
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
}
.media-store { display: flex; flex-direction: column; gap: 8px; }
.media-store-row { display: flex; justify-content: space-between; gap: 8px; }
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
