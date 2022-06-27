module.exports = {
  stories: ["../assets/js/**/*.stories.js"],
  addons: [
    "@storybook/addon-links",
    "@storybook/addon-essentials",
    "@storybook/addon-interactions",
  ],
  framework: "@storybook/vue3",
  core: {
    builder: "@storybook/builder-vite",
  },
  staticDirs: ["../assets"],
  features: {
    storyStoreV7: true,
  },
};
