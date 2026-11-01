import SocChart, { type SocChartEntry } from "./SocChart.vue";
import Card from "../Helper/Card.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import type { EvoptData } from "./TimeSeriesDataTable.vue";

export default {
  title: "Optimize/SocChart",
  component: SocChart,
  parameters: {
    layout: "padded",
  },
} as Meta<typeof SocChart>;

const START_HOUR = 6;
const HOURS = 30;
const SLOT_SECONDS = 15 * 60;
const SLOTS = (HOURS * 3600) / SLOT_SECONDS;
const dt = Array(SLOTS).fill(SLOT_SECONDS);
const timestamp = new Date(2025, 5, 12, START_HOUR).toISOString();
const hours = Array.from({ length: SLOTS }, (_, i) => START_HOUR + ((i + 1) * SLOT_SECONDS) / 3600);

interface Device extends Omit<SocChartEntry, "index"> {
  soc: (h: number) => number; // percent at slot end
}

const clamp = (v: number) => Math.min(100, Math.max(0, v));

const vehicle: Device = {
  title: "Carport (ID.3)",
  capacity: 58,
  color: "#0077B6",
  soc: (h) => clamp(45 + Math.max(0, Math.min(h, 15) - 11) * 8),
};
const secondVehicle: Device = {
  title: "Garage (e-Golf)",
  capacity: 35,
  color: "#F8961E",
  soc: (h) => clamp(70 + Math.max(0, Math.min(h, 24) - 22) * 10),
};
const battery: Device = {
  title: "Home Battery",
  capacity: 10,
  color: "#40A578",
  soc: (h) => clamp(60 + 35 * Math.sin(((h - 8) / 24) * 2 * Math.PI)),
};
const secondBattery: Device = {
  title: "Garage Battery",
  capacity: 5,
  color: "#9D4EDD",
  soc: (h) => clamp(50 + 40 * Math.sin(((h - 10) / 24) * 2 * Math.PI)),
};

function evopt(devices: Device[]): EvoptData {
  const wh = (d: Device, h: number) => (d.soc(h) / 100) * d.capacity * 1000;
  return {
    req: {
      time_series: { dt, gt: [], ft: [], p_N: [], p_E: [] },
      batteries: devices.map((d) => ({ s_initial: wh(d, START_HOUR) })),
    },
    res: {
      batteries: devices.map((d) => ({
        charging_power: [],
        discharging_power: [],
        state_of_charge: hours.map((h) => wh(d, h)),
      })),
    },
  } as unknown as EvoptData;
}

const Template: StoryFn<{ devices: Device[] }> = (args) => ({
  components: { SocChart, Card },
  setup() {
    const entries = args.devices.map((d, index) => ({ ...d, index }));
    return { evopt: evopt(args.devices), entries, timestamp };
  },
  template: `<div class="container px-0"><Card title="SoC Projections" subtitle="%" edge-to-edge class="box-pull-out">
    <div v-for="(entry, idx) in entries" :key="entry.index" :class="{ 'mb-3': idx < entries.length - 1 }">
      <SocChart :evopt="evopt" :entry="entry" :timestamp="timestamp" :show-x-axis="idx === entries.length - 1" />
    </div>
  </Card></div>`,
});

export const VehicleAndBattery = Template.bind({});
VehicleAndBattery.args = { devices: [vehicle, battery] };

export const MultipleDevices = Template.bind({});
MultipleDevices.args = { devices: [vehicle, secondVehicle, battery, secondBattery] };
