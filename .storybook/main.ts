import { StorybookConfig } from "@storybook/vue3-vite";

export default {
  stories: ["../assets/js/**/*.stories.@(js|ts)"],
  addons: [],
  framework: {
    name: "@storybook/vue3-vite",
    options: {},
  },
  core: {
    disableTelemetry: true,
  },
} satisfies StorybookConfig;
