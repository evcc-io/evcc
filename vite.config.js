import { defineConfig } from "vite";
import vuePlugin from "@vitejs/plugin-vue";
import { ViteToml } from "vite-plugin-toml";
import legacy from "@vitejs/plugin-legacy";
import { visualizer } from "rollup-plugin-visualizer";

export default defineConfig({
  root: "./assets",
  publicDir: "public",
  base: "./",
  build: {
    outDir: "../dist/",
    emptyOutDir: true,
    assetsInlineLimit: 1024,
    chunkSizeWarningLimit: 800, // legacy build increases file size
  },
  server: {
    port: 7071,
    proxy: {
      "/api": "http://localhost:7070",
      "/i18n": "http://localhost:7070",
      "/ws": { target: "ws://localhost:7070", ws: true },
    },
  },
  plugins: [
    legacy({
      targets: ["defaults", "iOS >= 14"],
    }),
    vuePlugin({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag.startsWith("shopicon-"),
        },
      },
    }),
    ViteToml(),
    visualizer({ filename: "asset-stats.html" }),
  ],
});
