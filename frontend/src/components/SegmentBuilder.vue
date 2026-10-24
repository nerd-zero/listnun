<template>
  <section class="ln-card seg" :aria-label="$t('subscribers.segments.title')" data-cy="segment-builder">
    <div class="seg-body">
      <div class="seg-head">
        <strong v-if="title" class="seg-title">{{ title }}</strong>
        <span class="seg-match">
          {{ $t('subscribers.segments.matchLead') }}
          <PvSelect
            :model-value="modelValue.match || 'all'"
            :options="matchOptions"
            option-label="label"
            option-value="value"
            size="small"
            class="seg-match-select"
            :aria-label="$t('subscribers.segments.matchLead')"
            @update:model-value="(m) => emitSeg({ ...modelValue, match: m })"
          />
          {{ $t('subscribers.segments.matchTail') }}
        </span>
      </div>

      <div class="seg-chips">
        <span v-if="!modelValue.conditions.length" class="ln-small">{{ $t('subscribers.segments.noConditions') }}</span>
        <span v-for="(c, i) in modelValue.conditions" :key="i" class="seg-chip" data-cy="condition">
          <span class="seg-chip-field">{{ fieldLabel(c) }}</span>
          <span class="seg-chip-op">{{ $t(`subscribers.segments.op.${c.op}`) }}</span>
          <strong v-if="valueLabel(c)">{{ valueLabel(c) }}</strong>
          <button type="button" class="seg-chip-x" :aria-label="$t('subscribers.segments.removeCondition')" @click="remove(i)">
            <i class="pi pi-times" aria-hidden="true" />
          </button>
        </span>
        <button type="button" class="seg-add" data-cy="btn-add-condition" @click="(e) => adder?.toggle(e)">
          <i class="pi pi-plus" aria-hidden="true" /> {{ $t('subscribers.segments.addCondition') }}
        </button>
      </div>
    </div>

    <div class="seg-foot">
      <span class="seg-count">
        <strong data-cy="segment-count">{{ count === null ? '…' : $t('subscribers.segments.peopleMatch', { n: $utils.niceNumber(count) }, count) }}</strong>
        <span v-if="count !== null && total" class="ln-small">{{ $t('subscribers.segments.shareOfEveryone', { pct: share }) }}</span>
      </span>
      <span class="seg-actions">
        <a v-if="canDelete" href="#" class="ln-small seg-delete" @click.prevent="emit('delete')">{{ $t('subscribers.segments.delete') }}</a>
        <a v-if="canSql" href="#" class="ln-small" data-cy="btn-edit-sql" @click.prevent="emit('sql')">{{ $t('subscribers.segments.editAsSql') }}</a>
        <PvButton v-if="canSave" :label="$t('subscribers.segments.save')" size="small" severity="secondary" outlined data-cy="btn-save-segment" @click="emit('save')" />
      </span>
    </div>

    <PvPopover ref="adder">
      <form class="seg-form" @submit.prevent="add">
        <PvSelect v-model="draft.field" :options="fieldOptions" option-label="label" option-value="value" :aria-label="$t('subscribers.segments.fieldLabel')" @change="onField" />
        <PvInputText v-if="draft.field === 'attribute'" v-model="draft.key" :placeholder="$t('subscribers.segments.attributeKey')" :aria-label="$t('subscribers.segments.attributeKey')" required />
        <PvSelect v-model="draft.op" :options="opOptions" option-label="label" option-value="value" :aria-label="$t('subscribers.segments.opLabel')" />
        <template v-if="needsValue">
          <PvSelect
            v-if="draft.field === 'list'"
            v-model="draft.value"
            :options="listOptions"
            option-label="name"
            option-value="id"
            filter
            :aria-label="$t('subscribers.segments.field.list')"
          />
          <PvSelect
            v-else-if="draft.field === 'status'"
            v-model="draft.value"
            :options="statusOptions"
            option-label="label"
            option-value="value"
            :aria-label="$t('subscribers.segments.field.status')"
          />
          <PvInputNumber v-else-if="isDays" v-model="draftDays" :min="1" :max="3650" :suffix="` ${$t('subscribers.segments.daysUnit')}`" :aria-label="$t('subscribers.segments.value')" />
          <PvInputText v-else-if="isDate" v-model="draft.value" type="date" :aria-label="$t('subscribers.segments.value')" required />
          <PvInputText v-else v-model="draft.value" :placeholder="$t('subscribers.segments.value')" :aria-label="$t('subscribers.segments.value')" required />
        </template>
        <PvButton type="submit" :label="$t('subscribers.segments.add')" size="small" data-cy="btn-condition-add" />
      </form>
    </PvPopover>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

export interface SegCondition { field: string; op: string; key?: string; value?: string }
export interface Seg { match: 'all' | 'any'; conditions: SegCondition[] }
interface ListRef { id: number; name: string }

// Which operators each field offers, first one the default.
const FIELD_OPS: Record<string, string[]> = {
  attribute: ['is', 'is_not', 'contains', 'exists'],
  email: ['contains', 'is', 'is_not'],
  name: ['contains', 'is', 'is_not'],
  joined: ['in_last', 'not_in_last', 'before', 'after'],
  list: ['is', 'is_not'],
  status: ['is', 'is_not'],
  unconfirmed: ['is'],
  last_opened: ['in_last', 'not_in_last', 'never'],
  last_clicked: ['in_last', 'not_in_last', 'never'],
};

