import AuthProvidersCard from "./AuthProvidersCard.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Config/AuthProvidersCard",
  component: AuthProvidersCard,
  parameters: {
    layout: "centered",
  },
} as Meta<typeof AuthProvidersCard>;

// narrow: column width in the config device grid
const Narrow: StoryFn<typeof AuthProvidersCard> = (args) => ({
  components: { AuthProvidersCard },
  setup() {
    return { args };
  },
  template: '<div style="width: 320px"><AuthProvidersCard v-bind="args" /></div>',
});

export const Mixed = Narrow.bind({});
Mixed.args = {
  providers: {
    "Tesla (Model Y)": { id: "tesla", authenticated: true },
    "Home Assistant (homeassistant.local)": { id: "ha", authenticated: false },
  },
};

export const LongTitles = Narrow.bind({});
LongTitles.args = {
  providers: {
    "Home Assistant (https://homeassistant.example.duckdns.org)": {
      id: "ha1",
      authenticated: true,
    },
    "Home Assistant (https://homeassistant.another-very-long-domain.duckdns.org)": {
      id: "ha2",
      authenticated: false,
    },
  },
};
