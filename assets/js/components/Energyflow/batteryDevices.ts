import type { BatteryMeter } from "@/types/evcc";

export interface BatteryDeviceRow {
  device: BatteryMeter;
  depth: number;
}

export const batteryDeviceRows = (devices: BatteryMeter[]): BatteryDeviceRow[] => {
  const names = new Set(devices.flatMap((device) => (device.name ? [device.name] : [])));
  const children = new Map<string, BatteryMeter[]>();
  const roots: BatteryMeter[] = [];

  for (const device of devices) {
    const master = device.master;
    if (master && master !== device.name && names.has(master)) {
      const siblings = children.get(master) ?? [];
      siblings.push(device);
      children.set(master, siblings);
    } else {
      roots.push(device);
    }
  }

  const rows: BatteryDeviceRow[] = [];
  const visited = new Set<BatteryMeter>();
  const append = (device: BatteryMeter, depth: number) => {
    if (visited.has(device)) return;
    visited.add(device);
    rows.push({ device, depth });
    if (!device.name) return;
    for (const child of children.get(device.name) ?? []) {
      append(child, depth + 1);
    }
  };

  roots.forEach((device) => append(device, 0));
  devices.forEach((device) => append(device, 0));
  return rows;
};
