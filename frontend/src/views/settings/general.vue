<template>
  <div class="items">
    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.siteName') }}</label>
      <PvInputText v-model="data['app.site_name']" name="app.site_name" :maxlength="300" required class="w-full" />
    </div>

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.rootURL') }}</label>
      <PvInputText v-model="data['app.root_url']" name="app.root_url" placeholder="https://listnun.yoursite.com"
        :maxlength="300" required type="url" pattern="https?://.*" class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.rootURLHelp') }}</small>
    </div>

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.logoURL') }}</label>
      <PvInputText v-model="data['app.logo_url']" name="app.logo_url" placeholder="https://listnun.yoursite.com/logo.png"
        :maxlength="300" type="url" pattern="https?://.*" class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.logoURLHelp') }}</small>
    </div>

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.faviconURL') }}</label>
      <PvInputText v-model="data['app.favicon_url']" name="app.favicon_url"
        placeholder="https://listnun.yoursite.com/favicon.png" :maxlength="300" type="url" pattern="https?://.*"
        class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.faviconURLHelp') }}</small>
    </div>

    <hr />

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.fromEmail') }}</label>
      <PvInputText v-model="data['app.from_email']" name="app.from_email"
        placeholder="listnun <noreply@listnun.yoursite.com>" pattern="((.+?)\s)?<(.+?)@(.+?)>" :maxlength="300"
        class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.fromEmailHelp') }}</small>
    </div>

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.adminNotifEmails') }}</label>
      <PvAutoComplete v-model="data['app.notify_emails']" name="app.notify_emails" :typeahead="false"
        :suggestions="[]" multiple placeholder="you@yoursite.com"
        @before-add="(v) => v.match(/(.+?)@(.+?)/)" class="w-full" />
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.adminNotifEmailsHelp') }}</small>
    </div>

    <hr />

    <p class="settings-section-label">{{ $t('globals.terms.subscriptions', 2) }}</p>
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.enable_public_subscription_page']" name="app.enable_public_subscription_page" />
        <span>{{ $t('settings.general.enablePublicSubPage') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.enablePublicSubPageHelp') }}</small>
    </div>
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.send_optin_confirmation']" name="app.send_optin_confirmation" />
        <span>{{ $t('settings.general.sendOptinConfirm') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.sendOptinConfirmHelp') }}</small>
    </div>
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.show_optin_page']" name="app.show_optin_page" />
        <span>{{ $t('settings.general.showOptinPage') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.showOptinPageHelp') }}</small>
    </div>

    <hr />

    <p class="settings-section-label">{{ $t('campaigns.archive') }}</p>
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.enable_public_archive']" name="app.enable_public_archive" />
        <span>{{ $t('settings.general.enablePublicArchive') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.enablePublicArchiveHelp') }}</small>
    </div>
    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.enable_public_archive_rss_content']" name="app.enable_public_archive_rss_content" />
        <span>{{ $t('settings.general.enablePublicArchiveRSSContent') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.enablePublicArchiveRSSContentHelp') }}</small>
    </div>

    <hr />

    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['app.check_updates']" name="app.check_updates" />
        <span>{{ $t('settings.general.checkUpdates') }}</span>
      </div>
      <small class="block mt-1 text-color-secondary">{{ $t('settings.general.checkUpdatesHelp') }}</small>
    </div>

    <hr />

    <div class="field">
      <label class="block mb-1 text-sm font-medium">{{ $t('settings.general.language') }}</label>
      <PvSelect v-model="data['app.lang']" name="app.lang"
        :options="serverConfig.langs" option-label="name" option-value="code" class="w-full" />
      <p class="mt-2">
        <a href="https://listmonk.app/docs/i18n/#additional-language-packs" target="_blank" rel="noopener noreferer">
          {{ $t('globals.buttons.more') }} &rarr;
        </a>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useMainStore } from '@/store';

const props = defineProps<{ form?: any }>();
const data = props.form;
const { serverConfig } = storeToRefs(useMainStore());
</script>
