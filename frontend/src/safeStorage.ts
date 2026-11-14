// localStorage that never throws. Storage can be unavailable (blocked by the
// browser or a privacy setting, a full quota, some private modes) and a
// saved value can be corrupt; UI preferences are conveniences, so a failure
// just means the default is used and the change lasts for this visit only.

export function readItem(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function writeItem(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Storage unavailable: see the note at the top.
  }
}

export function removeItem(key: string): void {
  try {
    localStorage.removeItem(key);
  } catch {
    // Storage unavailable: see the note at the top.
  }
}

// readJSON returns the parsed value saved under key, or null when there is
// none, storage is unavailable or the saved text isn't valid JSON. Callers
// check its shape.
export function readJSON(key: string): unknown {
  const raw = readItem(key);
  if (raw === null) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function writeJSON(key: string, value: unknown): void {
  let raw: string;
  try {
    raw = JSON.stringify(value);
  } catch {
    return;
  }
  writeItem(key, raw);
}
