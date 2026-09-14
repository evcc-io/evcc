import { defineConfig } from "oxfmt";

export default defineConfig({
  printWidth: 100,
  trailingComma: "es5",
  ignorePatterns: ["tests/custom-css.css"],
  overrides: [{ files: ["**/*.vue"], options: { tabWidth: 4 } }],
});
