<template>
  <div class="set-cards">
    <set-card :title="$t('settings.speed.title')">
      <set-row :label="$t('settings.nav.performance')" :help="speedHelp">
        <PvSelectButton
          :model-value="preset"
          :options="presetOptions"
          option-label="label"
          option-value="value"
          :allow-empty="false"
          :aria-label="$t('settings.nav.performance')"
          @update:model-value="pickPreset"
        />
      </set-row>
      <template v-if="preset === 'custom'">
        <set-row :label="$t('settings.speed.workers')" :help="$t('settings.speed.workersHelp')">
          <PvInputNumber v-model="data['app.concurrency']" name="app.concurrency" :min="1" :max="10000" :aria-label="$t('settings.speed.workers')" />
        </set-row>
        <set-row :label="$t('settings.speed.rate')" :help="$t('settings.speed.rateHelp')">
          <PvInputNumber v-model="data['app.message_rate']" name="app.message_rate" :min="1" :max="100000" :aria-label="$t('settings.speed.rate')" />
        </set-row>
      </template>
    </set-card>

    <set-card :title="$t('settings.speed.limits')" :help="$t('settings.speed.limitsHelp')">
      <set-row :label="$t('settings.speed.respect')" toggle>
        <PvToggleSwitch v-model="data['app.message_sliding_window']" name="app.message_sliding_window" :aria-label="$t('settings.speed.respect')" />
      </set-row>
      <template v-if="data['app.message_sliding_window']">
        <set-row :label="$t('settings.speed.per')">
          <PvSelect
            v-model="data['app.message_sliding_window_duration']"
            name="sliding_window_duration"
            :options="periodOptions"
            option-label="label"
            option-value="value"
            :aria-label="$t('settings.speed.per')"
          />
        </set-row>
        <set-row :label="$t('settings.speed.upTo')" :help="$t('settings.speed.upToHelp')">
          <PvInputNumber v-model="data['app.message_sliding_window_rate']" name="sliding_window_rate" :min="1" :max="10000000" :aria-label="$t('settings.speed.upTo')" />
        </set-row>
      </template>
    </set-card>

    <set-card :title="$t('settings.ws.advanced')" :help="$t('settings.speed.advancedHelp')" collapsible :summary="$t('settings.speed.advancedSummary')">
      <set-row :label="$t('settings.speed.batch')" :help="$t('settings.speed.batchHelp')">
        <PvInputNumber v-model="data['app.batch_size']" name="app.batch_size" :min="1" :max="100000" :aria-label="$t('settings.speed.batch')" />
      </set-row>
      <set-row :label="$t('settings.speed.maxErrors')" :help="$t('settings.speed.maxErrorsHelp')">
        <PvInputNumber v-model="data['app.max_send_errors']" name="app.max_send_errors" :min="0" :max="100000" :aria-label="$t('settings.speed.maxErrors')" />
      </set-row>
      <set-row :label="$t('settings.speed.cache')" :help="$t('settings.speed.cacheHelp')" toggle>
        <PvToggleSwitch v-model="data['app.cache_slow_queries']" name="app.cache_slow_queries" :aria-label="$t('settings.speed.cache')" />
      </set-row>
      <set-row v-if="data['app.cache_slow_queries']" :label="$t('settings.speed.cacheWhen')" :help="$t('settings.speed.cacheWhenHelp')" label-for="set-cache-cron">
        <PvInputText id="set-cache-cron" v-model="data['app.cache_slow_queries_interval']" placeholder="0 3 * * *" class="ln-mono" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

// The settings object is the server's flat, dotted-key map.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// Speed presets: workers x emails a second per worker. Anything else is
// Custom, so existing values are kept as they are.
const PRESETS: Record<string, { workers: number; rate: number }> = {
  gentle: { workers: 2, rate: 10 },
  standard: { workers: 5, rate: 20 },
  fast: { workers: 10, rate: 50 },
};

const props = defineProps<{ form?: Loose }>();
const data = props.form as Loose;
const { t } = useI18n();

// Custom chosen while the numbers still match a preset.
const customChosen = ref(false);

const preset = computed(() => {
  if (customChosen.value) return 'custom';
  const hit = Object.entries(PRESETS).find(([, p]) => p.workers === data['app.concurrency'] && p.rate === data['app.message_rate']);
  return hit ? hit[0] : 'custom';
});
const presetOptions = computed(() => ['gentle', 'standard', 'fast', 'custom'].map((v) => ({ value: v, label: t(`settings.speed.${v}`) })));

// How the chosen speed reads in emails, e.g. "About 100 emails a second".
const speedHelp = computed(() => {
  const n = (Number(data['app.concurrency']) || 0) * (Number(data['app.message_rate']) || 0);
  return t('settings.speed.perSecond', { n: n.toLocaleString() }, n);
});

// The limit's period: the common ones, plus whatever is set if it's not
// one of them.
const periodOptions = computed(() => {
  const opts = [
    { value: '1m', label: t('settings.speed.minute') },
    { value: '1h', label: t('settings.speed.hour') },
    { value: '24h', label: t('settings.speed.day') },
  ];
  const cur = data['app.message_sliding_window_duration'];
  if (cur && !opts.some((o) => o.value === cur)) opts.push({ value: cur, label: cur });
  return opts;
});

function pickPreset(v: string) {
  const p = PRESETS[v];
  // Custom keeps the current numbers and shows them for editing.
  customChosen.value = !p;
  if (!p) return;
  data['app.concurrency'] = p.workers;
  data['app.message_rate'] = p.rate;
}
</script>
