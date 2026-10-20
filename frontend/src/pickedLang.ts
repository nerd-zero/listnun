// The language a visitor picked on the sign-in pages, from the ln_lang
// cookie (see internal/i18n.LangCookie). Returns '' when none was picked or
// the value isn't a plausible language code.
export function pickedLang(cookie: string): string {
  const m = cookie.match(/(?:^|;\s*)ln_lang=([^;]*)/);
  if (!m) return '';
  let v = '';
  try {
    v = decodeURIComponent(m[1]);
  } catch {
    return '';
  }
  return /^[a-zA-Z0-9_-]{2,6}$/.test(v) ? v : '';
}

export default pickedLang;
