import Footer from "./Footer.vue";

export default {
  title: "Main/Footer",
  component: Footer,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Footer },
  template: '<Footer v-bind="$props"></Footer>',
});

export const KeinUpdate = Template.bind({});
KeinUpdate.args = {
  version: { installed: "0.40" },
};

export const UpdateVerfuegbar = Template.bind({});
UpdateVerfuegbar.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    releaseNotes: "Lorem ipsum dolor sit amet consectetur",
  },
};

export const Sponsor = Template.bind({});
Sponsor.args = {
  version: {
    installed: "0.36",
    available: "0.40",
  },
  sponsor: "naltatis",
};

export const Updater = Template.bind({});
Updater.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    releaseNotes: "Lorem ipsum dolor sit amet consectetur",
    hasUpdater: true,
  },
};

export const Upgrade = Template.bind({});
Upgrade.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    hasUpdater: true,
  },
};
