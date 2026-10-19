// Typed edge over the orval-generated Developers client. The shared
// mutator camelCases responses at runtime, so the generated snake_case
// models don't describe what callers get; these interfaces do.
import { getDevelopers } from './generated/endpoints/developers/developers';

export type APIKeyScope = 'tx:send' | 'tx:logs' | 'subscribers:read';

// Every scope a key can be given, in the order the UI lists them.
export const API_KEY_SCOPES: APIKeyScope[] = ['tx:send', 'tx:logs', 'subscribers:read'];

export interface APIKey {
  id: number;
  name: string;
  secretHint: string;
  scopes: APIKeyScope[];
  status: 'active' | 'revoked';
  createdBy: number | null;
  createdByName: string;
  prevExpiresAt: string | null;
  lastUsedAt: string | null;
  revokedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface APIKeyWithSecret extends APIKey {
  secret: string;
}

const dev = getDevelopers();

export const listAPIKeys = (query = '') => dev.listAPIKeys({ query }) as unknown as Promise<APIKey[]>;

export const createAPIKey = (name: string, scopes: APIKeyScope[]) => (
  dev.createAPIKey({ name, scopes }) as unknown as Promise<APIKeyWithSecret>
);

export const updateAPIKeyScopes = (id: number, scopes: APIKeyScope[]) => (
  dev.updateAPIKey(id, { scopes }) as unknown as Promise<APIKey>
);

export const rotateAPIKey = (id: number) => dev.rotateAPIKey(id) as unknown as Promise<APIKeyWithSecret>;

export const revokeAPIKey = (id: number) => dev.revokeAPIKey(id) as unknown as Promise<APIKey>;

export const deleteAPIKey = (id: number) => dev.deleteAPIKey(id) as unknown as Promise<boolean>;

// isRotating reports whether a key's previous secret is still accepted.
export function isRotating(k: APIKey): boolean {
  return k.status === 'active' && !!k.prevExpiresAt && new Date(k.prevExpiresAt).getTime() > Date.now();
}

// maskedKey shows the prefix and last four characters of a secret.
export function maskedKey(k: Pick<APIKey, 'secretHint'>): string {
  return `lnk_live_••••${k.secretHint}`;
}
