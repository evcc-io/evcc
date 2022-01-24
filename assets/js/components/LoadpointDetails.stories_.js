import LoadpointDetails from "./LoadpointDetails.vue";
import i18n from "../i18n";
import "../tooltip";

export default {
  title: "Main/LoadpointDetails",
  component: LoadpointDetails,
  argTypes: {
    climater: { control: { type: "inline-radio", options: ["on", "heating", "cooling"] } },
  },
};

const Template = (args, { argTypes }) => ({
  i18n,
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
  vehiclePresent: true,
  chargeRemainingDuration: 5 * 3600,
};

export const VehicleRange = Template.bind({});
VehicleRange.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 5 * 3600,
};

export const VehicleClimater = Template.bind({});
VehicleClimater.args = {
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 5 * 3600,
  climater: "on",
};

export const TimerPhaseScaleDown = Template.bind({});
TimerPhaseScaleDown.args = {
  chargePower: 4400,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 5 * 3600,
  activePhases: 3,
  phaseAction: "scale1p",
  phaseRemaining: 80,
  pvAction: "enable",
  pvRemaining: 90,
};

export const TimerPhaseScaleUp = Template.bind({});
TimerPhaseScaleUp.args = {
  chargePower: 3900,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 5 * 3600,
  activePhases: 1,
  phaseAction: "scale3p",
  phaseRemaining: 25,
  pvAction: "inactive",
  pvRemaining: 0,
};

export const TimerPvDisable = Template.bind({});
TimerPvDisable.args = {
  chargePower: 1300,
  chargedEnergy: 4e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 5 * 3600,
  activePhases: 1,
  phaseAction: "inactive",
  phaseRemaining: 0,
  pvAction: "disable",
  pvRemaining: 55,
};

export const TimerPvEnable = Template.bind({});
TimerPvEnable.args = {
  chargePower: 0,
  chargedEnergy: 0,
  chargeDuration: 0,
  vehiclePresent: true,
  vehicleRange: 240.123,
  chargeRemainingDuration: 0,
  activePhases: 1,
  phaseAction: "inactive",
  phaseRemaining: 0,
  pvAction: "enable",
  pvRemaining: 55,
};
