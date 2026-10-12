import SolarShareSlider from "./SolarShareSlider.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Loadpoints/SolarShareSlider",
  component: SolarShareSlider,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    modelValue: { control: { type: "range", min: 0, max: 100, step: 10 } },
  },
} as Meta<typeof SolarShareSlider>;

const Template: StoryFn<typeof SolarShareSlider> = (args) => ({
  components: { SolarShareSlider },
  setup() {
    return { args };
  },
  template: '<div style="width: 320px"><SolarShareSlider v-bind="args" /></div>',
});

export const Default = Template.bind({});
Default.args = { modelValue: 50 };

const states = [
  { label: "0%", value: 0, disabled: false },
  { label: "50%", value: 50, disabled: false },
  { label: "100%", value: 100, disabled: false },
  { label: "disabled", value: 50, disabled: true },
];

export const AllStates = () => ({
  components: { SolarShareSlider },
  setup() {
    return { states };
  },
  template: `
    <div style="width: 320px; display: flex; flex-direction: column; gap: 16px;">
      <div v-for="s in states" :key="s.label">
        <small style="font-family: monospace; color: #666; font-size: 12px;">{{ s.label }}</small>
        <SolarShareSlider :modelValue="s.value" :disabled="s.disabled" :class="{ 'opacity-50': s.disabled }" />
      </div>
    </div>
  `,
});
