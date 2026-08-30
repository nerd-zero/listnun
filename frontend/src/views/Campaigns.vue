<template>
  <div class="campaigns-page">
    <!-- Page header -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">
          {{ $t('globals.terms.campaigns') }}
          <span v-if="!isNaN(campaigns.total)" class="page-title-count">{{ campaigns.total }}</span>
        </h1>
      </div>
      <router-link v-if="$can('campaigns:manage')" :to="{ name: 'campaign', params: { id: 'new' } }" data-cy="btn-new">
        <PvButton severity="primary" icon="pi pi-plus" :label="$t('globals.buttons.new')" />
      </router-link>
    </div>

    <!-- Table card -->
    <div class="table-card">
      <PvDataTable
        :value="campaigns.results"
        :loading="loading.campaigns"
        :row-class="highlightedRow"
        :rows="campaigns.perPage"
        :total-records="campaigns.total"
        paginator
        paginator-position="bottom"
        @page="(e) => onPageChange(e.page + 1)"
        :first="(queryParams.page - 1) * campaigns.perPage"
        data-key="id"
        v-model:selection="bulk.checked"
        selection-mode="multiple"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        lazy
      >
        <template #header>
          <div class="table-toolbar">
            <form class="search-form" @submit.prevent="fetchCampaigns">
              <PvIconField>
                <PvInputIcon class="pi pi-search" />
                <PvInputText v-model="queryParams.query" name="query" class="search-input"
                  :placeholder="$t('campaigns.queryPlaceholder')" ref="query" />
              </PvIconField>
            </form>

            <div v-if="bulk.checked.length > 0" class="bulk-bar">
              <span class="bulk-count">
                {{ $t('globals.messages.numSelected', numSelectedCampaigns, { num: numSelectedCampaigns }) }}
                <template v-if="!bulk.all && campaigns.total > campaigns.perPage">
                  &mdash;
                  <a href="#" @click.prevent="onSelectAll" data-cy="select-all-campaigns">
                    {{ $t('globals.messages.selectAll', campaigns.total, { num: campaigns.total }) }}
                  </a>
                </template>
              </span>
              <button type="button" class="bulk-btn bulk-btn--danger" @click.prevent="onDeleteCampaigns" data-cy="btn-delete-campaigns">
                <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
              </button>
            </div>
          </div>
        </template>

        <PvColumn selection-mode="multiple" header-style="width:3rem" />

        <PvColumn field="status" :header="$t('globals.fields.status')" header-class="cy-status" style="width:11%" sortable>
          <template #body="{ data }">
            <router-link :to="{ name: 'campaign', params: { id: data.id } }" class="status-link">
              <PvTag
                :severity="statusSeverity(data.status)" :value="$t(`campaigns.status.${data.status}`)"
                v-tooltip.bottom="data.pauseReason ? $t(`campaigns.pauseReason.${data.pauseReason}`) : null"
              />
              <PvProgressSpinner v-if="isRunning(data.id)" style="width:1rem;height:1rem" />
            </router-link>
            <div v-if="isSheduled(data)" class="scheduled-info">
              <i class="pi pi-clock" />
              <span v-if="!isDone(data) && !isRunning(data)">{{ $utils.duration(new Date(), data.sendAt, true) }}<br /></span>
              <span>{{ $utils.niceDate(data.sendAt, true) }}</span>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="name" :header="$t('globals.fields.name')" header-class="cy-name" style="width:26%" sortable>
          <template #body="{ data }">
            <div class="name-cell">
              <div class="name-row">
                <PvTag v-if="data.type === 'optin'" severity="secondary" size="small" :value="$t('lists.optin')" />
                <router-link class="row-name" :to="{ name: 'campaign', params: { id: data.id } }">
                  {{ data.name }}
                  <copy-text :text="data.name" hide-text />
                </router-link>
              </div>
              <div class="subject-row">
                <copy-text :text="data.subject" />
              </div>
              <div v-if="data.tags?.length" class="tag-row">
                <PvTag v-for="t in data.tags" :key="t" :value="t" severity="secondary" size="small" />
              </div>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="lists" :header="$t('globals.terms.lists')" style="width:13%">
          <template #body="{ data }">
            <div class="lists-cell">
              <router-link v-for="l in data.lists" :key="l.id" class="list-link"
                :to="{ name: 'subscribers_list', params: { listID: l.id } }">
                {{ l.name }}
              </router-link>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('campaigns.timestamps')" header-class="cy-timestamp" style="width:16%" sortable>
          <template #body="{ data }">
            <div class="ts-cell" :set="stats = getCampaignStats(data)">
              <div class="ts-row"><span class="ts-label">{{ $t('globals.fields.createdAt') }}</span><span>{{ $utils.niceDate(data.createdAt, true) }}</span></div>
              <div v-if="stats.startedAt" class="ts-row"><span class="ts-label">{{ $t('campaigns.startedAt') }}</span><span>{{ $utils.niceDate(stats.startedAt, true) }}</span></div>
              <div v-if="isDone(data)" class="ts-row"><span class="ts-label">{{ $t('campaigns.ended') }}</span><span>{{ $utils.niceDate(stats.updatedAt, true) }}</span></div>
              <div v-if="stats.startedAt && stats.updatedAt" class="ts-row"><i class="pi pi-clock ts-label" /><span>{{ $utils.duration(stats.startedAt, stats.updatedAt) }}</span></div>
            </div>
          </template>
        </PvColumn>

        <PvColumn field="stats" :header="$t('campaigns.stats')" style="width:15%">
          <template #body="{ data }">
            <div class="stats-cell" :set="stats = getCampaignStats(data)">
              <div class="stats-row"><span class="stats-label">{{ $t('campaigns.views') }}</span><span>{{ $utils.formatNumber(data.views) }}</span></div>
              <div class="stats-row"><span class="stats-label">{{ $t('campaigns.clicks') }}</span><span>{{ $utils.formatNumber(data.clicks) }}</span></div>
              <div class="stats-row"><span class="stats-label">{{ $t('campaigns.sent') }}</span><span>{{ $utils.formatNumber(stats.sent) }} / {{ $utils.formatNumber(stats.toSend) }}</span></div>
              <div class="stats-row">
                <span class="stats-label">{{ $t('globals.terms.bounces') }}</span>
                <router-link :to="{ name: 'bounces', query: { campaign_id: data.id } }">{{ $utils.formatNumber(data.bounces) }}</router-link>
              </div>
              <div v-if="stats.rate" class="stats-row">
                <span class="stats-label"><i class="pi pi-gauge" /></span>
                <span v-tooltip.bottom="`${stats.netRate} / ${$t('campaigns.rateMinuteShort')} @ ${$utils.duration(stats.startedAt, stats.updatedAt)}`">
                  {{ stats.rate.toFixed(0) }} / {{ $t('campaigns.rateMinuteShort') }}
                </span>
              </div>
              <div v-if="isRunning(data.id)" class="stats-row stats-row--progress">
                <span class="stats-label">{{ $t('campaigns.progress') }} <PvProgressSpinner style="width:0.8rem;height:0.8rem" /></span>
                <PvProgressBar :value="stats.sent / stats.toSend * 100" style="height:5px;flex:1" />
              </div>
            </div>
          </template>
        </PvColumn>

        <PvColumn style="width:9rem; text-align:right">
          <template #body="{ data }">
            <div class="row-actions">
              <template v-if="$can('campaigns:send')">
                <button v-if="canStart(data)" type="button" class="row-action-btn row-action-btn--primary"
                  data-cy="btn-start" v-tooltip.bottom="$t('campaigns.start')"
                  @click="$utils.confirm(null, () => changeCampaignStatus(data, 'running'))">
                  <i class="pi pi-send" />
                </button>
                <button v-if="canPause(data)" type="button" class="row-action-btn"
                  data-cy="btn-pause" v-tooltip.bottom="$t('campaigns.pause')"
                  @click="$utils.confirm(null, () => changeCampaignStatus(data, 'paused'))">
                  <i class="pi pi-pause" />
                </button>
                <button v-if="canResume(data)" type="button" class="row-action-btn row-action-btn--primary"
                  data-cy="btn-resume" v-tooltip.bottom="$t('campaigns.send')"
                  @click="$utils.confirm(null, () => changeCampaignStatus(data, 'running'))">
                  <i class="pi pi-send" />
                </button>
                <button v-if="canSchedule(data)" type="button" class="row-action-btn"
                  data-cy="btn-schedule" v-tooltip.bottom="$t('campaigns.schedule')"
                  @click="$utils.confirm($t('campaigns.confirmSchedule'), () => changeCampaignStatus(data, 'scheduled'))">
                  <i class="pi pi-clock" />
                </button>
                <button v-if="canCancel(data)" type="button" class="row-action-btn row-action-btn--danger"
                  data-cy="btn-cancel" v-tooltip.bottom="$t('globals.buttons.cancel')"
                  @click="$utils.confirm(null, () => changeCampaignStatus(data, 'cancelled'))">
                  <i class="pi pi-times-circle" />
                </button>
              </template>

              <button type="button" class="row-action-btn" data-cy="btn-preview"
                v-tooltip.bottom="$t('campaigns.preview')" @click="previewCampaign(data)">
                <i class="pi pi-eye" />
              </button>
              <button v-if="$can('campaigns:manage')" type="button" class="row-action-btn" data-cy="btn-clone"
                v-tooltip.bottom="$t('globals.buttons.clone')"
                @click="$utils.prompt($t('globals.buttons.clone'),
                  { placeholder: $t('globals.fields.name'), value: $t('campaigns.copyOf', { name: data.name }) },
                  (name) => cloneCampaign(name, data))">
                <i class="pi pi-copy" />
              </button>
              <router-link v-if="$can('campaigns:get_analytics')" class="row-action-btn"
                :to="{ name: 'campaignAnalytics', query: { id: data.id } }" v-tooltip.bottom="$t('globals.terms.analytics')">
                <i class="pi pi-chart-bar" />
              </router-link>
              <button v-if="$can('campaigns:manage')" type="button" class="row-action-btn row-action-btn--danger"
                data-cy="btn-delete" v-tooltip.bottom="$t('globals.buttons.delete')"
                @click="$utils.confirm($t('campaigns.confirmDelete', { name: data.name }), () => onDeleteCampaign(data))">
                <i class="pi pi-trash" />
              </button>
            </div>
          </template>
        </PvColumn>

        <template #empty v-if="!loading.campaigns">
          <empty-placeholder />
        </template>
      </PvDataTable>
    </div>

    <campaign-preview v-if="previewItem" type="campaign" :id="previewItem.id" :title="previewItem.name"
      @close="closePreview" />
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, onMounted, onUnmounted,
} from 'vue';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import CampaignPreview from '../components/CampaignPreview.vue';
import CopyText from '../components/CopyText.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';

