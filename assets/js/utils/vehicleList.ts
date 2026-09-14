import type { Vehicle } from "@/types/evcc";

// convert the vehicles state record into a list, keys become vehicle names
export default function vehicleList(vehicles?: Record<string, Vehicle>): Vehicle[] {
  return Object.entries(vehicles || {}).map(([name, vehicle]) => ({ ...vehicle, name }));
}
