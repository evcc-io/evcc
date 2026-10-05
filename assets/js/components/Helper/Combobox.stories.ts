import Combobox from "./Combobox.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Helper/Combobox",
  component: Combobox,
} as Meta<typeof Combobox>;

const Template: StoryFn<typeof Combobox> = (args) => ({
  components: { Combobox },
  setup() {
    return { args };
  },
  template: '<Combobox v-bind="args" v-model="args.modelValue" />',
});

const options = [
  { value: "192.168.31.8", label: "homeassistant.local", hint: "Home Assistant", used: true },
  {
    value: "192.168.31.1",
    label: "a-very-long-hostname-of-some-device-in-the-network.fritz.box",
    hint: "AVM Audiovisuelles Marketing und Computersysteme GmbH",
  },
  { value: "192.168.31.20", label: "sma3009876543", hint: "SMA Solar Technology" },
  { value: "192.168.31.21", label: "fronius-symo-gen24", hint: "Fronius International" },
  { value: "192.168.31.42", label: "shellyplusplug-a8032ab", hint: "Shelly", used: true },
  { value: "192.168.31.50", label: "kostal-plenticore", hint: "KOSTAL Solar Electric" },
  { value: "192.168.31.77" },
];

export const Default = Template.bind({});
Default.args = { id: "host", options };

export const Matching = Template.bind({});
Matching.args = {
  ...Default.args,
  options: options.map((o) => ({ ...o, match: o.hint?.startsWith("Fronius") })),
};

export const Selected = Template.bind({});
Selected.args = { ...Default.args, modelValue: "192.168.31.21" };

export const Empty = Template.bind({});
Empty.args = { ...Default.args, options: [] };

export const Loading = Template.bind({});
Loading.args = { ...Default.args, loading: true };
