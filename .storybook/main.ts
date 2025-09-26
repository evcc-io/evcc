import { StorybookConfig } from "@storybook/vue3-vite";

export default {
  stories: ["../assets/js/**/*.stories.@(js|ts)"],
  addons: ["@chromatic-com/storybook"],
  framework: {
    name: "@storybook/vue3-vite",
    options: {},
  },
} satisfies StorybookConfig;
