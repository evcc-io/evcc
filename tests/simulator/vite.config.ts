import { defineConfig } from "vite-plus";
import vue from "@vitejs/plugin-vue";
import api from "./api";

export default defineConfig({
  plugins: [vue(), api()],
  preview: { port: 7072, host: true },
  // no lightningcss pass: it does not know the @custom-media rules in app.css
  build: { cssMinify: false },
});
