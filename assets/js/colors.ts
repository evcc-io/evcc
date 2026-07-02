import { reactive } from "vue";
import type { DeviceColorEntry, DeviceColors } from "./types/evcc";

export function deviceColorMap(list: DeviceColorEntry[] | undefined): DeviceColors {
  const m: DeviceColors = {};
  for (const { title, color } of list ?? []) m[title] = color;
  return m;
}

// alternatives
// const COLORS = [ "#40916C", "#52B788", "#74C69D", "#95D5B2", "#B7E4C7", "#D8F3DC", "#081C15", "#1B4332", "#2D6A4F"];
// const COLORS = ["#577590", "#43AA8B", "#90BE6D", "#F9C74F", "#F8961E", "#F3722C", "#F94144"];
// const COLORS = ["#0077b6", "#00b4d8", "#90e0ef", "#caf0f8", "#03045e"];
// const COLORS = [ "#0077B6FF", "#0096C7FF", "#00B4D8FF", "#48CAE4FF", "#90E0EFFF", "#ADE8F4FF", "#CAF0F8FF", "#03045EFF", "#023E8AFF",
// const COLORS = [ "#0077B6FF", "#00B4D8FF", "#90E0EFFF", "#40A578FF", "#9DDE8BFF", "#F8961EFF", "#F9C74FFF", "#E6FF94FF"];

const colors: {
  text: string | null;
  muted: string | null;
  border: string | null;
  self: string | null;
  grid: string | null;
  co2PerKWh: string | null;
  pricePerKWh: string | null;
  price: string | null;
  co2: string | null;
  export: string | null;
  background: string | null;
  light: string | null;
  selfPalette: string[];
  palette: string[];
  batteryPalette: string[];
} = reactive({
  text: null,
  muted: null,
  border: null,
  self: null,
  grid: null,
  co2PerKWh: null,
  pricePerKWh: null,
  price: null,
  co2: null,
  export: null,
  background: null,
  light: null,
  selfPalette: ["#0FDE41", "#FFBD2F", "#FD6158", "#03C1EF", "#0F662D", "#FF922E"],
  palette: [
    // light
    "#60A5FA", // blue
    "#FBBF24", // amber
    "#22D3EE", // cyan
    "#F472B6", // pink
    "#34D399", // green
    "#A78BFA", // violet
    "#94A3B8", // gray
    // mid
    "#2563EB", // blue
    "#F59E0B", // amber
    "#06B6D4", // cyan
    "#EC4899", // pink
    "#10B981", // green
    "#8B5CF6", // violet
    "#64748B", // gray
    // dark
    "#1E40AF", // blue
    "#B45309", // amber
    "#0E7490", // cyan
    "#BE185D", // pink
    "#047857", // green
    "#6D28D9", // violet
    "#334155", // gray
  ],
  batteryPalette: ["#0BA631", "#7FC41B", "#0A6E63", "#34D399", "#0E9E8F"],
});

// normalize 6-digit hex to 8-digit, then replace alpha
export const setAlpha = (color: string | null, alpha: string): string | undefined => {
  if (!color) return undefined;
  const c = color.trim().toLowerCase();
  // #rrggbb → append alpha, #rrggbbaa → replace alpha
  if (c.length === 7) return c + alpha;
  if (c.length === 9) return c.slice(0, 7) + alpha;
  return c;
};

// regex for raw hex (no leading #): 6 digits, used for input validation
export const HEX_RE = /^[0-9a-fA-F]{6}$/;

// normalize hex to uppercase 7-char #RRGGBB (strips alpha if present)
export const normalizeHex = (color?: string | null): string => {
  if (!color) return "";
  let c = color.trim().toUpperCase();
  if (!c.startsWith("#")) c = "#" + c;
  return c.slice(0, 7);
};

// override wins; rest get next free palette entry, wrap-around when exhausted
export function resolveColors(ids: string[], overrides: DeviceColors = {}): DeviceColors {
  const taken = new Set(Object.values(overrides).map(normalizeHex));
  const free = colors.palette.filter((c) => !taken.has(normalizeHex(c)));
  const result: DeviceColors = {};
  let idx = 0;
  for (const id of ids) {
    const ov = overrides[id];
    if (ov) {
      result[id] = ov;
    } else if (free.length) {
      result[id] = free[idx % free.length];
      idx++;
    } else {
      result[id] = colors.palette[idx % colors.palette.length];
      idx++;
    }
  }
  return result;
}

// dedicated battery palette, assigned by index (battery page + history battery group)
export function batteryColor(index: number): string {
  const p = colors.batteryPalette;
  return p[index % p.length] || "";
}

export const dimColor = (color: string | null) => setAlpha(color, "20");

export const lighterColor = (color: string | null) => setAlpha(color, "aa");

export const fullColor = (color: string | null) => setAlpha(color, "ff");

// Darken an opaque hex color: scale rgb toward black by `factor` (0-1), stays opaque.
export function darken(color: string, factor: number): string {
  if (!/^#[0-9a-f]{6}$/i.test(color)) return color;
  const f = Math.max(0, Math.min(1, factor));
  const channel = (o: number) =>
    Math.round(parseInt(color.slice(o, o + 2), 16) * f)
      .toString(16)
      .padStart(2, "0");
  return `#${channel(1)}${channel(3)}${channel(5)}`;
}

export function updateCssColors() {
  const style = window.getComputedStyle(document.documentElement);
  colors.text = style.getPropertyValue("--evcc-default-text");
  colors.muted = style.getPropertyValue("--bs-gray-medium");
  colors.border = style.getPropertyValue("--bs-border-color-translucent");
  colors.self = style.getPropertyValue("--evcc-self");
  colors.grid = style.getPropertyValue("--evcc-grid");
  colors.price = style.getPropertyValue("--evcc-price");
  colors.co2 = style.getPropertyValue("--evcc-co2");
  colors.export = style.getPropertyValue("--evcc-export-contrast");
  colors.background = style.getPropertyValue("--evcc-background");
  colors.pricePerKWh = style.getPropertyValue("--bs-gray-medium");
  colors.co2PerKWh = style.getPropertyValue("--bs-gray-medium");
  colors.light = style.getPropertyValue("--bs-gray-light");
}

// initialize colors
updateCssColors();
window.requestAnimationFrame(updateCssColors);

export default colors;
