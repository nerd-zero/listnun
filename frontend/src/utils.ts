import dayjs from 'dayjs';
import dayDuration from 'dayjs/plugin/duration';
import relativeTime from 'dayjs/plugin/relativeTime';
import updateLocale from 'dayjs/plugin/updateLocale';
import { showToast, showConfirm, showPrompt } from './toastService';

dayjs.extend(updateLocale);
dayjs.extend(relativeTime);
dayjs.extend(dayDuration);

const reEmail = /(.+?)@(.+?)/ig;
const prefKey = 'listmonk_pref';

// Shared page/sort click handlers for the query-params-plus-refetch
// pattern repeated across every paginated table view (Lists, Media,
// Bounces, Campaigns, Subscribers): mutate the reactive query params,
// then refetch. A plain function, not a composable -- it uses no Vue
// reactivity of its own, just closures over what the caller passes in.
export function makeQueryHandlers(
  queryParams: { page: number; orderBy?: string; order?: string },
  fetch: () => void,
) {
  function onPageChange(page: number) {
    Object.assign(queryParams, { page });
    fetch();
  }

  function onSort(field: string, direction: string) {
    Object.assign(queryParams, { orderBy: field, order: direction });
    fetch();
  }

  return { onPageChange, onSort };
}

export default class Utils {
  i18n: ReturnType<typeof import('vue-i18n').useI18n>;

  intlNumFormat: Intl.NumberFormat;

  constructor(i18n: ReturnType<typeof import('vue-i18n').useI18n>) {
    this.i18n = i18n;
    this.intlNumFormat = new Intl.NumberFormat();

    if (i18n) {
      dayjs.updateLocale('en', {
        relativeTime: {
          future: '%s',
          past: '%s',
          s: `${i18n.t('globals.terms.second', 2)}`,
          m: `1 ${i18n.t('globals.terms.minute', 1)}`,
          mm: `%d ${i18n.t('globals.terms.minute', 2)}`,
          h: `1 ${i18n.t('globals.terms.hour', 1)}`,
          hh: `%d ${i18n.t('globals.terms.hour', 2)}`,
          d: `1 ${i18n.t('globals.terms.day', 1)}`,
          dd: `%d ${i18n.t('globals.terms.day', 2)}`,
          M: `1 ${i18n.t('globals.terms.month', 1)}`,
          MM: `%d ${i18n.t('globals.terms.month', 2)}`,
          y: `${i18n.t('globals.terms.year', 1)}`,
          yy: `%d ${i18n.t('globals.terms.year', 2)}`,
        },
      });
    }
  }

  getDate = (d: string | Date) => dayjs(d);

  // Parses an ISO timestamp to a simpler form.
  niceDate = (stamp: string | null | undefined, showTime?: boolean): string => {
    if (!stamp) {
      return '';
    }

    const d = dayjs(stamp);
    const day = this.i18n.t(`globals.days.${d.day() + 1}`);
    const month = this.i18n.t(`globals.months.${d.month() + 1}`);
    let out = d.format(`[${day},] DD [${month}] YYYY`);
    if (showTime) {
      out += d.format(', HH:mm');
    }

    return out;
  };

  duration = (start: string | Date, end: string | Date): string => {
    const a = dayjs(start);
    const b = dayjs(end);
    const d = dayjs.duration(Math.abs(b.diff(a)));

    const parts = [
      Math.floor(d.asDays()) && `${Math.floor(d.asDays())}d`,
      d.hours() && `${d.hours()}h`,
      d.minutes() && `${d.minutes()}m`,
      d.seconds() && `${d.seconds()}s`,
    ].filter(Boolean);

    return `${b.isBefore(a) ? '-' : ''}${parts.join(' ')}`;
  };

  // Simple, naive, e-mail address check.
  validateEmail = (e: string) => e.match(reEmail);

  niceNumber = (n: number | null | undefined): number | string => {
    if (n === null || n === undefined) {
      return 0;
    }

    let pfx = '';
    let div = 1;

    if (n >= 1.0e+9) {
      pfx = 'b';
      div = 1.0e+9;
    } else if (n >= 1.0e+6) {
      pfx = 'm';
      div = 1.0e+6;
    } else if (n >= 1.0e+4) {
      pfx = 'k';
      div = 1.0e+3;
    } else {
      return n;
    }

    const out = (n / div);
    if (Math.floor(out) === out) {
      return out + pfx;
    }

    return out.toFixed(2) + pfx;
  };

  formatNumber(v: number): string {
    return this.intlNumFormat.format(v);
  }

  // Parse one or more numeric ids as query params and return as an array of ints.
  parseQueryIDs = (ids: string | number | string[] | number[] | null | undefined): number[] => {
    if (!ids) {
      return [];
    }

    if (typeof ids === 'string') {
      return [parseInt(ids, 10)];
    }

    if (typeof ids === 'number') {
      return [parseInt(String(ids), 10)];
    }

    return (ids as (string | number)[]).map((id) => parseInt(String(id), 10));
  };

  titleCase = (str: string): string => str[0].toUpperCase() + str.slice(1).toLowerCase();

  // UI shortcuts.
  confirm = (msg: string | null, onConfirm?: () => void, onCancel?: () => void): void => {
    showConfirm(
      !msg ? this.i18n.t('globals.messages.confirm') as string : msg,
      onConfirm,
      onCancel,
    );
  };

  prompt = (
    msg: string,
    _inputAttrs: unknown,
    onConfirm?: (value: string) => void,
    onCancel?: () => void,
  ): void => {
    showPrompt(msg, onConfirm, onCancel);
  };

  toast = (msg: string, typ?: string, duration?: number): void => {
    showToast(msg, typ || 'is-success', duration || 3000);
  };

  // Takes a props.row from a Buefy b-column <td> template and
  // returns a `data-id` attribute which Buefy then applies to the td.
  tdID = (row: { id: number | string }) => ({ 'data-id': row.id.toString() });

  camelString = (str: string): string => {
    const s = str.replace(/[-_\s]+(.)?/g, (_match, chr) => (chr ? chr.toUpperCase() : ''));
    return s.slice(0, 1).toLowerCase() + s.slice(1);
  };

  // camelKeys recursively camelCases all keys in a given object (array or {}).
  camelKeys = (obj: unknown, testFunc?: (keyPath: string) => boolean, keys?: string): unknown => {
    if (obj === null) {
      return obj;
    }

    if (Array.isArray(obj)) {
      return obj.map((o) => this.camelKeys(o, testFunc, `${keys || ''}.*`));
    }

    if (obj !== null && typeof obj === 'object' && (obj as object).constructor === Object) {
      return Object.keys(obj as object).reduce((result: Record<string, unknown>, key) => {
        const keyPath = `${keys || ''}.${key}`;
        let k = key;

        if (testFunc === undefined || testFunc(keyPath)) {
          k = this.camelString(key);
        }

        return {
          ...result,
          [k]: this.camelKeys((obj as Record<string, unknown>)[key], testFunc, keyPath),
        };
      }, {});
    }

    return obj;
  };

  getPref = (key: string): unknown => {
    if (localStorage.getItem(prefKey) === null) {
      return null;
    }

    const p = JSON.parse(localStorage.getItem(prefKey)!);
    return key in p ? p[key] : null;
  };

  setPref = (key: string, val: unknown): void => {
    let p: Record<string, unknown> = {};
    if (localStorage.getItem(prefKey) !== null) {
      p = JSON.parse(localStorage.getItem(prefKey)!);
    }

    p[key] = val;
    localStorage.setItem(prefKey, JSON.stringify(p));
  };
}
