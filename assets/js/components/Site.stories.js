import Site from "./Site.vue";

export default {
  title: "Main/Site",
  component: Site,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Site },
  template: '<Site v-bind="$props"></Site>',
});

export const Base = Template.bind({});
Base.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [],
};

export const Single = Template.bind({});
Single.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      socLevels: [20, 50, 80, 100],
    },
  ],
};

export const Multi = Template.bind({});
Multi.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      socLevels: [20, 50, 80, 100],
    },
    {
      title: "Ladepunkt 2",
    },
  ],
};
