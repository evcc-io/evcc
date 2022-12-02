import { defineConfig } from "vite";
import vuePlugin from "@vitejs/plugin-vue";
import { viteStaticCopy } from "vite-plugin-static-copy";
import { ViteToml } from "vite-plugin-toml";
//import analyze from "rollup-plugin-analyzer";

export default defineConfig({
  root: "./assets",
  publicDir: "public",
  base: "./",
  build: {
    outDir: "../dist/",
    emptyOutDir: true,
    assetsInlineLimit: 1024,
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
    viteStaticCopy({
      targets: [{ src: "i18n/*.toml", dest: "./assets/i18n" }],
    }),
    //analyze(),
  ],
});
