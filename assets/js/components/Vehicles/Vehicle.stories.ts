import { CHARGE_MODE } from "@/types/evcc";
import Vehicle from "./Vehicle.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

function getFutureTime(hours: number, minutes: number) {
  const now = new Date();
  now.setHours(now.getHours() + hours);
  now.setMinutes(now.getMinutes() + minutes);
  return now.toISOString();
}

const baseState = {
  vehicle: {
    title: "Mein Auto",
    icon: "car",
    capacity: 72,
    features: [],
    name: "",
    repeatingPlans: [],
    planStrategy: {
      continuous: false,
      precondition: 0,
    },
  },
  enabled: false,
  connected: true,
  vehicleName: "meinauto",
  vehicleSoc: 42.742,
  vehicleRange: 231,
  limitSoc: 90,
  chargedEnergy: 14123,
  socBasedCharging: true,
  id: 0,
};

export default {
  title: "Vehicles/Vehicle",
  component: Vehicle,
  argTypes: {
    chargedEnergy: { control: "number" },
    charging: { control: "boolean" },
    vehicleClimaterActive: { control: "boolean" },
    vehicleWelcomeActive: { control: "boolean" },
    connected: { control: "boolean" },
    currency: { control: "text" },
    effectiveLimitSoc: { control: "number" },
    effectivePlanSoc: { control: "number" },
    effectivePlanTime: { control: "text" },
    batteryBoostActive: { control: "boolean" },
    enabled: { control: "boolean" },
    heating: { control: "boolean" },
    id: { control: "text" },
    integratedDevice: { control: "boolean" },
    limitEnergy: { control: "number" },
    mode: { control: "text" },
    chargerStatusReason: { control: "text" },
    vehicle: { control: "object" },
    vehicleName: { control: "text" },
    vehicleRange: { control: "number" },
    vehicleSoc: { control: "number" },
    vehicleLimitSoc: { control: "number" },
    socBasedCharging: { control: "boolean" },
  },
} as Meta<typeof Vehicle>;

// white box background as in the app's loadpoint card
const cardStyle = "padding: 1rem; background: var(--evcc-box); border-radius: 0.5rem;";

const Template: StoryFn<typeof Vehicle> = (args) => ({
  components: { Vehicle },
  setup() {
    return { args, cardStyle };
  },
  template: '<div :style="cardStyle"><Vehicle v-bind="args" /></div>',
});

export const Disconnected = Template.bind({});
Disconnected.args = {
  ...baseState,
  connected: false,
};

export const Connected = Template.bind({});
Connected.args = {
  ...baseState,
};

export const Ready = Template.bind({});
Ready.args = {
  ...baseState,
  enabled: true,
};

export const Charging = Template.bind({});
Charging.args = {
  ...baseState,
  enabled: true,
  charging: true,
};

export const UnknownVehicle = Template.bind({});
UnknownVehicle.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleName: "",
  socBasedCharging: false,
  vehicle: { ...baseState.vehicle, capacity: undefined },
  mode: CHARGE_MODE.PV,
};

export const OfflineVehicle = Template.bind({});
OfflineVehicle.args = {
  ...baseState,
  enabled: true,
  charging: true,
  socBasedCharging: false,
  vehicle: {
    ...baseState.vehicle,
    title: "Opel Corsa-e",
    capacity: 72,
    features: ["Offline"],
  },
  mode: CHARGE_MODE.PV,
};

export const OfflineVehicleWithTarget = Template.bind({});
OfflineVehicleWithTarget.args = {
  ...baseState,
  enabled: true,
  charging: true,
  socBasedCharging: false,
  vehicle: {
    ...baseState.vehicle,
    title: "Opel Corsa-e",
    capacity: 72,
    features: ["Offline"],
  },
  mode: CHARGE_MODE.PV,
};

export const WaitingForAuthorization = Template.bind({});
WaitingForAuthorization.args = {
  ...baseState,
  enabled: true,
  chargerStatusReason: "waitingforauthorization",
};

export const VehicleLimit = Template.bind({});
VehicleLimit.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleLimitSoc: 80,
};

export const VehicleLimitReached = Template.bind({});
VehicleLimitReached.args = {
  ...baseState,
  enabled: true,
  vehicleLimitSoc: 80,
  vehicleSoc: 80,
};

export const MinSocCharging = Template.bind({});
MinSocCharging.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleSoc: 17.3,
  vehicleRange: 92,
  minSocNotReached: true,
  effectiveMinSoc: 30,
};

export const HeatingMinTemp = Template.bind({});
HeatingMinTemp.args = {
  ...baseState,
  vehicle: { ...baseState.vehicle, title: "Warmwasser", icon: "waterheater" },
  heating: true,
  integratedDevice: true,
  enabled: true,
  charging: true,
  vehicleSoc: 38,
  vehicleRange: 0,
  effectiveLimitSoc: 60,
  minSocNotReached: true,
  effectiveMinSoc: 40,
  ui: { minTemp: 35, maxTemp: 70 },
};

export const TargetChargePlanned = Template.bind({});
TargetChargePlanned.args = {
  ...baseState,
  mode: CHARGE_MODE.PV,
};

export const TargetChargeActive = Template.bind({});
TargetChargeActive.args = {
  ...baseState,
  enabled: true,
  charging: true,
  mode: CHARGE_MODE.PV,
};

export const SmartChargeCostLimitActive = Template.bind({});
SmartChargeCostLimitActive.args = {
  ...baseState,
  enabled: true,
  charging: true,
  smartCostLimit: 0.13,
  mode: CHARGE_MODE.PV,
};

export const SuggestionCharge = Template.bind({});
SuggestionCharge.args = {
  ...baseState,
  suggestion: { action: "charge", actionable: true },
};

export const SuggestionPause = Template.bind({});
SuggestionPause.args = {
  ...baseState,
  enabled: true,
  charging: true,
  suggestion: { action: "stop", actionable: true },
};

export const SuggestionCombination = Template.bind({});
SuggestionCombination.args = {
  ...baseState,
  enabled: true,
  charging: true,
  suggestion: { action: "stop", actionable: true },
  currency: "EUR",
  tariffGrid: 0.32,
  smartCostLimit: 0.12,
  smartCostType: "price",
  planActive: true,
  effectivePlanTime: getFutureTime(6, 54),
  planProjectedEnd: getFutureTime(5, 43),
};

export const PvEnableTimer = Template.bind({});
PvEnableTimer.args = {
  ...baseState,
  pvAction: "enable",
  pvRemainingInterpolated: 32,
};

export const PvDisableTimer = Template.bind({});
PvDisableTimer.args = {
  ...baseState,
  enabled: true,
  charging: true,
  pvAction: "disable",
  pvRemainingInterpolated: 155,
};

export const VehicleSwitch = Template.bind({});
VehicleSwitch.args = {
  ...baseState,
  vehicles: ["Blauer e-Golf", "Weißes Model 3"],
};
