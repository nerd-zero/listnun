// txDataVars lists the top-level .Tx.Data keys a template refers to, in
// the order they first appear, e.g. ["order_id", "total", "items"].
export function txDataVars(...sources: string[]): string[] {
  const re = /\.Tx\.Data\.([A-Za-z_][A-Za-z0-9_]*)/g;
  const seen = new Set<string>();
  sources.forEach((src) => {
    let m = re.exec(src);
    while (m) {
      seen.add(m[1]);
      m = re.exec(src);
    }
  });
  return [...seen];
}

// missingVars returns the variables data doesn't provide.
export function missingVars(vars: string[], data: Record<string, unknown>): string[] {
  return vars.filter((v) => !(v in data));
}
