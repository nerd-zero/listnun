<template>
  <div v-if="ic" class="ic">
    <label class="ln-card ic-enable" for="ic-enabled">
      <PvToggleSwitch v-model="ic.enabled" input-id="ic-enabled" />
      <span>
        <strong>{{ $t('settings.inactive.enable') }}</strong>
        <small class="text-color-secondary">{{ $t('settings.inactive.enableHelp') }}</small>
      </span>
    </label>

    <div class="ic-modes" role="radiogroup" :aria-label="$t('settings.inactive.mode')">
      <label v-for="m in modes" :key="m.value" class="ln-card ic-mode" :class="{ 'is-on': ic.mode === m.value }">
        <PvRadioButton v-model="ic.mode" :value="m.value" name="inactive_cleanup.mode" />
        <span>
          <strong>{{ m.label }}</strong>
          <small>{{ m.help }}</small>
        </span>
      </label>
    </div>

    <section class="ln-card ic-rules">
      <header class="ln-card-head">
        <h3>{{ managed ? $t('settings.inactive.managedRules') : $t('settings.inactive.customRules') }}</h3>
        <span class="ic-chip" :class="managed ? 'is-locked' : 'is-edit'">
          <i :class="managed ? 'pi pi-lock' : 'pi pi-pencil'" aria-hidden="true" />
          {{ managed ? $t('settings.inactive.setByListnun') : $t('settings.inactive.editable') }}
        </span>
      </header>

      <div class="ic-row">
        <div>
          <strong>{{ $t('settings.inactive.days') }}</strong>
          <small>{{ $t('settings.inactive.daysHelp') }}</small>
        </div>
        <locked-value v-if="managed" class="ln-enter" :text="daysLabel(status?.managed.days)" />
        <PvSelect v-else class="ln-enter" v-model="ic.days" :options="dayOptions" option-label="label" option-value="value"
          :aria-label="$t('settings.inactive.days')" />
      </div>

      <div class="ic-row">
        <div>
          <strong>{{ $t('settings.inactive.activity') }}</strong>
          <small>{{ $t('settings.inactive.activityHelp') }}</small>
        </div>
        <locked-value v-if="managed" class="ln-enter" :text="optionLabel(activityOptions, status?.managed.activity)" />
        <PvSelect v-else class="ln-enter" v-model="ic.activity" :options="activityOptions" option-label="label" option-value="value"
          :aria-label="$t('settings.inactive.activity')" />
      </div>

      <div class="ic-row">
        <div>
          <strong>{{ $t('settings.inactive.action') }}</strong>
          <small>{{ $t('settings.inactive.actionHelp') }}</small>
        </div>
        <locked-value v-if="managed" class="ln-enter" :text="optionLabel(actionOptions, status?.managed.action)" />
        <PvSelect v-else class="ln-enter" v-model="ic.action" :options="actionOptions" option-label="label" option-value="value"
          :aria-label="$t('settings.inactive.action')" />
      </div>

      <div class="ic-row">
        <div><strong>{{ $t('settings.inactive.lists') }}</strong></div>
        <locked-value v-if="managed" class="ln-enter" :text="listsLabel(status?.managed.list_ids)" />
        <PvMultiSelect v-else class="ln-enter" v-model="ic.list_ids" :options="allLists" option-label="name" option-value="id"
          :placeholder="$t('settings.inactive.allLists')" filter display="comma" :max-selected-labels="3"
          :aria-label="$t('settings.inactive.lists')" />
      </div>

      <div class="ic-row">
        <div><strong>{{ $t('settings.inactive.schedule') }}</strong></div>
        <locked-value v-if="managed" class="ln-enter" :text="scheduleLabel(status?.managed.schedule)" />
        <PvSelect v-else class="ln-enter" v-model="ic.schedule" :options="scheduleOptions" option-label="label" option-value="value"
          :aria-label="$t('settings.inactive.schedule')" />
      </div>
    </section>

    <p v-if="status && !status.tracking" class="ic-note is-bad ln-enter" role="alert">
      <i class="pi pi-exclamation-triangle" aria-hidden="true" />
      {{ $t('settings.inactive.noTracking') }}
    </p>
    <p v-else-if="managed" class="ic-note ln-enter">
      <i class="pi pi-info-circle" aria-hidden="true" />
      {{ $t('settings.inactive.managedNote') }}
    </p>
    <p v-else class="ic-note is-warn ln-enter">
      <i class="pi pi-exclamation-triangle" aria-hidden="true" />
      {{ ic.action === 'delete' ? $t('settings.inactive.deleteNote') : $t('settings.inactive.customNote') }}
    </p>

    <section class="ln-card ic-next">
      <div>
        <p>
          <strong>{{ $t('settings.inactive.nextRun') }}</strong>&emsp;
          <strong>{{ status?.next_run ? niceDate(status.next_run) : $t('settings.inactive.notScheduled') }}</strong>
        </p>
        <p class="ln-small">
          <template v-if="count !== null">
            {{ $t(managed ? 'settings.inactive.matchManaged' : 'settings.inactive.matchCustom', { n: count.toLocaleString() }) }}
          </template>
          <template v-if="status?.last_run">
            &emsp;{{ lastRunLabel(status.last_run) }}
          </template>
        </p>
      </div>
      <PvButton :label="$t('settings.inactive.preview')" severity="secondary" outlined :loading="previewing" @click="preview" />
    </section>
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
  { value: 'managed', label: t('settings.inactive.managed'), help: t('settings.inactive.managedHelp') },
  { value: 'custom', label: t('settings.inactive.custom'), help: t('settings.inactive.customHelp') },
]);
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
.ic { display: flex; flex-direction: column; gap: 16px; }
.ic small { display: block; font-size: 12px; line-height: 16px; color: var(--ln-text-3); font-weight: 400; }

