<template>
  <!-- FormEditor: what a form asks for and says. Each change goes up as a
       patch the page applies at once, so quick changes in a row all land. -->
  <section class="ln-card fe">
    <header class="fe-head">
      <PvInputText
        :model-value="draft.name"
        class="fe-name"
        :aria-label="$t('forms.name')"
        :maxlength="200"
        data-cy="form-name"
        @update:model-value="(v) => set('name', String(v ?? ''))"
      />
    </header>

    <div class="fe-sec">
      <h3 id="fe-lists">{{ $t('forms.listsTitle') }}</h3>
      <div role="group" aria-labelledby="fe-lists" class="fe-lists">
        <label v-for="l in shownLists" :key="l.id" class="fe-list" :for="`fe-list-${l.id}`" data-cy="form-list">
          <PvCheckbox
            :model-value="draft.listIds.includes(l.id)"
            binary
            :input-id="`fe-list-${l.id}`"
            @update:model-value="(on: boolean) => toggleList(l.id, on)"
          />
          <span>
            <span>{{ l.name }}</span>
            <span class="ln-small">{{ listNote(l) }}</span>
          </span>
        </label>
      </div>
      <!-- Many lists: the chosen ones above, the rest behind a search. -->
      <PvAutoComplete
        v-if="manyLists"
        v-model="listQuery"
        :suggestions="listSuggestions"
        option-label="name"
        dropdown
        force-selection
        :placeholder="$t('forms.addList')"
        :aria-label="$t('forms.addList')"
        class="fe-add-list"
        @complete="searchLists"
        @option-select="(e: { value: PublicList }) => addList(e.value)"
      />
      <p class="ln-small fe-help">{{ $t('forms.onlyPublic') }}</p>
    </div>

    <div class="fe-sec">
      <h3>{{ $t('forms.fields') }}</h3>
      <ol class="fe-fields">
        <li
          v-for="(f, i) in draft.fields"
          :key="f.key"
          class="fe-field"
          :class="{ 'is-dragging': dragIndex === i }"
          :draggable="i > 0"
          @dragstart="dragIndex = i"
          @dragover.prevent="onDragOver(i)"
          @dragend="dragIndex = -1"
        >
          <button
            v-if="i > 0"
            type="button"
            class="fe-handle"
            :aria-label="$t('forms.moveField', { name: f.label })"
            :title="$t('forms.moveHelp')"
            @keydown.up.prevent="move(i, -1)"
            @keydown.down.prevent="move(i, 1)"
          >
            <i class="pi pi-bars" aria-hidden="true" />
          </button>
          <span v-else class="fe-handle is-fixed" aria-hidden="true"><i class="pi pi-lock" /></span>
          <span class="fe-field-text">
            <input
              class="fe-field-label"
              :value="f.label"
              :aria-label="$t('forms.fieldLabel')"
              maxlength="200"
              @input="(e) => setField(i, { label: (e.target as HTMLInputElement).value })"
            >
            <span class="ln-mono ln-small">{{ f.key }}</span>
          </span>
          <button
            type="button"
            class="fe-req"
            :disabled="i === 0"
            :aria-pressed="f.required"
            @click="setField(i, { required: !f.required })"
          >
            {{ f.required ? $t('forms.required') : $t('forms.optionalField') }}
          </button>
          <button v-if="i > 0" type="button" class="fe-remove" :aria-label="$t('forms.removeField', { name: f.label })" @click="removeField(i)">
            <i class="pi pi-times" aria-hidden="true" />
          </button>
        </li>
      </ol>

      <form v-if="adding" class="fe-add-form" @submit.prevent="addCustom">
        <PvInputText
          ref="newLabelEl"
          v-model="newLabel"
          :placeholder="$t('forms.newFieldPlaceholder')"
          :aria-label="$t('forms.fieldLabel')"
          :maxlength="200"
        />
        <span class="ln-small fe-add-key">{{ newKey ? `attribs.${newKey}` : $t('forms.newFieldHelp') }}</span>
        <span class="fe-add-actions">
          <PvButton type="submit" size="small" :label="$t('forms.addField')" :disabled="!newKey || keyTaken" />
          <PvButton size="small" severity="secondary" text :label="$t('globals.buttons.cancel')" @click="adding = false" />
        </span>
        <span v-if="keyTaken" class="ln-small fe-add-taken">{{ $t('forms.fieldTaken') }}</span>
      </form>
      <span v-else-if="draft.fields.length < MAX_FIELDS" class="fe-add">
        <button v-if="!hasName" type="button" class="fe-link" @click="addName"><i class="pi pi-plus" aria-hidden="true" /> {{ $t('forms.addName') }}</button>
        <button type="button" class="fe-link" @click="startAdding"><i class="pi pi-plus" aria-hidden="true" /> {{ $t('forms.addCustom') }}</button>
      </span>
    </div>

    <div class="fe-sec">
      <label class="fe-label" for="fe-heading">{{ $t('forms.heading') }}</label>
      <PvInputText id="fe-heading" :model-value="draft.heading" :maxlength="200" :placeholder="draft.name" @update:model-value="(v) => set('heading', String(v ?? ''))" />
      <label class="fe-label" for="fe-intro">{{ $t('forms.introLabel') }} <span class="fe-opt">{{ $t('forms.optional') }}</span></label>
      <PvTextarea id="fe-intro" :model-value="draft.intro" rows="2" auto-resize :maxlength="1000" @update:model-value="(v) => set('intro', String(v ?? ''))" />
      <label class="fe-label" for="fe-button">{{ $t('forms.buttonLabel') }}</label>
      <PvInputText id="fe-button" :model-value="draft.buttonLabel" :maxlength="100" :placeholder="$t('public.sub')" @update:model-value="(v) => set('buttonLabel', String(v ?? ''))" />
      <label class="fe-label" for="fe-after">{{ $t('forms.after') }}</label>
      <PvTextarea
        id="fe-after"
        :model-value="draft.successMessage"
        rows="2"
        auto-resize
        :maxlength="1000"
        :placeholder="doubleOptin ? $t('public.subOptinPending') : $t('public.subConfirmed')"
        @update:model-value="(v) => set('successMessage', String(v ?? ''))"
      />
      <p v-if="doubleOptin" class="fe-note">
        <i class="pi pi-envelope" aria-hidden="true" />
        {{ $t('forms.optinNote', { n: doubleOptinCount }, doubleOptinCount) }}
      </p>
    </div>

    <div class="fe-sec fe-toggle">
      <span>
        <label for="fe-captcha">{{ $t('forms.bots') }}</label>
        <span class="ln-small">
          <template v-if="captchaOn">{{ $t('forms.botsHelp') }}</template>
          <template v-else>
            {{ $t('forms.botsOff') }}
            <router-link v-if="$can('settings:get')" :to="{ name: 'settings', params: { section: 'security' } }">{{ $t('settings.nav.security') }}</router-link>
          </template>
        </span>
      </span>
      <PvToggleSwitch
        input-id="fe-captcha"
        :model-value="draft.captcha && captchaOn"
        :disabled="!captchaOn"
        @update:model-value="(v: boolean) => set('captcha', v)"
      />
    </div>

    <footer class="fe-foot">
      <button type="button" class="fe-link fe-delete" @click="emit('delete')">{{ $t('forms.delete') }}</button>
      <slot name="actions" />
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  type FormDraft, type FormField, type PublicList, ATTRIB_PREFIX, attribKey,
} from './types';

