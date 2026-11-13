import { getHealth } from './index';

// How long to wait before the first health check: the backend restarts
// itself ~500ms after a settings save or a reload request
// (cmd/settings.go's handleSettingsRestart -> syscall.Exec in cmd/init.go),
// so an immediate check would hit the old process.
const FIRST_CHECK_DELAY_MS = 1500;
const POLL_INTERVAL_MS = 500;

// waitForServer resolves once the health endpoint answers after a restart,
// and rejects if it hasn't within timeoutMs. Requests made while the process
// is mid-exec fail (a proxy in front, e.g. Cloudflare, shows a gateway error),
// so callers wait on this before fetching anything else.
export function waitForServer(timeoutMs = 30000, health: () => Promise<unknown> = getHealth): Promise<void> {
  const start = Date.now();
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      let inFlight = false;
      const poll = setInterval(() => {
        if (inFlight) return;
        inFlight = true;
        health().then(() => {
          clearInterval(poll);
          resolve();
        }).catch(() => {
          if (Date.now() - start > timeoutMs) {
            clearInterval(poll);
            reject(new Error('timed out waiting for the server to restart'));
          }
        }).finally(() => {
          inFlight = false;
        });
      }, POLL_INTERVAL_MS);
    }, FIRST_CHECK_DELAY_MS);
  });
}

export default waitForServer;
