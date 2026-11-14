import { getCurrentInstance } from 'vue';
import type Utils from '../utils';
import type eventBus from '../eventBus';

// A call on the hand-written legacy client (api/index.js). Every one is an
// async axios call that resolves to the unwrapped, untyped response data.
type LegacyApiFn = (...args: unknown[]) => Promise<unknown>;

export interface GlobalProperties {
  $utils: Utils;
  $api: Record<string, LegacyApiFn>;
  $can: (...perms: string[]) => boolean;
  $canList: (id: number, perm: string) => boolean;
  $events: typeof eventBus;
}

export function useGlobal(): GlobalProperties {
  const instance = getCurrentInstance();
  if (!instance) throw new Error('useGlobal must be used inside setup()');
  return instance.appContext.config.globalProperties as unknown as GlobalProperties;
}
