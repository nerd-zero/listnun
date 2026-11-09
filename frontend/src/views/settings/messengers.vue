<template>
  <div class="set-cards messengers">
    <p v-if="!data.messengers.length" class="ln-small">{{ $t('settings.chan.none') }}</p>

    <template v-for="(item, n) in data.messengers" :key="n">
      <set-card :title="item.name || $t('settings.chan.channelN', { n: Number(n) + 1 })" :help="$t('settings.chan.help')">
        <set-row :label="$t('settings.chan.on')" toggle>
          <PvToggleSwitch v-model="item.enabled" name="enabled" :aria-label="$t('settings.chan.on')" />
        </set-row>
        <set-row :label="$t('settings.chan.name')" :help="$t('settings.chan.nameHelp')" :label-for="`chan-name-${n}`">
          <PvInputText :id="`chan-name-${n}`" v-model="item.name" name="name" placeholder="sms" :maxlength="200" />
        </set-row>
        <set-row :label="$t('settings.chan.endpoint')" :help="$t('settings.chan.endpointHelp')" :label-for="`chan-url-${n}`">
          <PvInputText :id="`chan-url-${n}`" v-model="item.root_url" name="root_url" placeholder="https://sms.example.com/send" :maxlength="200" type="url" pattern="https?://.*" class="ln-mono" />
        </set-row>
        <set-row :label="$t('settings.messengers.username')" :label-for="`chan-user-${n}`">
          <PvInputText :id="`chan-user-${n}`" v-model="item.username" name="username" :maxlength="200" />
        </set-row>
        <set-row :label="$t('settings.smtpx.password')" :help="$t('settings.chan.passwordHelp')">
          <PvPassword v-model="item.password" name="password" :feedback="false" :maxlength="200" :aria-label="$t('settings.smtpx.password')" />
        </set-row>
      </set-card>

      <set-card :title="$t('settings.ws.advanced')" :help="$t('settings.chan.advancedHelp')" collapsible :summary="$t('settings.chan.advancedSummary')">
        <set-row :label="$t('settings.mailserver.maxConns')" :help="$t('settings.smtpx.maxConnsHelp')">
          <PvInputNumber v-model="item.max_conns" name="max_conns" :min="1" :max="65535" :aria-label="$t('settings.mailserver.maxConns')" />
        </set-row>
        <set-row :label="$t('settings.smtp.retries')" :help="$t('settings.smtpx.retriesHelp')">
          <PvInputNumber v-model="item.max_msg_retries" name="max_msg_retries" :min="1" :max="1000" :aria-label="$t('settings.smtp.retries')" />
        </set-row>
        <set-row :label="$t('settings.messengers.timeout')" :help="$t('settings.chan.timeoutHelp')" :label-for="`chan-timeout-${n}`">
          <PvInputText :id="`chan-timeout-${n}`" v-model="item.timeout" name="timeout" placeholder="5s" :pattern="regDuration" :maxlength="10" />
        </set-row>
      </set-card>

      <p class="chan-delete">
        <button type="button" @click="$utils.confirm(null, () => removeMessenger(Number(n)))">
          {{ $t('settings.smtpx.delete', { name: item.name || $t('settings.chan.channelN', { n: Number(n) + 1 }) }) }}
        </button>
      </p>
    </template>

    <p>
      <PvButton icon="pi pi-plus" :label="$t('settings.chan.add')" severity="secondary" outlined @click="addMessenger" />
    </p>
  </div>
</template>

<script setup lang="ts">
import { nextTick } from 'vue';
import { regDuration } from '../../constants';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

const props = defineProps<{ form?: any }>();
const data = props.form;

function addMessenger() {
  data.messengers.push({
    enabled: true,
    root_url: '',
    name: '',
    username: '',
    password: '',
    max_conns: 25,
    max_msg_retries: 2,
    timeout: '5s',
  });
  nextTick(() => {
    const items = document.querySelectorAll('.messengers input[name="name"]');
    (items[items.length - 1] as HTMLInputElement).focus();
  });
}

function removeMessenger(i: number) { data.messengers.splice(i, 1); }
</script>

<style scoped lang="scss">
.chan-delete {
  margin: -4px 0 8px;

  button {
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
}
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
