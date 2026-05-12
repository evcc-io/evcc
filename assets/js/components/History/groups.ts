export const GROUP_ORDER = ["pv", "battery", "grid", "loadpoint", "meter"] as const;

const GROUP_COLOR_VAR: Record<string, string> = {
  pv: "--evcc-dark-green",
  forecast: "--evcc-dark-yellow",
  loadpoint: "--evcc-dark-green",
  grid: "--evcc-grid",
  battery: "--evcc-darker-green",
  meter: "--evcc-price",
};

const GROUP_EXPORT_COLOR_VAR: Record<string, string> = {
  grid: "--evcc-export-contrast",
};

function readCssVar(name: string): string {
  return window.getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

export function groupColor(group: string): string {
  const v = GROUP_COLOR_VAR[group];
  if (!v) return "";
  return readCssVar(v) || "";
}

export function groupExportColor(group: string): string | null {
  const v = GROUP_EXPORT_COLOR_VAR[group];
  if (!v) return null;
  return readCssVar(v) || null;
}
