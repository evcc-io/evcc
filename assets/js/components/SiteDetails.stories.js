import SiteDetails from "./SiteDetails.vue";

export default {
  title: "Main/SiteDetails",
  component: SiteDetails,
  argTypes: {
    gridConfigured: { control: { type: "boolean" } },
    pvConfigured: { control: { type: "boolean" } },
    batteryConfigured: { control: { type: "boolean" } },
  },
};

const Template = (args, { argTypes }) => ({
  props: ["state"],
  components: { SiteDetails },
  template: '<SiteDetails v-bind="$props"></SiteDetails>',
});

export const Base = Template.bind({});
Base.args = {};
// CaptionAndPV.args = { caption: true, pv: true, mode: "pv" };

// export const Minimal = Template.bind({});
// Minimal.args = {};