// The server's MaxFormFields.
const MAX_FIELDS = 10;
// Up to this many public lists show as checkboxes; more get a search.
const LISTS_INLINE = 8;

const props = defineProps<{ draft: FormDraft; lists: PublicList[]; captchaOn: boolean }>();
const emit = defineEmits<{ patch: [Partial<FormDraft>]; delete: [] }>();
const draft = computed(() => props.draft);
const { t } = useI18n();

const dragIndex = ref(-1);
const adding = ref(false);
const newLabel = ref('');
const newLabelEl = ref<{ $el: HTMLInputElement } | null>(null);

const hasName = computed(() => draft.value.fields.some((f) => f.key === 'name'));
const newKey = computed(() => attribKey(newLabel.value));
const keyTaken = computed(() => draft.value.fields.some((f) => f.key === `${ATTRIB_PREFIX}${newKey.value}`));
const chosen = computed(() => props.lists.filter((l) => draft.value.listIds.includes(l.id)));
const manyLists = computed(() => props.lists.length > LISTS_INLINE);
const shownLists = computed(() => (manyLists.value ? chosen.value : props.lists));
const listQuery = ref<string | PublicList>('');
const listSuggestions = ref<PublicList[]>([]);
const doubleOptinCount = computed(() => chosen.value.filter((l) => l.optin === 'double').length);
const doubleOptin = computed(() => doubleOptinCount.value > 0);

function set<K extends keyof FormDraft>(key: K, v: FormDraft[K]) {
  emit('patch', { [key]: v });
}

function setFields(fields: FormField[]) {
  set('fields', fields);
}

function setField(i: number, patch: Partial<FormField>) {
  setFields(draft.value.fields.map((f, j) => (j === i ? { ...f, ...patch } : f)));
}

function toggleList(id: number, on: boolean) {
  const ids = draft.value.listIds.filter((x) => x !== id);
  if (on) ids.push(id);
  set('listIds', ids);
}

function searchLists(e: { query: string }) {
  const q = (e.query || '').toLowerCase();
  listSuggestions.value = props.lists.filter((l) => !draft.value.listIds.includes(l.id) && l.name.toLowerCase().includes(q));
}

function addList(l: PublicList) {
  toggleList(l.id, true);
  // The field sets itself to the picked list after this; clear it after.
  nextTick(() => { listQuery.value = ''; });
}

function listNote(l: PublicList) {
  return l.optin === 'double' ? t('forms.optinDouble') : t('forms.optinSingle');
}

