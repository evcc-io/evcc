import { defineConfig } from "vite-plus";
import vue from "@vitejs/plugin-vue";
import api from "./api";

export default defineConfig({
  plugins: [vue(), api()],
  preview: { port: 7072, host: true },
});
