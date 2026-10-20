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
