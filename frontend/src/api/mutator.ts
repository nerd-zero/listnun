import Axios, { type AxiosRequestConfig } from 'axios';
import qs from 'qs';
import Utils from '../utils';
import { showToast } from '../toastService';

// Pass null for i18n — camelKeys doesn't need it.
const utils = new Utils(null as never);

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
  (resp) => {
    // Unwrap the listnun { data: T } envelope.
    let data: unknown = resp.data?.data ?? resp.data;
    if (data !== null && typeof data === 'object') {
      data = utils.camelKeys(data);
    }
    return data as never;
  },
  (err) => {
    const msg: string = err.response?.data?.message || err.toString();
    showToast(msg, 'is-danger', 4000);
    return Promise.reject(err);
  },
);

export const httpMutator = <T>(config: AxiosRequestConfig): Promise<T> => http(config) as Promise<T>;

export default httpMutator;
