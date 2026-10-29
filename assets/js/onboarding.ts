import { reactive } from "vue";
import api from "./api";
import store from "./store";
import auth from "./components/Auth/auth";
import type { OnboardingItem, OnboardingState } from "./types/evcc";
import type { ApiData, Template } from "./components/Config/DeviceModal";

export const ONBOARDING_ITEMS: OnboardingItem[] = [
  "battery",
  "pv",
  "grid",
  "loadpoint",
  "heater",
  "consumer",
  "gridImport",
  "gridExport",
  "co2",
  "solar",
];

function current(): OnboardingState {
  return store.state.onboarding ?? { done: false, dismissed: [] };
}

// wizard runs until finished or skipped, on fresh installs and when no admin password is set
export function wizardPending(): boolean {
  if (!auth.configured) return true;
  return !current().done && !!store.state.setupRequired;
}

export function isDismissed(item: OnboardingItem): boolean {
  return current().dismissed.includes(item);
}

async function save(patch: Partial<OnboardingState>) {
  const res = await api.put("/config/onboarding", { ...current(), ...patch });
  store.state.onboarding = res.data;
}

// no solar system means no export tariff and no solar forecast either
export function dismiss(item: OnboardingItem) {
  const implied: OnboardingItem[] = item === "pv" ? ["gridExport", "solar"] : [];
  const dismissed = new Set([...current().dismissed, item, ...implied]);
  return save({ dismissed: [...dismissed] });
}

export function finish() {
  return save({ done: true });
}

// devices created in this browser tab, offered as prefill for other usages of the same template
export interface CachedDevice {
  product: string;
  template: string;
  usages: string[];
  // connection values only, usage specific ones (capacity, maxacpower, ...) are stripped
  values: ApiData;
}

const STORAGE_KEY = "onboarding.devices";

function load(): CachedDevice[] {
  try {
    return JSON.parse(sessionStorage.getItem(STORAGE_KEY) ?? "[]");
  } catch {
    return [];
  }
}

export const sessionDevices = reactive(load());

function persist() {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(sessionDevices));
}

// hybrid inverters serve several of these, other meter types are standalone
const REUSE_USAGES = ["grid", "pv", "battery"];

export function remember(
  deviceType: string,
  product: string,
  template: Template | null,
  values: ApiData
) {
  if (deviceType !== "meter" || !template || !REUSE_USAGES.includes(values["usage"] ?? "")) return;
  const usageSpecific = new Set(template.Params.filter((p) => p.Usages).map((p) => p.Name));
  const shared = Object.fromEntries(
    Object.entries(values).filter(([key]) => !usageSpecific.has(key))
  ) as ApiData;
  delete shared["deviceTitle"];
  delete shared["deviceIcon"];
  sessionDevices.push({
    product,
    template: values["template"],
    usages: template.Params.find((p) => p.Name === "usage")?.Choice ?? [],
    values: shared,
  });
  persist();
}

// templates that support the usage but are not saved for it yet
export function suggestions(usage?: string): CachedDevice[] {
  if (!usage || !REUSE_USAGES.includes(usage)) return [];
  return sessionDevices.filter(
    (d) =>
      d.usages.includes(usage) &&
      !sessionDevices.some((o) => o.template === d.template && o.values["usage"] === usage)
  );
}

export function prefillFor(entry: CachedDevice, usage: string): ApiData {
  return { ...entry.values, usage: usage as ApiData["usage"] };
}

export function clearSession() {
  sessionDevices.splice(0);
  persist();
}
