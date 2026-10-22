<template>
  <div>
    <TransitionGroup name="ln-list" tag="div" class="smtp-list">
      <div class="smtp-card" v-for="(item, n) in form.smtp" :key="n">
        <!-- Card header -->
        <div class="smtp-card-header">
          <div class="flex items-center gap-2">
            <PvToggleSwitch v-model="item.enabled" name="enabled" data-cy="btn-enable-smtp" :aria-label="$t('globals.buttons.enabled')" />
            <span class="smtp-card-title">{{ item.name || `SMTP #${Number(n) + 1}` }}</span>
            <span v-if="item.host" class="smtp-card-host">{{ item.host }}:{{ item.port }}</span>
            <span v-if="!item.enabled" class="smtp-card-off">{{ $t('settings.smtp.off') }}</span>
          </div>
          <div class="flex items-center gap-2">
            <a v-if="form.smtp.length > 1" href="#" class="delete-link"
              @click.prevent="$utils.confirm(null, () => removeSMTP(Number(n)))" data-cy="btn-delete-smtp">
              <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
            </a>
            <button type="button" class="smtp-card-toggle" :aria-expanded="isOpen(Number(n))" @click="toggle(Number(n))">
              {{ isOpen(Number(n)) ? $t('settings.smtp.collapse') : $t('settings.smtp.expand') }}
            </button>
          </div>
        </div>

        <div v-show="isOpen(Number(n))" :class="{ disabled: !item.enabled }">
          <!-- Host / Port -->
          <div class="smtp-section">
            <div class="grid">
              <div class="col-9">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.host') }}</label>
                  <PvInputText v-model="item.host" name="host" placeholder="smtp.yourmailserver.net"
                    :maxlength="200" class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.hostHelp') }}</small>
                </div>
              </div>
              <div class="col-3">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.port') }}</label>
                  <PvInputNumber v-model="item.port" name="port" placeholder="25" :min="1" :max="65535" class="w-full" />
                </div>
              </div>
            </div>

            <div class="quick-links">
              <a href="#" @click.prevent="() => fillSettings(n, 'gmail')">Gmail</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'ses')">Amazon SES</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'azure')">Azure ACS</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'mailgun')">Mailgun</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'mailjet')">Mailjet</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'sendgrid')">Sendgrid</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'postmark')">Postmark</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'forwardemail')">Forward Email</a>
              <a href="#" @click.prevent="() => fillSettings(n, 'lettermint')">Lettermint</a>
            </div>
          </div>

          <!-- Authentication -->
          <div class="smtp-section">
            <p class="smtp-section-label">{{ $t('settings.mailserver.authProtocol') }}</p>
            <div class="grid">
              <div class="col-3">
                <div class="field">
                  <PvSelect v-model="item.auth_protocol" name="auth_protocol"
                    :options="[{ label: 'LOGIN', value: 'login' }, { label: 'CRAM', value: 'cram' }, { label: 'PLAIN', value: 'plain' }, { label: 'None', value: 'none' }]"
                    option-label="label" option-value="value" class="w-full" />
                </div>
              </div>
              <div class="col-4">
                <div class="field">
                  <PvInputText v-model="item.username" :class="`smtp-username-${n}`"
                    :disabled="item.auth_protocol === 'none'" name="username"
                    :placeholder="$t('settings.mailserver.username')" :maxlength="200" class="w-full" />
                </div>
              </div>
              <div class="col-5">
                <div class="field">
                  <PvPassword v-model="item.password" :disabled="item.auth_protocol === 'none'" name="password"
                    :input-class="`password-${n}`"
                    :placeholder="$t('settings.mailserver.passwordHelp')" :maxlength="200" :feedback="false"
                    class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.passwordHelp') }}</small>
                </div>
              </div>
            </div>
          </div>

          <!-- TLS -->
          <div class="smtp-section">
            <p class="smtp-section-label">TLS</p>
            <div class="grid">
              <div class="col-4">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.smtp.heloHost') }}</label>
                  <PvInputText v-model="item.hello_hostname" name="hello_hostname" placeholder="" :maxlength="200"
                    class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.smtp.heloHostHelp') }}</small>
                </div>
              </div>
              <div class="col-4">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.tls') }}</label>
                  <PvSelect v-model="item.tls_type" name="items.tls_type"
                    :options="[{ label: $t('globals.states.off'), value: 'none' }, { label: 'STARTTLS', value: 'STARTTLS' }, { label: 'SSL/TLS', value: 'TLS' }]"
                    option-label="label" option-value="value" class="w-full" />
                </div>
              </div>
              <div class="col-4">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium invisible">-</label>
                  <div class="flex items-center gap-2 mt-1">
                    <PvToggleSwitch v-model="item.tls_skip_verify" :disabled="item.tls_type === 'none'"
                      name="item.tls_skip_verify" />
                    <span class="text-sm">{{ $t('settings.mailserver.skipTLS') }}</span>
                  </div>
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.skipTLSHelp') }}</small>
                </div>
              </div>
            </div>
          </div>

          <!-- Limits -->
          <div class="smtp-section">
            <p class="smtp-section-label">{{ $t('settings.mailserver.maxConns') }} &amp; Timeouts</p>
            <div class="grid">
              <div class="col-3">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.maxConns') }}</label>
                  <PvInputNumber v-model="item.max_conns" name="max_conns" placeholder="25" :min="1" :max="65535"
                    class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.maxConnsHelp') }}</small>
                </div>
              </div>
              <div class="col-3">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.smtp.retries') }}</label>
                  <PvInputNumber v-model="item.max_msg_retries" name="max_msg_retries" placeholder="2" :min="1"
                    :max="1000" class="w-full" />
                </div>
              </div>
              <div class="col-2">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.smtp.retryDelay') }}</label>
                  <PvInputText v-model="item.msg_retry_delay" name="msg_retry_delay" placeholder="0s"
                    :pattern="regDuration" :maxlength="10" class="w-full" />
                </div>
              </div>
              <div class="col-2">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.idleTimeout') }}</label>
                  <PvInputText v-model="item.idle_timeout" name="idle_timeout" placeholder="15s"
                    :pattern="regDuration" :maxlength="10" class="w-full" />
                </div>
              </div>
              <div class="col-2">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.mailserver.waitTimeout') }}</label>
                  <PvInputText v-model="item.wait_timeout" name="wait_timeout" placeholder="5s"
                    :pattern="regDuration" :maxlength="10" class="w-full" />
                </div>
              </div>
            </div>
          </div>

          <!-- Identity -->
          <div class="smtp-section">
            <p class="smtp-section-label">{{ $t('globals.fields.name') }} &amp; Addresses</p>
            <div class="grid">
              <div class="col-5">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('globals.fields.name') }}</label>
                  <PvInputText v-model="item.name" name="name" placeholder="email-primary" :maxlength="100"
                    class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.mailserver.nameHelp') }}</small>
                </div>
              </div>
              <div class="col-7">
                <div class="field">
                  <label class="block mb-1 text-sm font-medium">{{ $t('settings.smtp.fromAddresses') }}</label>
                  <PvAutoComplete v-model="item.from_addresses" name="from_addresses" multiple :typeahead="false"
                    :placeholder="'user@example.com, anothersite.com'" class="w-full" />
                  <small class="block mt-1 text-color-secondary">{{ $t('settings.smtp.fromAddressesHelp') }}</small>
                </div>
              </div>
            </div>

            <div class="field">
              <p v-if="item.email_headers.length === 0 && !item.showHeaders">
                <a href="#" class="settings-link" @click.prevent="() => showSMTPHeaders(n)">
                  <i class="pi pi-plus" /> {{ $t('settings.smtp.setCustomHeaders') }}
                </a>
              </p>
              <div v-if="item.email_headers.length > 0 || item.showHeaders">
                <PvTextarea v-model="item.strEmailHeaders" name="email_headers" class="w-full"
                  placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}, {&quot;X-Custom2&quot;: &quot;value&quot;}]" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.smtp.customHeadersHelp') }}</small>
              </div>
            </div>
          </div>

          <!-- Test -->
          <div class="smtp-section smtp-section--test">
            <form @submit.prevent="() => doSMTPTest(item, n)">
              <div class="smtp-test-row">
                <template v-if="smtpTestItem === n">
                  <div class="smtp-test-from">
                    <span class="text-sm font-medium">{{ $t('settings.general.fromEmail') }}</span>
                    <span class="text-sm text-color-secondary">{{ settings['app.from_email'] }}</span>
                  </div>
                  <div class="field" style="flex:1">
                    <label class="block mb-1 text-sm font-medium">{{ $t('settings.smtp.toEmail') }}</label>
                    <PvInputText type="email" required v-model="testEmail"
                      placeholder="email@site.com" :class="`test-email-${n}`" class="w-full" />
                  </div>
                  <PvButton severity="primary" type="submit" :label="$t('settings.smtp.sendTest')" />
                </template>
                <a href="#" v-else class="settings-link" @click.prevent="showTestForm(n)">
                  <i class="pi pi-send" /> {{ $t('settings.smtp.testConnection') }}
                </a>
              </div>
              <div v-if="errMsg && smtpTestItem === n" class="field mt-3">
                <PvTextarea v-model="errMsg" class="w-full text-red-500 text-sm" readonly />
              </div>
            </form>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <PvButton @click="addSMTP" icon="pi pi-plus" severity="primary" :label="$t('globals.buttons.addNew')" />
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';
import { regDuration } from '../../constants';
import { getSettings as settingsApi } from '../../api/generated/endpoints/settings/settings';

