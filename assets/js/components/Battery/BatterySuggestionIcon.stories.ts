import BatterySuggestionIcon from "./BatterySuggestionIcon.vue";
import { ICON_SIZE } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

const actions = ["normal", "hold", "charge", "holdcharge", "discharge"] as const;

export default {
  title: "Battery/BatterySuggestionIcon",
  component: BatterySuggestionIcon,
  argTypes: {
    action: { control: "select", options: actions },
    size: { control: "select", options: Object.values(ICON_SIZE) },
  },
} as Meta<typeof BatterySuggestionIcon>;

const Single: StoryFn<typeof BatterySuggestionIcon> = (args) => ({
  components: { BatterySuggestionIcon },
  setup() {
    return { args };
  },
  template: '<BatterySuggestionIcon v-bind="args" />',
});

export const Single_ = Single.bind({});
Single_.args = { action: "normal", size: ICON_SIZE.L };

// all states side by side, reusing the MaterialIcon grid layout
export const AllStates = () => ({
  components: { BatterySuggestionIcon },
  setup() {
    return { actions };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 20px;">
      <div v-for="action in actions" :key="action" style="display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 15px;">
        <BatterySuggestionIcon :action="action" size="xl" />
        <small style="font-family: monospace; color: #666; text-align: center; font-size: 12px;">{{ action }}</small>
      </div>
    </div>
  `,
});