// Moves field i by one place; email stays first.
function move(i: number, by: number) {
  const to = i + by;
  if (i < 1 || to < 1 || to >= draft.value.fields.length) return;
  const fields = [...draft.value.fields];
  [fields[i], fields[to]] = [fields[to], fields[i]];
  setFields(fields);
  nextTick(() => (document.querySelectorAll('.fe-handle')[to - 1] as HTMLElement | undefined)?.focus());
}

function onDragOver(i: number) {
  const from = dragIndex.value;
  if (from < 1 || i < 1 || from === i) return;
  const fields = [...draft.value.fields];
  const [f] = fields.splice(from, 1);
  fields.splice(i, 0, f);
  dragIndex.value = i;
  setFields(fields);
}

function removeField(i: number) {
  setFields(draft.value.fields.filter((_, j) => j !== i));
}

function addName() {
  const fields = [...draft.value.fields];
  fields.splice(1, 0, { key: 'name', label: t('forms.nameLabel'), required: false });
  setFields(fields);
}

function startAdding() {
  newLabel.value = '';
  adding.value = true;
  nextTick(() => newLabelEl.value?.$el?.focus());
}

function addCustom() {
  if (!newKey.value || keyTaken.value) return;
  setFields([...draft.value.fields, { key: `${ATTRIB_PREFIX}${newKey.value}`, label: newLabel.value.trim(), required: false }]);
  adding.value = false;
}
</script>

<style scoped lang="scss">
.fe { overflow: hidden; }
.fe-head { padding: 12px 16px; border-bottom: 1px solid var(--ln-border); }
.fe-name {
  width: 100%;
  border-color: transparent;
  background: transparent;
  font-size: 15px;
  font-weight: 600;
  padding-left: 4px;

  &:hover, &:focus { border-color: var(--ln-border); }
}
.fe-sec {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid var(--ln-border);

  h3 { margin: 0 0 2px; font-size: 14px; font-weight: 600; }
}
.fe-lists { display: flex; flex-direction: column; gap: 10px; }
.fe-list {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  cursor: pointer;

  > span { display: flex; flex-direction: column; gap: 1px; }
  .ln-small { color: var(--ln-text-3); }
}
.fe-help { margin: 2px 0 0; color: var(--ln-text-3); }
.fe-add-list { width: 100%; }

.fe-fields { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }
.fe-field {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px 8px 6px;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: var(--ln-card);

  &.is-dragging { opacity: .5; }
}
.fe-handle {
  display: inline-grid;
  place-items: center;
  flex: none;
  width: 24px;
  height: 28px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--ln-text-3);
  cursor: grab;

  .pi { font-size: 11px; }
  &.is-fixed { cursor: default; color: var(--ln-text-4, var(--ln-text-3)); }
}
.fe-field-text { display: flex; flex: 1; flex-direction: column; min-width: 0; }
.fe-field-label {
  min-width: 0;
  padding: 1px 4px;
  margin-left: -4px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  font-weight: 500;

  &:hover, &:focus { border-color: var(--ln-border); outline: none; }
}
.fe-field-text .ln-small { color: var(--ln-text-3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.fe-req {
  flex: none;
  padding: 2px 6px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--ln-text-2);
  font: inherit;
  font-size: 13px;
  cursor: pointer;

  &:hover:not(:disabled) { background: var(--ln-hover); }
  &:disabled { cursor: default; }
}
.fe-remove {
  flex: none;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--ln-text-3);
  cursor: pointer;

  .pi { font-size: 10px; }
  &:hover { background: var(--ln-hover); color: var(--ln-text); }
}
.fe-add { display: flex; flex-wrap: wrap; gap: 4px 16px; }
.fe-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  font-weight: 500;
  cursor: pointer;

  .pi { font-size: 11px; }
  &:hover { text-decoration: underline; }
}
.fe-add-form {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border: 1px dashed var(--ln-border);
  border-radius: 8px;
}
.fe-add-key { color: var(--ln-text-3); }
.fe-add-actions { display: flex; gap: 6px; }
.fe-add-taken { color: var(--ln-warn-fg); }

.fe-label { margin-top: 4px; font-size: 14px; font-weight: 600; }
.fe-label:first-child { margin-top: 0; }
.fe-opt { font-weight: 400; color: var(--ln-text-3); }
.fe-note {
  display: flex;
  gap: 8px;
  margin: 4px 0 0;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--ln-muted);
  color: var(--ln-text-2);
  font-size: 13px;

  .pi { margin-top: 3px; font-size: 12px; }
}

.fe-toggle {
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  gap: 12px;

  > span { display: flex; flex-direction: column; gap: 2px; }
  label { font-weight: 600; font-size: 14px; }
  .ln-small { color: var(--ln-text-3); }
  a { color: var(--ln-text); text-decoration: underline; }
}
.fe-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--ln-muted);
}
.fe-delete { color: var(--ln-danger-fg); font-weight: 400; white-space: nowrap; }
</style>
