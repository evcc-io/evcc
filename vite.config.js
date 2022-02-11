import { defineConfig } from "vite";
import vuePlugin from "@vitejs/plugin-vue";

export default defineConfig({
  root: "./assets",
  publicDir: "public",
  base: "./",
  build: {
    outDir: "../dist/",
    emptyOutDir: true,
    assetsInlineLimit: 1024,
  },
  server: {
    port: 7071,
    proxy: {
      "/api": "http://localhost:7070",
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
  ],
});
