import { Config } from "prettier";

const config: Config = {
  printWidth: 100,
  trailingComma: "es5",
  plugins: ["prettier-plugin-sh"],
};

export default config;
