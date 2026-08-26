export function isDevelopment(version: string): boolean {
  return version === "0.0.0";
}

// untagged builds carry the commit as build metadata: 0.304.0-dev+abc1234
export function isNightly(version: string): boolean {
  return version.includes("+");
}

export function commitFromVersion(version: string): string {
  return version.split("+")[1] ?? "";
}

export function getReleaseName(version: string): string {
  if (isDevelopment(version)) return "development";
  if (isNightly(version)) return "nightly";
  return "stable";
}

export function getShortVersion(version: string): string {
  if (isDevelopment(version)) return "dev build";
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
