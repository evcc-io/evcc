// Form values entered before an OAuth login survive the redirect back to evcc,
// which may land in a new tab. Stored per device, consumed once, short-lived.

const STORAGE_KEY = "evcc.authStash";
const TTL_MS = 10 * 60 * 1000;

export type AuthStash = {
  key: string;
  templateName: string | null;
  values: Record<string, any>;
  ts: number;
};

export function stashAuthValues(
  key: string,
  templateName: string | null,
  values: Record<string, any>
): void {
  try {
    const stash: AuthStash = { key, templateName, values, ts: Date.now() };
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(stash));
  } catch {
    // storage unavailable
  }
}

// read the stash, dropping it once expired
function readAuthStash(now: number): AuthStash | null {
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  const stash = JSON.parse(raw) as AuthStash;
  if (now - stash.ts > TTL_MS) {
    window.localStorage.removeItem(STORAGE_KEY);
    return null;
  }
  return stash;
}

export function popAuthValues(key: string, now = Date.now()): AuthStash | null {
  try {
    const stash = readAuthStash(now);
    if (!stash || stash.key !== key) return null;
    window.localStorage.removeItem(STORAGE_KEY);
    return stash;
  } catch {
    return null;
  }
}

// an abandoned login never pops, so the entered secrets are cleaned on app start
export function cleanAuthStash(now = Date.now()): void {
  try {
    readAuthStash(now);
  } catch {
    // storage unavailable
  }
}
