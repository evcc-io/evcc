import api, { allowClientError } from "@/api";

// device classes exposed via GET /api/config/devices/{class}
const DEVICE_CLASSES = [
  "charger",
  "meter",
  "vehicle",
  "tariff",
  "hems",
  "circuit",
  "messenger",
  "curtailer",
] as const;

interface DeviceEntity {
  type?: string;
  deviceProduct?: string;
}

// normalize a product/type name for comparison, e.g. "Home Assistant" and
// "homeassistant" both become "homeassistant"
export function normalizeFeatureName(name: string): string {
  return name.toLowerCase().replace(/[^a-z0-9]/g, "");
}

// collects the product and type names of all devices the user actually configured.
// requires an authenticated session; resolves to an empty list otherwise (e.g. a
// password-protected instance viewed by a guest), so callers should treat an empty
// result as "unknown" rather than "nothing configured".
export async function fetchActiveFeatureNames(): Promise<string[]> {
  const lists = await Promise.all(
    DEVICE_CLASSES.map(async (cls) => {
      try {
        const { data, status } = await api.get(`config/devices/${cls}`, allowClientError);
        if (status !== 200 || !Array.isArray(data)) return [];
        return (data as DeviceEntity[]).flatMap((device) =>
          [device.deviceProduct, device.type].filter((v): v is string => !!v)
        );
      } catch {
        return [];
      }
    })
  );
  return lists.flat();
}
