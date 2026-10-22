import type { APIKeyScope } from '../../api/developers';

// The permissions each key scope grants; mirrors models.APIKeyScopes.
export const SCOPE_PERMISSIONS: Record<APIKeyScope, string[]> = {
  'tx:send': ['tx:send'],
  'tx:logs': ['tx:logs'],
  'subscribers:read': ['subscribers:get_all'],
};

// The built-in admin role holds every permission.
const ADMIN_ROLE_ID = 1;

export interface LegacyRole { id: number; permissions?: string[] | null }

// suggestScopes picks the key scopes a legacy API user's role already
// covers, so the replacement key can do the same things where it can.
export function suggestScopes(role: LegacyRole | null | undefined): APIKeyScope[] {
  const scopes = Object.keys(SCOPE_PERMISSIONS) as APIKeyScope[];
  if (!role) return [];
  if (role.id === ADMIN_ROLE_ID) return scopes;
  const held = new Set(role.permissions || []);
  return scopes.filter((s) => SCOPE_PERMISSIONS[s].every((p) => held.has(p)));
}

// uncoveredPermissions lists what the legacy user can do that a key with
// these scopes can't, so the switch doesn't silently cut access. For the
// admin role it returns ['*'].
export function uncoveredPermissions(role: LegacyRole | null | undefined, scopes: APIKeyScope[]): string[] {
  if (!role) return [];
  if (role.id === ADMIN_ROLE_ID) return ['*'];
  const granted = new Set(scopes.flatMap((s) => SCOPE_PERMISSIONS[s]));
  return (role.permissions || []).filter((p) => !granted.has(p));
}

// keyNameFor turns a username into a valid key name
// (/^[a-z0-9][a-z0-9._-]{1,62}$/).
export function keyNameFor(username: string): string {
  let n = username.toLowerCase().replace(/[^a-z0-9._-]+/g, '-').replace(/^[^a-z0-9]+/, '');
  if (n.length < 2) n = `${n}key`.padEnd(2, '0');
  return n.slice(0, 63);
}
