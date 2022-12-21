import { defineConfig } from "vite";
import vuePlugin from "@vitejs/plugin-vue";
import { ViteToml } from "vite-plugin-toml";
import { visualizer } from "rollup-plugin-visualizer";

export default defineConfig({
  root: "./assets",
  publicDir: "public",
  base: "./",
  build: {
    outDir: "../dist/",
    emptyOutDir: true,
    assetsInlineLimit: 1024,
    chunkSizeWarningLimit: 550,
    rollupOptions: {
      output: {
        manualChunks: undefined,
      },
    },
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
