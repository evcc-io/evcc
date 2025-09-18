import { reactive, watch } from "vue";
import type { LoadpointCompact } from "@/types/evcc";

export interface LoadpointDisplay {
  index: number;
  visible: boolean;
}

export interface LoadpointDisplayItem extends LoadpointDisplay {
  title: string;
  originalData: DisplayLoadpoint;
}

export type LoadpointWithId = LoadpointCompact & { id: number };

export type DisplayLoadpoint = LoadpointCompact & {
  id: number;
  displayTitle: string;
};

interface LoadpointSettings {
  order: number[];
  visibility: { [key: number]: boolean };
}

const STORAGE_KEY = "settings_lp_order_visibility";

const getStoredSettings = (): LoadpointSettings => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      return JSON.parse(stored);
    }
  } catch (error) {
    console.warn("Failed to parse stored loadpoint settings:", error);
  }
  return { order: [], visibility: {} };
};

const saveSettings = (settings: LoadpointSettings) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch (error) {
    console.warn("Failed to save loadpoint settings:", error);
  }
};

const initialSettings = getStoredSettings();

const loadpointSettings = reactive({
  order: initialSettings.order,
  visibility: initialSettings.visibility,
});

// Watch for changes and persist to localStorage
watch(
  loadpointSettings,
  (newSettings) => {
    saveSettings({
      order: newSettings.order,
      visibility: newSettings.visibility,
    });
  },
  { deep: true }
);

const initializeLoadpointSettings = (loadpointCount: number) => {
  // Initialize order if not set or if loadpoint count changed
  if (loadpointSettings.order.length === 0 || loadpointSettings.order.length !== loadpointCount) {
    loadpointSettings.order = Array.from({ length: loadpointCount }, (_, i) => i);
  }

  // Initialize visibility if not set (default to visible)
  for (let i = 0; i < loadpointCount; i++) {
    if (!(i in loadpointSettings.visibility)) {
      loadpointSettings.visibility[i] = true;
    }
  }
};

const getLoadpointOrder = () => loadpointSettings.order;

const setLoadpointOrder = (order: number[]) => {
  loadpointSettings.order = [...order];
};

const getLoadpointVisibility = (index: number): boolean => {
  return loadpointSettings.visibility[index] ?? true;
};

const setLoadpointVisibility = (index: number, visible: boolean) => {
  loadpointSettings.visibility[index] = visible;
};

const convertToDisplayLoadpoints = (loadpoints: LoadpointCompact[]): DisplayLoadpoint[] => {
  return loadpoints.map((loadpoint, index) => ({
    ...loadpoint,
    id: index + 1,
    displayTitle: loadpoint.title || "Charging point", // Using hardcoded fallback for now, could use i18n later
  }));
};

const filterAndSortDisplayLoadpoints = (
  displayLoadpoints: DisplayLoadpoint[]
): DisplayLoadpoint[] => {
  if (displayLoadpoints.length === 0) return [];

  initializeLoadpointSettings(displayLoadpoints.length);

  return loadpointSettings.order
    .filter((index) => index < displayLoadpoints.length && getLoadpointVisibility(index))
    .map((index) => displayLoadpoints[index])
    .filter((item): item is DisplayLoadpoint => item !== undefined);
};

const getDisplayLoadpointList = (displayLoadpoints: DisplayLoadpoint[]): LoadpointDisplayItem[] => {
  if (displayLoadpoints.length === 0) return [];

  initializeLoadpointSettings(displayLoadpoints.length);

  return loadpointSettings.order
    .filter((index) => index < displayLoadpoints.length)
    .map((index) => {
      const loadpoint = displayLoadpoints[index];
      if (!loadpoint) {
        throw new Error(`Loadpoint at index ${index} is undefined`);
      }
      return {
        index,
        visible: getLoadpointVisibility(index),
        title: loadpoint.displayTitle,
        originalData: loadpoint,
      };
    });
};

const getOrderedVisibleLoadpoints = (loadpoints: LoadpointCompact[]): LoadpointWithId[] => {
  const displayLoadpoints = convertToDisplayLoadpoints(loadpoints);
  return filterAndSortDisplayLoadpoints(displayLoadpoints);
};

const getLoadpointDisplayList = (loadpoints: LoadpointCompact[]): LoadpointDisplayItem[] => {
  const displayLoadpoints = convertToDisplayLoadpoints(loadpoints);
  return getDisplayLoadpointList(displayLoadpoints);
};

export {
  loadpointSettings,
  initializeLoadpointSettings,
  getLoadpointOrder,
  setLoadpointOrder,
  getLoadpointVisibility,
  setLoadpointVisibility,
  convertToDisplayLoadpoints,
  filterAndSortDisplayLoadpoints,
  getDisplayLoadpointList,
  getOrderedVisibleLoadpoints,
  getLoadpointDisplayList,
};