const { $utils } = useGlobal();
const {
  listCampaigns, getCampaign, createCampaign, deleteCampaign, deleteCampaigns,
  updateCampaignStatus, getRunningCampaignStats,
} = campaignsApi();
const { t, tc } = useI18n();
const router = useRouter();
const { refreshTick, campaigns, loading } = storeToRefs(useMainStore());

const previewItem = ref<any>(null);
const pollID = ref<any>(null);
const campaignStatsData = ref<Record<number, any>>({});
const bulk = reactive({ checked: [] as any[], all: false });
const queryParams = reactive({
  page: 1, query: '', orderBy: 'created_at', order: 'desc',
});

const numSelectedCampaigns = computed(() => (bulk.all ? (campaigns.value as any).total : bulk.checked.length));

function statusSeverity(status: string) {
  const map: Record<string, string> = {
    running: 'success',
    finished: 'info',
    scheduled: 'warn',
    paused: 'secondary',
    cancelled: 'danger',
    draft: 'secondary',
  };
  return map[status] || 'secondary';
}

const canStart = (c: any) => c.status === 'draft' && !c.sendAt;
const canSchedule = (c: any) => c.status === 'draft' && c.sendAt;
const canPause = (c: any) => c.status === 'running';
const canCancel = (c: any) => c.status === 'running' || c.status === 'paused';
const canResume = (c: any) => c.status === 'paused';
const isSheduled = (c: any) => c.status === 'scheduled' || c.sendAt !== null;
const isDone = (c: any) => c.status === 'finished' || c.status === 'cancelled';
const isRunning = (id: number) => id in campaignStatsData.value;

