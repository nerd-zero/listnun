<template>
  <!-- Custom code is rare, so each area starts closed and says whether
       anything is set. -->
  <div class="set-cards">
    <set-card
      v-for="a in areas"
      :key="a.key"
      :title="a.title"
      :help="a.help"
      collapsible
      :summary="a.summary"
    >
      <set-row :label="$t('settings.appearance.customCSS')" wide>
        <code-editor v-model="data[`appearance.${a.key}.custom_css`]" lang="css" name="body" :key="`editor-${a.key}-css`" />
      </set-row>
      <set-row :label="$t('settings.appearance.customJS')" wide>
        <code-editor v-model="data[`appearance.${a.key}.custom_js`]" lang="javascript" name="body" :key="`editor-${a.key}-js`" />
      </set-row>
    </set-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import CodeEditor from '../../components/CodeEditor.vue';
import SetCard from '../../components/ui/SetCard.vue';
import SetRow from '../../components/ui/SetRow.vue';

const props = defineProps<{ form?: any }>();
const { t } = useI18n();
const data = props.form;

const has = (k: string) => !!(data[k] || '').trim();

const areas = computed(() => (['public', 'admin'] as const).map((key) => {
  const css = has(`appearance.${key}.custom_css`);
  const js = has(`appearance.${key}.custom_js`);
  let summary = t('settings.look.none');
  if (css && js) summary = t('settings.look.both');
  else if (css) summary = t('settings.look.css');
  else if (js) summary = t('settings.look.js');
  return {
    key,
    title: t(`settings.look.${key}`),
    help: t(`settings.look.${key}Help`),
    summary,
  };
}));
</script>
