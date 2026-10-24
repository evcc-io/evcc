import { describe, expect, test } from "vite-plus/test";
import type { BatteryMeter } from "@/types/evcc";
import { batteryDeviceRows } from "./batteryDevices";

const battery = (name: string, master?: string): BatteryMeter => ({
  name,
  title: name,
  power: 0,
  soc: 50,
  capacity: 10,
  controllable: true,
  master,
});

describe("batteryDeviceRows", () => {
  test("places slaves below their master regardless of input order", () => {
    const rows = batteryDeviceRows([
      battery("slave-a", "master"),
      battery("other"),
      battery("master"),
      battery("slave-b", "slave-a"),
    ]);

    expect(rows.map(({ device, depth }) => [device.name, depth])).toEqual([
      ["other", 0],
      ["master", 0],
      ["slave-a", 1],
      ["slave-b", 2],
    ]);
  });

  test("keeps missing masters and cycles visible without recursion loops", () => {
    const rows = batteryDeviceRows([
      battery("orphan", "missing"),
      battery("cycle-a", "cycle-b"),
      battery("cycle-b", "cycle-a"),
    ]);

    expect(rows.map(({ device, depth }) => [device.name, depth])).toEqual([
      ["orphan", 0],
      ["cycle-a", 0],
      ["cycle-b", 1],
    ]);
  });
});
