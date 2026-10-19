<template>
  <div class="ln-page dev-brand">
    <page-header :title="$t('developers.branding.title')" :subtitle="$t('developers.branding.intro')">
      <PvButton
        v-if="canManage"
        :label="$t('developers.branding.save')"
        :loading="saving"
        :disabled="!dirty || !linkValid"
        data-cy="btn-save"
        @click="save"
      />
    </page-header>

    <error-state v-if="loadError" :title="$t('states.couldntLoad')" :error="loadError" @retry="load" />

    <div v-else-if="form" class="dev-brand-grid">
      <div class="dev-brand-form">
        <section class="ln-card">
          <div class="ln-card-head dev-brand-card-head">
            <div>
              <h2 class="ln-card-title">{{ $t('developers.branding.logo') }}</h2>
              <p class="ln-small">{{ $t('developers.branding.logoHelp') }}</p>
            </div>
          </div>
          <div class="ln-card-pad">
            <logo-field v-model="form.logo" :disabled="!canManage" />
          </div>
        </section>

        <section class="ln-card">
          <div class="ln-card-head dev-brand-card-head">
            <div>
              <h2 class="ln-card-title">
                {{ $t('developers.branding.logoDark') }}
                <span class="ln-muted dev-brand-optional">· {{ $t('developers.branding.optional') }}</span>
              </h2>
              <p class="ln-small">{{ $t('developers.branding.logoDarkHelp') }}</p>
            </div>
          </div>
          <div class="ln-card-pad">
            <logo-field v-model="form.logoDark" dark :disabled="!canManage" />
          </div>
        </section>

        <section class="ln-card ln-card-pad dev-brand-layout">
          <div class="dev-brand-row">
            <div class="ln-field">
              <label for="brand-width">{{ $t('developers.branding.width') }}</label>
              <div class="dev-brand-slider">
                <PvSlider
                  id="brand-width"
                  v-model="form.width"
                  :min="BRAND_WIDTH_MIN"
                  :max="BRAND_WIDTH_MAX"
                  :step="4"
                  :disabled="!canManage"
                  :aria-label="$t('developers.branding.width')"
                />
                <span class="ln-mono">{{ form.width }} px</span>
              </div>
            </div>
            <div class="ln-field">
              <span class="ln-field-label">{{ $t('developers.branding.alignment') }}</span>
              <PvSelectButton
                v-model="form.align"
                :options="alignOptions"
                option-label="label"
                option-value="value"
                :allow-empty="false"
                :disabled="!canManage"
                :aria-label="$t('developers.branding.alignment')"
              />
            </div>
          </div>
          <div class="ln-field">
            <label for="brand-link">{{ $t('developers.branding.linksTo') }}</label>
            <PvInputText
              id="brand-link"
              v-model="form.linkUrl"
              class="ln-mono"
              placeholder="https://acme.example"
              :invalid="!linkValid"
              :disabled="!canManage"
            />
            <p v-if="!linkValid" class="ln-field-error">{{ $t('developers.branding.invalidLink') }}</p>
          </div>
        </section>
      </div>

      <!-- Sample receipt -->
      <div class="dev-brand-preview">
        <div class="dev-brand-preview-head">
          <h2 class="ln-card-title">{{ $t('developers.branding.previewTitle') }}</h2>
          <span v-if="sampleTemplate" class="ln-small">{{ $t('developers.branding.previewTemplate', { name: sampleTemplate }) }}</span>
        </div>
        <div class="dev-brand-clients">
          <div v-for="c in clients" :key="c.key" class="dev-brand-client">
            <span class="ln-label">{{ c.label }}</span>
            <div class="dev-brand-mail" :class="`is-${c.key}`">
              <div class="dev-brand-mail-card">
                <div class="dev-brand-mail-logo" :style="{ textAlign: form.align }">
                  <span v-if="logoFor(c.key)" class="dev-brand-mail-tile" :class="{ 'has-tile': c.key === 'dark' && !form.logoDark }">
                    <img :src="logoFor(c.key)!" :alt="orgName" :style="{ width: `${form.width}px` }">
                  </span>
                  <span v-else class="ln-small">{{ $t('developers.branding.noLogo') }}</span>
                </div>
                <h3>{{ $t('developers.branding.sampleHeading') }}</h3>
                <p>{{ $t('developers.branding.sampleText') }}</p>
                <div class="dev-brand-mail-total"><strong>{{ $t('developers.branding.sampleTotal') }}</strong><strong>$49.00</strong></div>
              </div>
            </div>
          </div>
        </div>
        <p class="ln-card ln-card-pad dev-brand-note">
          <i class="pi pi-info-circle" aria-hidden="true" />
          <span>
            <i18n-t keypath="developers.branding.usage" tag="span">
              <template #block><code class="ln-mono">brand-logo</code></template>
            </i18n-t>
            {{ usageLine }}
          </span>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { storeToRefs } from 'pinia';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import {
  getBranding, listTxTemplates, updateBranding, type Branding, type TxTemplate,
} from '../../api/developers';
import PageHeader from '../../components/ui/PageHeader.vue';
import ErrorState, { type RequestError } from '../../components/ui/ErrorState.vue';
import LogoField from './LogoField.vue';

