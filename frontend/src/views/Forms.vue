<template>
  <div class="ln-page forms">
    <page-header :title="$t('forms.title')" :subtitle="$t('forms.intro')">
      <PvButton
        v-if="$can('lists:manage_all') && publicLists.length"
        icon="pi pi-plus"
        :label="$t('forms.new')"
        :loading="creating"
        data-cy="btn-new-form"
        @click="newForm"
      />
    </page-header>

    <div v-if="loading" class="forms-skel">
      <PvSkeleton height="84px" />
      <PvSkeleton height="520px" />
    </div>

    <empty-state
      v-else-if="!publicLists.length"
      icon="pi-list"
      :title="$t('forms.noPublicListsTitle')"
      :text="$t('forms.noPublicLists')"
      :action-label="$can('lists:manage_all') ? $t('forms.goToLists') : ''"
      @action="$router.push({ name: 'lists' })"
    />

    <empty-state
      v-else-if="!forms.length"
      icon="pi-file-edit"
      :title="$t('forms.noneTitle')"
      :text="$t('forms.noneText')"
      :action-label="$can('lists:manage_all') ? $t('forms.new') : ''"
      @action="newForm"
    />

    <template v-else>
      <!-- Every form; the chosen one is outlined. -->
      <div class="forms-cards" role="list">
        <button
          v-for="f in forms"
          :key="f.id"
          type="button"
          role="listitem"
          class="ln-card forms-card"
          :class="{ 'is-on': f.id === selectedId }"
          :aria-current="f.id === selectedId ? 'true' : undefined"
          data-cy="form-card"
          @click="select(f.id)"
        >
          <span class="forms-card-top">
            <strong>{{ f.name }}</strong>
            <span class="forms-chip" :class="`is-${f.status}`">{{ f.status === 'live' ? $t('forms.live') : $t('forms.draft') }}</span>
          </span>
          <span v-if="f.status === 'live'" class="forms-card-num"><strong>{{ n(f.signups) }}</strong> {{ $t('forms.signups30', f.signups) }}</span>
          <span v-else class="forms-card-num"><strong>{{ n(f.signups) }}</strong> {{ $t('forms.notPublished') }}</span>
        </button>
      </div>

      <div v-if="current && draft" class="forms-work">
        <form-editor
          :draft="draft"
          :lists="publicLists"
          :captcha-on="captchaOn"
          class="forms-editor"
          @patch="(p) => draft && (draft = { ...draft, ...p })"
          @delete="confirmDelete = true"
        >
          <template #actions>
            <PvButton :label="$t('forms.save')" :disabled="!dirty" :loading="saving" data-cy="btn-save-form" @click="save()" />
          </template>
        </form-editor>

        <form-preview :draft="draft" :lists="chosenLists" :site-name="siteName" :captcha="draft.captcha && captchaOn" class="forms-preview" />

        <div class="forms-side">
          <form-share :form="current" :busy="saving" @publish="save('live')" @unpublish="save('draft')" />

          <section v-if="current.status === 'live' && stats" class="ln-card forms-stats">
            <h2>{{ $t('forms.last30') }}</h2>
            <dl>
              <template v-for="s in statRows" :key="s.label">
                <dt>{{ s.label }}</dt>
                <dd>{{ n(s.value) }}</dd>
              </template>
            </dl>
          </section>
        </div>
      </div>
    </template>

    <confirm-action
      v-model:visible="confirmDelete"
      :title="$t('forms.deleteTitle', { name: current?.name })"
      :text="$t('forms.deleteText')"
      :confirm-label="$t('forms.delete')"
      danger
      :busy="saving"
      @confirm="removeForm"
    />
    <confirm-action
      v-model:visible="confirmDiscard"
      :title="$t('forms.discardTitle')"
      :text="$t('forms.discardText')"
      :confirm-label="$t('forms.discard')"
      @confirm="discardAndGo"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import FormEditor from '../components/forms/FormEditor.vue';
