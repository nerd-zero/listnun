import React from 'react';
import ReactDOM from 'react-dom/client';
import App, { AppProps, DEFAULT_SOURCE } from './App';
import { setDocument, resetDocument } from './documents/editor/EditorContext';

import { CssBaseline, ThemeProvider } from '@mui/material';
import { listnunTheme } from './theme';
import { setThemeMode, useThemeMode } from './themeMode';

function isRendered(containerId: string): boolean {
  const container = document.getElementById(containerId);
  if (!container) {
    console.error(`Container with id ${containerId} not found`);
    return false;
  }
  return container.hasChildNodes();
}

/**
 * The builder root. The theme mode belongs to the admin app, which sets it
 * through the exported setThemeMode (see VisualEditor.vue); the theme is
 * re-derived whenever it changes.
 */
function BuilderRoot(props: AppProps) {
  const mode = useThemeMode();

  return (
    <ThemeProvider theme={listnunTheme(mode)}>
      <CssBaseline />
      <App {...props} />
    </ThemeProvider>
  );
}

function render(containerId: string, props: AppProps, force: boolean = false) {
  if (!isRendered(containerId) || force) {
    const container = document.getElementById(containerId);
    if (!container) return;

    ReactDOM.createRoot(container).render(
      <React.StrictMode>
        <BuilderRoot {...props} />
      </React.StrictMode>
    );
  }
}

export { App, setDocument, resetDocument, render, isRendered, setThemeMode, DEFAULT_SOURCE };
