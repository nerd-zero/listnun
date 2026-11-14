import { defineStore } from 'pinia';
import { models, type ModelName } from '../constants';
import type { ServerConfig } from '../api/generated/model';

type LoadingState = Record<ModelName, boolean>;

// The signed-in user, as the legacy client stores it (camelCased).
export interface Profile {
  id?: number;
  username?: string;
  name?: string;
  email?: string;
  userRole?: { id: number; name?: string; permissions: string[] };
  listRole?: { lists: { id: number; permissions: string[] }[] } | null;
}

// serverConfig is stored as the server sends it (snake_case): the legacy
// client fetches it with camelCase off.
type ModelState = Record<Exclude<ModelName, 'serverConfig' | 'profile'>, unknown> & {
  serverConfig: ServerConfig;
  profile: Profile;
};

interface MainStoreState extends ModelState {
  loading: LoadingState;
  refreshTick: number;
}

// eslint-disable-next-line import/prefer-default-export
export const useMainStore = defineStore('main', {
  state: (): MainStoreState => ({
    // Model data keyed by model name (lists, campaigns, etc.)
    ...(Object.keys(models).reduce((obj, cur) => ({ ...obj, [cur]: [] }), {}) as ModelState),

    // Per-model loading flags driven by API interceptors.
    loading: Object.keys(models).reduce(
      (obj, cur) => ({ ...obj, [cur]: false }),
      {} as LoadingState,
    ),

    // Incremented by the topbar refresh button; views watch this to re-fetch.
    refreshTick: 0,
  }),

  actions: {
    setModelResponse({ model, data }: { model: ModelName; data: unknown }) {
      (this as unknown as Record<string, unknown>)[model] = data;
    },

    setLoading({ model, status }: { model: ModelName; status: boolean }) {
      this.loading[model] = status;
    },

    refresh() {
      this.refreshTick += 1;
    },
  },
});
