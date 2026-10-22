import { describe, expect, it } from 'vitest';
import { keyNameFor, suggestScopes, uncoveredPermissions } from './legacyKeys';

describe('legacy API user to key', () => {
  it('suggests only scopes the role fully holds', () => {
    const role = { id: 5, permissions: ['tx:send', 'campaigns:manage'] };
    expect(suggestScopes(role)).toEqual(['tx:send']);
  });

  it('gives the admin role every scope and flags it as uncoverable', () => {
    expect(suggestScopes({ id: 1 })).toEqual(['tx:send', 'tx:logs', 'subscribers:read']);
    expect(uncoveredPermissions({ id: 1 }, ['tx:send'])).toEqual(['*']);
  });

  it('lists permissions a key with the chosen scopes cannot use', () => {
    const role = { id: 5, permissions: ['tx:send', 'subscribers:get_all', 'campaigns:manage'] };
    expect(uncoveredPermissions(role, ['tx:send', 'subscribers:read'])).toEqual(['campaigns:manage']);
    expect(uncoveredPermissions(role, ['tx:send'])).toEqual(['subscribers:get_all', 'campaigns:manage']);
  });

  it('makes a valid key name from any username', () => {
    const valid = /^[a-z0-9][a-z0-9._-]{1,62}$/;
    ['Checkout_Svc', '@@api user', 'x', 'a'.repeat(90)].forEach((u) => {
      expect(keyNameFor(u)).toMatch(valid);
    });
    expect(keyNameFor('Checkout Svc')).toBe('checkout-svc');
  });
});
