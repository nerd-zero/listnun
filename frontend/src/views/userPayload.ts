// A user as the API returns it, camelCased.
type Loose = Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any

// userPayload builds the create/update body for a user from a camelCased
// user row or form. Passwords are only sent when password login is on and
// one was typed, so an edit never clears an existing password.
export function userPayload(u: Loose) {
  const passwordLogin = !!u.passwordLogin;
  return {
    username: u.username,
    email: u.type === 'api' ? undefined : u.email,
    name: u.name,
    type: u.type || 'user',
    status: u.status || 'enabled',
    password_login: passwordLogin,
    password: passwordLogin && u.password ? u.password : null,
    user_role_id: u.userRoleId ?? u.userRole?.id,
    list_role_id: (u.listRoleId ?? u.listRole?.id) || null,
  };
}

export default userPayload;
