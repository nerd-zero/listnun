import { useSyncExternalStore } from 'react';

import { ThemeMode } from './theme';

/**
 * The builder's theme mode.
 *
 * The admin app owns the mode. The builder runs in a same-origin frame, so
 * the app calls the exported setThemeMode directly before rendering and on
 * every toggle (see VisualEditor.vue); the builder never picks one itself.
 */

type ModeListener = () => void;

let currentMode: ThemeMode = 'light';
const listeners = new Set<ModeListener>();

/** Reads the current mode. */
export function getThemeMode(): ThemeMode {
  return currentMode;
}

/** Sets the mode; anything else than 'dark' means light. */
export function setThemeMode(mode: ThemeMode): void {
  const next: ThemeMode = mode === 'dark' ? 'dark' : 'light';
  if (next === currentMode) {
    return;
  }
  currentMode = next;
  listeners.forEach((listener) => listener());
}

/** The message the admin app posts when the user toggles the theme. */
export const THEME_MESSAGE = 'listnun:theme';

/**
 * Applies a theme message from the admin app (handover: `{ type:
 * "listnun:theme", mode: "light" | "dark" }`). Only the embedding page may
 * send it: same origin as this frame (a srcdoc frame shares its parent's)
 * and from the parent window itself.
 */
export function handleThemeMessage(event: MessageEvent): boolean {
  if (event.source !== window.parent || event.origin !== window.location.origin) {
    return false;
  }
  const data = event.data as { type?: unknown; mode?: unknown } | null;
  if (!data || data.type !== THEME_MESSAGE || (data.mode !== 'light' && data.mode !== 'dark')) {
    return false;
  }
  setThemeMode(data.mode);
  return true;
}

/** Subscribes the root component to mode changes. */
export function useThemeMode(): ThemeMode {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getThemeMode,
    getThemeMode,
  );
}
