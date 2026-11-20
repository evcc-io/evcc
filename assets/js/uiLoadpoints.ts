import settings from "./settings";
import type { UiLoadpoint, SessionInfoKey, Loadpoint, Vehicle } from "./types/evcc";
import { distanceValue } from "./units";

const get = (id: string) => {
  if (!settings.loadpoints[id]) {
    settings.loadpoints[id] = {};
  }
  return settings.loadpoints[id];
};

export const vehicleHasSoc = (vehicle?: Vehicle): boolean => {
  return !vehicle?.features?.includes("Offline");
};

export const vehicleNotReachable = (vehicle?: Vehicle): boolean => {
  const features = vehicle?.features || [];
  return features.includes("Offline") && features.includes("Retryable");
};

export const convertToUiLoadpoints = (
  loadpoints: Loadpoint[],
  vehicles: Record<string, Vehicle>
): UiLoadpoint[] => {
  if (loadpoints.length === 0) return [];

  const mappedLoadpoints = loadpoints.map((lp, originalIndex) => {
    const vehicle = vehicles[lp.vehicleName] as Vehicle | undefined;
    const id = `${originalIndex + 1}`;

    const vehicleRange = lp.vehicleRange;
    const vehicleSoc = lp.vehicleSoc;
    const capacity = vehicle?.capacity || 0;
    const range = distanceValue(vehicleRange);
    const vehicleKnown = !!lp.vehicleName;
    const hasSoc = vehicleKnown && vehicleHasSoc(vehicle);
    const socBasedCharging = hasSoc || lp.vehicleSoc > 0;
    const socBasedPlanning = !!(socBasedCharging && capacity > 0);

    return {
      ...lp,
      id,
      range,
      vehicleRange,
      vehicleSoc,
      capacity,
      vehicleKnown,
      vehicleHasSoc: hasSoc,
      socBasedCharging,
      socBasedPlanning,
      displayTitle: vehicle?.title || lp.title || "Charging point",
      icon: lp.chargerIcon || vehicle?.icon || "car",
      order: getLoadpointOrder(id),
      visible: isLoadpointVisible(id),
      sessionInfo: getLoadpointSessionInfo(id),
      lastSmartCostLimit: getLoadpointLastSmartCostLimit(id),
      lastSmartFeedInPriorityLimit: getLoadpointLastSmartFeedInPriorityLimit(id),
      rangePerSoc:
        vehicleSoc > 10 && range ? Math.round((range / vehicleSoc) * 1e2) / 1e2 : undefined,
      socPerKwh: capacity > 0 ? 100 / capacity : 0,
      vehicleNotReachable: vehicleNotReachable(vehicle),
    } satisfies UiLoadpoint;
  });

  // Order: either as added (server order, per-device override wins) or by loadpoint priority.
  // `id` was captured as originalIndex + 1 *before* sorting, and every consumer reads it from the
  // object — so reordering the display can never misaddress an API call. The server order, the API
  // index and the MQTT topics stay untouched; this is presentation only.
  const manualOrder = (a: UiLoadpoint, b: UiLoadpoint) =>
    (a.order ?? Infinity) - (b.order ?? Infinity);
  const byPriority = settings.loadpointOrderMode === "priority";
  return mappedLoadpoints.sort((a, b) => {
    if (byPriority) {
      const priorityDiff = (b.priority ?? 0) - (a.priority ?? 0);
      if (priorityDiff !== 0) return priorityDiff;
    }
    return manualOrder(a, b);
  });
};

export const getLoadpointOrder = (id: string): number | null => {
  return get(id).order ?? null;
};

export const setLoadpointOrder = (orderedIds: string[]) => {
  // Update order for all loadpoints in the ordered list
  orderedIds.forEach((id, index) => {
    get(id).order = index;
  });
};

export const isLoadpointVisible = (id: string): boolean => {
  return get(id).visible ?? true; // Default to visible
};

export const setLoadpointVisibility = (id: string, visible: boolean) => {
  get(id).visible = visible;
};

export const getLoadpointSessionInfo = (id: string): SessionInfoKey | undefined => {
  return get(id).info;
};

export const getLoadpointLastSmartCostLimit = (id: string): number | undefined => {
  return get(id).lastSmartCostLimit;
};

export const setLoadpointSessionInfo = (id: string, value: SessionInfoKey) => {
  get(id).info = value;
};

export const setLoadpointLastSmartCostLimit = (id: string, value: number) => {
  get(id).lastSmartCostLimit = value;
};

export const getLoadpointLastSmartFeedInPriorityLimit = (id: string): number | undefined => {
  return get(id).lastSmartFeedInPriorityLimit;
};

export const setLoadpointLastSmartFeedInPriorityLimit = (id: string, value: number) => {
  get(id).lastSmartFeedInPriorityLimit = value;
};

export const resetLoadpointsOrder = () => {
  Object.keys(settings.loadpoints).forEach((id) => {
    if (settings.loadpoints[id]) {
      settings.loadpoints[id].order = undefined;
    }
  });
};

export const resetLoadpointsVisible = () => {
  Object.keys(settings.loadpoints).forEach((id) => {
    if (settings.loadpoints[id]) {
      settings.loadpoints[id].visible = undefined;
    }
  });
};
