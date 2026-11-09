import dayjs from 'dayjs';
import { useI18n } from 'vue-i18n';

const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;

// useTime formats timestamps the way the redesign shows them: "Just now",
// "4 min ago", "3 h ago", "Yesterday", then "19 Oct" (with the year once
// it isn't this year).
// eslint-disable-next-line import/prefer-default-export
export function useTime() {
  const { t } = useI18n();

  function shortDate(stamp: string | Date | null | undefined, withTime = false): string {
    if (!stamp) return '';
    const d = dayjs(stamp);
    const month = t(`globals.months.${d.month() + 1}`);
    let out = `${d.date()} ${month}`;
    if (d.year() !== dayjs().year()) out += ` ${d.year()}`;
    if (withTime) out += ` ${d.format('HH:mm')}`;
    return out;
  }

  function ago(stamp: string | Date | null | undefined): string {
    if (!stamp) return '';
    const diff = Date.now() - dayjs(stamp).valueOf();
    if (diff < MINUTE) return t('time.justNow');
    if (diff < HOUR) return t('time.minAgo', { n: Math.floor(diff / MINUTE) });
    if (diff < 24 * HOUR) return t('time.hAgo', { n: Math.floor(diff / HOUR) });
    if (dayjs(stamp).isSame(dayjs().subtract(1, 'day'), 'day')) return t('time.yesterday');
    return shortDate(stamp);
  }

  // timeLeft is the remaining time until stamp, e.g. "18 h" or "40 min".
  function timeLeft(stamp: string | Date | null | undefined): string {
    if (!stamp) return '';
    const diff = Math.max(0, dayjs(stamp).valueOf() - Date.now());
    if (diff >= HOUR) return t('time.hLeft', { n: Math.floor(diff / HOUR) });
    return t('time.minLeft', { n: Math.max(1, Math.floor(diff / MINUTE)) });
  }

  return { shortDate, ago, timeLeft };
}
