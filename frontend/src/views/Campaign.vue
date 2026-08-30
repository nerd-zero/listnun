<template>
  <section class="campaign">
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">
          <template v-if="isEditing">{{ data.name }}</template>
          <template v-else>{{ $t('campaigns.newCampaign') }}</template>
        </h1>
        <div v-if="isEditing && data.status" class="header-meta">
          <PvTag
            :class="data.status" :value="$t(`campaigns.status.${data.status}`)"
            v-tooltip.bottom="data.pauseReason ? $t(`campaigns.pauseReason.${data.pauseReason}`) : null"
          />
          <PvTag v-if="data.type === 'optin'" :class="data.type" :value="$t('lists.optin')" />
          <span class="id-meta" :data-campaign-id="data.id">
            {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
            &nbsp;{{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
          </span>
        </div>
      </div>
      <div v-if="(canManage || canSend) && isEditing && canEdit" class="header-actions">
        <PvButton v-if="canManage" @click="() => onSubmit('update')" :loading="loading.campaigns"
          severity="primary" data-cy="btn-save" aria-keyshortcuts="ctrl+s">
          <i class="pi pi-save" /><span class="has-kbd">{{ $t('globals.buttons.saveChanges') }} <span class="kbd">Ctrl+S</span></span>
        </PvButton>
        <PvButton v-if="canSend && canStart" @click="startCampaign" :loading="loading.campaigns"
          severity="primary" icon="pi pi-send" data-cy="btn-start" :label="$t('campaigns.start')" />
        <PvButton v-if="canSend && canSchedule" @click="startCampaign" :loading="loading.campaigns"
          severity="primary" icon="pi pi-clock" data-cy="btn-schedule" :label="$t('campaigns.schedule')" />
        <PvButton v-if="canSend && canUnSchedule" @click="$utils.confirm(null, unscheduleCampaign)"
          :loading="loading.campaigns" severity="primary" icon="pi pi-clock"
          data-cy="btn-unschedule" :label="$t('campaigns.unSchedule')" />
      </div>
    </div>

    <div v-if="loading.campaigns" class="flex justify-center p-8">
      <PvProgressSpinner />
    </div>

    <PvTabs class="lm-tabs" v-model:value="activeTab" @update:value="onTab">
      <PvTabList>
        <PvTab value="campaign">
          <i class="pi pi-send mr-1" />{{ $t('globals.terms.campaign') }}
        </PvTab>
        <PvTab value="content" :disabled="isNew">
          <i class="pi pi-file mr-1" />{{ $t('campaigns.content') }}
        </PvTab>
        <PvTab value="attribs" :disabled="isNew">
          <i class="pi pi-code mr-1" />{{ $t('globals.terms.attribs') }}
        </PvTab>
        <PvTab value="archive" :disabled="isNew">
          <i class="pi pi-file mr-1" />{{ $t('campaigns.archive') }}
        </PvTab>
      </PvTabList>

      <PvTabPanels>
        <!-- campaign tab -->
        <PvTabPanel value="campaign">
          <div class="campaign-layout">
            <form class="box campaign-form" @submit.prevent="() => onSubmit(isNew ? 'create' : 'update')">
              <div class="field">
                <label class="field-label">{{ $t('globals.fields.name') }}</label>
                <PvInputText :maxlength="200" ref="focusEl" v-model="form.name" name="name" :disabled="!canEdit"
                  :placeholder="$t('globals.fields.name')" required autofocus class="w-full" />
              </div>

              <div class="field">
                <label class="field-label">{{ $t('campaigns.subject') }}</label>
                <PvInputText :maxlength="5000" v-model="form.subject" name="subject" :disabled="!canEdit"
                  :placeholder="$t('campaigns.subject')" required class="w-full" />
              </div>

              <div class="field">
                <label class="field-label">{{ $t('campaigns.fromAddress') }}</label>
                <PvInputText :maxlength="200" v-model="form.fromEmail" name="from_email" :disabled="!canEdit"
                  :placeholder="$t('campaigns.fromAddressPlaceholder')" required class="w-full" />
              </div>

              <div class="field">
                <list-selector v-model="form.lists" :selected="form.lists" :all="lists.results" :disabled="!canEdit"
                  :label="$t('globals.terms.lists')" :placeholder="$t('campaigns.sendToLists')" />
              </div>

              <div class="form-row">
                <div class="field">
                  <label class="field-label">{{ $t('globals.terms.messenger') }}</label>
                  <PvSelect v-model="form.messenger" :options="allMessengers" :disabled="!canEdit"
                    required class="w-full" />
                </div>
                <div class="field">
                  <label class="field-label">{{ $t('campaigns.format') }}</label>
                  <PvSelect v-model="form.content.contentType" :options="contentTypeOptions"
                    option-label="label" option-value="value"
                    :disabled="!canEdit || isEditing" class="w-full" />
                </div>
              </div>

              <div class="field">
                <label class="field-label">{{ $t('globals.terms.tags') }}</label>
                <PvAutoComplete v-model="form.tags" name="tags" :disabled="!canEdit" :typeahead="false"
                  :placeholder="$t('globals.terms.tags')" multiple class="w-full" />
              </div>

              <PvDivider />

              <div class="field" data-cy="btn-send-later">
                <div class="toggle-row">
                  <PvToggleSwitch v-model="form.sendLater" :disabled="!canEdit" />
                  <span class="toggle-label">{{ $t('campaigns.sendLater') }}</span>
                </div>
                <div v-if="form.sendLater" data-cy="send_at" class="mt-2">
                  <PvDatePicker v-model="form.sendAtDate" :disabled="!canEdit" show-time hour-format="24"
                    :placeholder="$t('campaigns.dateAndTime')" required />
                  <small v-if="form.sendAtDate" class="block mt-1 text-color-secondary">
                    {{ $utils.duration(Date(), form.sendAtDate) }}
                  </small>
                </div>
              </div>

              <div class="field">
                <a href="#" class="form-link" @click.prevent="onShowHeaders" data-cy="btn-headers">
                  <i class="pi pi-plus" />{{ $t('settings.smtp.setCustomHeaders') }}
                </a>
                <div v-if="form.headersStr !== '[]' || isHeadersVisible" class="mt-2">
                  <PvTextarea v-model="form.headersStr" name="headers"
                    placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}, {&quot;X-Custom2&quot;: &quot;value&quot;}]"
                    :disabled="!canEdit" class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('campaigns.customHeadersHelp') }}</small>
                </div>
              </div>

              <div v-if="isNew" class="form-footer">
                <PvButton type="submit" severity="primary" :loading="loading.campaigns" data-cy="btn-continue"
                  icon="pi pi-arrow-right" icon-pos="right" :label="$t('campaigns.continue')" />
              </div>
            </form>

            <div v-if="canManage" class="campaign-sidebar">
              <div class="box test-message-card">
                <div class="test-message-card__title">
                  <i class="pi pi-send" />
                  <span>{{ $t('campaigns.sendTest') }}</span>
                </div>
                <small class="test-message-card__help">{{ $t('campaigns.sendTestHelp') }}</small>
                <PvAutoComplete v-model="form.testEmails" :disabled="isNew" :typeahead="false"
                  :placeholder="$t('campaigns.testEmails')" multiple class="w-full"
                  @blur="onTestEmailBlur" />
                <PvButton @click="() => onSubmit('test')" :loading="loading.campaigns" :disabled="isNew"
                  severity="secondary" outlined icon="pi pi-send" :label="$t('campaigns.send')" class="w-full" />
              </div>
            </div>
          </div>
        </PvTabPanel><!-- campaign -->

        <!-- content tab -->
        <PvTabPanel value="content">
          <editor v-if="data.id" v-model="form.content" :id="data.id" :title="data.name" :disabled="!canEdit"
            :templates="templates" :content-types="contentTypes" />

          <div class="grid">
            <div class="col-6">
              <p v-if="!isAttachFieldVisible" style="font-size:0.9rem;color:var(--lm-text-muted)">
                <a href="#" @click.prevent="onShowAttachField()" data-cy="btn-attach">
                  <i class="pi pi-upload" />
                  {{ $t('campaigns.addAttachments') }}
                </a>
              </p>

              <div class="field" v-if="isAttachFieldVisible" data-cy="media">
                <label class="block mb-1 text-sm font-medium">{{ $t('campaigns.attachments') }}</label>
                <PvAutoComplete v-model="form.media" name="media" ref="media" option-label="filename"
                  @focus="onOpenAttach" :disabled="!canEdit" multiple />
              </div>
            </div>
            <div class="col" style="text-align:right">
              <a href="https://listmonk.app/docs/templating/#template-expressions" target="_blank"
                rel="noopener noreferer">
                <i class="pi pi-code" /> {{ $t('campaigns.templatingRef') }}</a>
              <span v-if="canEdit && form.content.contentType !== 'plain'" style="font-size:0.9rem;color:var(--lm-text-muted);margin-left:1.5rem">
                <a v-if="form.altbody === null" href="#" @click.prevent="onAddAltBody">
                  <i class="pi pi-file" /> {{ $t('campaigns.addAltText') }}
                </a>
                <a v-else href="#" @click.prevent="$utils.confirm(null, onRemoveAltBody)">
                  <i class="pi pi-trash" />
                  {{ $t('campaigns.removeAltText') }}
                </a>
              </span>
            </div>
          </div>

          <div v-if="canEdit && form.content.contentType !== 'plain'" class="alt-body">
            <PvTextarea v-if="form.altbody !== null" v-model="form.altbody" :disabled="!canEdit" class="w-full" />
          </div>
        </PvTabPanel><!-- content -->

        <!-- attribs tab -->
        <PvTabPanel value="attribs">
          <section class="wrap">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('globals.terms.attribs') }}</label>
              <PvTextarea v-model="form.attribsStr" :disabled="!canEdit" rows="15" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('campaigns.attribsHelp') }}</small>
            </div>
          </section>
        </PvTabPanel><!-- attribs -->

        <!-- archive tab -->
        <PvTabPanel value="archive">
          <section class="wrap">
            <div class="grid">
              <div class="col-4">
                <div class="field" data-cy="btn-archive">
                  <label class="block mb-1 text-sm font-medium">{{ $t('campaigns.archiveEnable') }}</label>
                  <small class="block mt-1 text-color-secondary">{{ $t('campaigns.archiveHelp') }}</small>
                  <div class="flex items-center gap-2 mt-2">
                    <PvToggleSwitch data-cy="btn-archive" v-model="form.archive" :disabled="!canArchive" />
                    <a :href="`${serverConfig.root_url}/archive/${data.uuid}`" target="_blank" rel="noopener noreferer"
                      :style="{ color: !form.archive ? 'var(--lm-text-subtle)' : 'inherit' }" :aria-label="$t('campaigns.archive')">
                      <i class="pi pi-external-link" />
                    </a>
                  </div>
                </div>
              </div>
              <div class="col-8">
                <div style="display:flex; justify-content: flex-end;">
                  <PvButton v-if="!canEdit && canArchive" @click="onUpdateCampaignArchive" :loading="loading.campaigns" severity="primary"
                    icon="pi pi-save" data-cy="btn-save" :label="$t('globals.buttons.saveChanges')" />
                </div>
              </div>
            </div>

            <div class="grid">
              <div class="col-6">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('globals.terms.template') }}</label>
                  <PvSelect v-model="form.archiveTemplateId" :options="campaignTemplates"
                    option-label="name" option-value="id"
                    :disabled="!canArchive || !form.archive || form.content.contentType === 'visual'"
                    required class="w-full" />
                </div>
              </div>

              <div class="col-6">
                <div style="display:flex; justify-content: flex-end; gap: 0.5rem; align-items: center;">
                  <PvButton v-if="form.archive && (!form.archiveMetaStr || form.archiveMetaStr === '{}')"
                    severity="secondary" outlined @click.prevent="onFillArchiveMeta" aria-label="{}" icon="pi pi-code" />
                  <PvButton v-if="form.archive" @click="onToggleArchivePreview" severity="primary" icon="pi pi-eye"
                    data-cy="btn-preview" :label="$t('campaigns.preview')" />
                </div>
              </div>
            </div>

            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('campaigns.archiveSlug') }}</label>
              <small class="block mt-1 text-color-secondary">{{ $t('campaigns.archiveSlugHelp') }}</small>
              <PvInputText :maxlength="200" ref="focus" v-model="form.archiveSlug" name="archive_slug"
                data-cy="archive-slug" :disabled="!canArchive || !form.archive" class="w-full" />
            </div>
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('campaigns.archiveMeta') }}</label>
              <small class="block mt-1 text-color-secondary">{{ $t('campaigns.archiveMetaHelp') }}</small>
              <PvTextarea v-model="form.archiveMetaStr" name="archive_meta" data-cy="archive-meta"
                :disabled="!canArchive || !form.archive" rows="15" class="w-full" />
            </div>
          </section>
        </PvTabPanel><!-- archive -->
      </PvTabPanels>
    </PvTabs>

    <PvDialog v-model:visible="isAttachModalOpen" :style="{ width: '900px' }" :closable="true" modal>
      <media is-modal @selected="onAttachSelect" @close="isAttachModalOpen = false" />
    </PvDialog>

    <campaign-preview v-if="isPreviewingArchive" @close="onToggleArchivePreview" type="campaign" :id="data.id"
      :archive-meta="form.archiveMetaStr" :title="data.name" :content-type="data.contentType"
      :template-id="form.archiveTemplateId" is-post is-archive />
  </section>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, nextTick, onMounted, onBeforeUnmount,
} from 'vue';
import dayjs from 'dayjs';
import htmlToPlainText from 'textversionjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import CampaignPreview from '../components/CampaignPreview.vue';
import CopyText from '../components/CopyText.vue';
import Editor from '../components/Editor.vue';
import ListSelector from '../components/ListSelector.vue';
import Media from './Media.vue';

