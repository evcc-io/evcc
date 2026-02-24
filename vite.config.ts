import { defineConfig } from "vite";
import vuePlugin from "@vitejs/plugin-vue";
import legacy from "@vitejs/plugin-legacy";
import { visualizer } from "rollup-plugin-visualizer";
import path from "path";

export default defineConfig({
  root: "./assets",
  publicDir: "public",
  base: "./",
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./assets/js"),
    },
  },
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
      "/providerauth": "http://localhost:7070",
      "/ws": { target: "ws://localhost:7070", ws: true },
    },
  },
  plugins: [
    legacy({
      targets: ["defaults", "iOS >= 14"],
      modernPolyfills: ["es.promise.all-settled"],
    }),
    vuePlugin({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag.startsWith("shopicon-"),
        },
      },
    }),
    visualizer({ filename: "asset-stats.html" }),
  ],
});
