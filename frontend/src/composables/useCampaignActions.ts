import { computed, reactive, ref } from 'vue';
import dayjs from 'dayjs';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import type { MenuItem } from 'primevue/menuitem';
import { useGlobal } from './useGlobal';
import { getCampaigns as campaignsApi } from '../api/generated/endpoints/campaigns/campaigns';

// CampaignRef is what the actions need of a campaign: the list's rows and
// the overview's items both have it.
export interface CampaignRef {
  id: number;
  name: string;
  status: string;
  sendAt?: string | null;
  toSend?: number;
  sent?: number;
}

// RowAction is the one action a row or card shows as a button.
export interface RowAction { label: string; icon: string; cy: string; run: () => void }

export type CampaignStatus = 'running' | 'paused' | 'scheduled' | 'draft' | 'cancelled';

interface PopupMenu { toggle: (e: Event) => void }

// The full campaign, as clone copies it.
type FullCampaign = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// useCampaignActions is what a campaign row or card can do -- start,
// schedule, pause, resume, cancel, preview, clone, delete, open analytics
// -- with the confirm and clone dialogs they go through. The Campaigns
// list and the overview share it, so both offer the same actions.
// onChanged runs after a campaign changes status, is deleted or cloned.
export function useCampaignActions(onChanged: () => void) {
  const { $utils, $can } = useGlobal();
  const { t } = useI18n();
  const router = useRouter();
  const {
    getCampaign, createCampaign, deleteCampaign, updateCampaignStatus,
  } = campaignsApi();

  const previewItem = ref<CampaignRef | null>(null);
  const confirm = reactive({
    open: false, title: '', text: '', label: '', typed: '', danger: true, busy: false, run: () => {},
  });
  const clone = reactive({
    open: false, busy: false, name: '', source: null as CampaignRef | null,
  });
  const menu = ref<PopupMenu | null>(null);
  const menuFor = ref<CampaignRef | null>(null);

  const canSend = () => $can('campaigns:send');
  const canManage = () => $can('campaigns:manage');
  const canAnalyse = () => $can('campaigns:get_analytics');

  function edit(c: CampaignRef) {
    router.push({ name: 'campaign', params: { id: c.id } });
  }

  function analytics(c: CampaignRef) {
    router.push({ name: 'campaignAnalytics', query: { id: String(c.id) } });
  }

  function preview(c: CampaignRef) {
    previewItem.value = c;
  }

  function setStatus(c: CampaignRef, status: CampaignStatus) {
    confirm.busy = true;
    updateCampaignStatus(c.id, { status } as never)
      .then(() => {
        confirm.open = false;
        $utils.toast(t('campaigns.statusChanged', { name: c.name, status: t(`campaigns.status.${status}`) }));
        onChanged();
      })
      .finally(() => { confirm.busy = false; });
  }

  // askStatus confirms a status change, naming the campaign and saying
  // what happens. Resuming a paused campaign says how many are still
  // waiting.
  function askStatus(c: CampaignRef, status: CampaignStatus) {
    const resume = status === 'running' && c.status === 'paused';
    const left = Math.max(0, (c.toSend || 0) - (c.sent || 0));
    Object.assign(confirm, {
      open: true,
      title: resume ? t('campaigns.overview.resumeTitle', { name: c.name }) : t(`campaigns.confirm.${status}.title`, { name: c.name }),
      text: resume
        ? t('campaigns.overview.resumeText', { n: $utils.niceNumber(left) })
        : t(`campaigns.confirm.${status}.text`, { n: (c.toSend || 0).toLocaleString() }),
      label: resume ? t('campaigns.overview.resume') : t(`campaigns.confirm.${status}.action`),
      typed: '',
      danger: status === 'cancelled',
      run: () => setStatus(c, status),
    });
  }

  function askDelete(c: CampaignRef) {
    Object.assign(confirm, {
      open: true,
      title: t('campaigns.confirm.delete.title', { name: c.name }),
      text: t('campaigns.confirm.delete.text'),
      label: t('globals.buttons.delete'),
      typed: '',
      danger: true,
      run: () => {
        confirm.busy = true;
        deleteCampaign(c.id)
          .then(() => {
            confirm.open = false;
            $utils.toast(t('globals.messages.deleted', { name: c.name }));
            onChanged();
          })
          .finally(() => { confirm.busy = false; });
      },
    });
  }

  function askClone(c: CampaignRef) {
    clone.source = c;
    clone.name = t('campaigns.copyOf', { name: c.name });
    clone.open = true;
  }

  // doClone copies the full campaign (rows and cards carry only a summary)
  // into a new draft and opens it.
  async function doClone() {
    const src = clone.source;
    if (!src) return;
    clone.busy = true;
    try {
      const c = await getCampaign(src.id) as unknown as FullCampaign;
      const now = dayjs();
      const sendLater = !!c.sendAt;
      let sendAt = null;
      if (sendLater) sendAt = dayjs(c.sendAt).isAfter(now) ? c.sendAt : now.add(7, 'day');
      const name = clone.name.trim();
      const data: Record<string, unknown> = {
        name,
        subject: c.subject,
        lists: (c.lists || []).map((l: { id: number }) => l.id),
        type: c.type,
        from_email: c.fromEmail,
        content_type: c.contentType,
        messenger: c.messenger,
        tags: c.tags,
        template_id: c.templateId,
        body: c.body,
        body_source: c.bodySource,
        altbody: c.altbody,
        headers: c.headers,
        tracking: c.tracking,
        send_later: sendLater,
        send_at: sendAt,
        archive: c.archive,
        archive_template_id: c.archiveTemplateId,
        archive_meta: c.archiveMeta,
        media: (c.media || []).map((m: { id: number }) => m.id),
      };
      if (c.archive) data.archive_slug = `${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
      const created = await createCampaign(data as never) as unknown as FullCampaign;
      clone.open = false;
      onChanged();
      router.push({ name: 'campaign', params: { id: created.id } });
    } finally {
      clone.busy = false;
    }
  }

  // primary is the one action a row shows directly, by status.
  function primary(c: CampaignRef): RowAction | null {
    if (c.status === 'running' && canSend()) {
      return {
        label: t('campaigns.pause'), icon: 'pi-pause', cy: 'btn-pause', run: () => askStatus(c, 'paused'),
      };
    }
    if (c.status === 'paused' && canSend()) {
      return {
        label: t('campaigns.send'), icon: 'pi-play', cy: 'btn-resume', run: () => askStatus(c, 'running'),
      };
    }
    if ((c.status === 'finished' || c.status === 'cancelled') && canAnalyse()) {
      return {
        label: t('globals.terms.analytics'), icon: 'pi-chart-bar', cy: 'btn-analytics', run: () => analytics(c),
      };
    }
    return {
      label: t('campaigns.preview'), icon: 'pi-eye', cy: 'btn-preview', run: () => preview(c),
    };
  }

  // sendAction is a draft's start or schedule action, when it has lists
  // and the user may send.
  function sendAction(c: CampaignRef): RowAction | null {
    if (c.status !== 'draft' || !canSend()) return null;
    return c.sendAt
      ? {
        label: t('campaigns.schedule'), icon: 'pi-clock', cy: 'btn-schedule', run: () => askStatus(c, 'scheduled'),
      }
      : {
        label: t('campaigns.start'), icon: 'pi-send', cy: 'btn-start', run: () => askStatus(c, 'running'),
      };
  }

  function itemsFor(c: CampaignRef): MenuItem[] {
    const send = canSend();
    const items: MenuItem[] = [
      { label: t('globals.buttons.edit'), icon: 'pi pi-pencil', command: () => edit(c) },
      { label: t('campaigns.preview'), icon: 'pi pi-eye', command: () => preview(c) },
    ];
    const start = sendAction(c);
    if (start) items.push({ label: start.label, icon: `pi ${start.icon}`, command: start.run });
    if (send && c.status === 'scheduled') {
      items.push({ label: t('campaigns.unschedule'), icon: 'pi pi-clock', command: () => askStatus(c, 'draft') });
    }
    if (canAnalyse() && c.status !== 'draft') {
      items.push({ label: t('globals.terms.analytics'), icon: 'pi pi-chart-bar', command: () => analytics(c) });
    }
    if (send && (c.status === 'running' || c.status === 'paused')) {
      items.push({ separator: true }, {
        label: t('globals.buttons.cancel'), icon: 'pi pi-times-circle', class: 'is-danger', command: () => askStatus(c, 'cancelled'),
      });
    }
    if (canManage()) {
      items.push({ separator: true }, {
        label: t('globals.buttons.delete'), icon: 'pi pi-trash', class: 'is-danger', command: () => askDelete(c),
      });
    }
    return items;
  }

  const menuItems = computed<MenuItem[]>(() => (menuFor.value ? itemsFor(menuFor.value) : []));

  function openMenu(e: Event, c: CampaignRef) {
    menuFor.value = c;
    menu.value?.toggle(e);
  }

  return {
    previewItem,
    confirm,
    clone,
    menu,
    menuItems,
    openMenu,
    edit,
    analytics,
    preview,
    askStatus,
    askDelete,
    askClone,
    doClone,
    primary,
    sendAction,
    itemsFor,
  };
}
