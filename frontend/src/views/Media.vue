<template>
  <section class="media-page">
    <div class="page-header">
      <h1 class="page-title">
        {{ $t('media.title') }}
        <span v-if="media.results && media.results.length > 0" class="page-title-count">
          {{ media.results.length }}
        </span>
        <span class="provider-label">/ {{ serverConfig.media_provider }}</span>
      </h1>
      <PvButton v-if="$can('media:manage')" @click="onToggleForm"
        :icon="showUploadForm ? 'pi pi-times' : 'pi pi-upload'"
        :label="showUploadForm ? $t('globals.buttons.cancel') : $t('media.upload')"
        :severity="showUploadForm ? 'secondary' : 'primary'"
        data-cy="btn-toggle-upload" />
    </div>

    <!-- Upload panel -->
    <div v-if="$can('media:manage') && showUploadForm" class="upload-card">
      <form @submit.prevent="onSubmit" data-cy="upload">
        <PvFileUpload
          mode="advanced"
          :multiple="true"
          :auto="false"
          :custom-upload="true"
          @select="onFilesSelect"
          @remove="onFileRemove"
          :show-upload-button="false"
          :show-cancel-button="false"
        >
          <template #empty>
            <div class="upload-empty">
              <i class="pi pi-cloud-upload upload-empty__icon" />
              <p class="upload-empty__text">{{ $t('media.uploadHelp') }}</p>
            </div>
          </template>
        </PvFileUpload>

        <div v-if="form.files.length > 0" class="upload-footer">
          <div class="upload-tags">
            <PvTag v-for="(f, i) in form.files" :key="i" class="upload-tag">
              <template #default>
                {{ f.name }}
                <i class="pi pi-times upload-tag__remove" @click="removeUploadFile(i)" />
              </template>
            </PvTag>
          </div>
          <PvButton type="submit" severity="primary" icon="pi pi-upload"
            :loading="isProcessing" :label="$t('media.upload')" />
        </div>
      </form>
    </div>

    <!-- Gallery card -->
    <div class="table-card">
      <!-- Toolbar -->
      <div class="gallery-toolbar">
        <PvIconField class="gallery-search">
          <PvInputIcon class="pi pi-search" />
          <PvInputText v-model="queryParams.query" @keyup.enter="onQueryMedia"
            placeholder="Search…" data-cy="query" ref="query" />
        </PvIconField>
      </div>

      <!-- Loading -->
      <div v-if="loading.media" class="gallery-spinner">
        <PvProgressSpinner />
      </div>

      <!-- Grid -->
      <div v-else-if="media.results && media.results.length > 0" class="media-grid">
        <div v-for="item in media.results" :key="item.id" class="media-item">
          <a class="media-item__thumb" @click="(e) => onMediaSelect(item, e)"
            :href="item.url" target="_blank" rel="noopener noreferrer">
            <img v-if="item.thumbUrl" :src="item.thumbUrl" :alt="item.filename" />
            <div v-else class="media-item__placeholder">
              <span class="media-item__ext">{{ item.filename.split('.').pop().toUpperCase() }}</span>
            </div>
            <div class="media-item__overlay">
              <button type="button" class="media-item__delete" data-cy="btn-delete"
                @click.prevent.stop="$utils.confirm(null, () => onDeleteMedia(item.id))"
                :aria-label="$t('globals.buttons.delete')">
                <i class="pi pi-trash" />
              </button>
            </div>
          </a>
          <div class="media-item__info">
            <p class="media-item__filename" :title="item.filename">{{ item.filename }}</p>
            <p class="media-item__date">{{ $utils.niceDate(item.createdAt, false) }}</p>
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="gallery-empty">
        <empty-placeholder />
      </div>

      <!-- Pagination -->
      <div v-if="media.total > media.perPage" class="gallery-paginator">
        <PvPaginator
          :rows="media.perPage"
          :total-records="media.total"
          :first="(media.page - 1) * media.perPage"
          @page="(e) => onPageChange(e.page + 1)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { makeQueryHandlers } from '../utils';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

const props = withDefaults(defineProps<{
  isModal?: boolean;
  type?: string;
}>(), { isModal: false, type: '' });

const emit = defineEmits(['selected', 'close']);
const { $api, $utils } = useGlobal();
const {
  refreshTick, loading, media, serverConfig,
} = storeToRefs(useMainStore());

const form = reactive({ files: [] as File[] });
const toUpload = ref(0);
const uploaded = ref(0);
const showUploadForm = ref(false);
const queryParams = reactive({ page: 1, query: '' });

const isProcessing = computed(() => toUpload.value > 0 && uploaded.value < toUpload.value);

