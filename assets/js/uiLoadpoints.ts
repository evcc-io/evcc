import settings from "./settings";
import type { UiLoadpoint, SessionInfoKey, Loadpoint, Vehicle } from "./types/evcc";
import { distanceValue } from "./units";

const get = (id: string) => {
  if (!settings.loadpoints[id]) {
    settings.loadpoints[id] = {};
  }
  return settings.loadpoints[id];
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
    const vehicleHasSoc = vehicleKnown && !vehicle?.features?.includes("Offline");
    const socBasedCharging = vehicleHasSoc || lp.vehicleSoc > 0;
    const socBasedPlanning = !!(socBasedCharging && capacity > 0);

    return {
      ...lp,
      id,
      range,
      vehicleRange,
      vehicleSoc,
      capacity,
      vehicleKnown,
      vehicleHasSoc,
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
      vehicleNotReachable:
        (vehicle?.features || []).includes("Offline") &&
        (vehicle?.features || []).includes("Retryable"),
    } satisfies UiLoadpoint;
  });

  // Sort by order (loadpoints with no order go to the end)
  return mappedLoadpoints.sort((a, b) => (a.order ?? Infinity) - (b.order ?? Infinity));
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
