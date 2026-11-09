<template>
  <div class="set-cards">
    <!-- Every server, then the one being edited below it. -->
    <section class="ln-card smtp-servers">
      <div v-for="(item, n) in data.smtp" :key="n" class="smtp-server" :class="{ 'is-editing': editing === Number(n) }">
        <PvToggleSwitch v-model="item.enabled" name="enabled" data-cy="btn-enable-smtp" :aria-label="$t('settings.smtpx.serverOn', { name: serverName(item, n) })" />
        <span class="smtp-server-name">
          <strong>{{ serverName(item, n) }}</strong>
          <span class="ln-mono ln-small">{{ item.host ? `${item.host}:${item.port}` : $t('settings.smtpx.noHost') }}</span>
        </span>
        <span v-if="!item.enabled" class="smtp-server-off">{{ $t('settings.smtp.off') }}</span>
        <span class="smtp-server-actions">
          <PvButton :label="$t('settings.smtpx.test')" size="small" severity="secondary" outlined @click="openTest(Number(n))" />
          <PvButton
            :label="editing === Number(n) ? $t('settings.smtpx.editing') : $t('globals.buttons.edit')"
            size="small"
            :severity="editing === Number(n) ? undefined : 'secondary'"
            :text="editing !== Number(n)"
            @click="editing = Number(n)"
          />
        </span>
      </div>
      <button type="button" class="smtp-add" @click="addSMTP">
        <i class="pi pi-plus" aria-hidden="true" /> {{ $t('settings.smtpx.add') }}
      </button>
    </section>

    <template v-if="current">
      <h3 class="smtp-editing">{{ $t('settings.smtpx.editingName', { name: serverName(current, editing) }) }}</h3>

      <set-card :title="$t('settings.smtpx.connection')">
        <set-row :label="$t('settings.mailserver.host')" label-for="smtp-host">
          <template #help>
            <span class="smtp-presets">
              {{ $t('settings.smtpx.fillFor') }}
              <a v-for="p in PRESETS" :key="p.key" href="#" @click.prevent="fillSettings(editing, p.key)">{{ p.label }}</a>
            </span>
          </template>
          <PvInputText id="smtp-host" v-model="current.host" name="host" placeholder="smtp.yourmailserver.net" :maxlength="200" class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.mailserver.port')" label-for="smtp-port">
          <PvInputNumber v-model="current.port" input-id="smtp-port" name="port" placeholder="587" :min="1" :max="65535" :use-grouping="false" />
        </set-row>
        <set-row :label="$t('settings.smtpx.encryption')" :help="encryptionHelp">
          <PvSelect
            v-model="current.tls_type"
            name="tls_type"
            :options="[{ label: $t('globals.states.off'), value: 'none' }, { label: 'STARTTLS', value: 'STARTTLS' }, { label: 'SSL/TLS', value: 'TLS' }]"
            option-label="label"
            option-value="value"
            :aria-label="$t('settings.smtpx.encryption')"
          />
        </set-row>
      </set-card>

      <set-card :title="$t('settings.smtpx.signIn')">
        <set-row :label="$t('settings.smtpx.method')">
          <PvSelect
            v-model="current.auth_protocol"
            name="auth_protocol"
            :options="[{ label: 'Login', value: 'login' }, { label: 'CRAM', value: 'cram' }, { label: 'Plain', value: 'plain' }, { label: $t('settings.smtpx.noSignIn'), value: 'none' }]"
            option-label="label"
            option-value="value"
            :aria-label="$t('settings.smtpx.method')"
          />
        </set-row>
        <template v-if="current.auth_protocol !== 'none'">
          <set-row :label="$t('settings.mailserver.username')" :label-for="`smtp-user-${editing}`">
            <PvInputText :id="`smtp-user-${editing}`" v-model="current.username" :class="`smtp-username-${editing}`" name="username" :maxlength="200" />
          </set-row>
          <set-row :label="$t('settings.smtpx.password')" :help="$t('settings.mailserver.passwordHelp')">
            <PvPassword v-model="current.password" name="password" :input-class="`password-${editing}`" :maxlength="200" :feedback="false" :aria-label="$t('settings.smtpx.password')" />
          </set-row>
        </template>
      </set-card>

      <set-card :title="$t('settings.smtpx.sendTest')" :help="$t('settings.smtpx.sendTestHelp')">
        <form class="smtp-test" @submit.prevent="doSMTPTest(current, editing)">
          <PvInputText v-model="testEmail" type="email" required placeholder="you@yoursite.com" :class="`test-email-${editing}`" :aria-label="$t('settings.smtp.toEmail')" />
          <PvButton type="submit" :label="$t('settings.smtpx.sendTestButton')" severity="secondary" outlined />
        </form>
        <p v-if="errMsg" class="smtp-error ln-small" role="alert">{{ errMsg }}</p>
      </set-card>

      <set-card :title="$t('settings.ws.advanced')" :help="$t('settings.smtpx.advancedHelp')" collapsible :summary="advancedSummary">
        <set-row :label="$t('globals.fields.name')" :help="$t('settings.smtpx.nameHelp')" label-for="smtp-name">
          <PvInputText id="smtp-name" v-model="current.name" name="name" placeholder="email-primary" :maxlength="100" />
        </set-row>
        <set-row :label="$t('settings.smtp.fromAddresses')" :help="$t('settings.smtpx.fromHelp')">
          <PvAutoComplete v-model="current.from_addresses" name="from_addresses" multiple :typeahead="false" placeholder="user@example.com" :aria-label="$t('settings.smtp.fromAddresses')" />
        </set-row>
        <set-row :label="$t('settings.mailserver.maxConns')" :help="$t('settings.smtpx.maxConnsHelp')">
          <PvInputNumber v-model="current.max_conns" name="max_conns" :min="1" :max="65535" :aria-label="$t('settings.mailserver.maxConns')" />
        </set-row>
        <set-row :label="$t('settings.smtp.retries')" :help="$t('settings.smtpx.retriesHelp')">
          <PvInputNumber v-model="current.max_msg_retries" name="max_msg_retries" :min="1" :max="1000" :aria-label="$t('settings.smtp.retries')" />
        </set-row>
        <set-row :label="$t('settings.smtp.retryDelay')" label-for="smtp-retry-delay">
          <PvInputText id="smtp-retry-delay" v-model="current.msg_retry_delay" name="msg_retry_delay" placeholder="0s" :pattern="regDuration" :maxlength="10" />
        </set-row>
        <set-row :label="$t('settings.mailserver.idleTimeout')" :help="$t('settings.smtpx.idleHelp')" label-for="smtp-idle">
          <PvInputText id="smtp-idle" v-model="current.idle_timeout" name="idle_timeout" placeholder="15s" :pattern="regDuration" :maxlength="10" />
        </set-row>
        <set-row :label="$t('settings.mailserver.waitTimeout')" label-for="smtp-wait">
          <PvInputText id="smtp-wait" v-model="current.wait_timeout" name="wait_timeout" placeholder="5s" :pattern="regDuration" :maxlength="10" />
        </set-row>
        <set-row :label="$t('settings.smtp.heloHost')" :help="$t('settings.smtpx.heloHelp')" label-for="smtp-helo">
          <PvInputText id="smtp-helo" v-model="current.hello_hostname" name="hello_hostname" :maxlength="200" />
        </set-row>
        <set-row v-if="current.tls_type !== 'none'" :label="$t('settings.mailserver.skipTLS')" :help="$t('settings.smtpx.skipTlsHelp')" toggle>
          <PvToggleSwitch v-model="current.tls_skip_verify" name="tls_skip_verify" :aria-label="$t('settings.mailserver.skipTLS')" />
        </set-row>
        <set-row :label="$t('settings.smtpx.headers')" :help="$t('settings.smtp.customHeadersHelp')" wide>
          <PvTextarea
            v-model="current.strEmailHeaders"
            name="email_headers"
            rows="3"
            class="ln-mono"
            placeholder="[{&quot;X-Custom&quot;: &quot;value&quot;}]"
            :aria-label="$t('settings.smtpx.headers')"
          />
        </set-row>
      </set-card>

      <p v-if="data.smtp.length > 1">
        <button type="button" class="smtp-delete" data-cy="btn-delete-smtp" @click="$utils.confirm(null, () => removeSMTP(editing))">
          {{ $t('settings.smtpx.delete', { name: serverName(current, editing) }) }}
        </button>
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useGlobal } from '../../composables/useGlobal';
import { regDuration } from '../../constants';
import { getSettings as settingsApi } from '../../api/generated/endpoints/settings/settings';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

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

