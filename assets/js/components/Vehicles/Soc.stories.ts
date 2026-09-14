import Soc from "./Soc.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

// non-heating SoC bar (percentage scale). current soc steps up across the
// variants; the heating variants use the matching position on the 20..60 scale.
const baseState = {
  connected: true,
  socBasedCharging: true,
  socBasedPlanning: true,
  vehicleSoc: 40,
  effectiveLimitSoc: 90, // manual limit
  vehicleLimitSoc: 0,
  enabled: false,
  charging: false,
  heating: false,
};

// heating bar: vehicleSoc is the current temperature, ui defines the scale
// (20..60), effectiveLimitSoc / vehicleLimitSoc are target temperatures.
// temp = 20 + soc * 0.4, so positions line up with the charging column.
const heatingBase = {
  ...baseState,
  heating: true,
  ui: { minTemp: 20, maxTemp: 60 },
  vehicleSoc: 36, // ~40%
  effectiveLimitSoc: 56, // manual limit, ~90%
};

export default {
  title: "Vehicles/Soc",
  component: Soc,
  argTypes: {
    connected: { control: "boolean" },
    enabled: { control: "boolean" },
    charging: { control: "boolean" },
    heating: { control: "boolean" },
    socBasedCharging: { control: "boolean" },
    socBasedPlanning: { control: "boolean" },
    vehicleSoc: { control: "number" },
    vehicleLimitSoc: { control: "number" },
    effectiveLimitSoc: { control: "number" },
    effectivePlanSoc: { control: "number" },
    minSoc: { control: "number" },
    minSocNotReached: { control: "boolean" },
    ui: { control: "object" },
  },
} as Meta<typeof Soc>;

// give the bar a fixed width and the card background (var(--evcc-box)) it lives on
// in the app, so the grey track (var(--evcc-background)) reads correctly
const cardStyle =
  "width: 340px; padding: 1rem; background: var(--evcc-box); border-radius: 0.5rem;";

const Template: StoryFn<typeof Soc> = (args) => ({
  components: { Soc },
  setup() {
    return { args, cardStyle };
  },
  template: '<div :style="cardStyle"><Soc v-bind="args" /></div>',
});

// --- charging (percentage) variants, current soc increasing ---

export const MinSocNotReached = Template.bind({});
MinSocNotReached.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleSoc: 10,
  minSoc: 20,
  minSocNotReached: true,
};

// very low / low current soc, actively charging
export const ChargingVeryLow = Template.bind({});
ChargingVeryLow.args = { ...baseState, enabled: true, charging: true, vehicleSoc: 5 };

export const ChargingLow = Template.bind({});
ChargingLow.args = { ...baseState, enabled: true, charging: true, vehicleSoc: 20 };

export const Connected = Template.bind({});
Connected.args = { ...baseState, vehicleSoc: 30 };

export const Ready = Template.bind({});
Ready.args = { ...baseState, enabled: true, vehicleSoc: 45 };

export const Charging = Template.bind({});
Charging.args = { ...baseState, enabled: true, charging: true, vehicleSoc: 60 };

// manual limit (slider) below the vehicle's own limit -> vehicle marker sits right of the thumb
export const VehicleLimitManualBelow = Template.bind({});
VehicleLimitManualBelow.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleSoc: 65,
  vehicleLimitSoc: 80,
  effectiveLimitSoc: 70,
};

// manual limit above the vehicle's own limit -> fill clamps at the vehicle marker, thumb beyond
export const VehicleLimitManualAbove = Template.bind({});
VehicleLimitManualAbove.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleSoc: 70,
  vehicleLimitSoc: 80,
  effectiveLimitSoc: 90,
};

// current soc nearly at the top of the scale, limit raised to make room
export const NearlyFull = Template.bind({});
NearlyFull.args = {
  ...baseState,
  enabled: true,
  charging: true,
  vehicleSoc: 96,
  effectiveLimitSoc: 100,
};

export const Disconnected = Template.bind({});
Disconnected.args = { ...baseState, connected: false };

// no vehicle SoC: energy-based bar (chargedEnergy vs limitEnergy), no slider
export const NoSoc = Template.bind({});
NoSoc.args = {
  ...baseState,
  socBasedCharging: false,
  socBasedPlanning: false,
  enabled: true,
  charging: true,
  chargedEnergy: 14000,
};

// --- heating (temperature range) variants, current temp increasing ---

export const MinTempNotReached = Template.bind({});
MinTempNotReached.args = {
  ...heatingBase,
  enabled: true,
  charging: true,
  vehicleSoc: 24,
  minSoc: 28,
  minSocNotReached: true,
};

