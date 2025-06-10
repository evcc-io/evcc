import { Config } from "@jest/types";

const config: Config.InitialOptions = {
  preset: "@vue/cli-plugin-unit-jest/presets/no-babel",
  testMatch: ["**/*.spec.js"],
};

export default config;