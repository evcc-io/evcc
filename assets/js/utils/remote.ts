const ACTIVE_THRESHOLD_MS = 5 * 60 * 1000; // 5 minutes

export function isRemoteClientActive(
  lastSeen: Record<string, string> | undefined,
  username: string
): boolean {
  const seen = lastSeen?.[username];
  if (!seen) return false;
  return Date.now() - new Date(seen).getTime() < ACTIVE_THRESHOLD_MS;
}
