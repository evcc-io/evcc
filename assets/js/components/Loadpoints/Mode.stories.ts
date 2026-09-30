import Mode from "./Mode.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import sk from "../../../../i18n/sk.json";
import { reactive, ref } from "vue";

export default {
  title: "Loadpoints/Mode",
  component: Mode,
  argTypes: {
    mode: {
      control: "select",
      options: ["off", "smart", "now"],
      description: "Charging mode",
    },
    pvPossible: { control: "boolean", description: "Whether PV is possible" },
    smartCostAvailable: { control: "boolean", description: "Whether smart cost is available" },
    alwaysCharge: {
      control: "select",
      options: ["off", "on", "once"],
      description: "Always charge state",
    },
    switchDevice: { control: "boolean", description: "Charger without current control" },
    continuous: { control: "boolean", description: "Continuously operating heatpump" },
  },
  parameters: {
    layout: "centered",
  },
} as Meta<typeof Mode>;

const scenarios = [
  { name: "Minimal", args: { mode: "now" } },
  { name: "Smart", args: { mode: "smart", pvPossible: true, effectiveMinCurrent: 6 } },
  {
    name: "AlwaysCharge",
    args: { mode: "smart", pvPossible: true, alwaysCharge: "on", effectiveMinCurrent: 6 },
  },
  {
    name: "AlwaysChargeCharging",
    args: {
      mode: "smart",
      pvPossible: true,
      alwaysCharge: "on",
      charging: true,
      effectiveMinCurrent: 6,
    },
  },
  { name: "SwitchDevice", args: { mode: "smart", pvPossible: true, switchDevice: true } },
  {
    name: "HeatpumpSGReady",
    args: { mode: "smart", pvPossible: true, heating: true, continuous: true, switchDevice: true },
  },
  {
    name: "HeatpumpControllable",
    args: {
      mode: "smart",
      pvPossible: true,
      heating: true,
      continuous: true,
      alwaysCharge: "on",
      effectiveMinCurrent: 6,
    },
  },
];

const byName = (name: string) => scenarios.find((s) => s.name === name)?.args;

type Scenario = (typeof scenarios)[number];

// shared mode across all examples, so switching one switches all
function overviewState() {
  const mode = ref("smart");
  const rows = reactive(scenarios.map((s) => ({ ...s, args: { ...s.args } })));
  const smartPossible = (s: Scenario) => "pvPossible" in s.args || "smartCostAvailable" in s.args;
  return {
    rows,
    setMode: (value: string) => (mode.value = value),
    modeFor: (s: Scenario) => (smartPossible(s) || mode.value !== "smart" ? mode.value : "now"),
  };
}

const overviewTemplate = `
  <div class="p-3" style="display: grid; grid-template-columns: auto minmax(440px, 1fr) minmax(0, 2fr); gap: 1rem 1.5rem; align-items: center">
    <div></div>
    <div class="evcc-gray">condensed</div>
    <div class="evcc-gray">full width</div>
    <template v-for="s in rows" :key="s.name">
      <div class="evcc-gray">{{ s.name }}</div>
      <div class="d-flex">
        <Mode v-bind="s.args" :mode="modeFor(s)" @updated="setMode" @always-charge-updated="s.args.alwaysCharge = $event" />
      </div>
      <div class="d-flex">
        <Mode v-bind="s.args" :mode="modeFor(s)" class="flex-grow-1" @updated="setMode" @always-charge-updated="s.args.alwaysCharge = $event" />
      </div>
    </template>
  </div>
`;

// rows: scenario, columns: condensed (wide screens) and stretched (narrow screens)
export const Overview: StoryFn<typeof Mode> = () => ({
  components: { Mode },
  setup: overviewState,
  template: overviewTemplate,
});
Overview.parameters = { controls: { disable: true }, layout: "fullscreen" };

// same matrix with a verbose locale, shows how long labels behave
export const OverviewLongLabels: StoryFn<typeof Mode> = () => ({
  components: { Mode },
  setup: overviewState,
  data() {
    return { previousLocale: "" };
  },
  mounted() {
    this.previousLocale = this.$i18n.locale;
    this.$i18n.setLocaleMessage("sk", sk);
    this.$i18n.locale = "sk";
  },
  unmounted() {
    this.$i18n.locale = this.previousLocale;
  },
  template: overviewTemplate,
});
OverviewLongLabels.parameters = { controls: { disable: true }, layout: "fullscreen" };

const Template: StoryFn<typeof Mode> = (args) => ({
  components: { Mode },
  setup() {
    return { args };
  },
  template: '<Mode v-bind="args" />',
});

export const Minimal = Template.bind({});
Minimal.args = byName("Minimal");

export const Smart = Template.bind({});
Smart.args = byName("Smart");

export const AlwaysCharge = Template.bind({});
AlwaysCharge.args = byName("AlwaysCharge");

export const AlwaysChargeCharging = Template.bind({});
AlwaysChargeCharging.args = byName("AlwaysChargeCharging");

export const SwitchDevice = Template.bind({});
SwitchDevice.args = byName("SwitchDevice");

export const HeatpumpSGReady = Template.bind({});
HeatpumpSGReady.args = byName("HeatpumpSGReady");

export const HeatpumpControllable = Template.bind({});
HeatpumpControllable.args = byName("HeatpumpControllable");
