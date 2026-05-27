import { Config } from "@jest/types";

export default {
  preset: "@vue/cli-plugin-unit-jest/presets/no-babel",
  testMatch: ["**/*.spec.ts"],
} satisfies Config.InitialOptions;