.ic-enable {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  cursor: pointer;
  strong { font-weight: 500; }
}

.ic-modes { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
@media (max-width: 640px) { .ic-modes { grid-template-columns: 1fr; } }
.ic-mode {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 18px;
  cursor: pointer;
  strong { font-weight: 600; }
  small { font-size: 13px; line-height: 20px; color: var(--ln-text-2); margin-top: 2px; }
  // A 2px ring drawn as border + inset shadow, so selecting doesn't shift
  // the layout and the change can ease (motion.scss).
  &.is-on { border-color: var(--ln-ink); box-shadow: inset 0 0 0 1px var(--ln-ink); }
}

.ic-rules {
  overflow: hidden;
  h3 { margin: 0; font-size: 15px; font-weight: 600; }
}
.ic-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  i { font-size: 10px; }
  &.is-locked { background: var(--ln-muted); color: var(--ln-text-2); }
  &.is-edit { background: var(--ln-success-bg); color: var(--ln-success-fg); }
}
.ic-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 18px;
  border-top: 1px solid var(--ln-border);
  strong { font-weight: 500; }
  > :last-child { width: 280px; flex: none; }
  .ln-card-head + & { border-top: 0; }

  @media (max-width: 640px) {
    flex-direction: column;
    align-items: stretch;
    > :last-child { width: 100%; }
  }
}
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

.ic-note {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  padding: 12px 16px;
  border-radius: 8px;
  background: var(--ln-muted);
  color: var(--ln-text-2);
  font-size: 13px;
  &.is-warn { background: var(--ln-warn-bg); color: var(--ln-warn-fg); }
  &.is-bad { background: var(--ln-danger-bg); color: var(--ln-danger-fg); }
}

.ic-next {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  p { margin: 0; }
  strong { font-weight: 500; }
  .ln-small { color: var(--ln-text-2); margin-top: 2px; }

  @media (max-width: 640px) { flex-direction: column; align-items: stretch; }
}
</style>
