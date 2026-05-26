export const GROUP_ORDER = ["pv", "battery", "grid", "loadpoint", "consumer"] as const;

const GROUP_COLOR_VAR: Record<string, string> = {
  pv: "--evcc-dark-green",
  forecast: "--evcc-dark-yellow",
  loadpoint: "--evcc-dark-green",
  grid: "--evcc-grid",
  battery: "--evcc-darker-green",
  consumer: "--evcc-price",
};

export function groupColor(group: string): string {
  const v = GROUP_COLOR_VAR[group];
  if (!v) return "";
  return window.getComputedStyle(document.documentElement).getPropertyValue(v).trim() || "";
}
