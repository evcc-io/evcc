import { reactive } from "vue";

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
  selfPalette: ["#0FDE41FF", "#FFBD2FFF", "#FD6158FF", "#03C1EFFF", "#0F662DFF", "#FF922EFF"],
  palette: [
    // Dynamic palette (vehicles, loadpoints, …): optimized for neighbor contrast, avoids overlap with system colors (solar, battery, grid, …).
    "#06B6D4FF", // Cyan
    "#2B7FFFFF", // Blue
    "#6366F1FF", // Indigo
    "#A855F7FF", // Violet
    "#D946EFFF", // Magenta
    "#EC4899FF", // Pink
    "#FB7185FF", // Coral
    "#475569FF", // Slate
    "#1E40AFFF", // Royal
    "#6D28D9FF", // Purple
    "#BE185DFF", // Crimson
    "#D2691EFF", // Sienna
    "#67E8F9FF", // Glacier
    "#C084FCFF", // Lilac
    "#FFD580FF", // Sand
    "#94A3B8FF", // Steel
  ],
});

// normalize 6-digit hex to 8-digit, then replace alpha
const setAlpha = (color: string | null, alpha: string): string | undefined => {
  if (!color) return undefined;
  const c = color.trim().toLowerCase();
  // #rrggbb → append alpha, #rrggbbaa → replace alpha
  if (c.length === 7) return c + alpha;
  if (c.length === 9) return c.slice(0, 7) + alpha;
  return c;
};

export const dimColor = (color: string | null) => setAlpha(color, "20");

export const lighterColor = (color: string | null) => setAlpha(color, "aa");

export const fullColor = (color: string | null) => setAlpha(color, "ff");

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
