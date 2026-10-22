<template>
  <div class="items">
    <div class="grid">
      <div class="col-3">
        <div class="field" data-cy="btn-enable-bounce">
          <div class="flex items-center gap-2">
            <PvToggleSwitch v-model="data['bounce.enabled']" name="bounce.enabled" />
            <span>{{ $t('settings.bounces.enable') }}</span>
          </div>
        </div>
      </div>
      <div class="col-9">
        <div v-for="typ in bounceTypes" :key="typ" class="grid mb-2">
          <div class="col-2" :class="{ disabled: !data['bounce.enabled'] }">
            <span class="text-sm font-medium">{{ $t(`bounces.${typ}`) }}</span>
          </div>
          <div class="col-5" :class="{ disabled: !data['bounce.enabled'] }">
            <div class="field" data-cy="btn-bounce-count">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.count') }}</label>
              <PvInputNumber v-model="data['bounce.actions'][typ]['count']" name="bounce.count"
                placeholder="3" :min="1" :max="1000" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('settings.bounces.countHelp') }}</small>
            </div>
          </div>
          <div class="col-5" :class="{ disabled: !data['bounce.enabled'] }">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.action') }}</label>
              <PvSelect v-model="data['bounce.actions'][typ]['action']" name="bounce.action"
                :options="bounceActionOptions" option-label="label" option-value="value" class="w-full" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <hr />

    <div class="field" data-cy="btn-enable-bounce-webhook">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-model="data['bounce.webhooks_enabled']" :disabled="!data['bounce.enabled']"
          name="webhooks_enabled" />
        <span>{{ $t('settings.bounces.enableWebhooks') }}</span>
      </div>
      <a href="https://listmonk.app/docs/bounces" target="_blank" rel="noopener noreferer" class="settings-link mt-1 block">
        {{ $t('globals.buttons.learnMore') }} &rarr;
      </a>
    </div>

    <ln-collapse :open="data['bounce.webhooks_enabled']">
      <div class="settings-card">
        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.ses_enabled']" name="ses_enabled" data-cy="btn-enable-bounce-ses" />
                <span>{{ $t('settings.bounces.enableSES') }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.azure'].enabled" name="azure_enabled" data-cy="btn-enable-bounce-azure" />
                <span>{{ $t('settings.bounces.enableAzure') }}</span>
              </div>
            </div>
          </div>
          <div class="col-4">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.azureSharedSecret') }}</label>
              <PvPassword v-model="data['bounce.azure'].shared_secret" :feedback="false"
                :disabled="!data['bounce.azure'].enabled" name="azure_shared_secret"
                data-cy="bounce-azure-shared-secret" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('settings.bounces.azureSharedSecretHelp') }}</small>
            </div>
          </div>
          <div class="col-5">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.azureSharedSecretHeader') }}</label>
              <PvInputText v-model="data['bounce.azure'].shared_secret_header"
                :disabled="!data['bounce.azure'].enabled" name="azure_shared_secret_header"
                data-cy="bounce-azure-shared-secret-header" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('settings.bounces.azureSharedSecretHeaderHelp') }}</small>
            </div>
          </div>
        </div>

        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.sendgrid_enabled']" name="sendgrid_enabled" data-cy="btn-enable-bounce-sendgrid" />
                <span>{{ $t('settings.bounces.enableSendgrid') }}</span>
              </div>
            </div>
          </div>
          <div class="col-9">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.sendgridKey') }}</label>
              <PvPassword v-model="data['bounce.sendgrid_key']" :feedback="false"
                :disabled="!data['bounce.sendgrid_enabled']" name="sendgrid_enabled"
                data-cy="btn-enable-bounce-sendgrid" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('globals.messages.passwordChange') }}</small>
            </div>
          </div>
        </div>

        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.postmark'].enabled" name="postmark_enabled" data-cy="btn-enable-bounce-postmark" />
                <span>{{ $t('settings.bounces.enablePostmark') }}</span>
              </div>
            </div>
          </div>
          <div class="col-4">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.postmarkUsername') }}</label>
              <PvInputText v-model="data['bounce.postmark'].username"
                :disabled="!data['bounce.postmark'].enabled" name="postmark_username"
                data-cy="btn-enable-bounce-postmark" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('settings.bounces.postmarkUsernameHelp') }}</small>
            </div>
          </div>
          <div class="col-5">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.postmarkPassword') }}</label>
              <PvPassword v-model="data['bounce.postmark'].password" :feedback="false"
                :disabled="!data['bounce.postmark'].enabled" name="postmark_password"
                data-cy="btn-enable-bounce-postmark" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('globals.messages.passwordChange') }}</small>
            </div>
          </div>
        </div>

        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.forwardemail'].enabled" name="forwardemail_enabled" data-cy="btn-enable-bounce-forwardemail" />
                <span>{{ $t('settings.bounces.enableForwardemail') }}</span>
              </div>
            </div>
          </div>
          <div class="col-9">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.forwardemailKey') }}</label>
              <PvPassword v-model="data['bounce.forwardemail'].key" :feedback="false"
                :disabled="!data['bounce.forwardemail'].enabled" name="forwardemail_enabled"
                data-cy="btn-enable-bounce-forwardemail" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('globals.messages.passwordChange') }}</small>
            </div>
          </div>
        </div>

        <div class="grid">
          <div class="col-3">
            <div class="field">
              <div class="flex items-center gap-2">
                <PvToggleSwitch v-model="data['bounce.lettermint'].enabled" name="lettermint_enabled" data-cy="btn-enable-bounce-lettermint" />
                <span>{{ $t('settings.bounces.enableLettermint') }}</span>
              </div>
            </div>
          </div>
          <div class="col-9">
            <div class="field">
              <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.lettermintKey') }}</label>
              <PvPassword v-model="data['bounce.lettermint'].key" :feedback="false"
                :disabled="!data['bounce.lettermint'].enabled" name="lettermint_key"
                data-cy="bounce-lettermint-key" class="w-full" />
              <small class="block mt-1 text-color-secondary">{{ $t('globals.messages.passwordChange') }}</small>
            </div>
          </div>
        </div>
      </div>
    </ln-collapse>

    <hr />

    <div class="field">
      <div class="flex items-center gap-2">
        <PvToggleSwitch v-if="data['bounce.mailboxes']" v-model="data['bounce.mailboxes'][0].enabled"
          :disabled="!data['bounce.enabled']" name="enabled" data-cy="btn-enable-bounce-mailbox" />
        <span v-if="data['bounce.mailboxes']">{{ $t('settings.bounces.enableMailbox') }}</span>
      </div>
    </div>

    <ln-collapse :open="data['bounce.enabled'] && data['bounce.mailboxes'][0].enabled">
      <div class="settings-card" v-for="(item, n) in data['bounce.mailboxes']" :key="n">
        <div :class="{ disabled: !item.enabled }">
          <div class="grid">
            <div class="col-3">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.type') }}</label>
                <PvSelect v-model="item.type" name="type"
                  :options="[{ label: 'POP', value: 'pop' }]" option-label="label" option-value="value"
                  class="w-full" />
              </div>
            </div>
            <div class="col-6">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.host') }}</label>
                <PvInputText v-model="item.host" name="host" placeholder="bounce.yourmailserver.net"
                  :maxlength="200" class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.hostHelp') }}</small>
              </div>
            </div>
            <div class="col-3">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.port') }}</label>
                <PvInputNumber v-model="item.port" name="port" placeholder="25" :min="1" :max="65535"
                  class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.portHelp') }}</small>
              </div>
            </div>
          </div>

          <div class="grid">
            <div class="col-3">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.authProtocol') }}</label>
                <PvSelect v-model="item.auth_protocol" name="auth_protocol"
                  :options="getAuthProtocolOptions(item.type)" option-label="label" option-value="value"
                  class="w-full" />
              </div>
            </div>
            <div class="col-4">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.username') }}</label>
                <PvInputText v-model="item.username" :disabled="item.auth_protocol === 'none'" name="username"
                  placeholder="mysmtp" :maxlength="200" class="w-full" />
              </div>
            </div>
            <div class="col-5">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.password') }}</label>
                <PvPassword v-model="item.password" :disabled="item.auth_protocol === 'none'" name="password"
                  :feedback="false" :placeholder="$t('settings.mailserver.passwordHelp')" :maxlength="200"
                  class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.passwordHelp') }}</small>
              </div>
            </div>
          </div>

          <div class="grid">
            <div class="col-3">
              <div class="field">
                <div class="flex items-center gap-2">
                  <PvToggleSwitch v-model="item.tls_enabled" name="item.tls_enabled" />
                  <span>{{ $t('settings.mailserver.tls') }}</span>
                </div>
                <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.tlsHelp') }}</small>
              </div>
            </div>
            <div class="col-3">
              <div class="field">
                <div class="flex items-center gap-2">
                  <PvToggleSwitch v-model="item.tls_skip_verify" :disabled="!item.tls_enabled"
                    name="item.tls_skip_verify" />
                  <span>{{ $t('settings.mailserver.skipTLS') }}</span>
                </div>
                <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.skipTLSHelp') }}</small>
              </div>
            </div>
            <div class="col-4 col-offset-2">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.bounces.scanInterval') }}</label>
                <PvInputText v-model="item.scan_interval" name="scan_interval" placeholder="15m"
                  :pattern="regDuration" :maxlength="10" class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.bounces.scanIntervalHelp') }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
    </ln-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { regDuration } from '@/constants.ts';
import LnCollapse from '../../components/ui/LnCollapse.vue';

const props = defineProps<{ form?: any }>();
const { t } = useI18n();
const data = props.form;
const bounceTypes = ['soft', 'hard', 'complaint'];

const bounceActionOptions = computed(() => [
  { label: t('globals.terms.none'), value: 'none' },
  { label: t('email.unsub'), value: 'unsubscribe' },
  { label: t('settings.bounces.blocklist'), value: 'blocklist' },
  { label: t('globals.buttons.delete'), value: 'delete' },
]);

function getAuthProtocolOptions(type: string) {
  const opts: any[] = [{ label: 'none', value: 'none' }];
  if (type === 'pop') {
    opts.push({ label: 'userpass', value: 'userpass' });
  } else {
    opts.push({ label: 'cram', value: 'cram' });
    opts.push({ label: 'plain', value: 'plain' });
    opts.push({ label: 'login', value: 'login' });
  }
  return opts;
}
</script>

<style scoped lang="scss">
:deep(.p-password) { width: 100%; }
:deep(.p-password-input) { width: 100%; }
</style>