import FormPreview from '../components/forms/FormPreview.vue';
import FormShare from '../components/forms/FormShare.vue';
import {
  type FormDraft, type PublicList, type SignupForm, toDraft, toReq,
} from '../components/forms/types';
import { getForms } from '../api/generated/endpoints/forms/forms';

// Lists and server config are the API's loosely typed objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface FormStats { views: number; signups: number; confirmed: number; blocked: number }

const api = getForms();
const { $utils } = useGlobal();
const { t, n } = useI18n();
const route = useRoute();
const router = useRouter();
const store = useMainStore();
const { lists, serverConfig } = storeToRefs(store);

const loading = ref(true);
const creating = ref(false);
const saving = ref(false);
const confirmDelete = ref(false);
const confirmDiscard = ref(false);
const forms = ref<SignupForm[]>([]);
const selectedId = ref(0);
const draft = ref<FormDraft | null>(null);
const stats = ref<FormStats | null>(null);
let pendingId = 0;

const sc = computed<Loose>(() => serverConfig.value as Loose);
const siteName = computed(() => sc.value.organization_name || sc.value.site_name || 'listnun');
const captchaOn = computed(() => !!sc.value.public_subscription?.captcha_enabled);
const publicLists = computed<PublicList[]>(() => ((lists.value as Loose).results || [])
  .filter((l: Loose) => l.type === 'public' && l.status !== 'archived'));

const current = computed(() => forms.value.find((f) => f.id === selectedId.value) || null);
const chosenLists = computed(() => publicLists.value.filter((l) => draft.value?.listIds.includes(l.id)));
const dirty = computed(() => !!current.value && !!draft.value
  && JSON.stringify(draft.value) !== JSON.stringify(toDraft(current.value)));

const statRows = computed(() => {
  const s = stats.value;
  if (!s || !current.value) return [];
  const out = [
    { label: t('forms.views'), value: s.views },
    { label: t('forms.signupsLabel'), value: s.signups },
  ];
  if (chosenLists.value.some((l) => l.optin === 'double')) out.push({ label: t('forms.confirmed'), value: s.confirmed });
  if (current.value.captcha && captchaOn.value) out.push({ label: t('forms.blocked'), value: s.blocked });
  return out;
});

function loadStats() {
  stats.value = null;
  const f = current.value;
  if (!f || f.status !== 'live') return;
  api.getFormStats(f.id).then((data) => {
    if (selectedId.value === f.id) stats.value = data as unknown as FormStats;
  }, () => {});
}

function show(id: number) {
  selectedId.value = id;
  draft.value = current.value ? toDraft(current.value) : null;
  if (Number(route.query.form) !== id) router.replace({ query: { ...route.query, form: String(id) } });
  loadStats();
}

function select(id: number) {
  if (id === selectedId.value) return;
  if (dirty.value) {
    pendingId = id;
    confirmDiscard.value = true;
    return;
  }
  show(id);
}

function discardAndGo() {
  confirmDiscard.value = false;
  show(pendingId);
}

function load() {
  return api.getForms().then((data) => {
    forms.value = data as unknown as SignupForm[];
  }).finally(() => { loading.value = false; });
}

function replace(f: SignupForm) {
  // A saved form keeps its 30-day signup count, which saving doesn't return.
  const old = forms.value.find((x) => x.id === f.id);
  const merged = { ...f, signups: old?.signups ?? 0 };
  forms.value = old ? forms.value.map((x) => (x.id === f.id ? merged : x)) : [...forms.value, merged];
}

function newForm() {
  creating.value = true;
  api.createForm(toReq({
    name: t('forms.untitled'),
    status: 'draft',
    heading: '',
    intro: '',
    listIds: publicLists.value.length === 1 ? [publicLists.value[0].id] : [],
    fields: [
      { key: 'email', label: t('forms.emailLabel'), required: true },
      { key: 'name', label: t('forms.nameLabel'), required: false },
    ],
    buttonLabel: '',
    successMessage: '',
    captcha: captchaOn.value,
  })).then((data) => {
    const f = data as unknown as SignupForm;
    replace(f);
    show(f.id);
    window.setTimeout(() => (document.querySelector('[data-cy=form-name]') as HTMLInputElement | null)?.select(), 50);
  }).finally(() => { creating.value = false; });
}

