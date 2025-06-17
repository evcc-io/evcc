import { StorybookConfig } from "@storybook/vue3-vite";

const config: StorybookConfig = {
  stories: ["../assets/js/**/*.stories.@(js|ts)"],
  addons: [
    "@storybook/addon-essentials",
    "@chromatic-com/storybook",
    "@storybook/addon-interactions",
  ],
  framework: {
    name: "@storybook/vue3-vite",
    options: {},
  },
};
export default config;
