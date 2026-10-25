import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/invoice";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/receivepayment";
import "@h2d2/shopicons/es/regular/eco1";
import { markRaw, type Component } from "vue";
import LoadpointIcon from "../../MaterialIcon/Loadpoint.vue";
import HeaterIcon from "../../VehicleIcon/Heater.vue";
import ForecastGraphIcon from "../../MaterialIcon/ForecastGraph.vue";
import ConsumerIcon from "../../VehicleIcon/Generic.vue";
import type { OnboardingItem } from "@/types/evcc";

// same icons as the config page cards, shopicons as tag names
export const ONBOARDING_ICONS: Record<OnboardingItem, string | Component> = {
  battery: "shopicon-regular-batterythreequarters",
  pv: "shopicon-regular-sun",
  grid: "shopicon-regular-powersupply",
  loadpoint: markRaw(LoadpointIcon),
  heater: markRaw(HeaterIcon),
  consumer: markRaw(ConsumerIcon),
  gridImport: "shopicon-regular-invoice",
  gridExport: "shopicon-regular-receivepayment",
  co2: "shopicon-regular-eco1",
  solar: markRaw(ForecastGraphIcon),
};
