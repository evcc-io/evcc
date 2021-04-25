import Vehicle from "./Vehicle.vue";

export default {
  title: "Main/Vehicle",
  component: Vehicle,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Vehicle },
  template: '<Vehicle v-bind="$props"></Vehicle>',
});

export const Base = Template.bind({});
Base.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: true,
  socCharge: 42,
  targetSoC: 90,
};

export const Connected = Template.bind({});
Connected.args = {
  socTitle: "Mein Auto",
  enabled: false,
  connected: true,
  hasVehicle: true,
  charging: false,
  socCharge: 66,
  targetSoC: 90,
};

export const ReadyToCharge = Template.bind({});
ReadyToCharge.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: true,
  charging: false,
  socCharge: 66,
  targetSoC: 90,
};

export const Charging = Template.bind({});
Charging.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: true,
  charging: true,
  socCharge: 66,
  targetSoC: 90,
};

const hoursFromNow = function (hours) {
  const now = new Date();
  now.setHours(now.getHours() + hours);
  return now.toISOString();
};

export const TargetChargePlanned = Template.bind({});
TargetChargePlanned.args = {
  socTitle: "Mein Auto",
  enabled: false,
  connected: true,
  hasVehicle: true,
  socCharge: 31,
  minSoC: 20,
  charging: false,
  timerSet: true,
  timerActive: false,
  targetSoC: 45,
  targetTime: hoursFromNow(14),
};

export const TargetChargeActive = Template.bind({});
TargetChargeActive.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: true,
  socCharge: 66,
  minSoC: 30,
  charging: true,
  timerSet: true,
  timerActive: true,
  targetSoC: 80,
  targetTime: hoursFromNow(2),
};

export const MinCharge = Template.bind({});
MinCharge.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: true,
  socCharge: 17,
  minSoC: 20,
  charging: true,
  targetSoC: 90,
};

export const UnknownVehicleConnected = Template.bind({});
UnknownVehicleConnected.args = {
  socTitle: "Mein Auto",
  enabled: false,
  connected: true,
  hasVehicle: false,
};

export const UnknownVehicleReadyToCharge = Template.bind({});
UnknownVehicleReadyToCharge.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: false,
  charging: false,
};

export const UnknownVehicleCharging = Template.bind({});
UnknownVehicleCharging.args = {
  socTitle: "Mein Auto",
  enabled: true,
  connected: true,
  hasVehicle: false,
  charging: true,
};

export const Disconnected = Template.bind({});
Disconnected.args = {
  socTitle: "Mein Auto",
  connected: false,
  hasVehicle: false,
};

export const DisconnectedKnownSoc = Template.bind({});
DisconnectedKnownSoc.args = {
  socTitle: "Mein Auto",
  connected: false,
  enabled: false,
  hasVehicle: true,
  socCharge: 17,
  targetSoC: 60,
};
