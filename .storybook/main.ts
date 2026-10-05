import { StorybookConfig } from "@storybook/vue3-vite";
import { createReadStream } from "node:fs";

// serve test logos so the custom branding stories can render them
const customLogos = {
  "/custom-logo-light": "tests/custom-logo-light.svg",
  "/custom-logo-dark": "tests/custom-logo-dark.svg",
};

export default {
  stories: ["../assets/js/**/*.stories.@(js|ts)"],
  addons: [],
  framework: {
    name: "@storybook/vue3-vite",
    options: {},
  },
  core: {
    disableTelemetry: true,
  },
  viteFinal: (config) => ({
    ...config,
    plugins: [
      ...(config.plugins ?? []),
      {
        name: "evcc-custom-logos",
        configureServer(server) {
          server.middlewares.use((req, res, next) => {
            const file = customLogos[req.url?.split("?")[0] as keyof typeof customLogos];
            if (!file) return next();
            res.setHeader("Content-Type", "image/svg+xml");
            createReadStream(file).pipe(res);
          });
        },
      },
    ],
  }),
} satisfies StorybookConfig;
