import {mergeConfig} from "vite";
import {defineConfig} from "vitest/config";
import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: "happy-dom",
      coverage: {
        reporters: ['default', "json", "json-summary"],
        enabled: true,
        reportOnFailure: true
      }
    },
  })
);