const data = props.form;

// Fill-in presets shown under Host, in this order.
const PRESETS = [
  { key: 'postmark', label: 'Postmark' }, { key: 'ses', label: 'Amazon SES' }, { key: 'gmail', label: 'Gmail' },
  { key: 'mailgun', label: 'Mailgun' }, { key: 'sendgrid', label: 'Sendgrid' }, { key: 'mailjet', label: 'Mailjet' },
  { key: 'azure', label: 'Azure ACS' }, { key: 'forwardemail', label: 'Forward Email' }, { key: 'lettermint', label: 'Lettermint' },
];

// The server whose settings show below the list.
const editing = ref(0);
const current = computed(() => data.smtp[editing.value] || null);

function serverName(item: any, n: number | string): string {
  return item.name || t('settings.smtpx.serverN', { n: Number(n) + 1 });
}

const encryptionHelp = computed(() => {
  const tls = current.value?.tls_type;
  if (tls === 'STARTTLS') return t('settings.smtpx.starttlsHelp');
  if (tls === 'TLS') return t('settings.smtpx.tlsHelp');
  return t('settings.smtpx.noTlsHelp');
});

// What the closed Advanced card says.
const advancedSummary = computed(() => {
  const c = current.value;
  if (!c) return '';
  return t('settings.smtpx.advancedSummary', { conns: c.max_conns, retries: c.max_msg_retries, idle: c.idle_timeout });
});
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
  editing.value = data.smtp.length - 1;
  nextTick(() => { (document.getElementById('smtp-host') as HTMLInputElement | null)?.focus(); });
}

