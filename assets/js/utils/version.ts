export function isDevelopment(version: string): boolean {
  return version === "0.0.0";
}

// untagged builds carry a build timestamp and the commit: 0.304.0-dev.1712345678+abc1234
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
  // the build timestamp only serves package ordering and is not worth displaying
  return `v${version.replace(/-dev\.\d+\+/, "-dev+")}`;
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
