<template>
  <div class="ln-page profile">
    <header class="profile-head">
      <span class="profile-avatar" aria-hidden="true">
        <img v-if="data.avatar" :src="data.avatar" alt="">
        <template v-else>{{ initials }}</template>
      </span>
      <div>
        <h1 class="ln-page-header-title">{{ data.name || data.username }}</h1>
        <p class="ln-small">
          <template v-if="data.userRole">{{ data.userRole.name }}</template>
          <template v-if="data.userRole && orgName">&emsp;</template>
          {{ orgName }}
        </p>
      </div>
    </header>

    <PvSkeleton v-if="!loaded" height="240px" />

    <template v-else>
      <!-- Details -->
      <form class="ln-card profile-card" @submit.prevent="saveDetails">
        <h2 class="ln-card-head ln-card-title">{{ $t('users.details') }}</h2>
        <div class="profile-body">
          <div class="ln-fields-2">
            <div class="ln-field">
              <label for="profile-name">{{ $t('globals.fields.name') }}</label>
              <PvInputText id="profile-name" v-model="form.name" name="name" :maxlength="200" />
            </div>
            <div v-if="data.type !== 'api'" class="ln-field">
              <label for="profile-email">{{ $t('subscribers.email') }}</label>
              <PvInputText id="profile-email" v-model="form.email" name="email" type="email" :maxlength="200" :disabled="!data.passwordLogin" required />
              <p v-if="!data.passwordLogin" class="ln-help">{{ $t('users.emailFromSSO') }}</p>
            </div>
          </div>
          <div class="ln-fields-2">
            <div class="ln-field">
              <label for="profile-username">{{ $t('users.username') }}</label>
              <PvInputText id="profile-username" :model-value="data.username" class="ln-mono" disabled />
              <p class="ln-help">{{ $t('users.cantChange') }}</p>
            </div>
          </div>
        </div>
        <footer class="profile-foot">
          <PvButton type="submit" :label="$t('users.saveDetails')" :loading="busy === 'details'" :disabled="!detailsDirty" data-cy="btn-save" />
        </footer>
      </form>

      <!-- Password -->
      <form v-if="data.passwordLogin" class="ln-card profile-card" @submit.prevent="savePassword">
        <h2 class="ln-card-head ln-card-title">{{ $t('users.password') }}</h2>
        <div class="profile-body profile-pw">
          <div class="ln-field">
            <label for="profile-pw-current">{{ $t('users.currentPassword') }}</label>
            <PvPassword v-model="pw.current" input-id="profile-pw-current" :feedback="false" required autocomplete="current-password" />
          </div>
          <div class="ln-field">
            <label for="profile-pw-new">{{ $t('users.newPassword') }}</label>
            <PvPassword v-model="pw.next" input-id="profile-pw-new" :feedback="false" :minlength="8" :maxlength="200" required autocomplete="new-password" />
          </div>
          <div class="ln-field">
            <label for="profile-pw-confirm">{{ $t('users.confirmPassword') }}</label>
            <PvPassword v-model="pw.confirm" input-id="profile-pw-confirm" :feedback="false" required autocomplete="new-password" />
          </div>
          <p v-if="pwError" class="ln-field-error profile-pw-error" role="alert">{{ pwError }}</p>
        </div>
        <footer class="profile-foot">
          <PvButton type="submit" severity="secondary" outlined :label="$t('users.changePassword')" :loading="busy === 'password'" />
        </footer>
      </form>

      <!-- Two-step verification -->
      <section v-if="data.passwordLogin" class="ln-card profile-card">
        <div class="ln-card-head">
          <h2 class="ln-card-title">{{ $t('users.twoFA') }}</h2>
          <status-tag :status="twofaStatus" :label="twofaLabel" />
        </div>

        <div v-if="totpOn" class="profile-body profile-row">
          <p class="ln-small profile-grow">{{ $t('users.twoFAEnabledDesc', { type: 'TOTP' }) }}</p>
          <PvButton v-if="!disabling" :label="$t('users.turnOff')" severity="danger" outlined @click="startDisable" />
        </div>
        <form v-if="totpOn && disabling" class="profile-body profile-row" @submit.prevent="confirmDisable">
          <div class="ln-field profile-grow">
            <label for="profile-2fa-pw">{{ $t('users.confirmWithPassword') }}</label>
            <PvPassword ref="disablePwEl" v-model="disablePw" input-id="profile-2fa-pw" :feedback="false" required />
          </div>
          <PvButton type="submit" :label="$t('users.turnOff')" severity="danger" :loading="busy === '2fa'" />
          <PvButton type="button" :label="$t('globals.buttons.cancel')" text @click="resetTotp" />
        </form>

        <div v-if="!totpOn && !totpQR" class="profile-body profile-row">
          <p class="ln-small profile-grow">{{ $t('users.twoFAOffText') }}</p>
          <PvButton :label="$t('users.setUp2fa')" :loading="busy === '2fa'" @click="startSetup" />
        </div>
        <div v-if="!totpOn && totpQR" class="profile-body profile-setup">
          <img :src="`data:image/png;base64,${totpQR}`" :alt="$t('users.totpScanQR')" class="profile-qr">
          <form class="profile-steps" @submit.prevent="confirmSetup">
            <ol>
              <li>{{ $t('users.totpStep1') }}</li>
              <li>{{ $t('users.totpStep2') }} <code class="ln-mono profile-secret">{{ groupedSecret }}</code></li>
              <li>{{ $t('users.totpStep3') }}</li>
            </ol>
            <div class="profile-row">
              <PvInputText
                ref="codeEl"
                v-model="totpCode"
                class="ln-mono profile-code"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                pattern="[0-9]{6}"
                placeholder="000000"
                :aria-label="$t('users.totpCode')"
                required
              />
              <PvButton type="submit" :label="$t('users.turnOn')" :loading="busy === '2fa'" />
              <PvButton type="button" :label="$t('globals.buttons.cancel')" text @click="resetTotp" />
            </div>
          </form>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  ref, reactive, computed, nextTick, onMounted,
} from 'vue';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import { useMainStore } from '../store';
import { useGlobal } from '../composables/useGlobal';
import StatusTag from '../components/ui/StatusTag.vue';
import { getUsers as usersApi } from '../api/generated/endpoints/users/users';