function removeSMTP(i: number) {
  data.smtp.splice(i, 1);
  editing.value = Math.max(0, Math.min(editing.value, data.smtp.length - 1));
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

// openTest switches to a server and puts the cursor in its test field.
function openTest(n: number) {
  editing.value = n;
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
.smtp-servers { overflow: hidden; }
.smtp-server {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--ln-border);

  &.is-editing { background: var(--ln-muted); }
}
.smtp-server-name {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong { font-weight: 600; }
  .ln-mono { color: var(--ln-text-3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
}
.smtp-server-off {
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--ln-neutral-bg);
  color: var(--ln-text-2);
  font-size: 12px;
}
.smtp-server-actions { display: flex; gap: 8px; flex: none; }
.smtp-add {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin: 10px 14px;
  padding: 6px 8px;
  border: 0;
  border-radius: var(--ln-radius);
  background: transparent;
  color: var(--ln-text);
  font: inherit;
  font-weight: 500;
  cursor: pointer;

  i { font-size: 12px; }
  &:hover { background: var(--ln-hover); }
}
.smtp-editing { margin: 8px 0 0; font-size: 15px; font-weight: 600; }
.smtp-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  color: var(--ln-text-3);

  a { color: var(--ln-text-2); text-decoration: underline; }
}
.smtp-test { display: flex; gap: 8px; padding: 14px 20px; .p-inputtext { flex: 1; } }
.smtp-error { margin: 0 20px 14px; color: var(--ln-danger-fg); white-space: pre-wrap; }
.smtp-delete {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ln-danger-fg);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  &:hover { text-decoration: underline; }
}
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
