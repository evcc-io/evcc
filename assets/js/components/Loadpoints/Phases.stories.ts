import Phases from "./Phases.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import { ref, onMounted, onUnmounted } from "vue";

export default {
  title: "Loadpoints/Phases",
  component: Phases,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    phasesActive: { control: { type: "number" } },
    phasesConfigured: { control: { type: "number" } },
    minCurrent: { control: { type: "number" } },
    maxCurrent: { control: { type: "number" } },
    offeredCurrent: { control: { type: "number" } },
    chargeCurrents: { control: { type: "object" } },
  },
} as Meta<typeof Phases>;

const Template: StoryFn<typeof Phases> = (args) => ({
  components: { Phases },
  setup() {
    return { args };
  },
  template: '<Phases v-bind="args" />',
});

const base = {
  phasesConfigured: 3,
  minCurrent: 6,
  maxCurrent: 16,
  offeredCurrent: 8,
  chargeCurrents: undefined,
};

// --- expected phases only (no per-phase measurement) ---

export const NoData = Template.bind({});
NoData.args = { ...base, phasesActive: undefined, offeredCurrent: 0 };

export const Expected1p = Template.bind({});
Expected1p.args = { ...base, phasesActive: 1 };

export const Expected2p = Template.bind({});
Expected2p.args = { ...base, phasesActive: 2 };

export const Expected3p = Template.bind({});
Expected3p.args = { ...base, phasesActive: 3 };

// --- measured per-phase currents ---

export const Measured1p = Template.bind({});
Measured1p.args = { ...base, phasesActive: 1, offeredCurrent: 12, chargeCurrents: [6, 0.2, 0] };

export const Measured2p = Template.bind({});
Measured2p.args = { ...base, phasesActive: 2, offeredCurrent: 16, chargeCurrents: [16, 16, 0.3] };

export const Measured3p = Template.bind({});
Measured3p.args = { ...base, phasesActive: 3, offeredCurrent: 13, chargeCurrents: [11, 9, 12] };

export const Asymmetric = Template.bind({});
Asymmetric.args = { ...base, phasesActive: 2, offeredCurrent: 16, chargeCurrents: [8, 0.9, 14] };

export const OnlyL2 = Template.bind({});
OnlyL2.args = { ...base, phasesActive: 1, offeredCurrent: 13, chargeCurrents: [0, 13, 0] };

export const MainlyL3 = Template.bind({});
MainlyL3.args = {
  ...base,
  phasesActive: 1,
  offeredCurrent: 20,
  maxCurrent: 20,
  chargeCurrents: [0.007, 0.009, 5.945],
};

// 1p device but 3 phases measured: configuration error, warning color + tooltip
export const Mismatch1p = Template.bind({});
Mismatch1p.args = {
  ...base,
  phasesConfigured: 1,
  phasesActive: 3,
  offeredCurrent: 10,
  chargeCurrents: [10, 10, 10],
};

// cycles 1p → 3p → 1p every few seconds to inspect the transition
export const Switching: StoryFn<typeof Phases> = (args) => ({
  components: { Phases },
  setup() {
    const phasesActive = ref(1);
    let timer: ReturnType<typeof setInterval>;
    onMounted(() => {
      timer = setInterval(() => {
        phasesActive.value = phasesActive.value === 1 ? 3 : 1;
      }, 3000);
    });
    onUnmounted(() => clearInterval(timer));
    return { args, phasesActive };
  },
  template: '<Phases v-bind="args" :phasesActive="phasesActive" />',
});
Switching.args = { ...base, phasesConfigured: 0, offeredCurrent: 12 };

// --- overview: rows = scenario, columns = device phase configuration ---

const scenarios = [
  { name: "NoData", args: NoData.args },
  { name: "Expected1p", args: Expected1p.args },
  { name: "Expected2p", args: Expected2p.args },
  { name: "Expected3p", args: Expected3p.args },
  { name: "Measured1p", args: Measured1p.args },
  { name: "Measured2p", args: Measured2p.args },
  { name: "Measured3p", args: Measured3p.args },
  { name: "Asymmetric", args: Asymmetric.args },
  { name: "OnlyL2", args: OnlyL2.args },
  { name: "MainlyL3", args: MainlyL3.args },
];

// 1p3p auto (0) behaves like 3p
const configs = [
  { name: "1p device", phasesConfigured: 1 },
  { name: "3p/auto", phasesConfigured: 3 },
];

export const Overview: StoryFn<typeof Phases> = () => ({
  components: { Phases },
  setup() {
    return { scenarios, configs };
  },
  template: `
    <div style="display: grid; grid-template-columns: 140px repeat(2, 120px); gap: 0.75rem 1.5rem; align-items: center; padding: 1rem; background: var(--evcc-box); border-radius: 0.5rem; color: var(--evcc-default-text); font-size: 0.85rem;">
      <div></div>
      <div v-for="c in configs" :key="c.name" style="opacity: 0.7;">{{ c.name }}</div>
      <template v-for="s in scenarios" :key="s.name">
        <div style="opacity: 0.7;">{{ s.name }}</div>
        <Phases v-for="c in configs" :key="c.name" v-bind="{ ...s.args, phasesConfigured: c.phasesConfigured }" />
      </template>
    </div>
  `,
});
Overview.parameters = { controls: { disable: true } };
