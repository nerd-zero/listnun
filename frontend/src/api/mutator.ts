import Axios, { type AxiosRequestConfig } from 'axios';
import qs from 'qs';
import Utils from '../utils';
import { showToast } from '../toastService';
import { isTwofaRequired, redirectToTwofa } from './twofaGuard';

// Pass null for i18n — camelKeys doesn't need it.
const utils = new Utils(null as never);

// Fields whose values are free-form JSON: the keys inside them belong to the
// user, a provider or an API caller, not to the API (subscriber and campaign
// attribs, campaign/template headers, bounce and subscription meta, campaign
// archive meta, tx template sample data, logged tx and webhook payloads).
// Their keys are kept as sent wherever they appear in a response, so an
// attribute like "first_name" isn't saved back as "firstName".
const FREE_FORM_FIELDS = ['attribs', 'headers', 'meta', 'archive_meta', 'sample_data', 'payload'];
const FREE_FORM_PATH = new RegExp(`\\.(${FREE_FORM_FIELDS.join('|')})\\.`);

// Responses that keep their keys as sent, by URL. A null path keeps every
// key in the response. Settings and server config are keyed by dotted
// setting names ("app.root_url") and i18n files by message keys; the legacy
// client (api/index.js) never camelCases them either.
const KEEP_KEYS_BY_URL: { url: RegExp; path: RegExp | null }[] = [
  { url: /^\/?api\/(settings|config)\/?$/, path: null },
  { url: /^\/?api\/lang\/[^/]+$/, path: null },
  // Inbox check params are i18n placeholders.
  { url: /^\/?api\/campaigns\/[^/]+\/inbox-check$/, path: /\.params\./ },
];

// keepsKey reports whether the key at keyPath (e.g. ".results.*.attribs.city")
// in a response from url keeps its original case.
export function keepsKey(url: string, keyPath: string): boolean {
  if (FREE_FORM_PATH.test(keyPath)) {
    return true;
  }
  const path = url.split('?')[0];
  return KEEP_KEYS_BY_URL.some((r) => r.url.test(path) && (r.path === null || r.path.test(keyPath)));
}

// transformResponse unwraps the listnun { data: T } envelope and camelCases
// the API's own keys, leaving free-form JSON (see keepsKey) untouched.
export function transformResponse(body: unknown, url = ''): unknown {
  const inner = (body as { data?: unknown } | null | undefined)?.data;
  let data: unknown = inner ?? body;
  if (data !== null && typeof data === 'object') {
    data = utils.camelKeys(data, (keyPath) => !keepsKey(url, keyPath));
  }
  return data;
}

// Dedicated axios instance for orval-generated API calls.
// Mirrors the interceptor logic in api/index.js but without the per-call
// loading/store flags (those belong to the hand-written layer).
const http = Axios.create({
  baseURL: import.meta.env.VUE_APP_ROOT_URL || '/',
  withCredentials: false,
  responseType: 'json',
  paramsSerializer: (params) => qs.stringify(params, { arrayFormat: 'repeat' }),
});

http.interceptors.response.use(
  (resp) => transformResponse(resp.data, resp.config.url) as never,
  (err) => {
    if (isTwofaRequired(err)) {
      redirectToTwofa();
      return Promise.reject(err);
    }
    const msg: string = err.response?.data?.message || err.toString();
    showToast(msg, 'is-danger', 4000);
    return Promise.reject(err);
  },
);

export const httpMutator = <T>(config: AxiosRequestConfig): Promise<T> => http(config) as Promise<T>;

export default httpMutator;
