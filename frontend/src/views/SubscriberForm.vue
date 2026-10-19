<template>
  <form class="lm-form" @submit.prevent="onSubmit">
    <div class="lm-form-header">
      <div class="lm-form-title-row">
        <h3 class="lm-form-title">{{ isEditing ? data.name : $t('subscribers.newSubscriber') }}</h3>
        <PvTag v-if="isEditing" :severity="data.status === 'enabled' ? 'success' : 'danger'" size="small"
          :value="$t(`subscribers.status.${data.status}`)" />
      </div>
      <p v-if="isEditing" class="lm-form-meta">
        ID: <copy-text :text="`${data.id}`" data-cy="id" /> &nbsp;·&nbsp;
        UUID: <copy-text :text="data.uuid" />
      </p>
    </div>

    <div class="lm-form-body">
      <div class="lm-field">
        <label class="lm-label">{{ $t('subscribers.email') }}</label>
        <PvInputText :maxlength="200" v-model="form.email" name="email" ref="focusEl"
          :placeholder="$t('subscribers.email')" class="w-full" required />
      </div>

      <div class="lm-field-row">
        <div class="lm-field">
          <label class="lm-label">{{ $t('globals.fields.name') }}</label>
          <PvInputText :maxlength="200" v-model="form.name" name="name"
            :placeholder="$t('globals.fields.name')" class="w-full" />
        </div>
        <div class="lm-field">
          <label class="lm-label">{{ $t('globals.fields.status') }}</label>
          <PvSelect v-model="form.status" name="status" :placeholder="$t('globals.fields.status')" required
            :options="statusOptions" option-label="label" option-value="value" class="w-full" />
          <small class="lm-help">{{ $t('subscribers.blocklistedHelp') }}</small>
        </div>
      </div>

      <PvTabs class="lm-tabs" v-model:value="activeTab">
        <PvTabList>
          <PvTab value="0">{{ $t('globals.terms.lists') }}</PvTab>
          <PvTab value="1">{{ `${$t('globals.terms.subscriptions', 2)} (${data.lists ? data.lists.length : 0})` }}</PvTab>
          <PvTab value="2" :disabled="bounces.length === 0">{{ `${$t('globals.terms.bounces')} (${bounces.length})` }}</PvTab>
          <PvTab value="3">{{ $t('globals.terms.attribs') }}</PvTab>
          <PvTab value="4" :disabled="!isEditing">{{ $t('subscribers.activity') }}</PvTab>
        </PvTabList>
        <PvTabPanels>
          <!-- lists -->
          <PvTabPanel value="0">
            <list-selector :label="$t('subscribers.lists')" :placeholder="$t('subscribers.listsPlaceholder')"
              :message="$t('subscribers.listsHelp')" v-model="form.lists" :selected="form.lists" :all="lists.results" />
            <div class="lm-field-row lm-field-row--preconfirm">
              <div class="lm-field">
                <small class="lm-help">{{ $t('subscribers.preconfirmHelp') }}</small>
                <div class="check-row">
                  <PvCheckbox v-model="form.preconfirm" :binary="true" :disabled="!hasOptinList" />
                  <span class="check-label">{{ $t('subscribers.preconfirm') }}</span>
                </div>
              </div>
              <div v-if="$can('subscribers:manage') && isEditing" class="optin-action">
                <a href="#" @click.prevent="sendOptinConfirmation"
                  :class="['optin-link', { 'optin-link--disabled': !hasOptinList }]">
                  <i class="pi pi-envelope" />
                  {{ $t('subscribers.sendOptinConfirm') }}
                </a>
              </div>
            </div>
          </PvTabPanel><!-- lists -->

          <!-- subscriptions -->
          <PvTabPanel value="1">
            <template v-if="data.lists">
              <PvDataTable :value="data.lists" hoverable sort-field="createdAt" class="subscriptions">
                <PvColumn field="name" :header="$t('globals.terms.list', 1)">
                  <template #body="{ data: row }">
                    <div class="sub-name-cell">
                      <router-link v-if="!row.restricted" :to="`/lists/${row.id}`">{{ row.name }}</router-link>
                      <span v-else class="sub-restricted">{{ row.name }}</span>
                      <PvTag :severity="row.optin === 'double' ? 'success' : 'secondary'" size="small"
                        :data-cy="`optin-${row.optin}`">
                        <i :class="row.optin === 'double' ? 'pi pi-check-circle' : 'pi pi-times-circle'" />
                        {{ ' ' }}{{ $t(`lists.optins.${row.optin}`) }}
                      </PvTag>
                    </div>
                  </template>
                </PvColumn>

                <PvColumn field="status" :header="$t('globals.fields.status')" class="status">
                  <template #body="{ data: row }">
                    <div class="sub-status-cell">
                      <PvTag :severity="subStatusSeverity(row.subscriptionStatus)" size="small"
                        :value="$t(`subscribers.status.${row.subscriptionStatus}`)" />
                      <span v-if="row.optin === 'double' && row.subscriptionMeta.optinIp" class="sub-ip">
                        {{ row.subscriptionMeta.optinIp }}
                      </span>
                    </div>
                  </template>
                </PvColumn>

                <PvColumn field="createdAt" :header="$t('globals.fields.createdAt')">
                  <template #body="{ data: row }">{{ $utils.niceDate(row.subscriptionCreatedAt, true) }}</template>
                </PvColumn>

                <PvColumn field="updatedAt" :header="$t('globals.fields.updatedAt')">
                  <template #body="{ data: row }">{{ $utils.niceDate(row.subscriptionCreatedAt, true) }}</template>
                </PvColumn>
              </PvDataTable>
            </template>
          </PvTabPanel><!-- subscriptions -->

          <!-- bounces -->
          <PvTabPanel value="2" class="bounces">
            <div class="bounces-header">
              <a v-if="isBounceVisible" href="#" class="delete-link" @click.prevent="deleteBounces">
                <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
              </a>
            </div>

            <PvDataTable :value="bounces" hoverable sort-field="createdAt">
              <PvColumn field="campaign" :header="$t('globals.terms.campaign', 1)">
                <template #body="{ data: row }">
                  <router-link v-if="row.campaign" :to="{ name: 'bounces', query: { campaign_id: row.campaign.id } }">
                    {{ row.campaign.name }}
                  </router-link>
                </template>
              </PvColumn>

              <PvColumn field="createdAt" :header="$t('globals.fields.createdAt')">
                <template #body="{ data: row }">{{ $utils.niceDate(row.createdAt, true) }}</template>
              </PvColumn>

              <PvColumn field="action" :header="$t('globals.fields.type')">
                <template #body="{ data: row }">
                  <div class="bounce-meta-row">
                    <a href="#" class="bounce-source" @click.prevent="toggleMeta(row.id)">
                      {{ row.source }}
                      <i :class="visibleMeta[row.id] ? 'pi pi-chevron-up' : 'pi pi-chevron-down'" />
                    </a>
                  </div>
                  <pre v-if="visibleMeta[row.id]" class="bounce-meta-pre">{{ row.meta }}</pre>
                </template>
              </PvColumn>
            </PvDataTable>
          </PvTabPanel><!-- bounces -->

          <!-- attributes -->
          <PvTabPanel value="3" class="attribs-panel">
            <small class="lm-help">{{ $t('subscribers.attribsHelp') + ' ' + egAttribs }}</small>
            <PvTextarea v-model="form.strAttribs" name="attribs" rows="6" class="w-full attribs-textarea" />
            <a href="https://listmonk.app/docs/concepts" target="_blank" rel="noopener noreferrer" class="learn-more-link">
              {{ $t('globals.buttons.learnMore') }} <i class="pi pi-external-link" />
            </a>
          </PvTabPanel><!-- attributes -->

          <!-- activity -->
          <PvTabPanel value="4" class="activity">
            <subscriber-activity v-if="isEditing && data.id" :subscriber-id="data.id" />
          </PvTabPanel><!-- activity -->
        </PvTabPanels>
      </PvTabs>
    </div>

    <div class="lm-form-footer">
      <PvButton @click="$emit('close')" :label="$t('globals.buttons.close')" severity="secondary" />
      <PvButton v-if="$can('subscribers:manage')" type="submit" severity="primary"
        :loading="loading.subscribers" :label="$t('globals.buttons.save')" />
    </div>
  </form>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import ListSelector from '../components/ListSelector.vue';
