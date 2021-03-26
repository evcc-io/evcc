import LoadpointDetails from "./LoadpointDetails.vue";

export default {
  title: "Main/LoadpointDetails",
  component: LoadpointDetails,
  argTypes: {
    climater: { control: { type: "inline-radio", options: ["on", "heating", "cooling"] } },
  },
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { LoadpointDetails },
  template: '<LoadpointDetails v-bind="$props"></LoadpointDetails>',
});

export const Base = Template.bind({});
Base.args = {};

export const Charging = Template.bind({});
Charging.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
};

export const Vehicle = Template.bind({});
Vehicle.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  hasVehicle: true,
  chargeEstimate: 5 * 3600,
};

export const VehicleRange = Template.bind({});
VehicleRange.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  hasVehicle: true,
  range: 240.123,
  chargeEstimate: 5 * 3600,
};

export const VehicleClimater = Template.bind({});
VehicleClimater.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  hasVehicle: true,
  range: 240.123,
  chargeEstimate: 5 * 3600,
  climater: "on",
};

export const VehicleTimer = Template.bind({});
VehicleTimer.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  hasVehicle: true,
  range: 240.123,
  chargeEstimate: 5 * 3600,
  timerSet: true,
  timerActive: true,
};