const {
  $api, $utils, $can, $events,
} = useGlobal();
const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const {
  serverConfig, loading, lists, templates,
} = storeToRefs(useMainStore());

const focusEl = ref<any>(null);
const isNew = ref(false);
const isEditing = ref(false);
const isHeadersVisible = ref(false);
const isAttachFieldVisible = ref(false);
const isAttachModalOpen = ref(false);
const isPreviewingArchive = ref(false);
const activeTab = ref('campaign');
const data = ref<any>({});
const selListIDs = ref<number[]>([]);

const form = reactive<any>({
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
  archiveMetaStr: '{}',
  archiveMeta: {},
  testEmails: [],
});

const contentTypes = computed(() => Object.freeze({
  richtext: t('campaigns.richText'),
  html: t('campaigns.rawHTML'),
  markdown: t('campaigns.markdown'),
  plain: t('campaigns.plainText'),
  visual: t('campaigns.visual'),
}));

const canManage = computed(() => $can('campaigns:manage_all', 'campaigns:manage'));
const canSend = computed(() => $can('campaigns:send'));
const canEdit = computed(() => isNew.value || data.value.status === 'draft' || data.value.status === 'scheduled' || data.value.status === 'paused');
const canSchedule = computed(() => (data.value.status === 'draft' || data.value.status === 'paused') && form.sendLater && form.sendAtDate);
const canUnSchedule = computed(() => data.value.status === 'scheduled');
const canStart = computed(() => (data.value.status === 'draft' || data.value.status === 'paused') && !form.sendLater);
const canArchive = computed(() => data.value.status !== 'cancelled' && data.value.type !== 'optin');
const selectedLists = computed(() => {
  if (selListIDs.value.length === 0 || !(lists.value as any).results) return [];
  return (lists.value as any).results.filter((l: any) => selListIDs.value.indexOf(l.id) > -1);
});
const allMessengers = computed(() => {
  const sc = serverConfig.value as any;
  const email = ['email', ...(sc.messengers || []).filter((m: string) => m.startsWith('email-'))];
  const others = (sc.messengers || []).filter((m: string) => m !== 'email' && !m.startsWith('email-'));
  return [...email, ...others];
});
const contentTypeOptions = computed(() => Object.entries(contentTypes.value).map(([value, label]) => ({ value, label })));
const campaignTemplates = computed(() => ((templates.value as any[]) || []).filter((tpl: any) => tpl.type === 'campaign'));

