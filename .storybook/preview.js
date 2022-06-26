import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";
import smoothscroll from "smoothscroll-polyfill";
import { app } from "@storybook/vue3";
import i18n from "../assets/js/i18n";
import "../assets/css/app.css";

smoothscroll.polyfill();
app.use(i18n);

export const parameters = {
  actions: { argTypesRegex: "^on[A-Z].*" },
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
  backgrounds: {
    default: "background",
    values: [
      {
        name: "background",
        value: "var(--evcc-background)",
      },
      {
        name: "box",
        value: "var(--evcc-box)",
      },
    ],
  },
};
