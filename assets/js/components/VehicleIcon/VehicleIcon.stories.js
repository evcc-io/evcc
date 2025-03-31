import VehicleIcon from "./VehicleIcon.vue";

const icons = [
  "car",
  "bike",
  "bus",
  "moped",
  "motorcycle",
  "rocket",
  "scooter",
  "taxi",
  "tractor",
  "rickshaw",
  "shuttle",
  "van",
  "airpurifier",
  "battery",
  "bulb",
  "climate",
  "coffeemaker",
  "compute",
  "cooking",
  "cooler",
  "desktop",
  "device",
  "dishwasher",
  "dryer",
  "floorlamp",
  "generic",
  "heater",
  "heatexchange",
  "heatpump",
  "kettle",
  "laundry",
  "laundry2",
  "machine",
  "meter",
  "smartconsumer",
  "microwave",
  "pump",
  "tool",
  "waterheater",
];

export default {
  title: "VehicleIcon/VehicleIcon",
  component: VehicleIcon,
  argTypes: {
    name: {
      control: "select",
      options: icons,
    },
    names: { control: "object" },
    size: {
      control: "select",
      options: ["sm", "md", "lg", "xl"],
      defaultValue: "xl",
    },
  },
};

const Template = (args) => ({
  components: { VehicleIcon },
  setup() {
    return { args };
  },
  template: '<VehicleIcon v-bind="args" />',
});

// Single icon story that can be controlled via args
export const SingleIcon = Template.bind({});
SingleIcon.args = {
  name: "car",
  size: "xl",
};

// Multiple icons stories
export const TwoCars = Template.bind({});
TwoCars.args = {
  names: ["car", "car"],
  size: "xl",
};

export const CarAndThreeBikes = Template.bind({});
CarAndThreeBikes.args = {
  names: ["car", "bike", "bike", "bike"],
  size: "xl",
};

// Story showing all icons at once
export const AllIcons = () => ({
  components: { VehicleIcon },
  setup() {
    return { icons };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 30px;">
      <div v-for="icon in icons" :key="icon" style="display: flex; flex-direction: column; align-items: center; gap: 10px;">
        <VehicleIcon :name="icon" size="xl" />
        <small>{{ icon }}</small>
      </div>
    </div>
  `,
});
