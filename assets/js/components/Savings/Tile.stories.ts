import Tile from "./Tile.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Savings/Tile",
  component: Tile,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    icon: { control: "text", description: "Icon name to display" },
    title: { control: "text", description: "Title of the tile" },
    value: { control: "text", description: "Primary value to display" },
    unit: { control: "text", description: "Unit for the value" },
    sub1: { control: "text", description: "First subtitle" },
    sub2: { control: "text", description: "Second subtitle" },
  },
} as Meta<typeof Tile>;

// Template for rendering the component
const Template: StoryFn<typeof Tile> = (args) => ({
  components: { Tile },
  setup() {
    return { args };
  },
  template: '<Tile v-bind="args" />',
});

// Create default story with the example props
export const Default = Template.bind({});
Default.args = {
  icon: "coinjar",
  title: "Ersparnis",
  value: "14,2",
  unit: "€",
  sub1: "gegenüber Netzbezug",
  sub2: "seit Dezember 2022",
};
