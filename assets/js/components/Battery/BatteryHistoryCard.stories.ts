import BatteryHistoryCard from "./BatteryHistoryCard.vue";
import type { SocPoint, BatterySeries } from "./types";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Battery/BatteryHistoryCard",
  component: BatteryHistoryCard,
  parameters: {
    layout: "fullscreen",
    backgrounds: {
      default: "black",
      values: [{ name: "black", value: "#000" }],
    },
  },
} as Meta<typeof BatteryHistoryCard>;

const HOUR = 3600 * 1000;
const STEP = 15 * 60 * 1000; // 15min, matches the history/energy "15m" aggregate
const NOW = new Date("2026-07-01T12:30:00");

const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));
const smoothstep = (a: number, b: number, x: number) => {
  const t = clamp((x - a) / (b - a), 0, 1);
  return t * t * (3 - 2 * t);
};

// relative daily soc profile (hour -> 0..~1.1): flat low overnight, charge ramp late morning,
// overshoot midday so it plateaus at full, steep discharge in the evening. Not a sine.
const PROFILE: [number, number][] = [
  [0, 0.2],
  [5, 0.05],
  [8, 0.08],
  [11, 0.5],
  [14, 1.12],
  [17, 1.05],
  [19, 0.6],
  [21, 0.32],
  [24, 0.2],
];

function dayShape(hour: number): number {
  for (let i = 1; i < PROFILE.length; i++) {
    const [h0, v0] = PROFILE[i - 1]!;
    const [h1, v1] = PROFILE[i]!;
    if (hour <= h1) return v0 + (v1 - v0) * smoothstep(h0, h1, hour);
  }
  return PROFILE.at(-1)![1];
}

// per-battery soc timeline at 15min resolution, same shape as the /history/energy data.
// each battery shifts the daily rhythm and its floor/ceiling; a slow continuous wave adds
// day-to-day variation. clamping the overshoot creates realistic full/empty plateaus.
function generateTimeline(from: number, to: number, seed: number): SocPoint[] {
  const offset = seed * 1.6; // shift the daily rhythm per battery
  const floor = 8 + seed * 5;
  const ceil = 100 - seed * 8;
  const pts: SocPoint[] = [];
  for (let t = Math.ceil(from / STEP) * STEP; t <= to; t += STEP) {
    const d = new Date(t);
    const hour = (((d.getHours() + d.getMinutes() / 60 - offset) % 24) + 24) % 24;
    const drift = 6 * Math.sin(t / (37 * HOUR) + seed);
    const soc = floor + (ceil - floor) * clamp(dayShape(hour), 0, 1) + drift;
    pts.push({ t, soc: clamp(soc, 5, 100) });
  }
  return pts;
}

const TITLES = ["Sungrow", "Anker", "Fronius", "BYD"];
const CAPACITIES = [13.5, 7.5, 10, 5];

function buildBatteries(count: number, withForecast: boolean, now: number): BatterySeries[] {
  return Array.from({ length: count }, (_, i) => {
    const history = generateTimeline(now - 60 * HOUR, now, i);
    const title = TITLES[i] ?? `Battery ${i + 1}`;
    return {
      id: title,
      title,
      capacity: CAPACITIES[i] ?? 8,
      currentSoc: history.at(-1)?.soc ?? 50,
      history,
      forecast: withForecast ? generateTimeline(now, now + 36 * HOUR, i) : [],
    };
  });
}

function scenario(count: number, withForecast: boolean): StoryFn<typeof BatteryHistoryCard> {
  return () => ({
    components: { BatteryHistoryCard },
    setup() {
      return { batteries: buildBatteries(count, withForecast, NOW.getTime()), now: NOW };
    },
    template: `<div class="p-4" style="max-width: 900px">
      <BatteryHistoryCard :batteries="batteries" :now="now" kwh-available />
    </div>`,
  });
}

export const OneBattery = scenario(1, false);
export const OneBatteryForecast = scenario(1, true);

export const TwoBatteries = scenario(2, false);
export const TwoBatteriesForecast = scenario(2, true);

export const ThreeBatteries = scenario(3, false);
export const ThreeBatteriesForecast = scenario(3, true);

export const FourBatteries = scenario(4, false);
export const FourBatteriesForecast = scenario(4, true);
