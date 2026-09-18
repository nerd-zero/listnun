<template>
  <div class="items">
    <div class="grid">
      <div class="col-4">
        <div class="field">
          <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.provider') }}</label>
          <PvSelect v-model="data['upload.provider']" name="upload.provider"
            :options="[{ label: 'filesystem', value: 'filesystem' }, { label: 's3', value: 's3' }]"
            option-label="label" option-value="value" class="w-full" />
        </div>
      </div>
      <div class="col-8">
        <div class="field">
          <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.upload.extensions') }}</label>
          <PvAutoComplete v-model="data['upload.extensions']" name="tags" :typeahead="false"
            :suggestions="[]" multiple placeholder="jpg, png, gif .." class="w-full" />
        </div>
      </div>
    </div>

    <hr class="m-0" />

    <div v-if="data['upload.provider'] === 'filesystem'" class="flex flex-col gap-4">
      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.upload.path') }}</label>
        <PvInputText v-model="data['upload.filesystem.upload_path']" name="app.upload_path"
          placeholder="/home/listnun/uploads" :maxlength="200" required class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.upload.pathHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.upload.uri') }}</label>
        <PvInputText v-model="data['upload.filesystem.upload_uri']" name="app.upload_uri" placeholder="/uploads"
          :maxlength="200" required pattern="^\/(.+?)" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.upload.uriHelp') }}</small>
      </div>
    </div>

    <div v-if="data['upload.provider'] === 's3'" class="flex flex-col gap-4">
      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.region') }}</label>
        <PvInputText v-model="data['upload.s3.aws_default_region']" @input="onS3URLChange"
          name="upload.s3.aws_default_region" :maxlength="200" placeholder="ap-south-1" class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.key') }}</label>
        <PvInputText v-model="data['upload.s3.aws_access_key_id']" name="upload.s3.aws_access_key_id"
          :maxlength="200" class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.secret') }}</label>
        <PvPassword v-model="data['upload.s3.aws_secret_access_key']" name="upload.s3.aws_secret_access_key"
          :feedback="false" :maxlength="200" class="w-full" />
        <small class="block mt-1 text-color-secondary">Enter a value to change.</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.bucketType') }}</label>
        <PvSelect v-model="data['upload.s3.bucket_type']" name="upload.s3.bucket_type"
          :options="[{ label: $t('settings.media.s3.bucketTypePrivate'), value: 'private' }, { label: $t('settings.media.s3.bucketTypePublic'), value: 'public' }]"
          option-label="label" option-value="value" class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.bucket') }}</label>
        <PvInputText v-model="data['upload.s3.bucket']" @input="onS3URLChange" name="upload.s3.bucket"
          :maxlength="200" class="w-full" />
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.bucketPath') }}</label>
        <PvInputText v-model="data['upload.s3.bucket_path']" name="upload.s3.bucket_path" :maxlength="200"
          placeholder="/" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.s3.bucketPathHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.uploadExpiry') }}</label>
        <PvInputText v-model="data['upload.s3.expiry']" name="upload.s3.expiry" placeholder="14d"
          :pattern="regDuration" :maxlength="10" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.s3.uploadExpiryHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.url') }}</label>
        <PvInputText v-model="data['upload.s3.url']" name="upload.s3.url" required
          placeholder="https://s3.$region.amazonaws.com" :maxlength="200" type="url" pattern="https?://.*"
          class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.s3.urlHelp') }}</small>
      </div>

      <div class="field">
        <label class="block mb-1 text-sm font-medium">{{ $t('settings.media.s3.publicURL') }}</label>
        <PvInputText v-model="data['upload.s3.public_url']" name="upload.s3.public_url"
          placeholder="https://files.yourdomain.com" :maxlength="200" pattern="(https?://.*|/.+)" class="w-full" />
        <small class="block mt-1 text-color-secondary">{{ $t('settings.media.s3.publicURLHelp') }}</small>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { regDuration } from '../../constants';

const props = defineProps<{ form?: any }>();
const data = props.form;

function onS3URLChange() {
  if (data['upload.s3.url'] !== '' && !data['upload.s3.url'].match(/amazonaws\.com/)) {
    return;
  }
  data['upload.s3.url'] = `https://s3.${data['upload.s3.aws_default_region']}.amazonaws.com`;
}
</script>

<style scoped>
:deep(.p-password) { width: 100%; }
:deep(.p-password-input) { width: 100%; }
</style>