// Operators that take no value.
const NO_VALUE = new Set(['exists', 'never']);

const props = defineProps<{
  modelValue: Seg;
  title?: string;
  count: number | null;
  total?: number;
  lists: ListRef[];
  canSave?: boolean;
  canSql?: boolean;
  canDelete?: boolean;
}>();
const emit = defineEmits<{ 'update:modelValue': [s: Seg]; save: []; sql: []; delete: [] }>();

const { t } = useI18n();
const adder = ref<{ toggle:(e: Event) => void; hide: () => void } | null>(null);
const draft = reactive<SegCondition>({
  field: 'attribute', op: 'is', key: '', value: '',
});
const draftDays = ref<number>(30);

const matchOptions = computed(() => [
  { value: 'all', label: t('subscribers.segments.match.all') },
  { value: 'any', label: t('subscribers.segments.match.any') },
]);
const fieldOptions = computed(() => Object.keys(FIELD_OPS).map((f) => ({ value: f, label: t(`subscribers.segments.field.${f}`) })));
const opOptions = computed(() => (FIELD_OPS[draft.field] || []).map((o) => ({ value: o, label: t(`subscribers.segments.op.${o}`) })));
const statusOptions = computed(() => ['enabled', 'blocklisted'].map((s) => ({ value: s, label: t(`subscribers.status.${s}`) })));
const listOptions = computed(() => props.lists.map((l) => ({ id: String(l.id), name: l.name })));
const isDays = computed(() => draft.op === 'in_last' || draft.op === 'not_in_last');
const isDate = computed(() => draft.op === 'before' || draft.op === 'after');
const needsValue = computed(() => !NO_VALUE.has(draft.op) && draft.field !== 'unconfirmed');
const share = computed(() => {
  if (!props.total || props.count === null) return '';
  return `${((props.count / props.total) * 100).toFixed(1)}%`;
});

function emitSeg(s: Seg) {
  emit('update:modelValue', s);
}

function fieldLabel(c: SegCondition): string {
  return c.field === 'attribute' ? (c.key || '') : t(`subscribers.segments.field.${c.field}`);
}

function valueLabel(c: SegCondition): string {
  if (NO_VALUE.has(c.op) || c.field === 'unconfirmed') return '';
  if (c.op === 'in_last' || c.op === 'not_in_last') return t('subscribers.segments.days', { n: c.value }, Number(c.value));
  if (c.field === 'list') return props.lists.find((l) => String(l.id) === c.value)?.name || `#${c.value}`;
  if (c.field === 'status') return t(`subscribers.status.${c.value}`);
  return c.value || '';
}

function onField() {
  [draft.op] = FIELD_OPS[draft.field];
  draft.value = '';
  draft.key = '';
}

function remove(i: number) {
  emitSeg({ ...props.modelValue, conditions: props.modelValue.conditions.filter((_, j) => j !== i) });
}

function add() {
  const c: SegCondition = { field: draft.field, op: draft.op };
  if (draft.field === 'attribute') c.key = (draft.key || '').trim();
  if (needsValue.value) c.value = isDays.value ? String(draftDays.value || 1) : String(draft.value || '').trim();
  if ((draft.field === 'attribute' && !c.key) || (needsValue.value && !c.value)) return;
  emitSeg({ ...props.modelValue, conditions: [...props.modelValue.conditions, c] });
  adder.value?.hide();
  draft.value = '';
  draft.key = '';
}
</script>

<style scoped lang="scss">
.seg { display: flex; flex-direction: column; }
.seg-body { padding: 16px 18px; display: flex; flex-direction: column; gap: 12px; }
.seg-head { display: flex; flex-direction: column; gap: 6px; }
.seg-title { font-size: 15px; font-weight: 600; }
.seg-match { display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--ln-text-2); }
.seg-match-select { min-width: 72px; }
.seg-chips { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.seg-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 6px 0 12px;
  border-radius: var(--ln-radius);
  border: 1px solid var(--ln-border-strong);
  background: var(--ln-card);
  font-size: 13px;

  strong { font-weight: 600; }
}
.seg-chip-field { color: var(--ln-text-2); }
.seg-chip-op { color: var(--ln-text-3); }
.seg-chip-x {
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--ln-text-3);
  cursor: pointer;

  i { font-size: 11px; }
  &:hover { background: var(--ln-hover); color: var(--ln-text); }
}
.seg-add {
  height: 34px;
  padding: 0 12px;
  border-radius: var(--ln-radius);
  border: 1px dashed var(--ln-border-strong);
  background: transparent;
  color: var(--ln-text-2);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  i { font-size: 11px; }
  &:hover { color: var(--ln-text); border-color: var(--ln-text-3); }
}
.seg-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 10px 18px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
  border-radius: 0 0 var(--ln-radius-card) var(--ln-radius-card);
}
.seg-count { display: flex; align-items: baseline; gap: 10px; strong { font-weight: 600; } }
.seg-actions { display: flex; align-items: center; gap: 10px; a { color: var(--ln-text-2); } }
.seg-delete { color: var(--ln-danger-fg) !important; }
.seg-form { display: flex; flex-direction: column; gap: 10px; width: 260px; }
</style>
