import Tile from "./Tile.vue";

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
};

// Template for rendering the component
const Template = (args) => ({
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
