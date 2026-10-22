<template>
  <section class="ln-card act-log">
    <header class="act-log-head">
      <span class="ln-small">{{ $t('settings.logs.lastLines', { n: MAX_LINES }) }}</span>
      <label class="ln-check-row act-log-auto" for="act-log-auto">
        <PvToggleSwitch v-model="auto" input-id="act-log-auto" />
        {{ $t('settings.logs.autoRefresh') }}
      </label>
      <PvButton :label="$t('globals.buttons.refresh')" size="small" severity="secondary" outlined :loading="loading" @click="load" />
    </header>

    <table-skeleton v-if="loading && !lines.length" :rows="8" :cols="1" />
    <error-state v-else-if="error" :title="$t('states.couldntLoad')" :error="error" @retry="load" />
    <p v-else-if="!lines.length" class="ln-small act-log-empty">{{ $t('settings.logs.empty') }}</p>
    <ol v-else class="act-log-lines ln-mono" data-cy="activity-log">
      <li v-for="l in lines" :key="l.id" :class="`is-${l.level}`">
        <time :datetime="l.createdAt">{{ stamp(l.createdAt) }}</time>
        <span class="act-log-src">{{ l.source }}:</span>
        <span class="act-log-msg">{{ l.message }}</span>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import {
  onBeforeUnmount, onMounted, ref, watch,
} from 'vue';
import dayjs from 'dayjs';
import axios from 'axios';
import TableSkeleton from '../../components/ui/TableSkeleton.vue';
import ErrorState, { type RequestError } from '../../components/ui/ErrorState.vue';

// Settings › Logs: the workspace's own recent activity, read only
// (GET /api/settings/logs; tenant-scoped, see migration v6.31.0).
interface ActivityLine {
  id: number;
  level: 'info' | 'warn' | 'error';
  source: string;
  message: string;
  createdAt: string;
}

// The design shows the latest 500 lines.
const MAX_LINES = 500;
// How often auto-refresh reloads the log.
const REFRESH_MS = 10_000;

const lines = ref<ActivityLine[]>([]);
const loading = ref(false);
const error = ref<RequestError | null>(null);
const auto = ref(true);
let timer: number | undefined;

function stamp(t: string): string {
  return dayjs(t).format('YYYY/MM/DD HH:mm:ss');
}

function load() {
  loading.value = true;
  axios.get('/api/settings/logs', { params: { limit: MAX_LINES } })
    .then((r) => {
      const rows = (r.data?.data || []) as Record<string, unknown>[];
      lines.value = rows.map((x) => ({
        id: x.id as number,
        level: x.level as ActivityLine['level'],
        source: x.source as string,
        message: x.message as string,
        createdAt: x.created_at as string,
      }));
      error.value = null;
    })
    .catch((e) => {
      error.value = {
        status: e.response?.status, method: 'GET', url: '/api/settings/logs', message: e.response?.data?.message,
      };
    })
    .finally(() => { loading.value = false; });
}

function schedule() {
  window.clearInterval(timer);
  if (auto.value) timer = window.setInterval(load, REFRESH_MS);
}

watch(auto, schedule);
onMounted(() => { load(); schedule(); });
onBeforeUnmount(() => window.clearInterval(timer));
</script>

<style scoped lang="scss">
.act-log { display: flex; flex-direction: column; overflow: hidden; }
.act-log-head {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 12px 16px;
  border-bottom: 1px solid var(--ln-border);
  > .ln-small { flex: 1; margin: 0; }
}
.act-log-auto { font-size: 13px; }
.act-log-empty { margin: 0; padding: 24px 16px; }
.act-log-lines {
  list-style: none;
  margin: 0;
  padding: 12px 16px;
  max-height: 60vh;
  overflow: auto;
  font-size: 12px;
  line-height: 20px;

  li { display: flex; gap: 10px; white-space: pre-wrap; overflow-wrap: anywhere; }
  time { flex: none; color: var(--ln-text-3); }
  .act-log-src { flex: none; color: var(--ln-text-2); }
  .is-warn .act-log-msg { color: var(--ln-warn-fg); }
  .is-error .act-log-msg { color: var(--ln-danger-fg); }
}
</style>
