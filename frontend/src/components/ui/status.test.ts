import { describe, expect, it } from 'vitest';
import { statusSeverity, statusTone } from './status';

describe('statusTone', () => {
  it('maps campaign statuses to the dashboard tones', () => {
    expect(statusTone('running')).toBe('success');
    expect(statusTone('scheduled')).toBe('warn');
    expect(statusTone('finished')).toBe('neutral');
    expect(statusTone('draft')).toBe('plain');
    expect(statusTone('cancelled')).toBe('danger');
  });

  it('ignores case and falls back to neutral for unknown words', () => {
    expect(statusTone('Bounced')).toBe('warn');
    expect(statusTone('something-new')).toBe('neutral');
    expect(statusTone('')).toBe('neutral');
  });
});

describe('statusSeverity', () => {
  it('turns neutral and plain into the secondary severity', () => {
    expect(statusSeverity('blocklisted')).toBe('danger');
    expect(statusSeverity('draft')).toBe('secondary');
    expect(statusSeverity('paused')).toBe('secondary');
  });
});
