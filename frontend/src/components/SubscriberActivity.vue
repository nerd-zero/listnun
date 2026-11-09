<template>
  <div class="subscriber-activity">
    <div v-if="isLoading" class="activity-loading">
      <PvProgressSpinner style="width:2rem;height:2rem" />
    </div>

    <div v-else>
      <!-- Summary Stats -->
      <div class="activity-stats">
        <div class="stat-box">
          <span class="stat-heading">{{ $t('globals.terms.campaigns') }}</span>
          <span class="stat-value">{{ activity.campaignViews ? activity.campaignViews.length : 0 }}</span>
        </div>
        <div class="stat-box">
          <span class="stat-heading">{{ $t('campaigns.views') }}</span>
          <span class="stat-value">{{ totalViews }}</span>
        </div>
        <div class="stat-box">
          <span class="stat-heading">{{ $t('campaigns.clicks') }}</span>
          <span class="stat-value">{{ totalClicks }}</span>
        </div>
      </div>

      <!-- Campaign Views Section -->
      <h5 class="activity-section-title">{{ $t('campaigns.views') }}</h5>

      <div v-if="activity.campaignViews && activity.campaignViews.length > 0">
        <PvDataTable :value="activity.campaignViews" :hoverable="true" sort-field="lastViewedAt" :sort-order="-1"
          :paginator="true" :rows="10" class="campaign-views-table">
          <PvColumn field="subject" :header="$t('globals.terms.campaign', 1)" sortable>
            <template #body="{ data }">
              <div v-if="data.uuid">
                <router-link :to="{ name: 'campaign', params: { id: data.id } }">{{ data.name }}</router-link>
                <p class="cell-sub">{{ data.subject }}</p>
              </div>
              <div v-else>
                <em class="text-muted">{{ $t('subscribers.activity.campaignDeleted') }}</em>
              </div>
            </template>
          </PvColumn>
          <PvColumn field="viewCount" :header="$t('campaigns.views')" sortable>
            <template #body="{ data }">
              <PvTag severity="secondary" :value="String(data.viewCount)" />
            </template>
          </PvColumn>
          <PvColumn field="lastViewedAt" :header="$t('globals.fields.createdAt')" sortable>
            <template #body="{ data }">
              <span v-if="data.lastViewedAt">{{ $utils.niceDate(data.lastViewedAt, true) }}</span>
            </template>
          </PvColumn>
        </PvDataTable>
      </div>
      <div v-else class="activity-empty">
        <p>{{ $t('globals.messages.emptyState') }}</p>
      </div>

      <!-- Link Clicks Section -->
      <h5 class="activity-section-title">{{ $t('campaigns.clicks') }}</h5>

      <div v-if="activity.linkClicks && activity.linkClicks.length > 0">
        <PvDataTable :value="activity.linkClicks" :hoverable="true" sort-field="lastClickedAt" :sort-order="-1"
          :paginator="true" :rows="10" class="link-clicks-table">
          <PvColumn field="url" :header="$t('globals.terms.url')" sortable body-class="link-click-url">
            <template #body="{ data }">
              <a :href="data.url" target="_blank" rel="noopener noreferrer">{{ data.url }}</a>
            </template>
          </PvColumn>
          <PvColumn field="campaignName" :header="$t('globals.terms.campaign', 1)" sortable>
            <template #body="{ data }">
              <div v-if="data.campaignUuid">
                <router-link :to="{ name: 'campaign', params: { id: data.campaignId } }">
                  {{ data.campaignSubject || data.campaignName }}
                </router-link>
              </div>
              <div v-else>&mdash;</div>
            </template>
          </PvColumn>
          <PvColumn field="clickCount" :header="$t('campaigns.clicks')" sortable>
            <template #body="{ data }">
              <PvTag severity="secondary" :value="String(data.clickCount)" />
            </template>
          </PvColumn>
          <PvColumn field="lastClickedAt" :header="$t('globals.fields.createdAt')" sortable>
            <template #body="{ data }">
              <span v-if="data.lastClickedAt">{{ $utils.niceDate(data.lastClickedAt, true) }}</span>
            </template>
          </PvColumn>
        </PvDataTable>
      </div>
      <div v-else class="activity-empty">
        <p>{{ $t('globals.messages.emptyState') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';

const props = defineProps<{
  subscriberId: number;
}>();

const { getSubscriberActivity } = subscribersApi();

const isLoading = ref(false);
const activity = ref<{ campaignViews: any[]; linkClicks: any[] }>({
  campaignViews: [],
  linkClicks: [],
});

const totalViews = computed(() => {
  if (!activity.value.campaignViews) return 0;
  return activity.value.campaignViews.reduce((sum: number, v: any) => sum + (v.viewCount || 0), 0);
});

const totalClicks = computed(() => {
  if (!activity.value.linkClicks) return 0;
  return activity.value.linkClicks.reduce((sum: number, c: any) => sum + (c.clickCount || 0), 0);
});

function getActivity() {
  isLoading.value = true;
  getSubscriberActivity(props.subscriberId).then((data: any) => {
    activity.value = data;
    isLoading.value = false;
  }).catch(() => {
    isLoading.value = false;
  });
}

onMounted(() => {
  getActivity();
});
</script>

<style scoped lang="scss">
.activity-loading { display: flex; justify-content: center; padding: 2rem; }

.activity-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  background: var(--ln-bg);
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  padding: 1rem;
}

.stat-heading {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--ln-text-3);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--ln-text);
  line-height: 1;
}

.activity-section-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--ln-text);
  margin: 1.25rem 0 0.75rem;
}

.cell-sub {
  font-size: 0.75rem;
  color: var(--ln-text-4);
  margin: 0.15rem 0 0;
}

.text-muted { color: var(--ln-text-4); font-style: italic; }

.activity-empty {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--ln-text-4);
  font-size: 0.875rem;
}

:deep(.p-tag-secondary) {
  background: var(--ln-muted);
  color: var(--ln-text-2);
  border: 1px solid var(--ln-border);
}
</style>
