import { defineConfig } from "oxfmt";

export default defineConfig({
  ignorePatterns: ["tests/custom-css.css"],
  overrides: [{ files: ["**/*.vue"], options: { tabWidth: 4 } }],
});
