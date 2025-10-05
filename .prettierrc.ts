import { type Config } from "prettier";

export default {
  printWidth: 100,
  trailingComma: "es5",
  plugins: ["prettier-plugin-sh"],
} satisfies Config;