function removeUploadFile(i: number) { form.files.splice(i, 1); }
function onFilesSelect(event: any) { form.files = event.files ? [...event.files] : []; }
function onFileRemove(event: any) { form.files = form.files.filter((f) => f !== event.file); }

function getMedia() {
  $api.getMedia({ page: queryParams.page, query: queryParams.query });
}

function onToggleForm() {
  showUploadForm.value = !showUploadForm.value;
  $utils.setPref('media.upload', showUploadForm.value);
}

function onQueryMedia() {
  queryParams.page = 1;
  getMedia();
}

function onMediaSelect(m: any, e: Event) {
  if (props.isModal) {
    e.preventDefault();
    emit('selected', m);
    emit('close');
  }
}

function onUploaded() {
  uploaded.value += 1;
  if (uploaded.value >= toUpload.value) {
    toUpload.value = 0;
    uploaded.value = 0;
    form.files = [];
    getMedia();
  }
}

function onSubmit() {
  toUpload.value = form.files.length;
  for (let i = 0; i < toUpload.value; i += 1) {
    const params = new FormData();
    params.set('file', form.files[i]);
    $api.uploadMedia(params).then(() => { onUploaded(); }, () => { onUploaded(); });
  }
}

function onDeleteMedia(id: number) {
  $api.deleteMedia(id).then(() => { getMedia(); });
}

const { onPageChange } = makeQueryHandlers(queryParams, getMedia);

watch(() => refreshTick.value, () => { getMedia(); });

onMounted(() => {
  $api.getMedia();
  if ($utils.getPref('media.upload')) { showUploadForm.value = true; }
});
</script>

<style scoped lang="scss">
.media-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.provider-label {
  font-size: 0.8rem;
  font-weight: 400;
  color: var(--lm-text-subtle);
  margin-left: 0.25rem;
}

// Upload panel
.upload-card {
  border: 1px solid var(--lm-border);
  border-radius: 10px;
  background: var(--lm-surface);
  overflow: hidden;

  :deep(.p-fileupload) {
    border: none;
    border-radius: 0;
    background: transparent;
  }
  :deep(.p-fileupload-header) { display: none; }
  :deep(.p-fileupload-content) {
    border: 2px dashed var(--lm-border);
    border-radius: 8px;
    margin: 1rem;
    background: var(--lm-bg-subtle);
    min-height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: border-color 0.15s;
    &:hover { border-color: var(--lm-primary); }
  }
}

.upload-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  color: var(--lm-text-muted);

  &__icon { font-size: 2rem; color: var(--lm-text-subtle); }
  &__text { font-size: 0.875rem; }
}

.upload-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1rem 1rem;
  flex-wrap: wrap;
}

.upload-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  flex: 1;
}

.upload-tag {
  font-size: 0.78rem;

  &__remove {
    margin-left: 0.4rem;
    cursor: pointer;
    opacity: 0.6;
    &:hover { opacity: 1; }
  }
}

// Gallery
.gallery-toolbar {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--lm-border);
}

.gallery-search {
  width: 280px;
}

.gallery-spinner {
  display: flex;
  justify-content: center;
  padding: 3rem;
}

.gallery-empty {
  padding: 2rem;
}

.gallery-paginator {
  border-top: 1px solid var(--lm-border);
  padding: 0.5rem 0;
}

// Media grid
.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 1rem;
  padding: 1rem;
}

.media-item {
  border: 1px solid var(--lm-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--lm-surface);
  transition: box-shadow 0.15s, transform 0.15s;

  &:hover {
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
    transform: translateY(-1px);

    .media-item__overlay { opacity: 1; }
  }

  &__thumb {
    display: block;
    position: relative;
    aspect-ratio: 1 / 1;
    background: var(--lm-bg-subtle);
    overflow: hidden;
    text-decoration: none;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      display: block;
    }
  }

  &__placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--lm-bg-subtle);
  }

  &__ext {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--lm-text-muted);
    letter-spacing: 0.05em;
    background: var(--lm-border);
    padding: 0.25rem 0.4rem;
    border-radius: 4px;
  }

  &__overlay {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: opacity 0.15s;
  }

  &__delete {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: 2px solid rgba(255, 255, 255, 0.8);
    background: rgba(239, 68, 68, 0.85);
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    font-size: 0.85rem;
    transition: background 0.15s;

    &:hover { background: rgb(239, 68, 68); }
  }

  &__info {
    padding: 0.45rem 0.6rem 0.5rem;
    border-top: 1px solid var(--lm-border);
    background: var(--lm-surface);
  }

  &__filename {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--lm-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin: 0;
  }

  &__date {
    font-size: 0.68rem;
    color: var(--lm-text-subtle);
    margin: 0.1rem 0 0;
  }
}
</style>
