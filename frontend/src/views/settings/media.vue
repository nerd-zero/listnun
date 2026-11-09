<template>
  <div class="set-cards">
    <set-card :title="$t('settings.mediax.uploads')" :help="$t('settings.mediax.uploadsHelp')">
      <set-row :label="$t('settings.mediax.types')" :help="$t('settings.media.upload.extensionsHelp')" wide>
        <PvAutoComplete
          v-model="data['upload.extensions']"
          name="tags"
          :typeahead="false"
          :suggestions="[]"
          multiple
          placeholder="jpg, png, gif .."
          :aria-label="$t('settings.mediax.types')"
        />
      </set-row>
      <set-row :label="$t('settings.media.maxFileSize')" :help="$t('settings.mediax.maxHelp')" label-for="upload-max-size">
        <PvInputNumber
          v-model="data['upload.max_file_size']"
          input-id="upload-max-size"
          name="upload.max_file_size"
          :min="0"
          :max="10485760"
          :use-grouping="false"
          suffix=" KB"
        />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.mediax.storage')">
      <set-row :label="$t('settings.media.provider')">
        <PvSelect
          v-model="data['upload.provider']"
          name="upload.provider"
          :options="providers"
          option-label="label"
          option-value="value"
          :aria-label="$t('settings.media.provider')"
        />
      </set-row>

      <template v-if="data['upload.provider'] === 'filesystem'">
        <set-row :label="$t('settings.media.upload.path')" :help="$t('settings.mediax.pathHelp')" label-for="upload-path">
          <PvInputText
            id="upload-path"
            v-model="data['upload.filesystem.upload_path']"
            name="app.upload_path"
            placeholder="/home/listnun/uploads"
            :maxlength="200"
            required
            class="ln-mono"
          />
        </set-row>
        <set-row :label="$t('settings.media.upload.uri')" :help="$t('settings.mediax.uriHelp')" label-for="upload-uri">
          <PvInputText
            id="upload-uri"
            v-model="data['upload.filesystem.upload_uri']"
            name="app.upload_uri"
            placeholder="/uploads"
            :maxlength="200"
            required
            pattern="^\/(.+?)"
            class="ln-mono"
          />
        </set-row>
      </template>

      <template v-else-if="data['upload.provider'] === 's3'">
        <set-row :label="$t('settings.media.s3.url')" :help="$t('settings.mediax.endpointHelp')" label-for="s3-url">
          <PvInputText
            id="s3-url"
            v-model="data['upload.s3.url']"
            name="upload.s3.url"
            required
            placeholder="https://s3.$region.amazonaws.com"
            :maxlength="200"
            type="url"
            pattern="https?://.*"
            class="ln-mono"
          />
        </set-row>
        <set-row :label="$t('settings.media.s3.region')" label-for="s3-region">
          <PvInputText
            id="s3-region"
            v-model="data['upload.s3.aws_default_region']"
            name="upload.s3.aws_default_region"
            :maxlength="200"
            placeholder="ap-south-1"
            class="ln-mono"
            @input="onS3URLChange"
          />
        </set-row>
        <set-row :label="$t('settings.media.s3.bucket')" label-for="s3-bucket">
          <PvInputText id="s3-bucket" v-model="data['upload.s3.bucket']" name="upload.s3.bucket" :maxlength="200" class="ln-mono" @input="onS3URLChange" />
        </set-row>
        <set-row :label="$t('settings.media.s3.key')" label-for="s3-key">
          <PvInputText id="s3-key" v-model="data['upload.s3.aws_access_key_id']" name="upload.s3.aws_access_key_id" :maxlength="200" class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.media.s3.secret')" :help="$t('settings.mediax.secretHelp')">
          <PvPassword
            v-model="data['upload.s3.aws_secret_access_key']"
            name="upload.s3.aws_secret_access_key"
            :feedback="false"
            :maxlength="200"
            :aria-label="$t('settings.media.s3.secret')"
          />
        </set-row>
        <set-row :label="$t('settings.media.s3.bucketType')" :help="$t('settings.mediax.bucketTypeHelp')">
          <PvSelect
            v-model="data['upload.s3.bucket_type']"
            name="upload.s3.bucket_type"
            :options="bucketTypes"
            option-label="label"
            option-value="value"
            :aria-label="$t('settings.media.s3.bucketType')"
          />
        </set-row>
      </template>
    </set-card>

    <set-card
      v-if="data['upload.provider'] === 's3'"
      :title="$t('settings.ws.advanced')"
      :help="$t('settings.mediax.advancedHelp')"
      collapsible
      :summary="$t('settings.mediax.advancedSummary')"
    >
      <set-row :label="$t('settings.media.s3.bucketPath')" :help="$t('settings.mediax.bucketPathHelp')" label-for="s3-path">
        <PvInputText id="s3-path" v-model="data['upload.s3.bucket_path']" name="upload.s3.bucket_path" :maxlength="200" placeholder="/" class="ln-mono" />
      </set-row>
      <set-row :label="$t('settings.mediax.publicURL')" :help="$t('settings.mediax.publicURLHelp')" label-for="s3-public">
        <PvInputText
          id="s3-public"
          v-model="data['upload.s3.public_url']"
          name="upload.s3.public_url"
          placeholder="https://files.yourdomain.com"
          :maxlength="200"
          pattern="(https?://.*|/.+)"
          class="ln-mono"
        />
      </set-row>
      <set-row :label="$t('settings.media.s3.uploadExpiry')" :help="$t('settings.mediax.expiryHelp')" label-for="s3-expiry">
        <PvInputText id="s3-expiry" v-model="data['upload.s3.expiry']" name="upload.s3.expiry" placeholder="14d" :pattern="regDuration" :maxlength="10" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { regDuration } from '../../constants';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

const props = defineProps<{ form?: any }>();
const { t } = useI18n();
const data = props.form;

const providers = computed(() => [
  { label: t('settings.media.providerFilesystem'), value: 'filesystem' },
  { label: t('settings.media.providerS3'), value: 's3' },
]);
const bucketTypes = computed(() => [
  { label: t('settings.media.s3.bucketTypePrivate'), value: 'private' },
  { label: t('settings.media.s3.bucketTypePublic'), value: 'public' },
]);

// Keeps an AWS endpoint in step with the region; any other endpoint is
// left alone.
function onS3URLChange() {
  if (data['upload.s3.url'] !== '' && !data['upload.s3.url'].match(/amazonaws\.com/)) {
    return;
  }
  data['upload.s3.url'] = `https://s3.${data['upload.s3.aws_default_region']}.amazonaws.com`;
}
</script>

<style scoped lang="scss">
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
