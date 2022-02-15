const { loadConfigFromFile } = require("vite");
const path = require("path");

module.exports = {
  stories: ["../assets/js/**/*.stories.mdx", "../assets/js/**/*.stories.@(js|jsx|ts|tsx)"],
  addons: ["@storybook/addon-links", "@storybook/addon-essentials", "@storybook/addon-postcss"],
  framework: "@storybook/vue3",
  core: {
    builder: "storybook-builder-vite",
  },
  staticDirs: ["../assets", "../node_modules/bootstrap/dist"],
  async viteFinal(config, b) {
    const { config: userConfig } = await loadConfigFromFile(
      path.resolve(__dirname, "../vite.config.ts")
    );
    const vuePlugin = userConfig.plugins.find((p) => p.name === "vite:vue");

    const vuePluginIndex = config.plugins.findIndex((p) => p.name === "vite:vue");
    config.plugins[vuePluginIndex] = vuePlugin;

    return config;
  },
};
