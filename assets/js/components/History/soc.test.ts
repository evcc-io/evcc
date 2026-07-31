import { describe, expect, it } from "vite-plus/test";
import { batteryCapacities, totalSocByStart } from "./soc";
import type { HistorySeries } from "./GroupChart.vue";
import type { BatteryMeter } from "@/types/evcc";

const t0 = "2026-01-01T00:00:00Z";
const t1 = "2026-01-01T00:15:00Z";

function series(title: string, socs: (number | null)[]): HistorySeries {
  return {
    title,
    group: "battery",
    data: socs.map((soc, i) => ({
      start: i === 0 ? t0 : t1,
      end: t1,
      energy: 0,
      returnEnergy: 0,
      socTemp: soc,
    })),
  };
}

const devices = [
  { title: "Home", capacity: 10 },
  { name: "garage", capacity: 5 },
] as BatteryMeter[];

describe("totalSocByStart", () => {
  it("weighs soc by capacity", () => {
    const res = totalSocByStart(
      [series("Home", [50, 40]), series("garage", [80, 60])],
      batteryCapacities(devices)
    );
    // (50%*10 + 80%*5) / 15 = 60%, (40%*10 + 60%*5) / 15 = ~46.7%
    expect(res.get(t0)).toBeCloseTo(60);
    expect(res.get(t1)).toBeCloseTo(46.67, 1);
  });

  it("skips slots not reported by every battery", () => {
    const res = totalSocByStart(
      [series("Home", [50, 40]), series("garage", [80, null])],
      batteryCapacities(devices)
    );
    expect(res.get(t0)).toBeCloseTo(60);
    expect(res.has(t1)).toBe(false);
  });

  it("is empty without a capacity for every battery", () => {
    const res = totalSocByStart(
      [series("Home", [50]), series("unknown", [80])],
      batteryCapacities(devices)
    );
    expect(res.size).toBe(0);
  });
});
