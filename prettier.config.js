/** @type {import("prettier").Config} */
export default {
  printWidth: 100,
  trailingComma: "es5",
  plugins: ["prettier-plugin-sh"],
  // mirror .editorconfig for editors that resolve prettier config without it (e.g. Zed)
  overrides: [
    { files: ["*.vue", "*.sh"], options: { useTabs: true, tabWidth: 4 } },
    { files: "*.css", options: { useTabs: true, tabWidth: 2 } },
  ],
};
