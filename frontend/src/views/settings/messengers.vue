<template>
  <div>
    <TransitionGroup name="ln-list" tag="div" class="items messengers">
      <div class="settings-card" v-for="(item, n) in data.messengers" :key="n">
        <div class="settings-card-header">
          <div class="flex items-center gap-2">
            <PvToggleSwitch v-model="item.enabled" name="enabled" />
            <span class="font-medium">{{ item.name || `Messenger #${n + 1}` }}</span>
          </div>
          <a href="#" class="delete-link" @click.prevent="$utils.confirm(null, () => removeMessenger(n))">
            <i class="pi pi-trash" /> {{ $t('globals.buttons.delete') }}
          </a>
        </div>

        <div class="flex flex-col gap-4" :class="{ disabled: !item.enabled }">
          <div class="field">
            <label class="block mb-1 text-sm font-medium">{{ $t('globals.fields.name') }}</label>
            <PvInputText v-model="item.name" name="name" placeholder="mymessenger" :maxlength="200" class="w-full" />
            <small class="block mt-1 text-color-secondary">{{ $t('settings.messengers.nameHelp') }}</small>
          </div>

          <div class="field">
            <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.url') }}</label>
            <PvInputText v-model="item.root_url" name="root_url" placeholder="https://postback.messenger.net/path"
              :maxlength="200" type="url" :pattern="'https?://.*'" class="w-full" />
            <small class="block mt-1 text-color-secondary">{{ $t('settings.messengers.urlHelp') }}</small>
          </div>

          <div class="field">
            <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.username') }}</label>
            <PvInputText v-model="item.username" name="username" :maxlength="200" class="w-full" />
          </div>

          <div class="field">
            <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.password') }}</label>
            <PvPassword v-model="item.password" name="password" :feedback="false"
              :placeholder="$t('globals.messages.passwordChange')" :maxlength="200" class="w-full" />
            <small class="block mt-1 text-color-secondary">{{ $t('globals.messages.passwordChange') }}</small>
          </div>

          <div class="grid">
            <div class="col-4">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.maxConns') }}</label>
                <PvInputNumber v-model="item.max_conns" name="max_conns" placeholder="25" :min="1" :max="65535"
                  class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.messengers.maxConnsHelp') }}</small>
              </div>
            </div>
            <div class="col-4">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.retries') }}</label>
                <PvInputNumber v-model="item.max_msg_retries" name="max_msg_retries" placeholder="2" :min="1"
                  :max="1000" class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.messengers.retriesHelp') }}</small>
              </div>
            </div>
            <div class="col-4">
              <div class="field">
                <label class="block mb-1 text-sm font-medium">{{ $t('settings.messengers.timeout') }}</label>
                <PvInputText v-model="item.timeout" name="timeout" placeholder="5s" :pattern="regDuration"
                  :maxlength="10" class="w-full" />
                <small class="block mt-1 text-color-secondary">{{ $t('settings.messengers.timeoutHelp') }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
    </TransitionGroup>

    <PvButton @click="addMessenger" icon="pi pi-plus" severity="primary" :label="$t('globals.buttons.addNew')" />
  </div>
</template>

<script setup lang="ts">
import { nextTick } from 'vue';
import { regDuration } from '../../constants';

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
.settings-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.delete-link {
  font-size: 0.8rem;
  color: var(--p-red-500);
  text-decoration: none;
  &:hover { text-decoration: underline; }
}

:deep(.p-password) { width: 100%; }
:deep(.p-password-input) { width: 100%; }
</style>