// The profile is the API's camelCased user.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const { $utils } = useGlobal();
const {
  getUserProfile, updateUserProfile, generateTotpQr, enableTotp, disableTotp,
} = usersApi();
const { t } = useI18n();
const store = useMainStore();
const { serverConfig } = storeToRefs(store);

const data = ref<Loose>({});
const loaded = ref(false);
const busy = ref<'' | 'details' | 'password' | '2fa'>('');
const form = reactive({ name: '', email: '' });
const pw = reactive({ current: '', next: '', confirm: '' });
const pwError = ref('');
const totpQR = ref('');
const totpSecret = ref('');
const totpCode = ref('');
const disabling = ref(false);
const disablePw = ref('');
const codeEl = ref<Loose | null>(null);
const disablePwEl = ref<Loose | null>(null);

const orgName = computed(() => (serverConfig.value as Loose).organization_name || '');
const totpOn = computed(() => data.value.twofaType === 'totp');
const detailsDirty = computed(() => form.name !== (data.value.name || '') || form.email !== (data.value.email || ''));
const twofaStatus = computed(() => {
  if (totpOn.value) return 'active';
  return totpQR.value ? 'pending' : 'disabled';
});
const twofaLabel = computed(() => {
  if (totpOn.value) return t('users.on');
  return totpQR.value ? t('users.settingUp') : t('users.off');
});
const groupedSecret = computed(() => (totpSecret.value.match(/.{1,4}/g) || []).join(' '));
const initials = computed(() => {
  const src = String(data.value.name || data.value.username || '?').trim();
  const parts = src.split(/\s+/).filter(Boolean);
  return (parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : src.slice(0, 2)).toUpperCase();
});

function setProfile(p: Loose) {
  data.value = { ...p };
  form.name = p.name || '';
  form.email = p.email || '';
  store.setModelResponse({ model: 'profile', data: p });
}

function reload() {
  return getUserProfile().then((p) => { setProfile(p as unknown as Loose); });
}

