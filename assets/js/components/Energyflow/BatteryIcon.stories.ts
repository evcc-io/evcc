import BatteryIcon from "./BatteryIcon.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Energyflow/BatteryIcon",
  component: BatteryIcon,
  argTypes: {
    soc: { control: { type: "range", min: 0, max: 100, step: 10 } },
    hold: { control: "boolean" },
    gridCharge: { control: "boolean" },
  },
  parameters: {
    layout: "centered",
  },
} as Meta<typeof BatteryIcon>;

const socLevels = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100];

interface RowConfig {
  label: string;
  hold: boolean;
  gridCharge: boolean;
}

const rows: RowConfig[] = [
  { label: "Normal", hold: false, gridCharge: false },
  { label: "Hold", hold: true, gridCharge: false },
  { label: "GridCharge", hold: false, gridCharge: true },
];

export const AllStates: StoryFn<typeof BatteryIcon> = () => ({
  components: { BatteryIcon },
  setup() {
    return { socLevels, rows };
  },
  template: `
    <div style="font-family: sans-serif; font-size: 12px;">
      <div style="display: grid; grid-template-columns: 100px repeat(11, 1fr); gap: 8px; align-items: center; text-align: center;">
        <div></div>
        <div v-for="soc in socLevels" :key="'h-'+soc" style="color: #888; font-weight: 600;">{{ soc }}%</div>
        <template v-for="row in rows" :key="row.label">
          <div style="text-align: right; font-weight: 600;">{{ row.label }}</div>
          <div v-for="soc in socLevels" :key="row.label+'-'+soc">
            <BatteryIcon
              :soc="soc"
              :hold="row.hold"
              :gridCharge="row.gridCharge"
              style="width: 48px; height: 48px;"
            />
          </div>
        </template>
      </div>
    </div>
  `,
});
AllStates.storyName = "All States (grid overview)";

const Template: StoryFn<typeof BatteryIcon> = (args) => ({
  components: { BatteryIcon },
  setup() {
    return { args };
  },
  template: '<BatteryIcon v-bind="args" style="width: 48px; height: 48px;" />',
});

export const Playground = Template.bind({});
Playground.args = {
  soc: 50,
  hold: false,
  gridCharge: false,
};
