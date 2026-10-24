<template>
  <div v-if="ic" class="set-cards">
    <set-card :title="$t('settings.inx.title')">
      <set-row :label="$t('settings.inactive.enable')" :help="$t('settings.inx.enableHelp')" toggle>
        <PvToggleSwitch v-model="ic.enabled" input-id="ic-enabled" :aria-label="$t('settings.inactive.enable')" />
        <template v-if="status && !status.tracking" #note>{{ $t('settings.inactive.noTracking') }}</template>
      </set-row>
      <set-row :label="$t('settings.inx.who')" :help="managed ? $t('settings.inactive.managedHelp') : $t('settings.inactive.customHelp')">
        <PvSelectButton
          v-model="ic.mode"
          :options="modes"
          option-label="label"
          option-value="value"
          :allow-empty="false"
          name="inactive_cleanup.mode"
          :aria-label="$t('settings.inx.who')"
        />
      </set-row>
      <set-row :label="$t('settings.inactive.days')" :help="$t('settings.inactive.daysHelp')">
        <locked-value v-if="managed" :text="daysLabel(status?.managed.days)" />
        <PvSelect v-else v-model="ic.days" :options="dayOptions" option-label="label" option-value="value" :aria-label="$t('settings.inactive.days')" />
      </set-row>
      <set-row :label="$t('settings.inactive.activity')" :help="$t('settings.inactive.activityHelp')">
        <locked-value v-if="managed" :text="optionLabel(activityOptions, status?.managed.activity)" />
        <PvSelect
          v-else
          v-model="ic.activity"
          :options="activityOptions"
          option-label="label"
          option-value="value"
          :aria-label="$t('settings.inactive.activity')"
        />
      </set-row>
      <set-row :label="$t('settings.inactive.action')" :help="$t('settings.inactive.actionHelp')">
        <locked-value v-if="managed" :text="optionLabel(actionOptions, status?.managed.action)" />
        <PvSelect v-else v-model="ic.action" :options="actionOptions" option-label="label" option-value="value" :aria-label="$t('settings.inactive.action')" />
        <template v-if="!managed && ic.action === 'delete'" #note>{{ $t('settings.inactive.deleteNote') }}</template>
      </set-row>
      <set-row :label="$t('settings.inactive.lists')">
        <locked-value v-if="managed" :text="listsLabel(status?.managed.list_ids)" />
        <PvMultiSelect
          v-else
          v-model="ic.list_ids"
          :options="allLists"
          option-label="name"
          option-value="id"
          :placeholder="$t('settings.inactive.allLists')"
          filter
          display="comma"
          :max-selected-labels="3"
          :aria-label="$t('settings.inactive.lists')"
        />
      </set-row>
      <set-row :label="$t('settings.inactive.schedule')">
        <locked-value v-if="managed" :text="scheduleLabel(status?.managed.schedule)" />
        <PvSelect
          v-else
          v-model="ic.schedule"
          :options="scheduleOptions"
          option-label="label"
          option-value="value"
          :aria-label="$t('settings.inactive.schedule')"
        />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.inx.next')">
      <set-row :label="`${$t('settings.inactive.nextRun')}: ${status?.next_run ? niceDate(status.next_run) : $t('settings.inactive.notScheduled')}`" :help="nextHelp" toggle>
        <PvButton :label="$t('settings.inactive.preview')" severity="secondary" outlined :loading="previewing" @click="preview" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import {
  computed, defineComponent, h, onMounted, ref,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import dayjs from 'dayjs';
import axios from 'axios';
import { useMainStore } from '../../store';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

// Settings › Inactive cleanup (the design's "Scrub" screen; named apart
// from the Scrub address-validation service). The rules live in the
// inactive_cleanup setting; managed values and the next run come from
// GET /api/settings/inactive-cleanup (cmd/cleanup.go).

interface Rules {
  days: number;
  activity: string;
  action: string;
  list_ids: number[] | null;
  schedule: string;
}
interface CleanupSettings extends Rules {
  enabled: boolean;
  mode: 'managed' | 'custom';
}
interface CleanupRun {
  started_at: string;
  finished_at: string | null;
  matched: number;
  done: number;
  error: string;
}
interface CleanupStatus {
  managed: Rules;
  matching: number;
  tracking: boolean;
  next_run: string | null;
  last_run: CleanupRun | null;
}
interface Option<T> { label: string; value: T }

// A read-only field with a lock, for Listnun's managed values.
const LockedValue = defineComponent({
  props: { text: { type: String, default: '' } },
  setup(p) {
    return () => h('span', { class: 'ic-locked' }, [p.text, h('i', { class: 'pi pi-lock', 'aria-hidden': 'true' })]);
  },
});

// Inactivity windows offered in custom mode (cleanup.MinDays is 30).
const DAY_CHOICES = [30, 60, 90, 120, 180, 270, 365];

const props = defineProps<{ form: Record<string, unknown> }>();
const { t } = useI18n();
const { lists } = storeToRefs(useMainStore());

const ic = computed(() => props.form.inactive_cleanup as CleanupSettings | undefined);
const managed = computed(() => ic.value?.mode !== 'custom');
const status = ref<CleanupStatus | null>(null);
const count = ref<number | null>(null);
const previewing = ref(false);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const allLists = computed(() => ((lists.value as any)?.results || []) as { id: number; name: string }[]);

const modes = computed(() => [
  { value: 'managed', label: t('settings.inx.listnun') },
  { value: 'custom', label: t('settings.inx.you') },
]);

// How many match today, and how the last run went.
const nextHelp = computed(() => {
  const parts: string[] = [];
  if (count.value !== null) parts.push(t('settings.inx.match', { n: count.value.toLocaleString() }, count.value));
  if (status.value?.last_run) parts.push(lastRunLabel(status.value.last_run));
  return parts.join(' ');
});
const dayOptions = computed<Option<number>[]>(() => DAY_CHOICES.map((d) => ({ label: daysLabel(d), value: d })));
const activityOptions = computed<Option<string>[]>(() => [
  { label: t('settings.inactive.activityEither'), value: 'either' },
  { label: t('settings.inactive.activityOpened'), value: 'opened' },
  { label: t('settings.inactive.activityClicked'), value: 'clicked' },
]);
const actionOptions = computed<Option<string>[]>(() => [
  { label: t('settings.inactive.actionUnsubscribe'), value: 'unsubscribe' },
  { label: t('settings.inactive.actionBlocklist'), value: 'blocklist' },
  { label: t('settings.inactive.actionDelete'), value: 'delete' },
]);
const scheduleOptions = computed<Option<string>[]>(() => [
  { label: t('settings.inactive.daily'), value: '0 2 * * *' },
  { label: t('settings.inactive.weekly'), value: '0 2 * * 0' },
  { label: t('settings.inactive.monthly'), value: '0 2 1 * *' },
]);

function daysLabel(d?: number): string {
  return d ? t('settings.inactive.nDays', { n: d }) : '';
}

function optionLabel<T>(opts: Option<T>[], v?: T): string {
  return opts.find((o) => o.value === v)?.label || String(v ?? '');
}

function listsLabel(ids?: number[] | null): string {
  if (!ids?.length) return t('settings.inactive.allLists');
  return ids.map((id) => allLists.value.find((l) => l.id === id)?.name || `#${id}`).join(', ');
}

// Managed schedules come from config, so show the cron itself when it
// isn't one of the presets.
function scheduleLabel(cron?: string): string {
  return scheduleOptions.value.find((o) => o.value === cron)?.label || cron || '';
}

function niceDate(s: string): string {
  return dayjs(s).format('ddd D MMM, HH:mm');
}

function lastRunLabel(r: CleanupRun): string {
  if (r.error) return t('settings.inactive.lastRunFailed', { date: niceDate(r.started_at), error: r.error });
  return t('settings.inactive.lastRun', { date: niceDate(r.started_at), n: r.done.toLocaleString() });
}

function rules(): Rules | null {
  if (!ic.value) return null;
  if (managed.value) return status.value?.managed || null;
  const {
    days, activity, action, list_ids: ids, schedule,
  } = ic.value;
  return {
    days, activity, action, list_ids: ids || [], schedule,
  };
}

function loadStatus() {
  return axios.get('/api/settings/inactive-cleanup').then((r) => {
    status.value = r.data.data as CleanupStatus;
    count.value = status.value.matching;
  });
}

// Counts against the rules on screen, so custom edits can be checked
// before saving.
function preview() {
  const r = rules();
  if (!r) return;
  previewing.value = true;
  axios.post('/api/settings/inactive-cleanup/preview', r)
    .then((res) => { count.value = res.data.data as number; })
    .finally(() => { previewing.value = false; });
}

onMounted(() => {
  if (ic.value && !ic.value.list_ids) ic.value.list_ids = [];
  loadStatus();
});
</script>

<style scoped lang="scss">
:deep(.ic-locked) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid var(--ln-border);
  border-radius: 6px;
  background: var(--ln-muted);
  color: var(--ln-text-2);
  font-size: 14px;

  i { font-size: 11px; color: var(--ln-text-3); }
}
</style>
