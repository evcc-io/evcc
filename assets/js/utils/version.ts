export function isDevelopment(version: string): boolean {
  return version === "0.0.0";
}

// untagged builds are pre-releases carrying the build timestamp: 0.304.0-dev.1712345678
export function isNightly(version: string): boolean {
  return version.includes("-dev.");
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
  const v = installed || "";
  return !!available && !isDevelopment(v) && !isNightly(v) && available !== installed;
}

export function isNewVersionUnacknowledged(
  installed?: string,
  available?: string,
  acknowledged?: string
): boolean {
  return isNewVersionAvailable(installed, available) && available !== acknowledged;
}
