import { defineConfig } from "oxfmt";

export default defineConfig({
  // keep in sync with the `fmt` section of vite.config.ts
  printWidth: 100,
  trailingComma: "es5",
  ignorePatterns: ["tests/custom-css.css"],
  overrides: [{ files: ["**/*.vue"], options: { tabWidth: 4 } }],
});