const smtpTemplates: Record<string, any> = {
  gmail: {
    host: 'smtp.gmail.com', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  ses: {
    host: 'email-smtp.YOUR-REGION.amazonaws.com', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  azure: {
    host: 'smtp.azurecomm.net', port: 587, auth_protocol: 'login', tls_type: 'STARTTLS',
  },
  mailjet: {
    host: 'in-v3.mailjet.com', port: 465, auth_protocol: 'cram', tls_type: 'TLS',
  },
  mailgun: {
    host: 'smtp.mailgun.org', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  sendgrid: {
    host: 'smtp.sendgrid.net', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  forwardemail: {
    host: 'smtp.forwardemail.net', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
  postmark: {
    host: 'smtp.postmarkapp.com', port: 587, auth_protocol: 'cram', tls_type: 'STARTTLS',
  },
  lettermint: {
    host: 'smtp.lettermint.co', port: 465, auth_protocol: 'login', tls_type: 'TLS',
  },
};

const props = defineProps<{ form?: any }>();
const { $utils } = useGlobal();
const { testSmtpSettings } = settingsApi();
const { t } = useI18n();
const { settings } = storeToRefs(useMainStore());

const data = props.form;

// Server cards fold to their header; the first starts open.
const collapsed = ref<Record<number, boolean>>({});
function isOpen(n: number): boolean {
  return collapsed.value[n] === undefined ? n === 0 : !collapsed.value[n];
}
function toggle(n: number) {
  collapsed.value = { ...collapsed.value, [n]: isOpen(n) };
}
const smtpTestItem = ref<number | null>(null);
const testEmail = ref('');
const errMsg = ref('');

function addSMTP() {
  data.smtp.push({
    name: '',
    enabled: true,
    host: '',
    hello_hostname: '',
    port: 587,
    auth_protocol: 'none',
    username: '',
    password: '',
    email_headers: [],
    from_addresses: [],
    max_conns: 10,
    max_msg_retries: 2,
    msg_retry_delay: '0s',
    idle_timeout: '15s',
    wait_timeout: '5s',
    tls_type: 'STARTTLS',
    tls_skip_verify: false,
  });
  nextTick(() => {
    const items = document.querySelectorAll('.smtp-list input[name="host"]');
    (items[items.length - 1] as HTMLInputElement).focus();
  });
}

function removeSMTP(i: number) { data.smtp.splice(i, 1); }

function showSMTPHeaders(i: number | string) {
  const n = Number(i);
  const s = data.smtp[n];
  s.showHeaders = true;
  data.smtp.splice(n, 1, s);
}

function isTestEnabled(item: any) {
  if (!item.host || !item.port) return false;
  if (item.auth_protocol !== 'none' && item.password.includes('•')) return false;
  return true;
}

function doSMTPTest(item: any, n: number | string) {
  const idx = Number(n);
  if (!isTestEnabled(item)) {
    $utils.toast(t('settings.smtp.testEnterEmail'), 'is-danger');
    nextTick(() => {
      const i = document.querySelector(`.password-${idx}`) as HTMLInputElement;
      data.smtp[idx].password = '';
      i.focus();
      i.select();
    });
    return;
  }
  errMsg.value = '';
  testSmtpSettings({ ...item, email: testEmail.value }).then(() => {
    $utils.toast(t('campaigns.testSent'));
  }).catch((err: any) => {
    if (err.response?.data?.message) { errMsg.value = err.response.data.message; }
  });
}

function showTestForm(n: number | string) {
  smtpTestItem.value = Number(n);
  errMsg.value = '';
  nextTick(() => { (document.querySelector(`.test-email-${n}`) as HTMLInputElement)?.focus(); });
}

function fillSettings(n: number | string, key: string) {
  const idx = Number(n);
  data.smtp.splice(idx, 1, {
    ...data.smtp[idx],
    ...smtpTemplates[key],
    username: '',
    password: '',
    hello_hostname: '',
    tls_skip_verify: false,
  });
  nextTick(() => { (document.querySelector(`.smtp-username-${idx}`) as HTMLInputElement)?.focus(); });
}
</script>

<style scoped lang="scss">
.smtp-list { display: flex; flex-direction: column; gap: 1.25rem; margin-bottom: 1.25rem; }

.smtp-card {
  background: var(--ln-card);
  border: 1px solid var(--ln-border);
  border-radius: 10px;
  overflow: hidden;
}

.smtp-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.85rem 1.25rem;
  background: var(--ln-muted);
  border-bottom: 1px solid var(--ln-border);
}

.smtp-card-title { font-weight: 600; font-size: 0.95rem; color: var(--ln-text); }
.smtp-card-host { font-family: var(--ln-mono); font-size: 12px; color: var(--ln-text-3); }
.smtp-card-off { font-size: 12px; color: var(--ln-text-3); }
.smtp-card-toggle { border: 0; background: none; font: inherit; font-size: 13px; font-weight: 500; color: var(--ln-text); cursor: pointer; padding: 4px 8px; border-radius: 6px; &:hover { background: var(--ln-hover); } }

.smtp-section {
  padding: 1rem 1.25rem;
  & + & { border-top: 1px solid var(--ln-border); }
  &--test { background: var(--ln-muted); }
}

.smtp-section-label {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--ln-text-3);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: 0.75rem;
}

.delete-link {
  font-size: 0.8rem;
  color: var(--p-red-500);
  text-decoration: none;
  &:hover { text-decoration: underline; }
}

.smtp-test-row {
  display: flex;
  align-items: flex-end;
  gap: 1rem;
  flex-wrap: wrap;
}

.smtp-test-from {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

:deep(.p-password) { width: 100%; }
:deep(.p-password-input) { width: 100%; }
</style>
