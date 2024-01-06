import { defineConfig } from "histoire";
import { HstVue } from "@histoire/plugin-vue";

export default defineConfig({
  plugins: [HstVue()],
  setupFile: "./histoire.setup.js",
  viteNodeInlineDeps: [/!axios/],
  routerMode: "hash",
  backgroundPresets: [
    {
      label: "default",
      color: "var(--evcc-background)",
      contrastColor: "var(--evcc-default-text)",
    },
    {
      label: "box",
      color: "var(--evcc-box)",
      contrastColor: "var(--evcc-default-text)",
    },
  ],
});
