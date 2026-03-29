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
  return version;
}