import CopyText from '../components/CopyText.vue';
import SubscriberActivity from '../components/SubscriberActivity.vue';
import { getSubscribers as subscribersApi } from '../api/generated/endpoints/subscribers/subscribers';
import { getBounces as bouncesApi } from '../api/generated/endpoints/bounces/bounces';

const props = withDefaults(defineProps<{
  data?: any;
  isEditing?: boolean;
}>(), { data: () => ({ lists: [] }), isEditing: false });

const emit = defineEmits(['finished', 'close']);

const { $utils } = useGlobal();
const {
  createSubscriber, updateSubscriber, sendSubscriberOptin, deleteSubscriberBounces,
} = subscribersApi();
const { getSubscriberBounces } = bouncesApi();
const { t } = useI18n();
const { lists, loading } = storeToRefs(useMainStore());

const focusEl = ref<any>(null);
const isBounceVisible = ref(false);
const bounces = ref<any[]>([]);
const visibleMeta = reactive<Record<number, boolean>>({});
const activeTab = ref('0');
const egAttribs = '{"job": "developer", "location": "Mars", "has_rocket": true}';

const form = reactive<any>({
  lists: [],
  strAttribs: '{}',
  status: 'enabled',
  preconfirm: false,
});

const statusOptions = computed(() => [
  { label: t('subscribers.status.enabled'), value: 'enabled' },
  { label: t('subscribers.status.blocklisted'), value: 'blocklisted' },
]);

const hasOptinList = computed(() => form.lists.some((l: any) => l.optin === 'double'));

function subStatusSeverity(status: string) {
  const map: Record<string, string> = {
    confirmed: 'success', unconfirmed: 'warn', unsubscribed: 'secondary', blocklisted: 'danger',
  };
  return map[status] || 'secondary';
}

