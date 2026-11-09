<template>
  <div class="set-cards">
    <set-card :title="$t('settings.bnc.when')" :help="$t('settings.bnc.whenHelp')">
      <set-row :label="$t('settings.bnc.process')" :help="$t('settings.bnc.processHelp')" toggle data-cy="btn-enable-bounce">
        <PvToggleSwitch v-model="data['bounce.enabled']" name="bounce.enabled" :aria-label="$t('settings.bnc.process')" />
      </set-row>
      <template v-if="data['bounce.enabled']">
        <set-row v-for="typ in bounceTypes" :key="typ" :label="$t(`settings.bnc.type.${typ}`)">
          <span class="bnc-action">
            <PvSelect
              v-model="data['bounce.actions'][typ]['action']"
              name="bounce.action"
              :options="bounceActionOptions"
              option-label="label"
              option-value="value"
              :aria-label="$t(`settings.bnc.type.${typ}`)"
            />
            <span class="bnc-after" data-cy="btn-bounce-count">
              {{ $t('settings.bnc.after') }}
              <PvInputNumber v-model="data['bounce.actions'][typ]['count']" name="bounce.count" :min="1" :max="1000" :aria-label="$t('settings.bounces.count')" />
            </span>
          </span>
        </set-row>
      </template>
    </set-card>

    <set-card v-if="data['bounce.enabled']" :title="$t('settings.bnc.sources')" :help="$t('settings.bnc.sourcesHelp')">
      <set-row label="Amazon SES" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-ses">
        <PvToggleSwitch v-model="data['bounce.ses_enabled']" name="ses_enabled" aria-label="Amazon SES" />
      </set-row>

      <set-row label="Postmark" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-postmark">
        <PvToggleSwitch v-model="data['bounce.postmark'].enabled" name="postmark_enabled" aria-label="Postmark" />
      </set-row>
      <template v-if="data['bounce.postmark'].enabled">
        <set-row :label="$t('settings.bounces.postmarkUsername')" :help="$t('settings.bnc.postmarkUserHelp')" class="bnc-sub">
          <PvInputText v-model="data['bounce.postmark'].username" name="postmark_username" :aria-label="$t('settings.bounces.postmarkUsername')" />
        </set-row>
        <set-row :label="$t('settings.bounces.postmarkPassword')" :help="$t('settings.chan.passwordHelp')" class="bnc-sub">
          <PvPassword v-model="data['bounce.postmark'].password" :feedback="false" name="postmark_password" :aria-label="$t('settings.bounces.postmarkPassword')" />
        </set-row>
      </template>

      <set-row label="SendGrid" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-sendgrid">
        <PvToggleSwitch v-model="data['bounce.sendgrid_enabled']" name="sendgrid_enabled" aria-label="SendGrid" />
      </set-row>
      <set-row v-if="data['bounce.sendgrid_enabled']" :label="$t('settings.bounces.sendgridKey')" :help="$t('settings.chan.passwordHelp')" class="bnc-sub">
        <PvPassword v-model="data['bounce.sendgrid_key']" :feedback="false" name="sendgrid_key" :aria-label="$t('settings.bounces.sendgridKey')" />
      </set-row>

      <set-row label="Azure Communication Services" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-azure">
        <PvToggleSwitch v-model="data['bounce.azure'].enabled" name="azure_enabled" aria-label="Azure" />
      </set-row>
      <template v-if="data['bounce.azure'].enabled">
        <set-row :label="$t('settings.bounces.azureSharedSecret')" :help="$t('settings.bounces.azureSharedSecretHelp')" class="bnc-sub">
          <PvPassword
            v-model="data['bounce.azure'].shared_secret"
            :feedback="false"
            name="azure_shared_secret"
            data-cy="bounce-azure-shared-secret"
            :aria-label="$t('settings.bounces.azureSharedSecret')"
          />
        </set-row>
        <set-row :label="$t('settings.bounces.azureSharedSecretHeader')" :help="$t('settings.bounces.azureSharedSecretHeaderHelp')" class="bnc-sub">
          <PvInputText
            v-model="data['bounce.azure'].shared_secret_header"
            name="azure_shared_secret_header"
            data-cy="bounce-azure-shared-secret-header"
            :aria-label="$t('settings.bounces.azureSharedSecretHeader')"
          />
        </set-row>
      </template>

      <set-row label="Forward Email" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-forwardemail">
        <PvToggleSwitch v-model="data['bounce.forwardemail'].enabled" name="forwardemail_enabled" aria-label="Forward Email" />
      </set-row>
      <set-row v-if="data['bounce.forwardemail'].enabled" :label="$t('settings.bounces.forwardemailKey')" :help="$t('settings.chan.passwordHelp')" class="bnc-sub">
        <PvPassword v-model="data['bounce.forwardemail'].key" :feedback="false" name="forwardemail_key" :aria-label="$t('settings.bounces.forwardemailKey')" />
      </set-row>

      <set-row label="Lettermint" :help="$t('settings.bnc.webhook')" toggle data-cy="btn-enable-bounce-lettermint">
        <PvToggleSwitch v-model="data['bounce.lettermint'].enabled" name="lettermint_enabled" aria-label="Lettermint" />
      </set-row>
      <set-row v-if="data['bounce.lettermint'].enabled" :label="$t('settings.bounces.lettermintKey')" :help="$t('settings.chan.passwordHelp')" class="bnc-sub">
        <PvPassword v-model="data['bounce.lettermint'].key" :feedback="false" name="lettermint_key" data-cy="bounce-lettermint-key" :aria-label="$t('settings.bounces.lettermintKey')" />
      </set-row>

      <set-row v-if="mailbox" :label="$t('settings.bnc.mailbox')" :help="$t('settings.bnc.mailboxHelp', { every: mailbox.scan_interval })" toggle data-cy="btn-enable-bounce-mailbox">
        <PvToggleSwitch v-model="mailbox.enabled" name="enabled" :aria-label="$t('settings.bnc.mailbox')" />
      </set-row>
    </set-card>

    <set-card
      v-if="data['bounce.enabled'] && mailbox?.enabled"
      :title="$t('settings.ws.advanced')"
      :help="$t('settings.bnc.advancedHelp')"
      collapsible
      :summary="mailbox.host ? `${mailbox.host}:${mailbox.port}` : $t('settings.bnc.advancedSummary')"
      :start-open="!mailbox.host"
    >
      <set-row :label="$t('settings.mailserver.host')" label-for="bnc-host">
        <PvInputText id="bnc-host" v-model="mailbox.host" name="host" placeholder="bounce.yourmailserver.net" :maxlength="200" class="ln-mono" />
      </set-row>
      <set-row :label="$t('settings.mailserver.port')">
        <PvInputNumber v-model="mailbox.port" name="port" :min="1" :max="65535" :use-grouping="false" :aria-label="$t('settings.mailserver.port')" />
      </set-row>
      <set-row :label="$t('settings.smtpx.method')">
        <PvSelect v-model="mailbox.auth_protocol" name="auth_protocol" :options="authOptions" option-label="label" option-value="value" :aria-label="$t('settings.smtpx.method')" />
      </set-row>
      <template v-if="mailbox.auth_protocol !== 'none'">
        <set-row :label="$t('settings.mailserver.username')" label-for="bnc-user">
          <PvInputText id="bnc-user" v-model="mailbox.username" name="username" :maxlength="200" />
        </set-row>
        <set-row :label="$t('settings.smtpx.password')" :help="$t('settings.chan.passwordHelp')">
          <PvPassword v-model="mailbox.password" name="password" :feedback="false" :maxlength="200" :aria-label="$t('settings.smtpx.password')" />
        </set-row>
      </template>
      <set-row :label="$t('settings.mailserver.tls')" toggle>
        <PvToggleSwitch v-model="mailbox.tls_enabled" name="tls_enabled" :aria-label="$t('settings.mailserver.tls')" />
      </set-row>
      <set-row v-if="mailbox.tls_enabled" :label="$t('settings.mailserver.skipTLS')" :help="$t('settings.smtpx.skipTlsHelp')" toggle>
        <PvToggleSwitch v-model="mailbox.tls_skip_verify" name="tls_skip_verify" :aria-label="$t('settings.mailserver.skipTLS')" />
      </set-row>
      <set-row :label="$t('settings.bnc.every')" :help="$t('settings.bnc.everyHelp')" label-for="bnc-every">
        <PvInputText id="bnc-every" v-model="mailbox.scan_interval" name="scan_interval" placeholder="15m" :pattern="regDuration" :maxlength="10" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { regDuration } from '@/constants.ts';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

