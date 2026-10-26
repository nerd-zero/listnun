<template>
  <div class="set-cards">
    <set-card :title="$t('settings.prv.records')">
      <!-- Shown as "track", the opposite of the stored disable_tracking,
           so the switch reads the same way as the rest. -->
      <set-row :label="$t('settings.prv.track')" :help="$t('settings.prv.trackHelp')" toggle>
        <PvToggleSwitch v-model="tracking" name="privacy.disable_tracking" :aria-label="$t('settings.prv.track')" />
      </set-row>
      <set-row v-if="tracking" :label="$t('settings.prv.perSub')" :help="$t('settings.prv.perSubHelp')" toggle>
        <PvToggleSwitch v-model="data['privacy.individual_tracking']" name="privacy.individual_tracking" :aria-label="$t('settings.prv.perSub')" />
      </set-row>
      <set-row :label="$t('settings.prv.ip')" :help="$t('settings.prv.ipHelp')" toggle>
        <PvToggleSwitch v-model="data['privacy.record_optin_ip']" name="privacy.record_optin_ip" :aria-label="$t('settings.prv.ip')" />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.prv.can')" :help="$t('settings.prv.canHelp')">
      <set-row :label="$t('settings.prv.oneClick')" :help="$t('settings.prv.oneClickHelp')" toggle>
        <PvToggleSwitch v-model="data['privacy.unsubscribe_header']" name="privacy.unsubscribe_header" :aria-label="$t('settings.prv.oneClick')" />
      </set-row>
      <set-row :label="$t('settings.prv.prefs')" toggle>
        <PvToggleSwitch v-model="data['privacy.allow_preferences']" name="privacy.allow_preferences" :aria-label="$t('settings.prv.prefs')" />
      </set-row>
      <set-row :label="$t('settings.prv.block')" toggle>
        <PvToggleSwitch v-model="data['privacy.allow_blocklist']" name="privacy.allow_blocklist" :aria-label="$t('settings.prv.block')" />
      </set-row>
      <set-row :label="$t('settings.prv.export')" toggle>
        <PvToggleSwitch v-model="data['privacy.allow_export']" name="privacy.allow_export" :aria-label="$t('settings.prv.export')" />
      </set-row>
      <set-row :label="$t('settings.prv.wipe')" toggle>
        <PvToggleSwitch v-model="data['privacy.allow_wipe']" name="privacy.allow_wipe" :aria-label="$t('settings.prv.wipe')" />
      </set-row>
    </set-card>

    <set-card :title="$t('settings.prv.who')" :help="$t('settings.prv.whoHelp')">
      <set-row :label="$t('settings.prv.never')" :help="$t('settings.prv.neverHelp')" label-for="prv-block" wide>
        <PvTextarea id="prv-block" v-model="data['privacy.domain_blocklist']" name="privacy.domain_blocklist" rows="3" class="ln-mono" placeholder="mailinator.com" />
      </set-row>
      <set-row :label="$t('settings.prv.only')" :help="$t('settings.prv.onlyHelp')" label-for="prv-allow" wide>
        <PvTextarea id="prv-allow" v-model="data['privacy.domain_allowlist']" name="privacy.domain_allowlist" rows="2" class="ln-mono" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

// The settings object is the server's flat, dotted-key map.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

const props = defineProps<{ form?: Loose }>();
const data = props.form as Loose;

const tracking = computed({
  get: () => !data['privacy.disable_tracking'],
  set: (on: boolean) => { data['privacy.disable_tracking'] = !on; },
});
</script>