// Bounds match the server's (models.BrandWidthMin / Max).
const BRAND_WIDTH_MIN = 40;
const BRAND_WIDTH_MAX = 600;

const { t } = useI18n();
const { $can, $utils } = useGlobal();
const { serverConfig } = storeToRefs(useMainStore());

const form = ref<Branding | null>(null);
const saved = ref('');
const saving = ref(false);
const loadError = ref<RequestError | null>(null);
const templates = ref<TxTemplate[]>([]);

const canManage = computed(() => $can('templates:manage'));
const orgName = computed(() => (serverConfig.value as { organization_name?: string }).organization_name || 'listnun');
const dirty = computed(() => !!form.value && JSON.stringify(form.value) !== saved.value);
const linkValid = computed(() => {
  const v = form.value?.linkUrl.trim() || '';
  return !v || /^https?:\/\/[^\s/]+/.test(v);
});

const alignOptions = computed(() => [
  { value: 'left', label: t('developers.branding.left') },
  { value: 'center', label: t('developers.branding.centre') },
]);

const clients = computed(() => [
  { key: 'light', label: t('developers.branding.lightClient') },
  {
    key: 'dark',
    label: form.value?.logoDark ? t('developers.branding.darkClient') : t('developers.branding.darkClientNoLogo'),
  },
]);

const usingBlock = computed(() => templates.value.filter((tpl) => tpl.body.includes('"brand-logo"')));
const sampleTemplate = computed(() => usingBlock.value[0]?.name || '');
const usageLine = computed(() => {
  const n = usingBlock.value.length;
  const total = templates.value.length;
  if (!total) return '';
  if (n === total) return t('developers.branding.usageAll', { n: total });
  return t('developers.branding.usageSome', { n, total });
});

function logoFor(client: string): string | null {
  if (!form.value?.logo) return null;
  if (client === 'dark' && form.value.logoDark) return form.value.logoDark.url;
  return form.value.logo.url;
}

function load() {
  loadError.value = null;
  getBranding()
    .then((b) => {
      form.value = { ...b, linkUrl: b.linkUrl || '' };
      saved.value = JSON.stringify(form.value);
    })
    .catch((err) => {
      const e = err as { response?: { status?: number }; config?: { method?: string; url?: string } };
      loadError.value = { status: e.response?.status, method: e.config?.method, url: e.config?.url };
    });
}

function save() {
  if (!form.value) return;
  saving.value = true;
  updateBranding({ ...form.value, linkUrl: form.value.linkUrl.trim() })
    .then((b) => {
      form.value = { ...b, linkUrl: b.linkUrl || '' };
      saved.value = JSON.stringify(form.value);
      $utils.toast(t('developers.branding.saved'));
    })
    .finally(() => { saving.value = false; });
}

onMounted(() => {
  load();
  listTxTemplates().then((list) => { templates.value = list; });
});
</script>

<style scoped lang="scss">
.dev-brand-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.3fr);
  gap: 20px;
  align-items: start;

  @media (max-width: 1100px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-brand-form { display: flex; flex-direction: column; gap: 16px; }
.dev-brand-card-head {
  align-items: flex-start;
  .ln-small { margin: 2px 0 0; }
}
.dev-brand-optional { font-weight: 400; }
.dev-brand-layout { display: flex; flex-direction: column; gap: 16px; }
.dev-brand-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 20px;
  align-items: end;

  @media (max-width: 600px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-brand-slider {
  display: flex;
  align-items: center;
  gap: 14px;
  height: 36px;

  .p-slider { flex: 1; }
  .ln-mono { font-size: 12px; white-space: nowrap; }
}

.dev-brand-preview { display: flex; flex-direction: column; gap: 12px; }
.dev-brand-preview-head { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; }
.dev-brand-clients {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;

  @media (max-width: 600px) { grid-template-columns: minmax(0, 1fr); }
}
.dev-brand-client { display: flex; flex-direction: column; gap: 8px; min-width: 0; }
.dev-brand-mail {
  border-radius: 10px;
  padding: 16px;
  background: #f1f0ed;
  border: 1px solid #dedad0;

  &.is-dark { background: #101318; border-color: #2b2f36; }
}
.dev-brand-mail-card {
  background: #ffffff;
  color: #1b1e24;
  border-radius: 6px;
  padding: 20px;
  font-size: 14px;

  .is-dark & { background: #1b1f26; color: #ece9e1; }
  h3 { margin: 0 0 12px; font-size: 17px; font-weight: 600; }
  p { margin: 0 0 16px; opacity: .85; }
}
.dev-brand-mail-logo {
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid rgba(127, 127, 127, .25);

  img { max-width: 100%; height: auto; display: inline-block; vertical-align: middle; }
}
.dev-brand-mail-tile {
  display: inline-block;
  max-width: 100%;

  &.has-tile { background: #fff; padding: 8px 12px; border-radius: 6px; }
}
.dev-brand-mail-total {
  display: flex;
  justify-content: space-between;
  border-top: 1px solid rgba(127, 127, 127, .25);
  padding-top: 12px;
}
.dev-brand-note {
  display: flex;
  gap: 10px;
  margin: 0;
  font-size: 13px;
  color: var(--ln-text-2);

  i { margin-top: 3px; }
}
</style>
