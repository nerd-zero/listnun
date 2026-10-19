// Typed edge over the orval-generated Developers client. The shared
// mutator camelCases responses at runtime, so the generated snake_case
// models don't describe what callers get; these interfaces do.
import { getDevelopers } from './generated/endpoints/developers/developers';
import { getTxMessage as legacyGetTxMessage } from './index';

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

export type TxStatus = 'queued' | 'sent' | 'delivered' | 'bounced' | 'failed';

// Statuses in the order the log's filter pills show them.
export const TX_STATUSES: TxStatus[] = ['delivered', 'sent', 'queued', 'bounced', 'failed'];

export interface TxLogEvent {
  type: 'accepted' | 'rendered' | 'sent' | 'delivered' | 'bounced' | 'failed';
  detail: string;
  createdAt: string;
}

export interface TxLogMessage {
  id: string;
  apiKeyId: number | null;
  sender: string;
  templateId: number | null;
  templateName: string;
  subscriberId: number | null;
  subscriberStatus?: string;
  recipient: string;
  subject: string;
  status: TxStatus;
  bounceType: string | null;
  error: string;
  messenger: string;
  sourceIp?: string;
  // The single-recipient request as sent, kept for inspection and resend.
  payload?: Record<string, unknown>;
  body?: string;
  createdAt: string;
  updatedAt: string;
  events?: TxLogEvent[];
}

export interface TxLogPage {
  results: TxLogMessage[];
  counts: Partial<Record<TxStatus, number>>;
  total: number;
  page: number;
  perPage: number;
}

export interface TxLogQuery {
  query?: string;
  status?: TxStatus | '';
  templateId?: number;
  apiKeyId?: number;
  from?: string;
  to?: string;
  page?: number;
  perPage?: number;
}

export interface TxStats {
  sent: number;
  sentPrev: number;
  delivered: number;
  bounced: number;
  failed: number;
  topError: string;
}

export const listTxMessages = (q: TxLogQuery) => dev.listTxMessages({
  query: q.query || undefined,
  status: q.status || undefined,
  template_id: q.templateId || undefined,
  api_key_id: q.apiKeyId || undefined,
  from: q.from || undefined,
  to: q.to || undefined,
  page: q.page,
  per_page: q.perPage,
}) as unknown as Promise<TxLogPage>;

// Fetched through the legacy client, which can leave the payload's keys
// (the caller's own request and data) as they were sent.
export const getTxMessage = (id: string) => legacyGetTxMessage(id) as unknown as Promise<TxLogMessage>;

export const resendTxMessage = (id: string) => dev.resendTxMessage(id) as unknown as Promise<string[]>;

export const getTxStats = () => dev.getTxStats() as unknown as Promise<TxStats>;
