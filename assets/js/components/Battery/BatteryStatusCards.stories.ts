import BatteryStatusCards from "./BatteryStatusCards.vue";
import type { Battery, BatteryMeter } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Battery/BatteryStatusCards",
  component: BatteryStatusCards,
  parameters: {
    layout: "fullscreen",
  },
} as Meta<typeof BatteryStatusCards>;

const Template: StoryFn<typeof BatteryStatusCards> = (args) => ({
  components: { BatteryStatusCards },
  setup() {
    return { args };
  },
  template: '<div class="p-4"><BatteryStatusCards v-bind="args" /></div>',
});

function device(overrides: Partial<BatteryMeter>): BatteryMeter {
  return { power: 0, soc: 50, controllable: false, capacity: 10, ...overrides };
}

function battery(devices: BatteryMeter[], forecast?: Battery["forecast"]): Battery {
  return {
    power: devices.reduce((s, d) => s + d.power, 0),
    capacity: devices.reduce((s, d) => s + d.capacity, 0),
    soc: 50,
    devices,
    forecast,
  };
}

export const Single = Template.bind({});
Single.args = {
  battery: battery([device({ title: "Sungrow", soc: 76, power: 800, capacity: 13.5 })]),
};

export const Multiple = Template.bind({});
Multiple.args = {
  battery: {
    ...battery([
      device({ title: "Sungrow", soc: 76, power: 800, capacity: 13.5, controllable: true }),
      device({ title: "Anker", soc: 40, power: -1200, capacity: 7.5 }),
    ]),
    soc: 63,
  },
};

// single battery: optimizer gets its own card, showing suggestion + high + low
export const WithForecast = Template.bind({});
WithForecast.args = {
  battery: battery([device({ title: "Battery", soc: 62, power: 900, capacity: 10 })], {
    highest: { soc: 100, time: "2026-07-01T16:30:00+02:00" },
    lowest: { soc: 12, time: "2026-07-02T06:00:00+02:00" },
  }),
  suggestion: { action: "charge" },
};

// forecast is a site-aggregate value, so it lands on the combined card
export const MultipleWithForecast = Template.bind({});
MultipleWithForecast.args = {
  battery: {
    ...battery(
      [
        device({ title: "Sungrow", soc: 76, power: 800, capacity: 13.5, controllable: true }),
        device({ title: "Anker", soc: 40, power: -1200, capacity: 7.5 }),
      ],
      {
        highest: { soc: 100, time: "2026-07-01T16:30:00+02:00" },
        lowest: { soc: 15, time: "2026-07-02T05:00:00+02:00" },
      }
    ),
    soc: 63,
  },
};

export const SocOnly = Template.bind({});
SocOnly.args = {
  battery: battery([device({ title: "Battery", soc: 55, power: 300, capacity: 0 })]),
};
