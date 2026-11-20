export const GROUP_ORDER = ["pv", "battery", "grid", "loadpoint", "consumer", "meter"] as const;

const BIDIRECTIONAL_GROUPS: ReadonlySet<string> = new Set(["grid", "battery"]);

// battery and grid are always bidirectional, additional meters when they export
export function isBidirectional(
  group: string,
  series: { data: { returnEnergy: number }[] }[]
): boolean {
  if (BIDIRECTIONAL_GROUPS.has(group)) return true;
  return group === "meter" && series.some((s) => s.data.some((slot) => slot.returnEnergy > 0));
}

// groups without their own label read like a meter
export function energyLabel(
  t: (key: string) => string,
  group: string,
  isTemp = false,
  direction: "energy" | "returnEnergy" = "energy"
): string {
  const key = `energy.direction.${group === "loadpoint" && isTemp ? "heating" : group}.${direction}`;
  const label = t(key);
  return label === key ? t(`energy.direction.meter.${direction}`) : label;
}

const COLOR_PICKER_GROUPS = ["loadpoint", "consumer", "meter"];

export function hasColorPicker(group: string): boolean {
  return COLOR_PICKER_GROUPS.includes(group);
}

const GROUP_COLOR_VAR: Record<string, string> = {
  pv: "--evcc-dark-green",
  forecast: "--evcc-dark-yellow",
  loadpoint: "--evcc-dark-green",
  grid: "--evcc-grid",
  battery: "--evcc-darker-green",
  consumer: "--evcc-price",
  meter: "--evcc-self",
};

export function groupColor(group: string): string {
  const v = GROUP_COLOR_VAR[group];
  if (!v) return "";
  return window.getComputedStyle(document.documentElement).getPropertyValue(v).trim() || "";
}
