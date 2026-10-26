<template>
  <!-- Each area is a way in: what it covers and where to change it. A
       status shows only when something needs attention, so a healthy
       workspace reads as a calm index rather than a wall of values. -->
  <div class="set-ov ln-enter">
    <section v-for="c in cards" :key="c.key" class="ln-card set-ov-card" :data-cy="`settings-card-${c.key}`">
      <div class="set-ov-head">
        <span class="set-ov-icon" aria-hidden="true"><i :class="`pi ${c.icon}`" /></span>
        <span class="set-ov-title">
          <h3>{{ c.title }}</h3>
          <span class="ln-small">{{ c.text }}</span>
        </span>
        <span v-if="c.status" class="set-ov-status" :class="`is-${c.status.tone}`" data-cy="settings-status">{{ c.status.label }}</span>
      </div>
      <span class="set-ov-links">
        <router-link v-for="l in c.links" :key="l.key" :to="l.to">{{ l.label }}</router-link>
      </span>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import axios from 'axios';
import dayjs from 'dayjs';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import type { RouteLocationRaw } from 'vue-router';
import { useMainStore } from '../../store';
import { useGlobal } from '../../composables/useGlobal';

// The settings object is the server's flat, dotted-key map.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

interface Link { key: string; label: string; to: RouteLocationRaw }
interface Status { label: string; tone: 'warn' | 'danger' }
interface Card {
  key: string; title: string; text: string; icon: string; links: Link[]; status?: Status;
}

// How many recent log lines to scan for today's errors.
const LOG_SCAN = 500;

const props = defineProps<{ form: Loose | null }>();

const { t } = useI18n();
const { $can } = useGlobal();
const { serverConfig } = storeToRefs(useMainStore());
const errorsToday = ref(0);

const sec = (key: string): Link => ({ key, label: t(`settings.nav.${key}`), to: { name: 'settings', params: { section: key } } });

const cards = computed<Card[]>(() => {
  const smtp = (props.form?.smtp || []) as Loose[];
  let sending: Status | undefined;
  if (smtp.length && !smtp.some((s) => s.enabled)) sending = { label: t('settings.status.noServer'), tone: 'danger' };
  else if ((serverConfig.value as Loose).needs_restart) sending = { label: t('settings.status.restart'), tone: 'warn' };

  const data: Link[] = [];
  if ($can('settings:maintain')) data.push({ key: 'maintenance', label: t('settings.nav.maintenance'), to: { name: 'maintenance' } });
  data.push({ key: 'logs', label: t('settings.nav.logs'), to: { name: 'settingsLogs' } });

  return [
    {
      key: 'sending',
      title: t('settings.group.sending'),
      text: t('settings.card.sending'),
      icon: 'pi-send',
      links: [sec('smtp'), sec('performance'), sec('messengers')],
      status: sending,
    },
    {
      key: 'deliverability',
      title: t('settings.group.deliverability'),
      text: t('settings.card.deliverability'),
      icon: 'pi-inbox',
      links: [sec('bounces'), sec('inactive'), sec('scrub')],
    },
    {
      key: 'privacy',
      title: t('settings.nav.privacy'),
      text: t('settings.card.privacy'),
      icon: 'pi-user',
      links: [sec('privacy')],
    },
    {
      key: 'security',
      title: t('settings.nav.security'),
      text: t('settings.card.security'),
      icon: 'pi-lock',
      links: [sec('security')],
    },
    {
      key: 'media',
      title: t('settings.nav.media'),
      text: t('settings.card.media'),
      icon: 'pi-box',
      links: [sec('media')],
    },
    {
      key: 'workspace',
      title: t('settings.card.workspaceTitle'),
      text: t('settings.card.workspace'),
      icon: 'pi-pencil',
      links: [sec('general'), sec('appearance')],
    },
    {
      key: 'data',
      title: t('settings.card.dataTitle'),
      text: t('settings.card.data'),
      icon: 'pi-file',
      links: data,
      status: errorsToday.value ? { label: t('settings.status.errors', { n: errorsToday.value }, errorsToday.value), tone: 'danger' } : undefined,
    },
  ];
});

onMounted(() => {
  axios.get('/api/settings/logs', { params: { limit: LOG_SCAN } }).then((r) => {
    const since = dayjs().subtract(1, 'day');
    const rows = (r.data?.data || []) as Loose[];
    errorsToday.value = rows.filter((x) => x.level === 'error' && dayjs(x.created_at).isAfter(since)).length;
  }).catch(() => {});
});
</script>

<style scoped lang="scss">
.set-ov {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;

  @media (max-width: 991px) { grid-template-columns: minmax(0, 1fr); }
}
.set-ov-card { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.set-ov-head { display: flex; gap: 12px; align-items: flex-start; }
.set-ov-icon {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: var(--ln-radius);
  background: var(--ln-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--ln-text-2);

  i { font-size: 14px; }
}
.set-ov-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;

  h3 { margin: 0; font-size: 15px; font-weight: 600; }
}
.set-ov-status {
  flex: none;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;

  &.is-warn { background: var(--ln-warn-bg); color: var(--ln-warn-fg); }
  &.is-danger { background: var(--ln-danger-bg); color: var(--ln-danger-fg); }
}
.set-ov-links {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  padding-left: 44px;

  a { font-size: 13px; color: var(--ln-text); text-decoration: underline; }
}
</style>
