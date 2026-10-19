<template>
  <div>
    <PvDialog :visible="isVisible" @update:visible="close" :aria-modal="true"
      :style="{ width: '900px', maxWidth: '95vw' }" :closable="true" modal>
      <template #header>
        <h4 class="preview-title">{{ title }}</h4>
      </template>

      <div class="preview-body">
        <div v-if="isLoading" class="preview-spinner">
          <PvProgressSpinner style="width:2rem;height:2rem" />
        </div>
        <form v-if="isPost" method="post" :action="previewURL" target="iframe" ref="formEl">
          <input v-if="templateId" type="hidden" name="template_id" :value="templateId" />
          <input v-if="contentType" type="hidden" name="content_type" :value="contentType" />
          <input v-if="templateType" type="hidden" name="template_type" :value="templateType" />
          <input v-if="archiveMeta" type="hidden" name="archive_meta" :value="archiveMeta" />
          <input v-if="body" type="hidden" name="body" :value="body" />
        </form>

        <iframe id="iframe" name="iframe" ref="iframeEl" :title="title" :src="isPost ? 'about:blank' : previewURL"
          @load="onLoaded" sandbox="allow-scripts" class="preview-iframe" />
      </div>

      <template #footer>
        <PvButton @click="close" :label="$t('globals.buttons.close')" severity="secondary" />
      </template>
    </PvDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { uris } from '../constants';

const props = withDefaults(defineProps<{
  isPost?: boolean;
  id?: number;
  title?: string;
  type?: string;
  templateType?: string;
  archiveMeta?: string | null;
  body?: string;
  contentType?: string;
  templateId?: number | null;
  isArchive?: boolean;
}>(), {
  isPost: false,
  id: 0,
  title: '',
  type: '',
  templateType: '',
  archiveMeta: null,
  body: '',
  contentType: '',
  templateId: null,
  isArchive: false,
});

const emit = defineEmits(['close']);

const isVisible = ref(true);
const isLoading = ref(true);
const formSubmitted = ref(false);
const formEl = ref<HTMLFormElement | null>(null);

const previewURL = computed(() => {
  let uri = 'about:blank';
  if (props.type === 'campaign') {
    uri = props.isArchive ? uris.previewCampaignArchive : uris.previewCampaign;
  } else if (props.type === 'template') {
    uri = props.id ? uris.previewTemplate : uris.previewRawTemplate;
  }
  return uri.replace(':id', String(props.id));
});

function close() {
  emit('close');
  isVisible.value = false;
}

function onLoaded() {
  if (!props.isPost) {
    isLoading.value = false;
    return;
  }
  if (formSubmitted.value) {
    isLoading.value = false;
  }
}

onMounted(() => {
  if (props.isPost) {
    setTimeout(() => {
      formEl.value?.submit();
      formSubmitted.value = true;
    }, 100);
  }
});
</script>

<style scoped lang="scss">
.preview-title { margin: 0; font-size: 1rem; font-weight: 600; }

.preview-body {
  position: relative;
  width: 100%;
  height: 72vh;
  display: flex;
  flex-direction: column;
}

.preview-spinner {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--ln-card);
  z-index: 1;
}

.preview-iframe {
  width: 100%;
  flex: 1;
  border: none;
  display: block;
}
</style>
