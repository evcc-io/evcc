import Loadpoint from "./Loadpoint.vue";

export default {
  title: "Loadpoints/Loadpoint",
  component: Loadpoint,
  parameters: {
    layout: "centered",
  },
};

const baseState = {
  id: 0,
  pvConfigured: true,
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehicleTitle: "Mein Auto",
  vehicleName: "meinauto",
  enabled: true,
  connected: true,
  mode: "pv",
  charging: true,
  vehicleSoc: 66,
  limitSoc: 90,
  offeredCurrent: 7,
  minCurrent: 6,
  maxCurrent: 16,
  activePhases: 2,
};

const createStory = (props) => {
  const story = (args) => ({
    components: { Loadpoint },
    setup() {
      return { args };
    },
    template: '<Loadpoint v-bind="args" />',
  });
  story.args = { ...baseState, ...props };
  return story;
};

export const Standard = createStory({});

export const WithoutSoc = createStory({
  vehicleTitle: "",
  vehicleName: "",
});

export const Idle = createStory({
  enabled: false,
  connected: false,
  vehicleName: "",
  mode: "off",
  charging: false,
  offeredCurrent: 0,
});

export const DisabledLongTitle = createStory({
  title: "Charging point with a very very very long title!!!1!",
  remoteDisabled: "soft",
  remoteDisabledSource: "Sunny Home Manager",
  mode: "now",
  enabled: false,
  charging: false,
  chargePower: 0,
});

export const ChargerIconNoVehicle = createStory({
  chargerIcon: "heater",
  title: "Heating device with long name",
  mode: "now",
  chargerFeatureIntegratedDevice: true,
});