function highlightedRow(data: any) {
  return data.status === 'running' ? ['running'] : '';
}

function getCampaignStats(c: any) {
  return c.id in campaignStatsData.value ? campaignStatsData.value[c.id] : c;
}

function fetchCampaigns() {
  listCampaigns({
    page: queryParams.page,
    query: queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
    order_by: queryParams.orderBy,
    order: queryParams.order,
    no_body: true,
  }).then((resp: any) => { campaigns.value = resp; });
}

function onPageChange(p: number) { queryParams.page = p; fetchCampaigns(); }
function onSort(field: string, direction: string) { queryParams.orderBy = field; queryParams.order = direction; fetchCampaigns(); }
function previewCampaign(c: any) { previewItem.value = c; }
function closePreview() { previewItem.value = null; }

function pollStats() {
  clearInterval(pollID.value);
  pollID.value = setInterval(() => {
    getRunningCampaignStats().then((data: any) => {
      if (data.length === 0) {
        clearInterval(pollID.value);
        if (Object.keys(campaignStatsData.value).length > 0) {
          fetchCampaigns();
          campaignStatsData.value = {};
        }
      } else {
        campaignStatsData.value = data.reduce((obj: any, cur: any) => ({ ...obj, [cur.id]: cur }), {});
      }
    }, () => { clearInterval(pollID.value); });
  }, 1000);
}

function changeCampaignStatus(c: any, status: string) {
  updateCampaignStatus(c.id, { status }).then(() => {
    $utils.toast(t('campaigns.statusChanged', { name: c.name, status }));
    fetchCampaigns();
    pollStats();
  });
}

