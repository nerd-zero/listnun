<template>
  <div class="ln-page tpls">
    <page-header :title="$t('globals.terms.templates')" :count="templates.length || null" :subtitle="$t('templates.intro')">
      <PvButton v-if="$can('templates:manage')" icon="pi pi-plus" :label="$t('templates.newTemplate')" data-cy="btn-new" @click="newTemplate" />
    </page-header>

    <div v-if="loading.templates && !templates.length" class="tpls-grid">
      <div v-for="n in 3" :key="n" class="ln-card tpls-card"><PvSkeleton height="190px" /></div>
    </div>

    <empty-state
      v-else-if="!shown.length"
      icon="pi-file"
      :title="$t('templates.emptyTitle')"
      :text="$t('templates.emptyText')"
      :action-label="$can('templates:manage') ? $t('templates.newTemplate') : ''"
      @action="newTemplate"
    />

    <div v-else class="tpls-grid">
      <article v-for="tpl in shown" :key="tpl.id" class="ln-card tpls-card" :data-cy="`template-${tpl.id}`">
        <router-link :to="editLink(tpl)" class="tpls-thumb" :aria-label="tpl.name" tabindex="-1">
          <iframe
            v-if="tpl.type !== 'campaign_visual' || tpl.body"
            :src="`/api/templates/${tpl.id}/preview`"
            loading="lazy"
            scrolling="no"
            sandbox=""
            tabindex="-1"
            aria-hidden="true"
            :title="tpl.name"
          />
        </router-link>
        <div class="tpls-body">
          <div class="tpls-title">
            <router-link :to="editLink(tpl)" class="ln-row-title" data-cy="btn-edit">{{ tpl.name }}</router-link>
            <status-tag v-if="tpl.isDefault" status="active" :label="$t('templates.default')" />
          </div>
          <div class="tpls-meta">
            <span v-if="tpl.type === 'campaign_visual'" class="ln-attr" data-cy="type-campaign_visual">{{ $t('templates.visual') }}</span>
            <span class="ln-small">{{ tpl.campaignCount ? $t('templates.usedBy', { num: tpl.campaignCount }, tpl.campaignCount) : $t('templates.unused') }}</span>
            <span class="ln-dialog-form-spacer" />
            <button
            v-if="$can('templates:manage')"
            type="button"
            class="ln-row-menu"
            :aria-label="$t('globals.buttons.rowActions')"
            aria-haspopup="true"
            data-cy="btn-row-menu"
            @click="(e) => openMenu(e, tpl)"
          >
              <i class="pi pi-ellipsis-h" />
            </button>
          </div>
        </div>
      </article>
    </div>
    <PvMenu ref="menu" :model="menuItems" popup />

    <dialog-form
      v-model:visible="clone.open"
      :title="$t('globals.buttons.clone')"
      :submit-label="$t('globals.buttons.clone')"
      :busy="clone.busy"
      :submit-disabled="!clone.name.trim()"
      @submit="doClone"
    >
      <div class="ln-field">
        <label for="tpl-clone-name">{{ $t('globals.fields.name') }}</label>
        <PvInputText id="tpl-clone-name" v-model="clone.name" autofocus />
      </div>
    </dialog-form>

    <confirm-action
      v-model:visible="confirm.open"
      :title="confirm.title"
      :text="confirm.text"
      :confirm-label="confirm.label"
      :danger="confirm.danger"
      :busy="confirm.busy"
      @confirm="confirm.run"
    />
  </div>
</template>

