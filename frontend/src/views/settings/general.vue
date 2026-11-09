<template>
  <div class="set-cards">
    <set-card :title="$t('settings.ws.organisation')">
      <set-row :label="$t('settings.ws.name')" :help="$t('settings.ws.nameHelp')" label-for="set-site-name">
        <PvInputText id="set-site-name" v-model="data['app.site_name']" name="app.site_name" :maxlength="300" required />
      </set-row>
      <set-row :label="$t('settings.ws.sender')" :help="$t('settings.ws.senderHelp')" label-for="set-from">
        <PvInputText
          id="set-from"
          v-model="data['app.from_email']"
          name="app.from_email"
          placeholder="Acme <news@acme.example>"
          pattern="((.+?)\s)?<(.+?)@(.+?)>"
          :maxlength="300"
        />
      </set-row>
      <!-- Set by the platform from the workspace's subdomain or custom
           domain, so it's shown, not edited, here. -->
      <set-row :label="$t('settings.ws.address')" :help="$t('settings.ws.addressHelp')">
        <PvInputText :model-value="data['app.root_url']" class="ln-mono" readonly :aria-label="$t('settings.ws.address')" />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.ws.notifications')">
      <set-row :label="$t('settings.ws.notify')" :help="$t('settings.ws.notifyHelp')">
        <PvAutoComplete
          v-model="data['app.notify_emails']"
          name="app.notify_emails"
          :typeahead="false"
          :suggestions="[]"
          multiple
          placeholder="you@yoursite.com"
          :aria-label="$t('settings.ws.notify')"
          @before-add="(v) => v.match(/(.+?)@(.+?)/)"
        />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.ws.publicPages')" :help="$t('settings.ws.publicPagesHelp')">
      <set-row :label="$t('settings.ws.formPage')" help="/subscription/form" toggle>
        <PvToggleSwitch v-model="data['app.enable_public_subscription_page']" name="app.enable_public_subscription_page" :aria-label="$t('settings.ws.formPage')" />
      </set-row>
      <set-row :label="$t('settings.ws.archive')" help="/archive" toggle>
        <PvToggleSwitch v-model="data['app.enable_public_archive']" name="app.enable_public_archive" :aria-label="$t('settings.ws.archive')" />
      </set-row>
      <set-row :label="$t('settings.ws.optinNow')" :help="$t('settings.ws.optinNowHelp')" toggle>
        <PvToggleSwitch v-model="data['app.send_optin_confirmation']" name="app.send_optin_confirmation" :aria-label="$t('settings.ws.optinNow')" />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.ws.advanced')" :help="$t('settings.ws.advancedHelp')" collapsible :summary="advancedSummary">
      <set-row :label="$t('settings.general.language')" label-for="set-lang">
        <PvSelect id="set-lang" v-model="data['app.lang']" name="app.lang" :options="langs" option-label="name" option-value="code" />
      </set-row>
      <set-row :label="$t('settings.ws.logo')" :help="$t('settings.ws.logoHelp')" label-for="set-logo">
        <PvInputText id="set-logo" v-model="data['app.logo_url']" name="app.logo_url" placeholder="https://…/logo.png" :maxlength="300" type="url" pattern="https?://.*" />
      </set-row>
      <set-row :label="$t('settings.ws.favicon')" label-for="set-favicon">
        <PvInputText id="set-favicon" v-model="data['app.favicon_url']" name="app.favicon_url" placeholder="https://…/favicon.png" :maxlength="300" type="url" pattern="https?://.*" />
      </set-row>
      <set-row :label="$t('settings.ws.optinPage')" :help="$t('settings.ws.optinPageHelp')" toggle>
        <PvToggleSwitch v-model="data['app.show_optin_page']" name="app.show_optin_page" :aria-label="$t('settings.ws.optinPage')" />
      </set-row>
      <set-row :label="$t('settings.ws.rssFull')" :help="$t('settings.ws.rssFullHelp')" toggle>
        <PvToggleSwitch v-model="data['app.enable_public_archive_rss_content']" name="app.enable_public_archive_rss_content" :aria-label="$t('settings.ws.rssFull')" />
      </set-row>
      <set-row :label="$t('settings.ws.updates')" toggle>
        <PvToggleSwitch v-model="data['app.check_updates']" name="app.check_updates" :aria-label="$t('settings.ws.updates')" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '@/store';
import SetCard from '@/components/ui/SetCard.vue';
import SetRow from '@/components/ui/SetRow.vue';

// The settings object is the server's flat, dotted-key map.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = defineProps<{ form?: Loose }>();
const data = props.form as Loose;
const { t } = useI18n();
const { serverConfig } = storeToRefs(useMainStore());
const langs = computed(() => ((serverConfig.value as Loose).langs || []) as Loose[]);

// What the closed Advanced card says: the language, and whether a custom
// logo is set.
const advancedSummary = computed(() => {
  const lang = langs.value.find((l) => l.code === data['app.lang'])?.name || data['app.lang'];
  return data['app.logo_url'] ? t('settings.ws.summaryLogo', { lang }) : t('settings.ws.summary', { lang });
});
</script>
