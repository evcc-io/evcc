import Badge from "./Badge.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

const variants = ["success", "muted"] as const;

export default {
  title: "Helper/Badge",
  component: Badge,
  argTypes: {
    variant: { control: "select", options: variants },
  },
} as Meta<typeof Badge>;

const Template: StoryFn<typeof Badge> = (args) => ({
  components: { Badge },
  setup() {
    return { args };
  },
  template: '<Badge v-bind="args">connected to Garage</Badge>',
});

export const Success = Template.bind({});
Success.args = { variant: "success" };

export const Muted = Template.bind({});
Muted.args = { variant: "muted" };
