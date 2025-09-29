import settings from "./settings";
import type { UiLoadpoint, SessionInfoKey, Loadpoint, Vehicle } from "./types/evcc";

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
    const vehicle = vehicles[lp.vehicleName];
    const id = `${originalIndex + 1}`;

    return {
      ...lp,
      id,
      displayTitle: vehicle?.title || lp.title || "Charging point",
      icon: lp.chargerIcon || vehicle?.icon || "car",
      order: getLoadpointOrder(id),
      visible: isLoadpointVisible(id),
      sessionInfo: getLoadpointSessionInfo(id),
    };
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

export const setLoadpointSessionInfo = (id: string, value: SessionInfoKey) => {
  get(id).info = value;
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
