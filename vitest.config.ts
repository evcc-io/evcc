import { mergeConfig } from "vite-plus";
import { defineConfig } from "vite-plus";
import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: "happy-dom",
    },
  })
);