function isUnsaved() {
  if (isNew.value) {
    return !!(form.name || form.subject || form.content.body);
  }
  return data.value.body !== form.content.body || data.value.contentType !== form.content.contentType;
}

function onToggleArchivePreview() { isPreviewingArchive.value = !isPreviewingArchive.value; }
function onAddAltBody() { form.altbody = htmlToPlainText(form.content.body); }
function onRemoveAltBody() { form.altbody = null; }
function onShowHeaders() { isHeadersVisible.value = !isHeadersVisible.value; }

function onShowAttachField() {
  isAttachFieldVisible.value = true;
}

function onOpenAttach() { isAttachModalOpen.value = true; }

function onAttachSelect(o: any) {
  if (!form.media.some((m: any) => m.id === o.id)) form.media.push(o);
}

function onTab(tab: string) {
  if (tab === 'content' && (window as any).tinymce && (window as any).tinymce.editors.length > 0) {
    nextTick(() => { (window as any).tinymce.editors[0].focus(); });
  }
  window.history.replaceState({}, '', `#${tab}`);
}

function onFillArchiveMeta() {
  const archiveStr = `{"email": "email@domain.com", "name": "${t('globals.fields.name')}", "attribs": {}}`;
  form.archiveMetaStr = $utils.getPref('campaign.archiveMetaStr') || JSON.stringify(JSON.parse(archiveStr), null, 4);
}

