import SetupChecklist from "./SetupChecklist.vue";
import { ONBOARDING_ITEMS } from "@/onboarding";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Config/SetupChecklist",
  component: SetupChecklist,
} as Meta<typeof SetupChecklist>;

const configured = (done: string[]) =>
  Object.fromEntries(ONBOARDING_ITEMS.map((item) => [item, done.includes(item)]));

const Template: StoryFn<typeof SetupChecklist> = (args) => ({
  components: { SetupChecklist },
  setup() {
    return { args };
  },
  template: '<div style="max-width: 760px"><SetupChecklist v-bind="args" /></div>',
});

export const Empty = Template.bind({});
Empty.args = { configured: configured([]) };

export const Partial = Template.bind({});
Partial.args = { configured: configured(["pv", "grid"]), dismissed: ["loadpoint"] };

export const OneLeft = Template.bind({});
OneLeft.args = {
  configured: configured([
    "battery",
    "pv",
    "grid",
    "loadpoint",
    "consumer",
    "gridImport",
    "gridExport",
    "co2",
  ]),
};
