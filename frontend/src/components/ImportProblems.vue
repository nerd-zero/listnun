<template>
  <!-- ImportProblems: the rows an import leaves out, a few at first, with
       a download so they can be fixed in the file and imported again. -->
  <div class="imp-problems">
    <header class="imp-problems-head">
      <span>
        <strong>{{ title }}</strong>
        <span v-if="help" class="ln-small">{{ help }}</span>
      </span>
      <PvButton :label="$t('import.downloadRows')" severity="secondary" outlined size="small" @click="download" />
    </header>
    <div class="imp-problems-table" role="table" :aria-label="title">
      <div class="imp-problems-row imp-problems-th" role="row">
        <span role="columnheader">{{ $t('import.colRow') }}</span>
        <span role="columnheader">{{ $t('import.colValue') }}</span>
        <span role="columnheader">{{ $t('import.colProblem') }}</span>
      </div>
      <div v-for="p in shown" :key="p.row" class="imp-problems-row" role="row">
        <span role="cell" class="imp-problems-num">{{ n(p.row) }}</span>
        <span role="cell" class="ln-mono imp-problems-value">{{ p.value || $t('import.empty') }}</span>
        <span role="cell" class="imp-problems-why">{{ reason(p.reason) }}</span>
      </div>
    </div>
    <p v-if="problems.length > FIRST_SHOWN || total > problems.length" class="imp-problems-more ln-small">
      <a v-if="problems.length > FIRST_SHOWN && !all" href="#" @click.prevent="all = true">{{ $t('import.showAll', { n: n(problems.length) }) }}</a>
      <span v-if="all && total > problems.length">{{ $t('import.listedFirst', { n: n(problems.length) }) }}</span>
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

interface ImportProblem { row: number; value: string; reason: string }

// How many rows show before "Show all".
const FIRST_SHOWN = 3;

const props = withDefaults(defineProps<{
  problems: ImportProblem[];
  // Every row left out; problems may list only the first of them.
  total: number;
  title: string;
  help?: string;
  fileName?: string;
}>(), { help: '', fileName: '' });

const { t, n } = useI18n();
const all = ref(false);

const shown = computed(() => (all.value ? props.problems : props.problems.slice(0, FIRST_SHOWN)));

function reason(r: string) {
  const key = `import.reason.${r}`;
  const s = t(key);
  return s === key ? r : s;
}

// A cell as CSV: quoted when it holds a comma, quote or line break.
function cell(v: string | number) {
  const s = String(v);
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

function download() {
  const lines = [[t('import.colRow'), t('import.colValue'), t('import.colProblem')].map(cell).join(',')]
    .concat(props.problems.map((p) => [p.row, p.value, reason(p.reason)].map(cell).join(',')));
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([`${lines.join('\n')}\n`], { type: 'text/csv' }));
  a.download = `${(props.fileName || 'import').replace(/\.(csv|zip)$/i, '')}-not-imported.csv`;
  a.click();
  URL.revokeObjectURL(a.href);
}
</script>

<style scoped lang="scss">
.imp-problems {
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  overflow: hidden;

  .ln-card > & { border: 0; border-radius: 0; }
}
.imp-problems-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;

  > span { display: flex; flex-direction: column; gap: 2px; }
  .ln-small { color: var(--ln-text-2); }
}
.imp-problems-row {
  display: grid;
  grid-template-columns: 80px minmax(0, 1.4fr) minmax(0, 1fr);
  gap: 12px;
  padding: 10px 16px;
  border-top: 1px solid var(--ln-border);
  font-size: 14px;
}
.imp-problems-th { background: var(--ln-muted); font-size: 13px; font-weight: 600; }
.imp-problems-num { color: var(--ln-text-3); font-variant-numeric: tabular-nums; }
.imp-problems-value { font-size: 13px; overflow-wrap: anywhere; }
.imp-problems-why { color: var(--ln-danger-fg); }
.imp-problems-more {
  margin: 0;
  padding: 12px 16px;
  border-top: 1px solid var(--ln-border);

  a { color: var(--ln-text); text-decoration: underline; font-weight: 500; }
}
</style>
