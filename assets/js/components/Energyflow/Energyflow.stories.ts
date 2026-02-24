import type { Meta, StoryFn } from "@storybook/vue3";
import Energyflow from "./Energyflow.vue";
import { CURRENCY } from "@/types/evcc";

export default {
  title: "Energyflow/Energyflow",
  component: Energyflow,
} as Meta<typeof Energyflow>;

const Template: StoryFn<typeof Energyflow> = (args: any) => ({
  components: { Energyflow },
  setup() {
    return { args };
  },
  template: '<Energyflow v-bind="args" />',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 7300,
  gridPower: -2300,
  homePower: 800,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "bike",
      charging: true,
      title: "Garage",
      chargePower: 1000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 2200,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  smartCostType: "price",
  currency: CURRENCY.EUR,
  pv: [{ power: 5000 }, { power: 2300 }],
  forecast: {
    solar: {
      today: { energy: 1000, complete: true },
      tomorrow: { energy: 1000, complete: false },
      dayAfterTomorrow: { energy: 1000, complete: false },
    },
  },
} as any;

function hoursFromNow(h: number): string {
  return new Date(Date.now() + h * 60 * 60 * 1000).toISOString();
}

const batteryBase = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  homePower: 800,
};

const bat = (power: number, soc: number, forecast: any, devices: any[] = []) => ({
  power,
  soc,
  capacity: 10,
  devices,
  forecast,
});

const dev = (title: string, power: number, soc: number, forecast: any) => ({
  title,
  power,
  soc,
  capacity: 10,
  controllable: true,
  forecast,
});

export const BatteryForecastDischarging = Template.bind({});
BatteryForecastDischarging.args = {
  ...batteryBase,
  gridPower: 500,
  homePower: 1800,
  battery: bat(1300, 62, { full: null, empty: hoursFromNow(0.4) }),
} as any;

export const BatteryForecastCharging = Template.bind({});
BatteryForecastCharging.args = {
  ...batteryBase,
  pvPower: 6000,
  gridPower: -1000,
  battery: bat(-4200, 45, { full: hoursFromNow(2.5), empty: null }),
} as any;

export const BatteryForecastBoth = Template.bind({});
BatteryForecastBoth.args = {
  ...batteryBase,
  pvPower: 3000,
  gridPower: -500,
  battery: bat(-1700, 70, { full: hoursFromNow(2), empty: hoursFromNow(36) }),
} as any;

export const BatteryForecastMultiDischarging = Template.bind({});
BatteryForecastMultiDischarging.args = {
  ...batteryBase,
  gridPower: 500,
  homePower: 2500,
  battery: bat(2000, 55, { full: null, empty: hoursFromNow(48) }, [
    dev("Powerwall", 1200, 60, { full: null, empty: hoursFromNow(48) }),
    dev("BYD", 800, 48, { full: null, empty: hoursFromNow(0.25) }),
  ]),
} as any;

export const BatteryForecastMultiCharging = Template.bind({});
BatteryForecastMultiCharging.args = {
  ...batteryBase,
  pvPower: 8000,
  gridPower: -1000,
  homePower: 1000,
  battery: bat(-6000, 40, { full: hoursFromNow(26), empty: null }, [
    dev("Powerwall", -3500, 35, { full: hoursFromNow(26), empty: null }),
    dev("BYD", -2500, 47, { full: hoursFromNow(0.7), empty: null }),
  ]),
} as any;

export const BatteryForecastGridChargeLimit = Template.bind({});
BatteryForecastGridChargeLimit.args = {
  ...batteryBase,
  pvPower: 5000,
  gridPower: -500,
  batteryGridChargeLimit: 0.15,
  smartCostType: "price",
  currency: CURRENCY.EUR,
  battery: bat(-3700, 50, { full: hoursFromNow(1.5), empty: null }),
} as any;

export const BatteryAndGrid = Template.bind({});
BatteryAndGrid.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 0,
  gridPower: 1200,
  homePower: 2000,
  batteryPower: 800,
  batterySoc: 77,
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  currency: CURRENCY.EUR,
  battery: [
    { soc: 44.999, capacity: 13.3, power: 350, controllable: true },
    { soc: 82.3331, capacity: 21, power: 450, controllable: false },
  ],
} as any;

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 5000,
  gridPower: -1300,
  homePower: 800,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1400,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  batteryPower: -1500,
  batterySoc: 75,
} as any;

export const GridPVAndBattery = Template.bind({});
GridPVAndBattery.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 1000,
  gridPower: 700,
  homePower: 3300,
  batteryPower: 1500,
  batterySoc: 30,
} as any;

export const BatteryThresholds = Template.bind({});
BatteryThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 8700,
  gridPower: -500,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 5000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "bus",
      charging: true,
      title: "Garage",
      chargePower: 2500,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  batteryPower: -700,
  batterySoc: 95,
} as any;

export const PVThresholds = Template.bind({});
PVThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 300,
  gridPower: 6500,
  homePower: 1000,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 5000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1600,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  batteryPower: 800,
  batterySoc: 76,
} as any;

export const GridOnly = Template.bind({});
GridOnly.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 0,
  gridPower: 6500,
  homePower: 1000,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 5500,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: false,
      title: "Garage",
      chargePower: 0,
      connected: false,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: false,
      title: "Garage",
      chargePower: 0,
      connected: false,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: false,
      title: "Garage",
      chargePower: 0,
      connected: false,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  batteryPower: 0,
  batterySoc: 0,
} as any;

export const LowPower = Template.bind({});
LowPower.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 700,
  gridPower: -300,
  homePower: 300,
  batteryPower: -100,
  batterySoc: 55,
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  currency: CURRENCY.EUR,
} as any;

export const CO2 = Template.bind({});
CO2.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 7300,
  gridPower: -2300,
  homePower: 800,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 2200,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  tariffCo2: 723,
  smartCostType: "co2",
  currency: CURRENCY.EUR,
  pv: [{ power: 5000 }, { power: 2300 }],
} as any;

export const UnknownInput = Template.bind({});
UnknownInput.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 2000,
  gridPower: -2000,
  loadpoints: [
    {
      icon: "car",
      charging: true,
      title: "Garage",
      chargePower: 1000,
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
} as any;

export const UnknownInputFill = Template.bind({});
UnknownInputFill.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 500,
  gridPower: 0,
  batteryPower: -1000,
  loadpoints: [],
} as any;

export const UnknownOutput = Template.bind({});
UnknownOutput.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 3000,
  gridPower: -1000,
  loadpoints: [
    {
      chargePower: 1700,
      icon: "car",
      charging: true,
      title: "Garage",
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
} as any;

export const UnknownOutputLessThan10Percent = Template.bind({});
UnknownOutputLessThan10Percent.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 3000,
  gridPower: -1000,
  loadpoints: [
    {
      chargePower: 1800,
      icon: "car",
      charging: true,
      title: "Garage",
      connected: true,
      vehicleName: "",
      vehicleSoc: 50,
      chargerFeatureHeating: false,
    },
  ],
} as any;