// very low / low current temp, actively heating (temp = 20 + soc% * 0.4)
export const HeatingChargingVeryLow = Template.bind({});
HeatingChargingVeryLow.args = { ...heatingBase, enabled: true, charging: true, vehicleSoc: 22 };

export const HeatingChargingLow = Template.bind({});
HeatingChargingLow.args = { ...heatingBase, enabled: true, charging: true, vehicleSoc: 28 };

export const HeatingIdle = Template.bind({});
HeatingIdle.args = { ...heatingBase, vehicleSoc: 32 };

export const HeatingReady = Template.bind({});
HeatingReady.args = { ...heatingBase, enabled: true, vehicleSoc: 38 };

export const HeatingCharging = Template.bind({});
HeatingCharging.args = { ...heatingBase, enabled: true, charging: true, vehicleSoc: 44 };

export const HeaterLimitManualBelow = Template.bind({});
HeaterLimitManualBelow.args = {
  ...heatingBase,
  enabled: true,
  charging: true,
  vehicleSoc: 46,
  vehicleLimitSoc: 52,
  effectiveLimitSoc: 48,
};

export const HeaterLimitManualAbove = Template.bind({});
HeaterLimitManualAbove.args = {
  ...heatingBase,
  enabled: true,
  charging: true,
  vehicleSoc: 48,
  vehicleLimitSoc: 52,
  effectiveLimitSoc: 56,
};

// current temp nearly at the top of the scale, limit raised to make room
export const HeatingNearlyFull = Template.bind({});
HeatingNearlyFull.args = {
  ...heatingBase,
  enabled: true,
  charging: true,
  vehicleSoc: 58,
  effectiveLimitSoc: 60,
};

export const HeatingDisconnected = Template.bind({});
HeatingDisconnected.args = { ...heatingBase, connected: false };

// no temperature reading yet: warm bar, no slider
export const HeatingNoReading = Template.bind({});
HeatingNoReading.args = { ...heatingBase, vehicleSoc: 0, enabled: true, charging: true };

// --- overview: charging (left) vs heating (right), each row the same situation ---

const chargingVariants = [
  { name: "Disconnected", args: Disconnected.args },
  { name: "ChargingVeryLow", args: ChargingVeryLow.args },
  { name: "MinSocNotReached", args: MinSocNotReached.args },
  { name: "ChargingLow", args: ChargingLow.args },
  { name: "Connected", args: Connected.args },
  { name: "Ready", args: Ready.args },
  { name: "Charging", args: Charging.args },
  { name: "VehicleLimitManualBelow", args: VehicleLimitManualBelow.args },
  { name: "VehicleLimitManualAbove", args: VehicleLimitManualAbove.args },
  { name: "NearlyFull", args: NearlyFull.args },
  { name: "NoSoc", args: NoSoc.args },
];

const heatingVariants = [
  { name: "HeatingDisconnected", args: HeatingDisconnected.args },
  { name: "HeatingChargingVeryLow", args: HeatingChargingVeryLow.args },
  { name: "MinTempNotReached", args: MinTempNotReached.args },
  { name: "HeatingChargingLow", args: HeatingChargingLow.args },
  { name: "HeatingIdle", args: HeatingIdle.args },
  { name: "HeatingReady", args: HeatingReady.args },
  { name: "HeatingCharging", args: HeatingCharging.args },
  { name: "HeaterLimitManualBelow", args: HeaterLimitManualBelow.args },
  { name: "HeaterLimitManualAbove", args: HeaterLimitManualAbove.args },
  { name: "HeatingNearlyFull", args: HeatingNearlyFull.args },
  { name: "HeatingNoReading", args: HeatingNoReading.args },
];

export const Overview: StoryFn<typeof Soc> = () => ({
  components: { Soc },
  setup() {
    return { columns: [chargingVariants, heatingVariants] };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(2, minmax(320px, 1fr)); gap: 1.5rem; align-items: start;">
      <div v-for="(column, i) in columns" :key="i" style="display: flex; flex-direction: column; gap: 1.5rem;">
        <div v-for="v in column" :key="v.name" style="padding: 1rem; background: var(--evcc-box); border-radius: 0.5rem;">
          <div style="margin-bottom: 0.75rem; color: var(--evcc-default-text); font-size: 0.85rem; opacity: 0.7;">{{ v.name }}</div>
          <Soc v-bind="v.args" />
        </div>
      </div>
    </div>
  `,
});
Overview.parameters = { controls: { disable: true } };