async function cloneCampaign(name: string, c: any) {
  let body = '';
  let bodySource = null;
  await getCampaign(c.id).then((data: any) => { body = data.body; bodySource = data.bodySource; });
  const now = $utils.getDate();
  const sendLater = !!c.sendAt;
  let sendAt = null;
  if (sendLater) sendAt = dayjs(c.sendAt).isAfter(now) ? c.sendAt : now.add(7, 'day');
  const data: any = {
    name,
    subject: c.subject,
    lists: c.lists.map((l: any) => l.id),
    type: c.type,
    from_email: c.fromEmail,
    content_type: c.contentType,
    messenger: c.messenger,
    tags: c.tags,
    template_id: c.templateId,
    body,
    body_source: bodySource,
    altbody: c.altbody,
    headers: c.headers,
    send_later: sendLater,
    send_at: sendAt,
    archive: c.archive,
    archive_template_id: c.archiveTemplateId,
    archive_meta: c.archiveMeta,
    media: c.media.map((m: any) => m.id),
  };
  if (c.archive) data.archive_slug = `${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
  createCampaign(data).then((d: any) => { router.push({ name: 'campaign', params: { id: d.id } }); });
}

function onDeleteCampaign(c: any) {
  deleteCampaign(c.id).then(() => {
    fetchCampaigns();
    $utils.toast(t('globals.messages.deleted', { name: c.name }));
  });
}

function onSelectAll() { bulk.all = true; }

function onDeleteCampaigns() {
  const name = tc('globals.terms.campaign', numSelectedCampaigns.value);
  const fn = () => {
    const params: any = {};
    if (!bulk.all && bulk.checked.length > 0) {
      params.id = bulk.checked.map((c: any) => c.id);
    } else {
      params.query = queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' ');
      params.all = bulk.all;
    }
    deleteCampaigns(params).then(() => {
      fetchCampaigns();
      $utils.toast(tc('globals.messages.deletedCount', numSelectedCampaigns.value, { num: numSelectedCampaigns.value, name }));
    });
  };
  $utils.confirm(tc('globals.messages.confirmDelete', numSelectedCampaigns.value, { num: numSelectedCampaigns.value, name: name.toLowerCase() }), fn);
}

watch(() => refreshTick.value, () => { fetchCampaigns(); });
onMounted(() => { fetchCampaigns(); pollStats(); });
onUnmounted(() => { clearInterval(pollID.value); });
</script>

<style scoped lang="scss">
.campaigns-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.table-toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}
.search-form { flex: 0 0 280px; }
.search-input { width: 100%; }

.bulk-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
}
.bulk-count {
  font-size: 0.85rem;
  color: var(--lm-text-muted);
  a { color: var(--lm-primary); text-decoration: none; }
}
.bulk-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--lm-border);
  border-radius: 6px;
  background: var(--lm-surface);
  font-size: 0.8rem;
  cursor: pointer;
  color: var(--lm-text-muted);

  &--danger { color: var(--lm-danger); border-color: var(--lm-danger-border); &:hover { background: var(--lm-danger-bg); } }
}

// Status column
.status-link {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  text-decoration: none;
}
.scheduled-info {
  margin-top: 0.35rem;
  font-size: 0.75rem;
  color: var(--lm-text-subtle);
  display: flex;
  align-items: flex-start;
  gap: 0.25rem;
}

// Name column
.name-cell { display: flex; flex-direction: column; gap: 0.25rem; }
.name-row { display: flex; align-items: center; gap: 0.4rem; }
.row-name { color: var(--lm-text); font-weight: 500; text-decoration: none; &:hover { color: var(--lm-primary); } }
.subject-row { font-size: 0.78rem; color: var(--lm-text-subtle); }
.tag-row { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-top: 0.2rem; }

// Lists column
.lists-cell { display: flex; flex-direction: column; gap: 0.2rem; }
.list-link { font-size: 0.82rem; color: var(--lm-primary); text-decoration: none; &:hover { text-decoration: underline; } }

// Timestamps column
.ts-cell { display: flex; flex-direction: column; gap: 0.2rem; }
.ts-row { display: flex; align-items: baseline; gap: 0.4rem; font-size: 0.8rem; }
.ts-label { font-size: 0.74rem; color: var(--lm-text-subtle); white-space: nowrap; }

// Status tag — make secondary (draft/paused) visually distinct
.status-link :deep(.p-tag-secondary) {
  background: var(--lm-bg-subtle);
  color: var(--lm-text-secondary);
  border: 1px solid var(--lm-border);
}

// Stats column
.stats-cell { display: flex; flex-direction: column; gap: 0.25rem; }
.stats-row {
  display: flex;
  align-items: baseline;
  gap: 0.45rem;
  font-size: 0.82rem;
}
.stats-label { font-size: 0.76rem; color: var(--lm-text-muted); white-space: nowrap; min-width: 52px; display: flex; align-items: center; gap: 0.2rem; }

// Row actions always visible, not just on row hover.
:deep(.row-actions) { opacity: 1; }
</style>