const props = defineProps<{ form?: any }>();
const { t } = useI18n();
const data = props.form;
const bounceTypes = ['soft', 'hard', 'complaint'];

const bounceActionOptions = computed(() => [
  { label: t('settings.bnc.act.none'), value: 'none' },
  { label: t('settings.bnc.act.unsubscribe'), value: 'unsubscribe' },
  { label: t('settings.bnc.act.blocklist'), value: 'blocklist' },
  { label: t('settings.bnc.act.delete'), value: 'delete' },
]);

const mailbox = computed(() => (data['bounce.mailboxes'] || [])[0] || null);
const authOptions = computed(() => getAuthProtocolOptions(mailbox.value?.type || 'pop'));

// Webhook reports are accepted whenever a webhook provider is on; the
// design has no separate switch for it.
watch(() => [
  data['bounce.ses_enabled'], data['bounce.sendgrid_enabled'], data['bounce.postmark']?.enabled,
  data['bounce.azure']?.enabled, data['bounce.forwardemail']?.enabled, data['bounce.lettermint']?.enabled,
], (on) => {
  const any = on.some(Boolean);
  if (data['bounce.webhooks_enabled'] !== any) data['bounce.webhooks_enabled'] = any;
}, { immediate: true });

function getAuthProtocolOptions(type: string) {
  const opts: any[] = [{ label: t('settings.smtpx.noSignIn'), value: 'none' }];
  if (type === 'pop') {
    opts.push({ label: t('settings.bnc.userpass'), value: 'userpass' });
  } else {
    opts.push({ label: 'cram', value: 'cram' });
    opts.push({ label: 'plain', value: 'plain' });
    opts.push({ label: 'login', value: 'login' });
  }
  return opts;
}
</script>

<style scoped lang="scss">
.bnc-action { display: flex; align-items: center; gap: 10px; .p-select { flex: 1; min-width: 0; } }
.bnc-after {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--ln-text-2);

  :deep(.p-inputnumber) { width: 64px; }
  :deep(.p-inputtext) { width: 64px; }
}
// A provider's credentials, set in under its switch.
.bnc-sub { padding-left: 36px; background: var(--ln-muted); }
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