function toggleMeta(id: number) { visibleMeta[id] = !visibleMeta[id]; }

function fetchBounces() {
  getSubscriberBounces(form.id).then((data: any) => { bounces.value = data; });
}

function deleteBounces() {
  $utils.confirm(null, () => {
    deleteSubscriberBounces(form.id).then(() => {
      fetchBounces();
      $utils.toast(t('globals.messages.deleted', { name: form.name }));
    });
  });
}

function validateAttribs(str: string) {
  let attribs: any = {};
  try {
    attribs = JSON.parse(str);
  } catch (e: any) {
    $utils.toast(`${t('subscribers.invalidJSON')}: ${e.toString()}`, 'is-danger', 3000);
    return null;
  }
  if (attribs instanceof Array) {
    $utils.toast('Attributes should be a map {} and not an array []', 'is-danger', 3000);
    return null;
  }
  return attribs;
}

function onCreateSubscriber() {
  let attribs = {};
  if (form.strAttribs) {
    attribs = validateAttribs(form.strAttribs);
    if (!attribs) return;
  }
  createSubscriber({
    email: form.email,
    name: form.name,
    status: form.status,
    attribs,
    preconfirm_subscriptions: form.preconfirm,
    lists: form.lists.map((l: any) => l.id),
  }).then((d: any) => {
    emit('finished'); emit('close');
    $utils.toast(t('globals.messages.created', { name: d.name }));
  });
}

function onUpdateSubscriber() {
  let attribs = {};
  if (form.strAttribs) {
    attribs = validateAttribs(form.strAttribs);
    if (!attribs) return;
  }
  updateSubscriber(form.id, {
    email: form.email,
    name: form.name,
    status: form.status,
    preconfirm_subscriptions: form.preconfirm,
    attribs,
    lists: form.lists.map((l: any) => l.id),
  }).then((d: any) => {
    emit('finished'); emit('close');
    $utils.toast(t('globals.messages.updated', { name: d.name }));
  });
}

function onSubmit() {
  if (props.isEditing) { onUpdateSubscriber(); return; }
  onCreateSubscriber();
}

function sendOptinConfirmation() {
  if (!hasOptinList.value) return;
  sendSubscriberOptin(form.id).then(() => {
    $utils.toast(t('subscribers.sentOptinConfirm'));
  });
}

onMounted(() => {
  if (props.isEditing) {
    Object.assign(form, { ...props.data, strAttribs: JSON.stringify(props.data.attribs, null, 4) });
  }
  if (form.id) fetchBounces();
  nextTick(() => { focusEl.value?.$el?.focus(); });
});
</script>

<style scoped lang="scss">
:deep(.p-tag-secondary) {
  background: var(--ln-muted);
  color: var(--ln-text-2);
  border: 1px solid var(--ln-border);
}

.lm-field { display: flex; flex-direction: column; gap: 0.35rem; margin-bottom: 0; }
.lm-field-row { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.lm-field-row--preconfirm { align-items: start; margin-top: 0.75rem; }

.lm-label { display: block; font-size: 0.8rem; font-weight: 600; color: var(--ln-text); }
.lm-help { display: block; font-size: 0.75rem; color: var(--ln-text-4); line-height: 1.4; }

.check-row { display: flex; align-items: center; gap: 0.5rem; }
.check-label { font-size: 0.875rem; color: var(--ln-text); }

.optin-action { display: flex; align-items: flex-end; padding-bottom: 0.1rem; }
.optin-link {
  font-size: 0.85rem; color: var(--ln-ink); text-decoration: none;
  display: inline-flex; align-items: center; gap: 0.35rem;
  &:hover { text-decoration: underline; }
  &--disabled { opacity: 0.4; pointer-events: none; }
}

.sub-name-cell { display: flex; flex-direction: column; gap: 0.25rem; }
.sub-restricted { color: var(--ln-text-4); font-style: italic; }
.sub-status-cell { display: flex; flex-direction: column; gap: 0.2rem; }
.sub-ip { font-size: 0.75rem; color: var(--ln-text-4); }

.bounces-header { display: flex; justify-content: flex-end; margin-bottom: 0.5rem; }
.delete-link { font-size: 0.85rem; color: var(--ln-danger); text-decoration: none; display: inline-flex; align-items: center; gap: 0.3rem; &:hover { text-decoration: underline; } }

.bounce-meta-row { display: flex; justify-content: flex-end; }
.bounce-source { font-size: 0.85rem; color: var(--ln-text-3); text-decoration: none; display: inline-flex; align-items: center; gap: 0.3rem; &:hover { color: var(--ln-text); } }
.bounce-meta-pre { font-size: 0.75rem; margin-top: 0.4rem; background: var(--ln-bg); border-radius: 4px; padding: 0.5rem; overflow-x: auto; }

.attribs-panel { display: flex; flex-direction: column; gap: 0.5rem; }
.attribs-textarea { font-family: monospace; font-size: 0.82rem; }
.learn-more-link { font-size: 0.78rem; color: var(--ln-ink); text-decoration: none; &:hover { text-decoration: underline; } }
</style>
