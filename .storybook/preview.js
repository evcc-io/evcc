import { app } from "@storybook/vue3";
import i18n from "../assets/js/i18n";

app.use(i18n);

export const parameters = {
  actions: { argTypesRegex: "^on[A-Z].*" },
  backgrounds: {
    default: "dark",
    values: [
      {
        name: "light",
        value: "#ffffff",
      },
      {
        name: "dark",
        value: "#28293e",
      },
    ],
  },
};
