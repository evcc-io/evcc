/** @type { import('@storybook/vue3-vite').StorybookConfig } */
const config = {
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
