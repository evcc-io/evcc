import { describe, it, expect } from "vitest";
import { historyToSeries, forecastToSeries, buildChartBatteries } from "./history";
import type { BatteryMeter, EvOpt } from "@/types/evcc";
import type { BatteryHistorySeries } from "./types";

const t = (iso: string) => new Date(iso).getTime();

describe("historyToSeries", () => {
  it("keys by title, drops slots without socTemp, maps start to epoch", () => {
    const raw: BatteryHistorySeries[] = [
      {
        title: "Anker",
        group: "battery",
        data: [
          { start: "2026-01-01T00:00:00Z", socTemp: 50 },
          { start: "2026-01-01T00:15:00Z", socTemp: null },
          { start: "2026-01-01T00:30:00Z" },
          { start: "2026-01-01T00:45:00Z", socTemp: 55 },
        ],
      },
    ];
    const res = historyToSeries(raw);
    expect(res).toHaveLength(1);
    expect(res[0]!.key).toBe("Anker");
    expect(res[0]!.points).toEqual([
      { t: t("2026-01-01T00:00:00Z"), soc: 50 },
      { t: t("2026-01-01T00:45:00Z"), soc: 55 },
    ]);
  });

  it("falls back to group when title is missing", () => {
    const res = historyToSeries([{ group: "battery", data: [] }]);
    expect(res[0]!.key).toBe("battery");
  });
});

describe("forecastToSeries", () => {
  const evopt = {
    res: {
      batteries: [
        { state_of_charge: [4000, 5000, 6000] }, // Wh
        { state_of_charge: [1000, 2000, 3000] },
      ],
    },
    details: {
      timestamp: ["2026-01-01T00:00:00Z", "2026-01-01T01:00:00Z", "2026-01-01T02:00:00Z"],
      batteryDetails: [
        { type: "battery", name: "bat1", title: "Anker", capacity: 10 },
        { type: "vehicle", name: "car", title: "EV", capacity: 60 },
      ],
    },
  } as unknown as EvOpt;

  it("keys by device name and converts Wh to percent of capacity", () => {
    const res = forecastToSeries(evopt, t("2026-01-01T00:00:00Z"));
    expect(res).toHaveLength(1); // vehicle skipped
    expect(res[0]!.key).toBe("bat1");
    // 10 kWh -> 5000 Wh = 50 %
    expect(res[0]!.points).toEqual([
      { t: t("2026-01-01T00:00:00Z"), soc: 40 },
      { t: t("2026-01-01T01:00:00Z"), soc: 50 },
      { t: t("2026-01-01T02:00:00Z"), soc: 60 },
    ]);
  });

  it("drops points before now", () => {
    const res = forecastToSeries(evopt, t("2026-01-01T01:00:00Z"));
    expect(res[0]!.points.map((p) => p.soc)).toEqual([50, 60]);
  });

  it("returns empty for missing optimizer data", () => {
    expect(forecastToSeries(undefined, 0)).toEqual([]);
  });
});

describe("buildChartBatteries", () => {
  const devices = [
    { name: "bat1", title: "Anker", soc: 60, capacity: 10, controllable: false },
  ] as BatteryMeter[];

  it("matches history by title and forecast by device name", () => {
    const history = [{ key: "Anker", points: [{ t: 1, soc: 42 }] }];
    const forecast = [{ key: "bat1", points: [{ t: 2, soc: 80 }] }];
    const res = buildChartBatteries(devices, history, forecast);
    expect(res).toHaveLength(1);
    expect(res[0]).toMatchObject({
      id: "Anker",
      title: "Anker",
      capacity: 10,
      currentSoc: 60,
      history: [{ t: 1, soc: 42 }],
      forecast: [{ t: 2, soc: 80 }],
    });
  });

  it("yields empty timelines when nothing matches", () => {
    const res = buildChartBatteries(devices, [], []);
    expect(res[0]!.history).toEqual([]);
    expect(res[0]!.forecast).toEqual([]);
  });
});
