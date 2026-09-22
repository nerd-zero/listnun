<template>
  <div class="bounces-page">
    <div class="page-header">
      <h1 class="page-title">
        {{ $t('globals.terms.bounces') }}
        <span v-if="bounces.total > 0" class="page-title-count">{{ bounces.total }}</span>
      </h1>
    </div>

    <div class="table-card">
      <PvDataTable :value="bounces.results" :loading="loading.bounces"
        sort-field="createdAt" :sort-order="-1"
        selection-mode="checkbox" v-model:selection="bulk.checked"
        @update:selection="onTableCheck"
        :rows="bounces.perPage" :paginator="true" paginator-position="bottom" :total-records="bounces.total"
        :lazy="true" @page="(e) => onPageChange(e.page + 1)"
        @sort="(e) => onSort(e.sortField, e.sortOrder === 1 ? 'asc' : 'desc')"
        data-key="id"
        :expanded-rows="expandedRows" @update:expanded-rows="expandedRows = $event">
        <template #header>
          <div v-if="bulk.checked.length > 0" class="bulk-bar">
            <span class="bulk-count">
              {{ $t('globals.messages.numSelected', { num: numSelectedBounces }) }}
              <template v-if="!bulk.all && bounces.total > bounces.perPage">
                &mdash; <a href="#" @click.prevent="selectAllBounces">{{ $t('subscribers.selectAll', { num: bounces.total }) }}</a>
              </template>
            </span>
            <button type="button" class="bulk-btn bulk-btn--danger" data-cy="btn-delete"
              @click="$utils.confirm(null, () => onDeleteBounces())">
              <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
            </button>
            <button type="button" class="bulk-btn bulk-btn--warn" data-cy="btn-manage-blocklist"
              @click="$utils.confirm(null, () => onBlocklistSubscribers())">
              <i class="pi pi-ban" /> {{ $t('import.blocklist') }}
            </button>
          </div>
        </template>

        <PvColumn selection-mode="multiple" header-style="width:3rem" />
        <PvColumn expander header-style="width:3rem" />

        <PvColumn field="email" :header="$t('subscribers.email')" sortable>
          <template #body="{ data }">
            <router-link class="row-name" :class="{ 'row-name--blocked': data.subscriberStatus === 'blocklisted' }"
              :to="{ name: 'subscriber', params: { id: data.subscriberId } }">
              {{ data.email }}
            </router-link>
            <PvTag v-if="data.subscriberStatus !== 'enabled'" severity="danger" size="small"
              data-cy="blocklisted" :value="$t(`subscribers.status.${data.subscriberStatus}`)" />
          </template>
        </PvColumn>

        <PvColumn field="campaign" :header="$t('globals.terms.campaign')" sortable>
          <template #body="{ data }">
            <router-link v-if="data.campaign" class="row-link"
              :to="{ name: 'bounces', query: { campaign_id: data.campaign.id } }">
              {{ data.campaign.name }}
            </router-link>
            <span v-else class="text-muted">-</span>
          </template>
        </PvColumn>

        <PvColumn field="source" :header="$t('bounces.source')" sortable>
          <template #body="{ data }">
            <router-link class="row-link" :to="{ name: 'bounces', query: { source: data.source } }">
              {{ data.source }}
            </router-link>
          </template>
        </PvColumn>

        <PvColumn field="type" :header="$t('globals.fields.type')" sortable style="width:8rem">
          <template #body="{ data }">
            <PvTag :severity="data.type === 'hard' ? 'danger' : 'warn'" size="small"
              :value="$t(`bounces.${data.type}`)" />
          </template>
        </PvColumn>

        <PvColumn field="created_at" :header="$t('globals.fields.createdAt')" sortable style="width:11rem">
          <template #body="{ data }">{{ $utils.niceDate(data.createdAt, true) }}</template>
        </PvColumn>

        <PvColumn style="width:4rem; text-align:right" align-frozen="right">
          <template #body="{ data }">
            <div class="row-actions">
              <button type="button" class="row-action-btn row-action-btn--danger"
                data-cy="btn-delete" v-tooltip.bottom="$t('globals.buttons.delete')"
                @click="$utils.confirm(null, () => onDeleteBounce(data))">
                <i class="pi pi-trash" />
              </button>
            </div>
          </template>
        </PvColumn>

        <template #expansion="{ data }">
          <pre class="meta-expansion">{{ data.meta }}</pre>
        </template>

        <template #empty v-if="!loading.bounces">
          <empty-placeholder />
        </template>
      </PvDataTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, watch, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { makeQueryHandlers } from '../utils';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import { getBounces as bouncesApi } from '../api/generated/endpoints/bounces/bounces';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

