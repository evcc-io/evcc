export function isDevelopment(version: string): boolean {
  return version === "[[.Version]]" || version === "0.0.0";
}

export function isNightly(version: string, commit?: string): boolean {
  return !isDevelopment(version) && !!commit;
}

export function getReleaseName(version: string, commit?: string): string {
  if (isDevelopment(version)) return "development";
  if (isNightly(version, commit)) return "nightly";
  return "stable";
}

export function shortCommit(commit?: string): string {
  return commit?.substring(0, 7) || "";
}

export function getShortVersion(version: string, commit?: string): string {
  if (isDevelopment(version)) return "dev build";
  if (isNightly(version, commit)) return `v${version} (${shortCommit(commit)})`;
  return `v${version}`;
}

export function isNewVersionAvailable(installed?: string, available?: string): boolean {
  return !!available && !isDevelopment(installed || "") && available !== installed;
}

export function isNewVersionUnacknowledged(
  installed?: string,
  available?: string,
  acknowledged?: string
): boolean {
  return isNewVersionAvailable(installed, available) && available !== acknowledged;
}
