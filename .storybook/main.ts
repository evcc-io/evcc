import { StorybookConfig } from "@storybook/vue3-vite";

const config: StorybookConfig = {
  stories: ["../assets/js/**/*.stories.@(js|ts)"],
  addons: [
    "@chromatic-com/storybook",
  ],
  framework: {
    name: "@storybook/vue3-vite",
    options: {},
  },
};
export default config;
