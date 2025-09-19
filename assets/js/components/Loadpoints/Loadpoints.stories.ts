import Loadpoints from "./Loadpoints.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import { SMART_COST_TYPE } from "@/types/evcc";

// Create LoadpointCompact structure for the Loadpoints component
const createLoadpoint = (opts = {}) => {
  const base = {
    icon: "car",
    title: "Carport",
    charging: true,
    soc: 66,
    power: 8050,
    chargePower: 8050,
    connected: true,
    index: 0,
    vehicleName: "vehicle_3",
    chargerIcon: "",
    vehicleSoc: 0,
    chargerFeatureHeating: false,
  };
  return { ...base, ...opts };
};

export default {
  title: "Loadpoints/Loadpoints",
  component: Loadpoints,
  parameters: {
    layout: "centered",
  },
} as Meta<typeof Loadpoints>;

const Template: StoryFn<typeof Loadpoints> = (args) => ({
  components: { Loadpoints },
  setup() {
    return { args };
  },
  template: '<Loadpoints v-bind="args" />',
});

const baseArgs = {
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
    {
      name: "vehicle_5",
      title: "schwarzes VanMoof",
      icon: "bike",
      capacity: 0.46,
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
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
};

export const OneLoadpoint = Template.bind({});
OneLoadpoint.args = {
  ...baseArgs,
  loadpoints: [
    createLoadpoint({
      title: "Main Charger",
      index: 0,
      vehicleName: "vehicle_3",
      charging: true,
      chargePower: 11000,
      power: 11000,
      connected: true,
      soc: 75,
      vehicleSoc: 75,
    }),
  ],
};

export const TwoLoadpoints = Template.bind({});
TwoLoadpoints.args = {
  ...baseArgs,
  loadpoints: [
    createLoadpoint({
      title: "Garage",
      index: 0,
      vehicleName: "vehicle_3",
      charging: true,
      chargePower: 8050,
      power: 8050,
      connected: true,
      soc: 66,
      vehicleSoc: 66,
    }),
    createLoadpoint({
      title: "Charging point with a very very very long title that tests layout compatibility!!!",
      index: 1,
      vehicleName: "vehicle_4",
      charging: false,
      chargePower: 0,
      power: 0,
      connected: true,
      soc: 89,
      vehicleSoc: 89,
    }),
  ],
};

export const ThreeLoadpoints = Template.bind({});
ThreeLoadpoints.args = {
  ...baseArgs,
  loadpoints: [
    createLoadpoint({
      title: "Garage",
      index: 0,
      vehicleName: "vehicle_3",
      charging: true,
      chargePower: 7400,
      power: 7400,
      connected: true,
      soc: 45,
      vehicleSoc: 45,
    }),
    createLoadpoint({
      title:
        "Extremely long charging point title that should test responsive layout behavior and text wrapping",
      index: 1,
      vehicleName: "vehicle_4",
      charging: false,
      chargePower: 0,
      power: 0,
      connected: true,
      soc: 92,
      vehicleSoc: 92,
    }),
    createLoadpoint({
      title: "Bike Charger",
      index: 2,
      vehicleName: "vehicle_5",
      charging: false,
      chargePower: 0,
      power: 0,
      connected: false,
      icon: "bike",
      soc: undefined,
      vehicleSoc: 0,
    }),
  ],
};
