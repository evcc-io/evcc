import DeviceList from "./DeviceList.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Config/DeviceList",
  component: DeviceList,
  // lives inside modals, which sit on the box background
  globals: { backgrounds: { value: "box" } },
  parameters: {
    docs: {
      description: {
        component:
          "Compact device rows for the setup wizard. A configured row shows its title, a formatted live value and the edit icon. A row with `add` shows the add button instead.",
      },
    },
  },
} as Meta<typeof DeviceList>;

const Template: StoryFn<typeof DeviceList> = (args) => ({
  components: { DeviceList },
  setup() {
    return { args };
  },
  template: '<div style="max-width: 460px"><DeviceList v-bind="args" /></div>',
});

export const Configured = Template.bind({});
Configured.args = {
  items: [
    { id: 1, title: "Grid Meter", value: "1.8 kW" },
    { id: 2, title: "Home battery", value: "72.0%" },
    { id: 3, title: "Solar Carport", value: "0.0 kW" },
  ],
};

export const Mixed = Template.bind({});
Mixed.args = {
  items: [
    { id: "grid", title: "Import tariff", value: "28.9 ct/kWh" },
    { id: "feedIn", title: "Export tariff", add: true },
  ],
};

export const Pending = Template.bind({});
Pending.args = {
  items: [
    { id: "grid", title: "Import tariff", add: true },
    { id: "feedIn", title: "Export tariff", add: true },
  ],
};

export const LongTitles = Template.bind({});
LongTitles.args = {
  items: [
    {
      id: 1,
      title: "A very long device title that does not fit into the row and gets truncated",
      value: "12.3 kW",
    },
    {
      id: 2,
      title: "Another long title for a device that is not configured yet and needs an add button",
      add: true,
    },
  ],
};
