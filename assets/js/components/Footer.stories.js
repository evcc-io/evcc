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
    totalCharged: 15231,
    selfConsumptionCharged: 12231,
    selfConsumptionPercent: 80.3,
  },
};

export const Sponsor = Template.bind({});
Sponsor.args = {
  version: {
    installed: "0.36",
  },
  savings: {
    since: 82800,
    totalCharged: 21000,
    selfConsumptionCharged: 12000,
    selfConsumptionPercent: 54,
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
    totalCharged: 15231,
    selfConsumptionCharged: 15000,
    selfConsumptionPercent: 74,
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
    totalCharged: 0,
    selfConsumptionCharged: 0,
    selfConsumptionPercent: 0,
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
    totalCharged: 0,
    selfConsumptionCharged: 0,
    selfConsumptionPercent: 0,
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
    totalCharged: 15231,
    selfConsumptionCharged: 12231,
    selfConsumptionPercent: 80.3,
  },
};
