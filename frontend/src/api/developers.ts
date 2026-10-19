// Typed edge over the orval-generated Developers client. The shared
// mutator camelCases responses at runtime, so the generated snake_case
// models don't describe what callers get; these interfaces do.
import axios from 'axios';
import { getDevelopers } from './generated/endpoints/developers/developers';
import {
  getTxMessage as legacyGetTxMessage,
  listTxTemplates as legacyListTxTemplates,
  createTxTemplate as legacyCreateTxTemplate,
  updateTxTemplate as legacyUpdateTxTemplate,
} from './index';

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

// rawHttp leaves responses untouched, for showing API calls as they are.
const rawHttp = axios.create({ baseURL: import.meta.env.VUE_APP_ROOT_URL || '/', responseType: 'json' });

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

export type HeaderSet = Record<string, string>;

export interface TxTemplate {
  id: number;
  name: string;
  subject: string;
  body: string;
  altBody: string;
  headers: HeaderSet[];
  // Sample `data` for previews and test sends; the caller's own shape.
  sampleData: Record<string, unknown>;
  usedThisWeek: number;
  createdAt: string;
  updatedAt: string;
}

export interface TxTemplateInput {
  name: string;
  subject: string;
  body: string;
  altBody: string;
  headers: HeaderSet[];
  sampleData: Record<string, unknown>;
}

export interface TxPreview {
  subject: string;
  html: string;
  text: string;
  from: string;
}

// Sample data and header names are the user's own keys, so they're sent
// and read back untouched rather than camelCased.
const toTemplateReq = (t: TxTemplateInput) => ({
  name: t.name,
  subject: t.subject,
  body: t.body,
  alt_body: t.altBody,
  headers: t.headers,
  sample_data: t.sampleData as never,
});

export const listTxTemplates = () => legacyListTxTemplates() as unknown as Promise<TxTemplate[]>;

export const createTxTemplate = (t: TxTemplateInput) => (
  legacyCreateTxTemplate(toTemplateReq(t)) as unknown as Promise<TxTemplate>
);

export const updateTxTemplate = (id: number, t: TxTemplateInput) => (
  legacyUpdateTxTemplate(id, toTemplateReq(t)) as unknown as Promise<TxTemplate>
);

export const previewTxTemplate = (p: { subject: string; body: string; altBody: string; data: Record<string, unknown> }) => (
  dev.previewTxTemplate({
    subject: p.subject, body: p.body, alt_body: p.altBody, data: p.data,
  }) as unknown as Promise<TxPreview>
);

// TxRequest is a POST /api/tx body as the docs describe it.
export interface TxRequest {
  subscriber_email?: string;
  subscriber_mode?: 'default' | 'fallback' | 'external';
  template_id: number;
  data: Record<string, unknown>;
  from_email?: string;
  subject?: string;
  content_type?: 'html' | 'markdown' | 'plain';
  headers?: HeaderSet[];
}

export interface TxResult {
  status: number;
  statusText: string;
  ms: number;
  // The raw response body, shown as the API returned it.
  body: unknown;
  messageIds: string[];
}

// sendTestTx calls POST /api/tx from the browser exactly as an integration
// would, returning the raw response rather than the app's unwrapped data.
// asKey attributes the send to an API key in the message log.
export async function sendTestTx(req: TxRequest, asKey?: string): Promise<TxResult> {
  const started = performance.now();
  const resp = await rawHttp.post('/api/tx', req, {
    headers: asKey ? { 'Listnun-Send-As': asKey } : {},
    validateStatus: () => true,
  });
  const ids = String(resp.headers['listnun-message-id'] || '').split(',').map((s) => s.trim()).filter(Boolean);
  return {
    status: resp.status,
    statusText: resp.statusText,
    ms: Math.round(performance.now() - started),
    body: resp.data,
    messageIds: ids,
  };
}

export interface BrandLogo {
  mediaId: number;
  url: string;
  filename: string;
  contentType: string;
  size: number;
  width: number;
  height: number;
  uploadedAt: string;
}

export interface Branding {
  logo: BrandLogo | null;
  logoDark: BrandLogo | null;
  width: number;
  align: 'left' | 'center';
  linkUrl: string;
}

const logoToReq = (l: BrandLogo | null) => (l ? {
  media_id: l.mediaId,
  url: l.url,
  filename: l.filename,
  content_type: l.contentType,
  size: l.size,
  width: l.width,
  height: l.height,
  uploaded_at: l.uploadedAt,
} : null);

export const getBranding = () => dev.getBranding() as unknown as Promise<Branding>;

export const updateBranding = (b: Branding) => dev.updateBranding({
  logo: logoToReq(b.logo) as never,
  logo_dark: logoToReq(b.logoDark) as never,
  width: b.width,
  align: b.align,
  link_url: b.linkUrl,
}) as unknown as Promise<Branding>;

// Logo uploads: SVG, PNG or JPG up to 1 MB, raster images at least 240 px
// wide so they stay sharp on high-density screens.
export const LOGO_TYPES = ['image/svg+xml', 'image/png', 'image/jpeg'];
export const LOGO_MAX_BYTES = 1024 * 1024;
export const LOGO_MIN_WIDTH = 240;