function onSubmit(typ: string) {
  if (form.headersStr && form.headersStr !== '[]') {
    try { form.headers = JSON.parse(form.headersStr); } catch (e: any) { $utils.toast(e.toString(), 'is-danger'); return; }
  } else { form.headers = []; }
  if (form.archive && form.archiveMetaStr) {
    try { form.archiveMeta = JSON.parse(form.archiveMetaStr); } catch (e: any) { $utils.toast(e.toString(), 'is-danger'); return; }
  }
  let attribs = null;
  if (form.attribsStr && form.attribsStr.trim()) {
    try { attribs = JSON.parse(form.attribsStr); } catch (e: any) { $utils.toast(`${t('subscribers.invalidJSON')}: ${e.toString()}`, 'is-danger', 3000); return; }
  }
  form.attribs = attribs;
  if (typ === 'create') { createCampaign(); } else if (typ === 'test') { sendTest(); } else { updateCampaign(); }
}

function getCampaign(id: string) {
  return $api.getCampaign(id).then((d: any) => {
    data.value = d;
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
    form.media = form.media.map((f: any) => (!f.id ? { ...f, filename: `❌ ${f.filename}` } : f));
  });
}

function onTestEmailBlur(e: FocusEvent) {
  const val = (e.target as HTMLInputElement).value?.trim();
  if (val) {
    form.testEmails.push(val);
    (e.target as HTMLInputElement).value = '';
  }
}