<script setup lang="ts">
import {
  computed, onMounted, reactive, ref, watch,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import type { MenuItem } from 'primevue/menuitem';
import { useRouter } from 'vue-router';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import PageHeader from '../components/ui/PageHeader.vue';
import EmptyState from '../components/ui/EmptyState.vue';
import StatusTag from '../components/ui/StatusTag.vue';
import DialogForm from '../components/ui/DialogForm.vue';
import ConfirmAction from '../components/ui/ConfirmAction.vue';
import { getTemplates as templatesApi } from '../api/generated/endpoints/templates/templates';

// Templates are the API's camelCased objects.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
interface PopupMenu { toggle: (e: Event) => void }

const { $utils } = useGlobal();
const store = useMainStore();
const {
  listTemplates, createTemplate, deleteTemplate, setDefaultTemplate,
} = templatesApi();
const { t } = useI18n();
const router = useRouter();
const { refreshTick, loading } = storeToRefs(store);
// Campaign templates only: transactional ones are edited under Developers.
const templates = computed<Loose[]>(() => ((store.templates as Loose[]) || []).filter((x) => x.type !== 'tx'));
const menu = ref<PopupMenu | null>(null);
const menuFor = ref<Loose | null>(null);
const clone = reactive({
  open: false, busy: false, name: '', source: null as Loose | null,
});
const confirm = reactive({
  open: false, title: '', text: '', label: '', danger: false, busy: false, run: () => {},
});

const shown = templates;

function editLink(tpl: Loose) {
  return { name: 'template', params: { id: String(tpl.id) } };
}

function newTemplate() {
  router.push({ name: 'template', params: { id: 'new' } });
}

function fetchTemplates() {
  listTemplates().then((data) => { store.setModelResponse({ model: 'templates', data }); });
}

function askDefault(tpl: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('templates.defaultTitle', { name: tpl.name }),
    text: t('templates.defaultText'),
    label: t('templates.makeDefault'),
    danger: false,
    run: () => {
      confirm.busy = true;
      setDefaultTemplate(tpl.id).then(() => {
        confirm.open = false;
        fetchTemplates();
        $utils.toast(t('globals.messages.updated', { name: tpl.name }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askDelete(tpl: Loose) {
  Object.assign(confirm, {
    open: true,
    title: t('templates.deleteTitle', { name: tpl.name }),
    text: tpl.type === 'tx' ? t('templates.deleteTxText') : t('templates.deleteText'),
    label: t('globals.buttons.delete'),
    danger: true,
    run: () => {
      confirm.busy = true;
      deleteTemplate(tpl.id).then(() => {
        confirm.open = false;
        fetchTemplates();
        $utils.toast(t('globals.messages.deleted', { name: tpl.name }));
      }).finally(() => { confirm.busy = false; });
    },
  });
}

function askClone(tpl: Loose) {
  clone.source = tpl;
  clone.name = t('campaigns.copyOf', { name: tpl.name });
  clone.open = true;
}

function doClone() {
  const tpl = clone.source;
  if (!tpl) return;
  clone.busy = true;
  createTemplate({
    name: clone.name.trim(), type: tpl.type, subject: tpl.subject, body: tpl.body, body_source: tpl.bodySource,
  } as never).then((d) => {
    clone.open = false;
    fetchTemplates();
    $utils.toast(t('globals.messages.created', { name: (d as unknown as Loose).name }));
  }).finally(() => { clone.busy = false; });
}

const menuItems = computed<MenuItem[]>(() => {
  const tpl = menuFor.value;
  if (!tpl) return [];
  const items: MenuItem[] = [
    { label: t('globals.buttons.clone'), icon: 'pi pi-copy', command: () => askClone(tpl) },
  ];
  if (!tpl.isDefault && tpl.type === 'campaign') {
    items.push({ label: t('templates.makeDefault'), icon: 'pi pi-check-circle', command: () => askDefault(tpl) });
  }
  if (!tpl.isDefault) {
    items.push({ separator: true }, {
      label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(tpl),
    });
  }
  return items;
});

function openMenu(e: Event, tpl: Loose) {
  menuFor.value = tpl;
  menu.value?.toggle(e);
}

watch(refreshTick, fetchTemplates);
onMounted(fetchTemplates);
</script>

<style scoped lang="scss">
.tpls-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
}
.tpls-card { display: flex; flex-direction: column; overflow: hidden; }
.tpls-thumb {
  display: block;
  position: relative;
  height: 170px;
  overflow: hidden;
  border: 0;
  border-bottom: 1px solid var(--ln-border);
  background: var(--ln-muted);
  padding: 0;
  cursor: pointer;

  iframe {
    width: 300%;
    height: 300%;
    transform: scale(0.3333);
    transform-origin: 0 0;
    border: 0;
    pointer-events: none;
    background: #fff;
  }
  &:focus-visible { outline: 2px solid var(--ln-ring); outline-offset: -2px; }
}
.tpls-body { padding: 14px 16px 10px; display: flex; flex-direction: column; gap: 6px; }
.tpls-title { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.tpls-meta { display: flex; align-items: center; gap: 8px; }
</style>