const { $utils } = useGlobal();
const {
  listBounces, deleteBounce, deleteBounces, blocklistBouncedSubscribers,
} = bouncesApi();
const { blocklistSubscribers } = subscribersApi();
const { t, tc } = useI18n();
const route = useRoute();
const { refreshTick, loading } = storeToRefs(useMainStore());

const bounces = ref<any>({});
const expandedRows = ref<any[]>([]);
const bulk = reactive({ checked: [] as any[], all: false });
const queryParams = reactive({
  page: 1, orderBy: 'created_at', order: 'desc', campaignId: 0, source: '',
});

const numSelectedBounces = computed(() => (bulk.all ? (bounces.value as any).total : bulk.checked.length));

function getBounces() {
  bulk.checked = [];
  bulk.all = false;
  listBounces({
    page: queryParams.page,
    order_by: queryParams.orderBy as any,
    order: queryParams.order as any,
    campaign_id: queryParams.campaignId,
    source: queryParams.source,
  }).then((data: any) => { bounces.value = data; });
}

const { onPageChange, onSort } = makeQueryHandlers(queryParams, getBounces);

function selectAllBounces() { bulk.all = true; }

function onTableCheck() {
  if (bulk.checked.length !== (bounces.value as any).total) bulk.all = false;
}

function onDeleteBounce(b: any) {
  deleteBounce(b.id).then(() => {
    getBounces();
    $utils.toast(t('globals.messages.deleted', { name: b.email }));
  });
}

function onDeleteBounces() {
  const params: any = {};
  if (!bulk.all && bulk.checked.length > 0) {
    params.id = bulk.checked.map((s: any) => s.id);
  } else if (bulk.all) {
    params.all = true;
  }
  deleteBounces(params).then(() => {
    getBounces();
    $utils.toast(t('globals.messages.deletedCount', { name: tc('globals.terms.bounces'), num: numSelectedBounces.value }));
  });
}

function onBlocklistSubscribers() {
  const cb = () => { getBounces(); $utils.toast(t('globals.messages.done')); };
  if (!bulk.all && bulk.checked.length > 0) {
    const subIds = bulk.checked.map((s: any) => s.subscriberId);
    blocklistSubscribers({ ids: subIds }).then(cb);
    return;
  }
  blocklistBouncedSubscribers().then(cb);
}

watch(() => refreshTick.value, () => { getBounces(); });

onMounted(() => {
  if (route.query.campaign_id) queryParams.campaignId = parseInt(route.query.campaign_id as string, 10);
  if (route.query.source) queryParams.source = route.query.source as string;
  getBounces();
});
</script>

<style scoped lang="scss">
.bounces-page { display: flex; flex-direction: column; gap: 1.5rem; }

.bulk-bar { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; }
.bulk-count { font-size: 0.85rem; color: var(--lm-text-muted); a { color: var(--lm-primary); text-decoration: none; } }
.bulk-btn {
  display: inline-flex; align-items: center; gap: 0.35rem;
  padding: 0.35rem 0.75rem; border: 1px solid var(--lm-border); border-radius: 6px;
  background: var(--lm-surface); font-size: 0.8rem; cursor: pointer; color: var(--lm-text-muted);
  &--danger { color: var(--lm-danger); border-color: var(--lm-danger-border); &:hover { background: var(--lm-danger-bg); } }
  &--warn   { color: var(--lm-warn, #d97706); border-color: var(--lm-warn-border); &:hover { background: var(--lm-warn-bg); } }
}

.row-name { color: var(--lm-text); font-weight: 500; text-decoration: none; &:hover { color: var(--lm-primary); } &--blocked { color: var(--lm-text-subtle); text-decoration: line-through; } }
.row-link  { color: var(--lm-primary); text-decoration: none; &:hover { text-decoration: underline; } }
.text-muted { color: var(--lm-text-subtle); }

.meta-expansion { font-size: 0.78rem; padding: 1rem; background: var(--lm-bg); border-radius: 6px; margin: 0.5rem 1rem; }
</style>