function sendTest() {
  $api.testCampaign({
    id: data.value.id,
    name: form.name,
    subject: form.subject,
    lists: form.lists.map((l: any) => l.id),
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
    media: form.media.map((m: any) => m.id),
  }).then(() => { $utils.toast(t('campaigns.testSent')); });
}

function createCampaign() {
  $api.createCampaign({
    archiveSlug: form.subject,
    name: form.name,
    subject: form.subject,
    lists: form.lists.map((l: any) => l.id),
    from_email: form.fromEmail,
    content_type: form.content.contentType,
    messenger: form.messenger,
    type: 'regular',
    tags: form.tags,
    send_at: form.sendLater ? form.sendAtDate : null,
    headers: form.headers,
    attribs: form.attribs,
    media: form.media.map((m: any) => m.id),
  }).then((d: any) => { router.push({ name: 'campaign', hash: '#content', params: { id: d.id } }); });
}

async function updateCampaign(typ?: string) {
  const typMsg = typ === 'start' ? 'campaigns.started' : 'globals.messages.updated';
  if (!form.sendAtDate) form.sendLater = false;
  return new Promise<void>((resolve) => {
    $api.updateCampaign(data.value.id, {
      archive_slug: form.archiveSlug,
      name: form.name,
      subject: form.subject,
      lists: form.lists.map((l: any) => l.id),
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
      media: form.media.map((m: any) => m.id),
    }).then((d: any) => {
      data.value = d;
      form.archiveSlug = d.archiveSlug;
      form.attribsStr = d.attribs ? JSON.stringify(d.attribs, null, 4) : '{}';
      $utils.toast(t(typMsg, { name: d.name }));
      resolve();
    });
  });
}

