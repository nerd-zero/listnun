// translateOr translates key, or returns fallback when there is no such
// message. vue-i18n's te() can't be used for this: our language packs use
// flat dotted keys ("globals.terms.campaign"), which te() looks up as a
// nested path and never finds, while t() falls back to the flat key. t()
// returns the key itself when a message is missing, so that's the test.
export function translateOr(
  t: (k: string, n?: number) => string,
  key: string,
  fallback: string,
  plural?: number,
): string {
  if (!key) return fallback;
  const v = plural === undefined ? t(key) : t(key, plural);
  return v && v !== key ? v : fallback;
}

export default translateOr;
