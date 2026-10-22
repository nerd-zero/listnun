export type StatusTone = 'success' | 'warn' | 'danger' | 'neutral' | 'plain';

// Status words to their tone, per the tokens board: green for things that
// went or are going well, amber for waiting or needing a look, red for
// failures and removals, grey for finished or inert. Drafts have no pill.
const tones: Record<string, StatusTone> = {
  running: 'success',
  importing: 'success',
  delivered: 'success',
  active: 'success',
  enabled: 'success',
  verified: 'success',
  confirmed: 'success',
  opened: 'success',
  clicked: 'success',
  ok: 'success',
  success: 'success',
  valid: 'success',

  scheduled: 'warn',
  queued: 'warn',
  pending: 'warn',
  retrying: 'warn',
  stopping: 'warn',
  bounced: 'warn',
  soft: 'warn',
  unconfirmed: 'warn',
  expiring: 'warn',
  paused: 'neutral',

  failed: 'danger',
  error: 'danger',
  revoked: 'danger',
  blocklisted: 'danger',
  unsubscribed: 'danger',
  hard: 'danger',
  complaint: 'danger',
  cancelled: 'danger',
  stopped: 'danger',
  disabled: 'danger',
  invalid: 'danger',

  draft: 'plain',
};

export function statusTone(status: string): StatusTone {
  return tones[(status || '').toLowerCase()] ?? 'neutral';
}

// PrimeVue Tag/Message severity for the same tone.
export function statusSeverity(status: string): 'success' | 'warn' | 'danger' | 'secondary' {
  const t = statusTone(status);
  if (t === 'success' || t === 'warn' || t === 'danger') return t;
  return 'secondary';
}