function saveDetails() {
  busy.value = 'details';
  updateUserProfile({ name: form.name, email: form.email } as never).then((p) => {
    setProfile(p as unknown as Loose);
    $utils.toast(t('globals.messages.updated', { name: data.value.username }));
  }).finally(() => { busy.value = ''; });
}

function savePassword() {
  pwError.value = '';
  if (pw.next.length < 8) {
    pwError.value = t('users.passwordTooShort');
    return;
  }
  if (pw.next !== pw.confirm) {
    pwError.value = t('users.passwordMismatch');
    return;
  }
  busy.value = 'password';
  // Name and e-mail go as saved, so unsaved detail edits aren't applied here.
  updateUserProfile({
    name: data.value.name, email: data.value.email, password: pw.next, password2: pw.confirm, current_password: pw.current,
  } as never).then(() => {
    Object.assign(pw, { current: '', next: '', confirm: '' });
    $utils.toast(t('users.passwordChanged'));
  }).finally(() => { busy.value = ''; });
}

function resetTotp() {
  totpQR.value = '';
  totpSecret.value = '';
  totpCode.value = '';
  disabling.value = false;
  disablePw.value = '';
}

function startSetup() {
  busy.value = '2fa';
  generateTotpQr(data.value.id).then((d) => {
    const r = d as unknown as { qr: string; secret: string };
    totpQR.value = r.qr;
    totpSecret.value = r.secret;
    nextTick(() => { codeEl.value?.$el?.focus(); });
  }).finally(() => { busy.value = ''; });
}

function confirmSetup() {
  busy.value = '2fa';
  enableTotp(data.value.id, { secret: totpSecret.value, code: totpCode.value.trim() }).then(() => {
    resetTotp();
    $utils.toast(t('users.twoFAEnabled'));
    return reload();
  }).finally(() => { busy.value = ''; });
}

function startDisable() {
  disabling.value = true;
  nextTick(() => { disablePwEl.value?.$el?.querySelector('input')?.focus(); });
}

function confirmDisable() {
  busy.value = '2fa';
  disableTotp(data.value.id, { password: disablePw.value }).then(() => {
    resetTotp();
    $utils.toast(t('users.twoFADisabled'));
    return reload();
  }).finally(() => { busy.value = ''; });
}

onMounted(() => {
  reload().finally(() => { loaded.value = true; });
});
</script>

<style scoped lang="scss">
.profile { max-width: 720px; }
.profile-head {
  display: flex;
  align-items: center;
  gap: 16px;
  .ln-small { margin: 2px 0 0; }
  h1 { margin: 0; }
}
.profile-avatar {
  flex: none;
  display: inline-grid;
  place-items: center;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--ln-ink);
  color: var(--ln-on-ink);
  font-size: 18px;
  font-weight: 600;
  img { width: 100%; height: 100%; object-fit: cover; }
}
.profile-card { display: flex; flex-direction: column; overflow: hidden; }
h2.ln-card-head { margin: 0; }
.profile-body { padding: 18px; display: flex; flex-direction: column; gap: 16px; }
.profile-pw {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  @media (max-width: 720px) { grid-template-columns: minmax(0, 1fr); }
}
.profile-pw-error { grid-column: 1 / -1; margin: 0; }
.profile-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid var(--ln-border);
  background: var(--ln-muted);
}
.profile-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; flex-direction: row; }
.profile-row .ln-field { margin: 0; }
.profile-grow { flex: 1; min-width: 200px; margin: 0; }
.profile-setup { flex-direction: row; align-items: flex-start; flex-wrap: wrap; gap: 20px; }
.profile-qr {
  width: 148px;
  height: 148px;
  padding: 6px;
  border: 1px solid var(--ln-border);
  border-radius: 8px;
  background: #fff;
}
.profile-steps {
  flex: 1;
  min-width: 240px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  ol { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; font-size: 14px; color: var(--ln-text-2); }
}
.profile-secret { padding: 2px 6px; border-radius: 4px; background: var(--ln-muted); font-size: 12px; color: var(--ln-text); }
.profile-code { width: 140px; letter-spacing: .3em; font-size: 16px; }
:deep(.p-password), :deep(.p-password-input) { width: 100%; }
</style>
