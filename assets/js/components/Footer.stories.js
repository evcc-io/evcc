import Footer from "./Footer.vue";
import i18n from "../i18n";

export default {
  title: "Main/Footer",
  component: Footer,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Footer },
  template: '<Footer v-bind="$props"></Footer>',
});

export const KeinUpdate = Template.bind({});
KeinUpdate.args = {
  version: { installed: "0.40" },
  savings: {
    since: 82800,
    chargedTotal: 15231,
    chargedSelfConsumption: 12231,
    selfPercentage: 80.3,
  },
};

export const Sponsor = Template.bind({});
Sponsor.args = {
  version: {
    installed: "0.36",
  },
  savings: {
    since: 82800,
    chargedTotal: 21000,
    chargedSelfConsumption: 12000,
    selfPercentage: 54,
  },
  sponsor: "naltatis",
};

export const UpdateVerfuegbar = Template.bind({});
UpdateVerfuegbar.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    releaseNotes: "Lorem ipsum dolor sit amet consectetur",
  },
  savings: {
    since: 82800,
    chargedTotal: 15231,
    chargedSelfConsumption: 15000,
    selfPercentage: 74,
  },
};

export const Updater = Template.bind({});
Updater.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    releaseNotes: "Lorem ipsum dolor sit amet consectetur",
    hasUpdater: true,
  },
  savings: {
    since: 82800,
    chargedTotal: 0,
    chargedSelfConsumption: 0,
    selfPercentage: 0,
  },
};

export const Upgrade = Template.bind({});
Upgrade.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    hasUpdater: true,
  },
  savings: {
    since: 82800,
    chargedTotal: 0,
    chargedSelfConsumption: 0,
    selfPercentage: 0,
  },
};

export const Savings = Template.bind({});
Savings.args = {
  version: {
    installed: "0.36",
    available: "0.40",
    hasUpdater: true,
  },
  savings: {
    since: 82800,
    chargedTotal: 15231,
    chargedSelfConsumption: 12231,
    selfPercentage: 80.3,
  },
};
