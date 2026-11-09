<template>
  <div class="set-cards">
    <set-card :title="$t('settings.scrubx.title')">
      <set-row :label="$t('settings.scrubx.on')" :help="$t('settings.scrubx.onHelp')" toggle>
        <PvToggleSwitch v-model="data.scrub.enabled" name="scrub.enabled" :disabled="locked" :aria-label="$t('settings.scrubx.on')" />
        <template v-if="locked" #note>{{ $t('settings.scrub.managedByPlatform') }}</template>
      </set-row>
      <template v-if="data.scrub.enabled && !locked">
        <set-row :label="$t('settings.scrub.url')" label-for="scrub-url">
          <PvInputText id="scrub-url" v-model="data.scrub.url" name="scrub.url" placeholder="https://api.thescrub.app" :maxlength="300" class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.scrub.apiKey')">
          <PvPassword
            v-model="data.scrub.api_key"
            name="scrub.api_key"
            :maxlength="300"
            :feedback="false"
            :placeholder="$t('settings.scrub.apiKeyPlaceholder')"
            :aria-label="$t('settings.scrub.apiKey')"
          />
        </set-row>
        <set-row :label="$t('settings.scrub.integrationId')" :help="$t('settings.scrubx.integrationHelp')" label-for="scrub-integration">
          <PvInputText id="scrub-integration" v-model="data.scrub.integration_id" name="scrub.integration_id" :maxlength="64" class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.scrub.testConnection')" :help="$t('settings.scrubx.testHelp')">
          <span>
            <PvButton
              severity="secondary"
              outlined
              :loading="isTesting"
              :disabled="!data.scrub.url || !data.scrub.api_key"
              :label="$t('settings.scrub.testConnection')"
              @click="testConnection"
            />
          </span>
        </set-row>
      </template>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../../composables/useGlobal';
import { getSettings as settingsApi } from '../../api/generated/endpoints/settings/settings';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

const props = defineProps<{ form?: any }>();
const { $utils } = useGlobal();
const { testScrubSettings } = settingsApi();
const { t } = useI18n();
const data = props.form;
const isTesting = ref(false);
const locked = computed(() => !!data.scrub.managed_by_platform);

async function testConnection() {
  isTesting.value = true;
  try {
    await testScrubSettings({ url: data.scrub.url, api_key: data.scrub.api_key });
    $utils.toast(t('settings.scrub.testSuccess'), 'is-success');
  } catch (e: any) {
    $utils.toast(e.response?.data?.message || t('settings.scrub.testError'), 'is-danger');
  } finally {
    isTesting.value = false;
  }
}
</script>

<style scoped lang="scss">
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
