import Loadpoint from "./Loadpoint.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import {
  SMART_COST_TYPE,
  CHARGE_MODE,
  CHARGER_STATUS_REASON,
  PHASE_ACTION,
  PV_ACTION,
} from "@/types/evcc";

export default {
  title: "Loadpoints/Loadpoint",
  component: Loadpoint,
  parameters: {
    layout: "centered",
  },
} as Meta<typeof Loadpoint>;

// Based on actual API state structure from demo.evcc.io
const baseState = {
  id: "1",
  title: "Carport",
  mode: CHARGE_MODE.PV,
  enabled: true,
  connected: true,
  socBasedCharging: true,
  charging: true,
  chargePower: 8050,
  chargedEnergy: 789.81,
  chargeDuration: 702,
  chargeRemainingDuration: 0,
  chargeRemainingEnergy: 0,
  vehicleName: "vehicle_3",
  vehicleSoc: 44,
  vehicleRange: 0,
  vehicleLimitSoc: 0,
  limitSoc: 80,
  effectiveLimitSoc: 80,
  limitEnergy: 0,
  minCurrent: 9,
  maxCurrent: 64,
  offeredCurrent: 34,
  phasesActive: 3,
  phasesConfigured: 3,
  sessionEnergy: 789.81,
  sessionPrice: 0.066,
  sessionPricePerKWh: 0.083,
  sessionCo2PerKWh: 11.813,
  sessionSolarPercentage: 95.312,
  batteryBoost: false,
  planActive: false,
  planEnergy: 0,
  planOverrun: 0,
  planPrecondition: 0,
  planProjectedEnd: undefined,
  planProjectedStart: undefined,
  planTime: undefined,
  priority: 0,
  phaseAction: PHASE_ACTION.INACTIVE,
  phaseRemaining: 0,
  pvAction: PV_ACTION.INACTIVE,
  pvRemaining: 0,
  smartCostActive: false,
  smartCostNextStart: undefined,
  smartFeedInPriorityActive: false,
  smartFeedInPriorityNextStart: undefined,
  vehicleClimaterActive: false,
  vehicleDetectionActive: false,
  vehicleWelcomeActive: false,
  chargerFeatureHeating: false,
  chargerFeatureIntegratedDevice: false,
  chargerIcon: "",
  chargerSinglePhase: false,
  chargerStatusReason: CHARGER_STATUS_REASON.UNKNOWN,
  connectedDuration: 0,
  // Global props that would typically come from parent
  vehicles: [
    {
      name: "vehicle_3",
      title: "grüner Honda e",
      icon: "car",
      capacity: 8,
      features: ["Offline"],
      repeatingPlans: [],
    },
    {
      name: "vehicle_4",
      title: "weißes Model 3",
      icon: "car",
      capacity: 80,
      features: ["Offline"],
      repeatingPlans: [],
    },
  ],
  smartCostType: SMART_COST_TYPE.PRICE_FORECAST,
  smartCostAvailable: true,
  smartFeedInPriorityAvailable: false,
  tariffGrid: 0.144,
  tariffCo2: 252,
  tariffFeedIn: 0.08,
  currency: "EUR",
  multipleLoadpoints: false,
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
};

const Template: StoryFn<typeof Loadpoint> = (args) => ({
  components: { Loadpoint },
  setup() {
    return { args };
  },
  template: '<Loadpoint v-bind="args" />',
});

export const Standard = Template.bind({});
Standard.args = baseState;

export const WithoutSoc = Template.bind({});
WithoutSoc.args = {
  ...baseState,
  vehicleName: "vehicle_4",
  vehicleSoc: 0,
  limitEnergy: 20,
};

export const Idle = Template.bind({});
Idle.args = {
  ...baseState,
  enabled: false,
  connected: false,
  vehicleName: "",
  vehicles: [],
  mode: CHARGE_MODE.OFF,
  charging: false,
  chargePower: 0,
  offeredCurrent: 0,
  sessionEnergy: 0,
  chargedEnergy: 0,
};

export const DisabledLongTitle = Template.bind({});
DisabledLongTitle.args = {
  ...baseState,
  title: "Charging point with a very very very long title!!!1!",
  remoteDisabled: "soft",
  remoteDisabledSource: "Sunny Home Manager",
  mode: CHARGE_MODE.NOW,
  enabled: false,
  charging: false,
  chargePower: 0,
};

export const ChargerIconNoVehicle = Template.bind({});
ChargerIconNoVehicle.args = {
  ...baseState,
  chargerIcon: "heater",
  title: "Heating device with long name",
  mode: CHARGE_MODE.NOW,
  chargerFeatureIntegratedDevice: true,
  vehicleName: "",
  vehicles: [],
};