function onUpdateCampaignArchive() {
  if (isEditing.value && canEdit.value) return;
  let archiveMeta;
  try {
    archiveMeta = JSON.parse(form.archiveMetaStr);
  } catch (e: any) {
    $utils.toast(e.toString(), 'is-danger');
    return;
  }
  $api.updateCampaignArchive(data.value.id, {
    archive: form.archive,
    archive_template_id: form.archiveTemplateId,
    archive_meta: archiveMeta,
    archive_slug: form.archiveSlug,
  }).then((d: any) => { form.archiveSlug = d.archiveSlug; });
}

function startCampaign() {
  if (!canStart.value && !canSchedule.value) return;
  $utils.confirm(null, () => {
    updateCampaign().then(() => {
      let status = '';
      if (canStart.value) { status = 'running'; } else if (canSchedule.value) { status = 'scheduled'; }
      if (!status) return;
      $api.changeCampaignStatus(data.value.id, status).then(() => { router.push({ name: 'campaigns' }); });
    });
  });
}

function unscheduleCampaign() {
  $api.changeCampaignStatus(data.value.id, 'draft').then((d: any) => { data.value = d; });
}

watch(selectedLists, (v) => { form.lists = v; });
watch(() => data.value.sendAt, (v) => {
  if (v !== null) { form.sendLater = true; form.sendAtDate = dayjs(v).toDate(); } else { form.sendLater = false; form.sendAtDate = null; }
});

onBeforeRouteLeave((_to, _from, next) => {
  if (isUnsaved()) {
    $utils.confirm(t('globals.messages.confirmDiscard'), () => next(true));
    return;
  }
  next(true);
});

onMounted(() => {
  window.onbeforeunload = () => isUnsaved() || null;
  form.fromEmail = (serverConfig.value as any).from_email;

  const { id } = route.params as { id: string };
  if (id === 'new') {
    isNew.value = true;
    if (route.query.list_id) {
      const strIds: string[] = typeof route.query.list_id === 'object'
        ? (route.query.list_id as string[]) : [route.query.list_id as string];
      selListIDs.value = strIds.map((v) => parseInt(v, 10));
    }
  } else {
    const intID = parseInt(id, 10);
    if (intID <= 0 || Number.isNaN(intID)) { $utils.toast(t('campaigns.invalid')); return; }
    isEditing.value = true;
  }

  $api.getTemplates().then((tpls: any) => {
    if (tpls.length > 0 && !form.content.templateId) {
      const tpl = tpls.find((i: any) => i.isDefault === true);
      if (tpl) form.content.templateId = tpl.id;
    }
  });

  if (isEditing.value) {
    getCampaign(id).then(() => {
      if (route.hash !== '') activeTab.value = route.hash.replace('#', '');
    });
  } else {
    form.messenger = 'email';
  }

  nextTick(() => { focusEl.value?.$el?.focus(); });
  $events.$on('campaign.update', () => { onSubmit('update'); });
});

onBeforeUnmount(() => { $events.$off('campaign.update'); });
</script>

<style scoped lang="scss">
.campaign {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

// Header
.page-header-left {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.header-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.id-meta {
  font-size: 0.75rem;
  color: var(--lm-text-subtle);
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

// Two-column layout
.campaign-layout {
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 1.5rem;
  align-items: start;
}

.campaign-sidebar {
  position: sticky;
  top: 1rem;
}

// Campaign form
.campaign-form {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;

  .field { margin-bottom: 0; }
}

.field-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--lm-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.4rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;

  .field { margin-bottom: 0; }
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}
.toggle-label { font-size: 0.9rem; font-weight: 500; color: var(--lm-text); }

.form-link {
  font-size: 0.85rem;
  color: var(--lm-text-muted);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;

  &:hover { color: var(--lm-primary); }
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 0.25rem;
}

// Send test message card
.test-message-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;

  &__title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--lm-text);

    .pi { color: var(--lm-text-muted); font-size: 0.9rem; }
  }

  &__help {
    font-size: 0.8rem;
    color: var(--lm-text-muted);
    line-height: 1.5;
  }
}
</style>
