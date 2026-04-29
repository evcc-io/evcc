import BatteryBoostButton from "./BatteryBoostButton.vue";
import { CHARGE_MODE } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Loadpoints/BatteryBoostButton",
  component: BatteryBoostButton,
  argTypes: {
    batteryBoost: { control: "boolean" },
    batteryBoostLimit: { control: { type: "range", min: 0, max: 100, step: 5 } },
    batterySoc: { control: { type: "range", min: 0, max: 100, step: 5 } },
    mode: { control: "select", options: Object.values(CHARGE_MODE) },
  },
  parameters: {
    layout: "centered",
  },
} as Meta<typeof BatteryBoostButton>;

const socLevels = [0, 20, 40, 60, 80, 100];

interface RowConfig {
  label: string;
  batteryBoost: boolean;
  batteryBoostLimit: number;
  mode: CHARGE_MODE;
}

const rows: RowConfig[] = [
  { label: "Ready (limit 0)", batteryBoost: false, batteryBoostLimit: 0, mode: CHARGE_MODE.PV },
  { label: "Enabled (limit 0)", batteryBoost: true, batteryBoostLimit: 0, mode: CHARGE_MODE.PV },
  { label: "Ready (limit 50)", batteryBoost: false, batteryBoostLimit: 50, mode: CHARGE_MODE.PV },
  { label: "Enabled (limit 50)", batteryBoost: true, batteryBoostLimit: 50, mode: CHARGE_MODE.PV },
  { label: "Disabled", batteryBoost: false, batteryBoostLimit: 50, mode: CHARGE_MODE.OFF },
];

export const AllStates: StoryFn<typeof BatteryBoostButton> = () => ({
  components: { BatteryBoostButton },
  setup() {
    return { socLevels, rows };
  },
  template: `
    <div style="font-family: sans-serif; font-size: 12px;">
      <div style="display: grid; grid-template-columns: 140px repeat(6, 1fr); gap: 16px; align-items: center; text-align: center;">
        <div></div>
        <div v-for="soc in socLevels" :key="'h-'+soc" style="color: #888; font-weight: 600;">{{ soc }}%</div>
        <template v-for="row in rows" :key="row.label">
          <div style="text-align: right; font-weight: 600;">{{ row.label }}</div>
          <div v-for="soc in socLevels" :key="row.label+'-'+soc">
            <BatteryBoostButton
              :batteryBoost="row.batteryBoost"
              :batteryBoostLimit="row.batteryBoostLimit"
              :mode="row.mode"
              :batterySoc="soc"
              @updated="() => {}"
            />
          </div>
        </template>
      </div>
    </div>
  `,
});
AllStates.storyName = "All States (grid overview)";

const Template: StoryFn<typeof BatteryBoostButton> = (args) => ({
  components: { BatteryBoostButton },
  setup() {
    return { args };
  },
  template: '<BatteryBoostButton v-bind="args" @updated="() => {}" />',
});

export const Playground = Template.bind({});
Playground.args = {
  batteryBoost: false,
  batteryBoostLimit: 50,
  batterySoc: 75,
  mode: CHARGE_MODE.PV,
};
