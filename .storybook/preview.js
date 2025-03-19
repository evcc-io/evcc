import { setup } from "@storybook/vue3";
import "bootstrap/dist/css/bootstrap.min.css";
import smoothscroll from "smoothscroll-polyfill";
import setupI18n from "../assets/js/i18n";
import "../assets/css/app.css";
import { watchThemeChanges } from "../assets/js/theme";

smoothscroll.polyfill();
watchThemeChanges();

// Setup global parameters
/** @type { import('@storybook/vue3').Preview } */
const preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/,
      },
    },
    backgrounds: {
      default: "default",
      values: [
        {
          name: "default",
          value: "var(--evcc-background)",
        },
        {
          name: "box",
          value: "var(--evcc-box)",
        },
      ],
    },
  },
};

setup((app) => {
  app.config.globalProperties.$hiddenFeatures = () => true;
  app.use(setupI18n());
});

export default preview;
