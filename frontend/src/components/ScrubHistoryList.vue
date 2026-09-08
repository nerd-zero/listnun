<template>
  <div v-if="(serverConfig as any).scrub_enabled" class="scrub-history">
    <div class="scrub-history__toolbar">
      <label class="scrub-history__filter">
        {{ $t('settings.scrub.invalidOnly') }}
        <PvToggleSwitch v-model="invalidOnly" @update:model-value="onFilterChange" />
      </label>
    </div>

    <PvDataTable :value="entries" :loading="loading" data-key="requestId" class="scrub-history__table">
      <template #empty>
        <p class="text-color-secondary" style="font-size:0.9rem">
          {{ invalidOnly ? $t('settings.scrub.historyEmptyInvalid') : $t('settings.scrub.historyEmpty') }}
        </p>
      </template>
      <PvColumn field="email" :header="$t('subscribers.email')" />
      <PvColumn field="status" :header="$t('globals.fields.status')">
        <template #body="{ data }">
          <PvTag
            :severity="data.status === 'invalid' ? 'danger' : 'success'"
            :value="$t(`settings.scrub.historyStatus.${data.status}`)"
          />
        </template>
      </PvColumn>
      <PvColumn field="errorCode" :header="$t('settings.scrub.historyReason')">
        <template #body="{ data }">{{ data.errorCode || '—' }}</template>
      </PvColumn>
      <PvColumn field="createdAt" :header="$t('globals.fields.createdAt')">
        <template #body="{ data }">{{ $utils.niceDate(data.createdAt) }}</template>
      </PvColumn>
    </PvDataTable>

    <div class="scrub-history__pager">
      <PvButton
        severity="secondary" outlined size="small"
        :disabled="cursors.length <= 1 || loading" :label="$t('globals.buttons.prev')"
        @click="onPrev"
      />
      <PvButton
        severity="secondary" outlined size="small"
        :disabled="!hasMore || loading" :label="$t('globals.buttons.next')"
        @click="onNext"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import { getSettings as settingsApi } from '../api/generated/endpoints/settings/settings';

const PAGE_SIZE = 20;

const { getScrubHistory } = settingsApi();
const { $utils } = useGlobal();
const { serverConfig } = storeToRefs(useMainStore());

const entries = ref<any[]>([]);
const loading = ref(false);
const invalidOnly = ref(false);
const hasMore = ref(false);
const nextCursor = ref<string | undefined>(undefined);
// cursors[0] is always undefined ("from the start"); cursors[cursors.length - 1]
// is the cursor for the page currently on screen. Next pushes nextCursor
// (captured from the last response) onto the stack; Previous pops it --
// same scheme as listnun's own ValidationHistoryCard, needed because
// Scrub's history API pages by opaque cursor, not offset.
const cursors = ref<(string | undefined)[]>([undefined]);

function fetchPage() {
  if (!(serverConfig.value as any).scrub_enabled) return;
  loading.value = true;
  getScrubHistory({
    invalid_only: invalidOnly.value,
    limit: PAGE_SIZE,
    cursor: cursors.value[cursors.value.length - 1],
  }).then((res: any) => {
    entries.value = res?.results || [];
    hasMore.value = !!res?.hasMore;
    nextCursor.value = res?.nextCursor;
  }).catch(() => {
    entries.value = [];
    hasMore.value = false;
    nextCursor.value = undefined;
  }).finally(() => { loading.value = false; });
}

function onFilterChange() {
  cursors.value = [undefined];
  fetchPage();
}

function onNext() {
  if (!hasMore.value || !nextCursor.value) return;
  cursors.value = [...cursors.value, nextCursor.value];
  fetchPage();
}

function onPrev() {
  if (cursors.value.length <= 1) return;
  cursors.value = cursors.value.slice(0, -1);
  fetchPage();
}

onMounted(fetchPage);
</script>

<style scoped>
.scrub-history__toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
}
.scrub-history__filter {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--lm-text-muted);
}
.scrub-history__pager {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}
</style>
