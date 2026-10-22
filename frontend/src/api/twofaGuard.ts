// The API refuses calls with 403 and X-Listnun-Error: twofa_required when
// the workspace requires two-step verification and the user hasn't set it
// up (cmd/twofa_policy.go). Both API clients send the user to their
// profile, where the setup lives, instead of showing an error toast.
export const TWOFA_PROFILE_PATH = '/admin/user/profile';

interface MaybeAxiosError { response?: { status?: number; headers?: Record<string, string> } }

export function isTwofaRequired(err: unknown): boolean {
  const r = (err as MaybeAxiosError)?.response;
  return r?.status === 403 && r.headers?.['x-listnun-error'] === 'twofa_required';
}

// redirectToTwofa navigates once; on the profile page itself it does
// nothing, so a blocked call there can't loop.
export function redirectToTwofa(): void {
  if (window.location.pathname.startsWith(TWOFA_PROFILE_PATH)) return;
  window.location.assign(`${TWOFA_PROFILE_PATH}?twofa=required`);
}
