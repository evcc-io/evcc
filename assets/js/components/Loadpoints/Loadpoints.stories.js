import Loadpoints from "./Loadpoints.vue";

function loadpoint(opts = {}) {
  const base = {
    id: 0,
    pvConfigured: true,
    chargePower: 2800,
    chargedEnergy: 11e3,
    chargeDuration: 95 * 60,
    vehicleName: "tesla",
    vehicleTitle: "Tesla Model 3",
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
  return { ...base, ...opts };
}

export default {
  title: "Loadpoints/Loadpoints",
  component: Loadpoints,
  parameters: {
    layout: "centered",
  },
};

const Template = (args) => ({
  components: { Loadpoints },
  setup() {
    return { args };
  },
  template: '<Loadpoints v-bind="args" />',
});

export const Standard = Template.bind({});
Standard.args = {
  vehicles: [],
  loadpoints: [
    loadpoint({ title: "Carport" }),
    loadpoint({
      title: "Water Heater",
      chargerFeatureIntegratedDevice: true,
      chargerIcon: "waterheater",
    }),
  ],
};