// Saves the draft; status publishes or unpublishes it at the same time.
function save(status?: 'live' | 'draft') {
  const f = current.value;
  if (!f || !draft.value) return;
  const body = { ...draft.value, status: status || draft.value.status };
  saving.value = true;
  api.updateForm(f.id, toReq(body)).then((data) => {
    const saved = data as unknown as SignupForm;
    replace(saved);
    draft.value = toDraft(saved);
    let msg = t('forms.saved');
    if (status === 'live' && f.status !== 'live') msg = t('forms.published');
    else if (status === 'draft' && f.status === 'live') msg = t('forms.unpublished');
    $utils.toast(msg);
    loadStats();
  }).finally(() => { saving.value = false; });
}

function removeForm() {
  const f = current.value;
  if (!f) return;
  saving.value = true;
  api.deleteForm(f.id).then(() => {
    confirmDelete.value = false;
    forms.value = forms.value.filter((x) => x.id !== f.id);
    $utils.toast(t('forms.deleted', { name: f.name }));
    if (forms.value.length) show(forms.value[0].id);
    else {
      selectedId.value = 0;
      draft.value = null;
    }
  }).finally(() => { saving.value = false; });
}

// Leaving with unsaved changes asks first.
function onBeforeUnload(e: BeforeUnloadEvent) {
  if (dirty.value) e.preventDefault();
}
watch(dirty, (d) => {
  if (d) window.addEventListener('beforeunload', onBeforeUnload);
  else window.removeEventListener('beforeunload', onBeforeUnload);
});

onMounted(() => {
  load().then(() => {
    if (!forms.value.length) return;
    const want = Number(route.query.form);
    show(forms.value.some((f) => f.id === want) ? want : forms.value[0].id);
  });
});
</script>

<style scoped lang="scss">
.forms-skel { display: flex; flex-direction: column; gap: 16px; }

.forms-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;

  @media (max-width: 767px) {
    display: flex;
    overflow-x: auto;
    scroll-snap-type: x mandatory;
    margin: 0 -16px;
    padding: 2px 16px 6px;

    .forms-card { flex: 0 0 78%; scroll-snap-align: start; }
  }
}
.forms-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  color: var(--ln-text);
  font: inherit;
  text-align: left;
  cursor: pointer;

  &:hover { border-color: var(--ln-text-3); }
  // A 2px ring drawn as border + inset shadow, so choosing doesn't shift.
  &.is-on { border-color: var(--ln-ink); box-shadow: inset 0 0 0 1px var(--ln-ink); }
}
.forms-card-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;

  strong { font-weight: 600; overflow-wrap: anywhere; }
}
.forms-chip {
  flex: none;
  padding: 1px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;

  &.is-live { background: var(--ln-success-bg); color: var(--ln-success-fg); }
  &.is-draft { background: var(--ln-muted); color: var(--ln-text-2); }
}
.forms-card-num {
  color: var(--ln-text-3);
  font-size: 13px;

  strong { margin-right: 4px; color: var(--ln-text); font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; }
}

.forms-work {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr) 340px;
  gap: 16px;
  align-items: start;

  @media (max-width: 1279px) {
    grid-template-columns: 300px minmax(0, 1fr);

    .forms-side { grid-column: 2; }
  }
  @media (max-width: 991px) {
    grid-template-columns: minmax(0, 1fr);

    .forms-side { grid-column: auto; }
  }
}
.forms-side { display: flex; flex-direction: column; gap: 16px; }

.forms-stats {
  padding: 14px 16px;

  h2 { margin: 0 0 10px; font-size: 15px; font-weight: 600; }
  dl {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 8px 12px;
    margin: 0;
  }
  dt { color: var(--ln-text-2); }
  dd { margin: 0; font-weight: 600; font-variant-numeric: tabular-nums; text-align: right; }
}
</style>
