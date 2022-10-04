import Vehicle from "./Vehicle.vue";

export default {
  title: "Main/Vehicle",
  component: Vehicle,
  parameters: { backgrounds: { default: "box" } },
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Vehicle },
  template: '<Vehicle v-bind="args"></Vehicle>',
  parameters: { backgrounds: { default: "box" } },
});

export const Base = Template.bind({});
Base.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 42,
  vehicleRange: 231,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

export const Connected = Template.bind({});
Connected.args = {
  vehicleTitle: "Mein Auto",
  enabled: false,
  connected: true,
  vehiclePresent: true,
  charging: false,
  vehicleSoC: 66,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

export const ReadyToCharge = Template.bind({});
ReadyToCharge.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  charging: false,
  vehicleSoC: 66,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

export const Charging = Template.bind({});
Charging.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  charging: true,
  vehicleSoC: 66,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

const hoursFromNow = function (hours) {
  const now = new Date();
  now.setHours(now.getHours() + hours);
  return now.toISOString();
};

export const TargetChargePlanned = Template.bind({});
TargetChargePlanned.args = {
  vehicleTitle: "Mein Auto",
  enabled: false,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 31,
  minSoC: 20,
  charging: false,
  targetTimeActive: false,
  targetSoC: 45,
  targetTime: hoursFromNow(14),
  socBasedCharging: true,
  id: 0,
};

export const TargetChargeActive = Template.bind({});
TargetChargeActive.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 66,
  minSoC: 30,
  charging: true,
  targetTimeActive: true,
  targetSoC: 80,
  targetTime: hoursFromNow(2),
  socBasedCharging: true,
  id: 0,
};

export const MinCharge = Template.bind({});
MinCharge.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 17,
  minSoC: 20,
  charging: true,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

export const VehicleTargetSoc = Template.bind({});
VehicleTargetSoc.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleTargetSoC: 80,
  vehicleSoC: 66,
  charging: true,
  targetSoC: 90,
  socBasedCharging: true,
  id: 0,
};

export const TimerPvEnable = Template.bind({});
TimerPvEnable.args = {
  vehicleTitle: "Mein Auto",
  enabled: false,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 17,
  charging: false,
  targetSoC: 90,
  phaseAction: "inactive",
  phaseRemainingInterpolated: 0,
  pvAction: "enable",
  pvRemainingInterpolated: 32,
  socBasedCharging: true,
  id: 0,
};

export const TimerPvDisable = Template.bind({});
TimerPvDisable.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 17,
  charging: true,
  targetSoC: 90,
  phaseAction: "inactive",
  phaseRemainingInterpolated: 0,
  pvAction: "disable",
  pvRemainingInterpolated: 155,
  socBasedCharging: true,
  id: 0,
};

export const UnknownVehicleConnected = Template.bind({});
UnknownVehicleConnected.args = {
  enabled: false,
  connected: true,
  vehiclePresent: false,
  targetSoC: 90,
  socBasedCharging: false,
  id: 0,
};

export const UnknownVehicleReadyToCharge = Template.bind({});
UnknownVehicleReadyToCharge.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: false,
  charging: false,
  targetSoC: 100,
  socBasedCharging: false,
  id: 0,
};

export const UnknownVehicleCharging = Template.bind({});
UnknownVehicleCharging.args = {
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  vehiclePresent: false,
  charging: true,
  targetSoC: 90,
  socBasedCharging: false,
  id: 0,
};

export const OfflineVehicleCharging = Template.bind({});
OfflineVehicleCharging.args = {
  vehicleTitle: "Polestar 2",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleFeatureOffline: true,
  vehicleCapacity: 72,
  chargedEnergy: 14123,
  charging: true,
  targetSoC: 90,
  socBasedCharging: false,
  id: 0,
};

export const OfflineVehicleTargetEnergy = Template.bind({});
OfflineVehicleTargetEnergy.args = {
  vehicleTitle: "Polestar 2",
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleFeatureOffline: true,
  vehicleCapacity: 72,
  chargedEnergy: 14123,
  targetEnergy: 30,
  charging: true,
  socBasedCharging: false,
  id: 0,
};

export const Disconnected = Template.bind({});
Disconnected.args = {
  vehicleTitle: "Mein Auto",
  connected: false,
  vehiclePresent: false,
  targetSoC: 75,
  id: 0,
};

export const DisconnectedKnownSoc = Template.bind({});
DisconnectedKnownSoc.args = {
  vehicleTitle: "Mein Auto",
  connected: false,
  enabled: false,
  vehiclePresent: true,
  vehicleSoC: 17,
  targetSoC: 60,
  id: 0,
};

export const SwitchBetweenVehicles = Template.bind({});
SwitchBetweenVehicles.args = {
  vehicleTitle: "Weißes Model 3",
  vehicles: ["Blauer e-Golf", "Weißes Model 3"],
  enabled: true,
  connected: true,
  vehiclePresent: true,
  vehicleSoC: 42,
  vehicleRange: 231,
  targetSoC: 90,
  id: 0,
};
