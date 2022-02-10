module.exports = {
  stories: ["../assets/js/**/*.stories.mdx", "../assets/js/**/*.stories.@(js|jsx|ts|tsx)"],
  addons: ["@storybook/addon-links", "@storybook/addon-essentials", "@storybook/addon-postcss"],
  framework: "@storybook/vue3",
  core: {
    builder: "storybook-builder-vite",
  },
};
